package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/e6qu/sharecrop/internal/agent"
	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/ledger"
	"github.com/e6qu/sharecrop/internal/submission"
	"github.com/e6qu/sharecrop/internal/task"
)

func TestInitializeReportsProtocolVersion(t *testing.T) {
	server := NewServer(fakeServices{})
	response := server.Handle(context.Background(), testSubject(t), agent.Credential{Scopes: allScopes()}, request(`1`, "initialize", `{}`))
	if response.Error != nil {
		t.Fatalf("initialize error: %s", response.Error.Message)
	}
	var result struct {
		ProtocolVersion string `json:"protocolVersion"`
	}
	if err := json.Unmarshal(mustResult(t, response), &result); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if result.ProtocolVersion != protocolVersion {
		t.Fatalf("protocol version = %q, want %q", result.ProtocolVersion, protocolVersion)
	}
}

func TestToolsListReturnsAllTools(t *testing.T) {
	server := NewServer(fakeServices{})
	response := server.Handle(context.Background(), testSubject(t), agent.Credential{Scopes: allScopes()}, request(`1`, "tools/list", `{}`))
	var result struct {
		Tools []struct {
			Name        string          `json:"name"`
			InputSchema json.RawMessage `json:"inputSchema"`
		} `json:"tools"`
	}
	if err := json.Unmarshal(mustResult(t, response), &result); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if len(result.Tools) != len(ToolNames()) {
		t.Fatalf("tool count = %d, want %d", len(result.Tools), len(ToolNames()))
	}
	if len(result.Tools[0].InputSchema) == 0 {
		t.Fatalf("tool input schema is empty")
	}
}

func TestUnknownMethodReturnsMethodNotFound(t *testing.T) {
	server := NewServer(fakeServices{})
	response := server.Handle(context.Background(), testSubject(t), agent.Credential{Scopes: allScopes()}, request(`1`, "resources/list", `{}`))
	if response.Error == nil || response.Error.Code != codeMethodNotFound {
		t.Fatalf("expected method-not-found error, got %+v", response.Error)
	}
}

func TestToolsCallEnforcesScope(t *testing.T) {
	server := NewServer(fakeServices{})
	onlyRead := agent.Credential{Scopes: agent.NewScopeSet([]agent.Scope{agent.ScopeTasksRead})}
	response := server.Handle(context.Background(), testSubject(t), onlyRead, request(`1`, "tools/call", `{"name":"sharecrop.submit_response","arguments":{"task_id":"`+testTaskID(t)+`","response_json":"{}"}}`))
	if response.Error == nil || response.Error.Code != codeScopeDenied {
		t.Fatalf("expected scope-denied error, got %+v", response.Error)
	}
}

func TestToolsCallRejectsTaskScopedCredentialForADifferentTask(t *testing.T) {
	server := NewServer(fakeServices{})
	scopedTaskID := testTaskID(t)
	credential := agent.Credential{Scopes: allScopes(), TaskID: taskIDPointer(t, scopedTaskID)}

	// Its own task: allowed.
	own := server.Handle(context.Background(), testSubject(t), credential, request(`1`, "tools/call", `{"name":"sharecrop.get_task","arguments":{"task_id":"`+scopedTaskID+`"}}`))
	if own.Error != nil {
		t.Fatalf("expected the task-scoped credential to work against its own task, got %+v", own.Error)
	}

	// A different task: rejected, not silently passed through to the service.
	otherTaskID := testTaskID(t)
	other := server.Handle(context.Background(), testSubject(t), credential, request(`2`, "tools/call", `{"name":"sharecrop.get_task","arguments":{"task_id":"`+otherTaskID+`"}}`))
	if other.Error == nil || other.Error.Code != codeScopeDenied {
		t.Fatalf("expected scope-denied error for a different task, got %+v", other.Error)
	}
}

func taskIDPointer(t *testing.T, raw string) *core.TaskID {
	t.Helper()
	parsed, matched := core.ParseTaskID(raw).(core.TaskIDCreated)
	if !matched {
		t.Fatalf("parse task id: %q", raw)
	}
	return &parsed.Value
}

func TestToolsCallListTasks(t *testing.T) {
	server := NewServer(fakeServices{})
	response := server.Handle(context.Background(), testSubject(t), agent.Credential{Scopes: allScopes()}, request(`1`, "tools/call", `{"name":"sharecrop.list_tasks","arguments":{"scope":"public"}}`))
	content := decodeToolText(t, response)
	if !strings.Contains(content, "\"tasks\"") {
		t.Fatalf("list tasks content missing tasks key: %s", content)
	}
}

func TestToolsCallReserveTask(t *testing.T) {
	server := NewServer(fakeServices{})
	response := server.Handle(context.Background(), testSubject(t), agent.Credential{Scopes: allScopes()}, request(`1`, "tools/call", `{"name":"sharecrop.reserve_task","arguments":{"task_id":"`+testTaskID(t)+`"}}`))
	content := decodeToolText(t, response)
	if !strings.Contains(content, "\"reservation\"") {
		t.Fatalf("reserve content missing reservation key: %s", content)
	}
	if !strings.Contains(content, "\"state\":\"active\"") {
		t.Fatalf("reserve content missing active state: %s", content)
	}
}

func TestToolsCallReserveTaskForOrganizationTeam(t *testing.T) {
	server := NewServer(fakeServices{})
	organizationID := testOrganizationID(t)
	teamID := testTeamID(t)
	response := server.Handle(context.Background(), testSubject(t), agent.Credential{Scopes: allScopes()}, request(`1`, "tools/call", `{"name":"sharecrop.reserve_task","arguments":{"task_id":"`+testTaskID(t)+`","assignee_kind":"organization_team","organization_id":"`+organizationID+`","team_id":"`+teamID+`"}}`))
	content := decodeToolText(t, response)
	if !strings.Contains(content, "\"assignee_kind\":\"organization_team\"") {
		t.Fatalf("reserve content missing organization team assignee: %s", content)
	}
	if !strings.Contains(content, "\"assignee_id\":\""+teamID+"\"") {
		t.Fatalf("reserve content missing team assignee id: %s", content)
	}
}

func TestToolsCallSubmitResponseReturnsReceipt(t *testing.T) {
	server := NewServer(fakeServices{})
	response := server.Handle(context.Background(), testSubject(t), agent.Credential{Scopes: allScopes()}, request(`1`, "tools/call", `{"name":"sharecrop.submit_response","arguments":{"task_id":"`+testTaskID(t)+`","response_json":"{\"answer\":\"done\"}"}}`))
	content := decodeToolText(t, response)
	if !strings.Contains(content, "receipt_token") {
		t.Fatalf("submit content missing receipt token: %s", content)
	}
}

func TestToolsCallRejectsUnknownTool(t *testing.T) {
	server := NewServer(fakeServices{})
	response := server.Handle(context.Background(), testSubject(t), agent.Credential{Scopes: allScopes()}, request(`1`, "tools/call", `{"name":"sharecrop.delete_everything","arguments":{}}`))
	if response.Error == nil || response.Error.Code != codeInvalidParams {
		t.Fatalf("expected invalid-params error, got %+v", response.Error)
	}
}

func TestToolsCallSurfacesDomainRejectionAsToolError(t *testing.T) {
	server := NewServer(fakeServices{rejectGet: true})
	response := server.Handle(context.Background(), testSubject(t), agent.Credential{Scopes: allScopes()}, request(`1`, "tools/call", `{"name":"sharecrop.get_task","arguments":{"task_id":"`+testTaskID(t)+`"}}`))
	if response.Error != nil {
		t.Fatalf("expected tool result, got protocol error: %s", response.Error.Message)
	}
	var result toolCallResult
	if err := json.Unmarshal(mustResult(t, response), &result); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if !result.IsError {
		t.Fatalf("expected isError tool result")
	}
}

type fakeServices struct {
	rejectGet bool
}

func (services fakeServices) ListTasks(_ context.Context, _ auth.UserSubject, _ task.ListScope, _ task.ListFilters) task.ListResult {
	return task.TasksListed{Values: []task.ListItem{}}
}

func (services fakeServices) GetTask(_ context.Context, subject auth.UserSubject, taskID core.TaskID) task.GetResult {
	if services.rejectGet {
		return task.GetRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "task view access denied")}
	}
	return task.TaskGot{Value: task.Task{
		ID:         taskID,
		Owner:      task.UserOwner{UserID: subject.ID},
		Visibility: task.UserVisibility{UserID: subject.ID},
		State:      task.StateOpen,
		CreatedBy:  subject.ID,
	}}
}

func (services fakeServices) CreateTask(_ context.Context, command task.CreateCommand) task.CreateResult {
	created := core.NewTaskID().(core.TaskIDCreated)
	return task.TaskCreated{Value: task.Task{
		ID:             created.Value,
		Owner:          command.Owner,
		Title:          command.Title,
		Description:    command.Description,
		Reward:         command.Reward,
		State:          task.StateDraft,
		Visibility:     command.Visibility,
		Placement:      command.Placement,
		ResponseSchema: command.ResponseSchema,
		Payload:        command.Payload,
		CreatedBy:      command.Actor.ID,
	}}
}

func (services fakeServices) OpenTask(_ context.Context, subject auth.UserSubject, taskID core.TaskID) task.ChangeStateResult {
	return task.TaskStateChanged{Value: task.Task{
		ID:         taskID,
		Owner:      task.UserOwner{UserID: subject.ID},
		Visibility: task.UserVisibility{UserID: subject.ID},
		State:      task.StateOpen,
		CreatedBy:  subject.ID,
	}}
}

func (services fakeServices) FundTask(_ context.Context, funder core.UserID, taskID core.TaskID, amount ledger.CreditAmount, _ ledger.IdempotencyKey) ledger.FundResult {
	return ledger.TaskFunded{Escrow: ledger.TaskEscrow{TaskID: taskID, Amount: amount, State: ledger.EscrowStateHeld}}
}

func (services fakeServices) SubmitResponse(_ context.Context, command submission.SubmitCommand) submission.SubmitResult {
	submissionID := core.NewSubmissionID().(core.SubmissionIDCreated)
	token := submission.NewReceiptTokenPlain().(submission.ReceiptTokenPlainAccepted)
	return submission.SubmissionCreated{
		Value: submission.Submission{
			ID:             submissionID.Value,
			TaskID:         command.TaskID,
			SubmitterID:    command.SubmitterID,
			State:          submission.StateSubmitted,
			ResponseSource: command.ResponseSource,
			Validation:     submission.ValidationPassed{},
		},
		ReceiptToken: token.Value,
	}
}

func (services fakeServices) GetSubmissionStatus(_ context.Context, _ submission.ReceiptTokenPlain) submission.ReceiptStatusResult {
	return submission.ReceiptStatusRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "unused")}
}

func (services fakeServices) ListTaskSubmissions(_ context.Context, _ auth.UserSubject, _ core.TaskID) submission.ListResult {
	return submission.SubmissionsListed{Values: []submission.Submission{}}
}

func (services fakeServices) AcceptSubmission(_ context.Context, _ core.UserID, taskID core.TaskID, submissionID core.SubmissionID, _ ledger.IdempotencyKey) ledger.AcceptResult {
	return ledger.SubmissionAccepted{TaskID: taskID, SubmissionID: submissionID, Payout: ledger.NoPayout{}, Tip: ledger.NoTip{}}
}

func (services fakeServices) ReviewAcceptSubmission(_ context.Context, _ core.UserID, taskID core.TaskID, submissionID core.SubmissionID, _ ledger.IdempotencyKey, _ ledger.CreditReviewSelection, _ ledger.TipSelection, _ ledger.CollectibleTipSelection) ledger.AcceptResult {
	return ledger.SubmissionAccepted{TaskID: taskID, SubmissionID: submissionID, Payout: ledger.NoPayout{}, Tip: ledger.NoTip{}}
}

func (services fakeServices) RequestChanges(_ context.Context, _ core.UserID, taskID core.TaskID, submissionID core.SubmissionID, note submission.ReviewNote) ledger.RequestChangesResult {
	return ledger.ChangesRequested{TaskID: taskID, SubmissionID: submissionID, ReviewNote: note.String()}
}

func (services fakeServices) RejectSubmission(_ context.Context, _ core.UserID, taskID core.TaskID, submissionID core.SubmissionID, _ ledger.IdempotencyKey, _ submission.ReviewNote, _ ledger.CreditReviewSelection, _ ledger.TipSelection, _ ledger.BanSelection) ledger.RejectResult {
	return ledger.SubmissionRejected{TaskID: taskID, SubmissionID: submissionID, Payout: ledger.NoPayout{}, Tip: ledger.NoTip{}}
}

func (services fakeServices) ListSeries(_ context.Context, _ auth.UserSubject) task.ListSeriesResult {
	return task.SeriesListed{Values: []task.Series{}}
}

func (services fakeServices) GetSeries(_ context.Context, _ auth.UserSubject, _ core.TaskSeriesID) task.GetSeriesResult {
	return task.GetSeriesRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "unused")}
}

func (services fakeServices) CreateSeries(_ context.Context, subject auth.UserSubject, title task.SeriesTitle, description task.SeriesDescription) task.SeriesMutationResult {
	seriesID := core.NewTaskSeriesID().(core.TaskSeriesIDCreated)
	return task.SeriesMutated{Value: task.SeriesDetail{Series: task.Series{
		ID:          seriesID.Value,
		Owner:       task.UserOwner{UserID: subject.ID},
		Title:       title,
		Description: description,
		State:       task.SeriesStateDraft,
		CreatedBy:   subject.ID,
	}}}
}

func (services fakeServices) ChangeSeriesState(_ context.Context, _ auth.UserSubject, _ core.TaskSeriesID, _ task.SeriesStateTransition) task.SeriesMutationResult {
	return task.SeriesMutationRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "unused")}
}

func (services fakeServices) AddTaskToSeries(_ context.Context, _ auth.UserSubject, _ core.TaskSeriesID, _ core.TaskID) task.SeriesMutationResult {
	return task.SeriesMutationRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "unused")}
}

func (services fakeServices) RemoveTaskFromSeries(_ context.Context, _ auth.UserSubject, _ core.TaskSeriesID, _ core.TaskID) task.SeriesMutationResult {
	return task.SeriesMutationRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "unused")}
}

func (services fakeServices) AddSeriesComment(_ context.Context, subject auth.UserSubject, seriesID core.TaskSeriesID, body task.CommentBody) task.SeriesCommentResult {
	commentID := core.NewSeriesCommentID().(core.SeriesCommentIDCreated)
	return task.SeriesCommentAdded{Value: task.SeriesComment{ID: commentID.Value, SeriesID: seriesID, AuthorID: subject.ID, Body: body}}
}

func (services fakeServices) ListSeriesComments(_ context.Context, _ auth.UserSubject, _ core.TaskSeriesID) task.SeriesCommentsResult {
	return task.SeriesCommentsListed{Values: nil}
}

func (services fakeServices) AddTaskComment(_ context.Context, subject auth.UserSubject, taskID core.TaskID, body task.CommentBody) task.TaskCommentResult {
	commentID := core.NewTaskCommentID().(core.TaskCommentIDCreated)
	return task.TaskCommentAdded{Value: task.TaskComment{ID: commentID.Value, TaskID: taskID, AuthorID: subject.ID, Body: body}}
}

func (services fakeServices) ListTaskComments(_ context.Context, _ auth.UserSubject, _ core.TaskID) task.TaskCommentsResult {
	return task.TaskCommentsListed{Values: nil}
}

func (services fakeServices) AddSubmissionComment(_ context.Context, subject auth.UserSubject, submissionID core.SubmissionID, body task.CommentBody) submission.SubmissionCommentResult {
	commentID := core.NewSubmissionCommentID().(core.SubmissionCommentIDCreated)
	return submission.SubmissionCommentAdded{Value: submission.SubmissionComment{ID: commentID.Value, SubmissionID: submissionID, AuthorID: subject.ID, Body: body}}
}

func (services fakeServices) ListSubmissionComments(_ context.Context, _ auth.UserSubject, _ core.SubmissionID) submission.SubmissionCommentsResult {
	return submission.SubmissionCommentsListed{Values: nil}
}

func (services fakeServices) UnpublishTask(_ context.Context, subject auth.UserSubject, taskID core.TaskID) task.ChangeStateResult {
	return task.TaskStateChanged{Value: task.Task{ID: taskID, Owner: task.UserOwner{UserID: subject.ID}, State: task.StateDraft, CreatedBy: subject.ID}}
}

func (services fakeServices) ReserveTask(_ context.Context, subject auth.UserSubject, taskID core.TaskID) task.ReservationResult {
	reservationID := core.NewTaskReservationID().(core.TaskReservationIDCreated)
	return task.ReservationCreated{Value: task.Reservation{
		ID:          reservationID.Value,
		TaskID:      taskID,
		Assignee:    task.UserAssignee{UserID: subject.ID},
		State:       task.ReservationStateActive,
		RequestedBy: subject.ID,
	}}
}

func (services fakeServices) ReserveTaskForOrganizationTeam(_ context.Context, subject auth.UserSubject, taskID core.TaskID, organizationID core.OrganizationID, teamID core.TeamID) task.ReservationResult {
	reservationID := core.NewTaskReservationID().(core.TaskReservationIDCreated)
	return task.ReservationCreated{Value: task.Reservation{
		ID:          reservationID.Value,
		TaskID:      taskID,
		Assignee:    task.OrganizationTeamAssignee{OrganizationID: organizationID, TeamID: teamID},
		State:       task.ReservationStateActive,
		RequestedBy: subject.ID,
	}}
}

func (services fakeServices) ListReservations(_ context.Context, subject auth.UserSubject, taskID core.TaskID) task.ReservationsListResult {
	reservationID := core.NewTaskReservationID().(core.TaskReservationIDCreated)
	return task.ReservationsListed{Values: []task.Reservation{{
		ID:          reservationID.Value,
		TaskID:      taskID,
		Assignee:    task.UserAssignee{UserID: subject.ID},
		State:       task.ReservationStateRequested,
		RequestedBy: subject.ID,
	}}}
}

func (services fakeServices) ApproveReservation(_ context.Context, subject auth.UserSubject, taskID core.TaskID, reservationID core.TaskReservationID) task.ReservationStateChangeResult {
	return fakeReservationStateChange(subject, taskID, reservationID, task.ReservationStateActive)
}

func (services fakeServices) DeclineReservation(_ context.Context, subject auth.UserSubject, taskID core.TaskID, reservationID core.TaskReservationID) task.ReservationStateChangeResult {
	return fakeReservationStateChange(subject, taskID, reservationID, task.ReservationStateDeclined)
}

func (services fakeServices) CancelReservation(_ context.Context, subject auth.UserSubject, taskID core.TaskID, reservationID core.TaskReservationID) task.ReservationStateChangeResult {
	return fakeReservationStateChange(subject, taskID, reservationID, task.ReservationStateCancelledByRequester)
}

func fakeReservationStateChange(subject auth.UserSubject, taskID core.TaskID, reservationID core.TaskReservationID, state task.ReservationState) task.ReservationStateChangeResult {
	return task.ReservationStateChanged{Value: task.Reservation{
		ID:          reservationID,
		TaskID:      taskID,
		Assignee:    task.UserAssignee{UserID: subject.ID},
		State:       state,
		RequestedBy: subject.ID,
	}}
}

func request(id string, method string, params string) Request {
	return Request{JSONRPC: jsonRPCVersion, ID: json.RawMessage(id), Method: method, Params: json.RawMessage(params)}
}

func mustResult(t *testing.T, response Response) json.RawMessage {
	t.Helper()
	if response.Error != nil {
		t.Fatalf("unexpected error: %s", response.Error.Message)
	}
	return response.Result
}

func decodeToolText(t *testing.T, response Response) string {
	t.Helper()
	var result toolCallResult
	if err := json.Unmarshal(mustResult(t, response), &result); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if len(result.Content) != 1 {
		t.Fatalf("content count = %d, want 1", len(result.Content))
	}
	return result.Content[0].Text
}

func allScopes() agent.ScopeSet {
	return agent.NewScopeSet([]agent.Scope{
		agent.ScopeTasksRead,
		agent.ScopeTasksWrite,
		agent.ScopeSubmissionsWrite,
		agent.ScopeSubmissionsRead,
		agent.ScopeSubmissionsReview,
	})
}

func testSubject(t *testing.T) auth.UserSubject {
	t.Helper()
	created, matched := core.NewUserID().(core.UserIDCreated)
	if !matched {
		t.Fatalf("user id rejected")
	}
	return auth.UserSubject{ID: created.Value}
}

func testTaskID(t *testing.T) string {
	t.Helper()
	created, matched := core.NewTaskID().(core.TaskIDCreated)
	if !matched {
		t.Fatalf("task id rejected")
	}
	return created.Value.String()
}

func testOrganizationID(t *testing.T) string {
	t.Helper()
	created, matched := core.NewOrganizationID().(core.OrganizationIDCreated)
	if !matched {
		t.Fatalf("organization id rejected")
	}
	return created.Value.String()
}

func testTeamID(t *testing.T) string {
	t.Helper()
	created, matched := core.NewTeamID().(core.TeamIDCreated)
	if !matched {
		t.Fatalf("team id rejected")
	}
	return created.Value.String()
}
