package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/e6qu/sharecrop/internal/agent"
	"github.com/e6qu/sharecrop/internal/assets"
	"github.com/e6qu/sharecrop/internal/audit"
	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/event"
	"github.com/e6qu/sharecrop/internal/ledger"
	"github.com/e6qu/sharecrop/internal/notification"
	"github.com/e6qu/sharecrop/internal/org"
	"github.com/e6qu/sharecrop/internal/orgcred"
	"github.com/e6qu/sharecrop/internal/submission"
	"github.com/e6qu/sharecrop/internal/task"
	"github.com/e6qu/sharecrop/internal/webhook"
)

// The webhook fakes run the real service over the in-memory store so tool
// tests observe genuine create/list/revoke behavior. The store is fresh per
// call because fakeServices is a value; tests that need continuity go
// through the HTTP layer instead.
func (services fakeServices) CreateWebhookSubscription(ctx context.Context, owner webhook.Owner, endpoint webhook.EndpointURL, kinds webhook.KindFilter, audience webhook.Audience) webhook.CreateResult {
	return webhook.NewService(webhook.NewMemoryStore()).Create(ctx, owner, endpoint, kinds, audience)
}

func (services fakeServices) ListWebhookSubscriptions(ctx context.Context, owner webhook.Owner, page core.Page) webhook.ListResult {
	return webhook.NewService(webhook.NewMemoryStore()).List(ctx, owner, page)
}

func (services fakeServices) RevokeWebhookSubscription(_ context.Context, _ webhook.Owner, _ core.WebhookSubscriptionID) webhook.RevokeResult {
	return webhook.RevokeRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "active webhook subscription was not found")}
}

func (services fakeServices) ListWebhookDeliveries(_ context.Context, _ webhook.Owner, _ core.WebhookSubscriptionID, _ core.Page) webhook.ListDeliveriesResult {
	return webhook.DeliveriesListed{Values: []webhook.Delivery{}}
}

func TestInitializeReportsProtocolVersion(t *testing.T) {
	server := NewServer(fakeServices{})
	response := server.Handle(context.Background(), testSubject(t), CallerCredential{Scopes: allScopes()}, request(`1`, "initialize", `{}`))
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

func TestToolsListReturnsAllToolsForAFullScopeCredential(t *testing.T) {
	server := NewServer(fakeServices{})
	response := server.Handle(context.Background(), testSubject(t), CallerCredential{Scopes: agent.NewScopeSet(agent.AllScopes())}, request(`1`, "tools/list", `{}`))
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

// TestToolsListIsFilteredByCredentialScopes pins the honesty of the tool
// surface: a credential only sees tools it is scope-eligible to call, so a
// worker credential is never shown grant_credits or the other
// platform_admin tools.
func TestToolsListIsFilteredByCredentialScopes(t *testing.T) {
	server := NewServer(fakeServices{})
	worker := CallerCredential{Scopes: agent.NewScopeSet([]agent.Scope{agent.ScopeTasksRead, agent.ScopeSubmissionsWrite, agent.ScopeSubmissionsRead})}
	response := server.Handle(context.Background(), testSubject(t), worker, request(`1`, "tools/list", `{}`))
	var result struct {
		Tools []struct {
			Name string `json:"name"`
		} `json:"tools"`
	}
	if err := json.Unmarshal(mustResult(t, response), &result); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	listed := make(map[string]bool, len(result.Tools))
	for _, tool := range result.Tools {
		listed[tool.Name] = true
	}
	for _, expected := range []string{toolListTasks, toolGetTask, toolSubmitResponse, toolGetSubmission, toolListTaskSubmissions} {
		if !listed[expected] {
			t.Fatalf("worker credential missing %s in tools/list", expected)
		}
	}
	for _, hidden := range []string{toolGrantCredits, toolGrantPlatformAdmin, toolListPlatformAdmins, toolAwardCollectible, toolCreateTask, toolListLedger, toolCreateWebhookSubscription} {
		if listed[hidden] {
			t.Fatalf("worker credential should not see %s in tools/list", hidden)
		}
	}

	// Calling an unlisted tool still fails the scope gate, matching the list.
	call := server.Handle(context.Background(), testSubject(t), worker, request(`2`, "tools/call", `{"name":"sharecrop.grant_credits","arguments":{}}`))
	if call.Error == nil || call.Error.Code != codeScopeDenied {
		t.Fatalf("expected scope-denied calling an unlisted tool, got %+v", call.Error)
	}
}

// TestInitializeCarriesInstructions pins the orientation text: the MCP spec
// instructions field must exist and cover the worker loop, the schema
// dialect, and the marketplace webhook channel.
func TestInitializeCarriesInstructions(t *testing.T) {
	server := NewServer(fakeServices{})
	response := server.Handle(context.Background(), testSubject(t), CallerCredential{Scopes: allScopes()}, request(`1`, "initialize", `{}`))
	var result struct {
		Instructions string `json:"instructions"`
	}
	if err := json.Unmarshal(mustResult(t, response), &result); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	for _, expected := range []string{"sharecrop.list_tasks", "sharecrop.reserve_task", "sharecrop.submit_response", "sharecrop.get_submission", "NOT JSON Schema", "marketplace", "next_offset"} {
		if !strings.Contains(result.Instructions, expected) {
			t.Fatalf("instructions missing %q:\n%s", expected, result.Instructions)
		}
	}
}

func TestUnknownMethodReturnsMethodNotFound(t *testing.T) {
	server := NewServer(fakeServices{})
	response := server.Handle(context.Background(), testSubject(t), CallerCredential{Scopes: allScopes()}, request(`1`, "resources/list", `{}`))
	if response.Error == nil || response.Error.Code != codeMethodNotFound {
		t.Fatalf("expected method-not-found error, got %+v", response.Error)
	}
}

func TestToolsCallEnforcesScope(t *testing.T) {
	server := NewServer(fakeServices{})
	onlyRead := CallerCredential{Scopes: agent.NewScopeSet([]agent.Scope{agent.ScopeTasksRead})}
	response := server.Handle(context.Background(), testSubject(t), onlyRead, request(`1`, "tools/call", `{"name":"sharecrop.submit_response","arguments":{"task_id":"`+testTaskID(t)+`","response_json":"{}"}}`))
	if response.Error == nil || response.Error.Code != codeScopeDenied {
		t.Fatalf("expected scope-denied error, got %+v", response.Error)
	}
}

func TestToolsCallRejectsTaskScopedCredentialForADifferentTask(t *testing.T) {
	server := NewServer(fakeServices{})
	scopedTaskID := testTaskID(t)
	credential := CallerCredential{Scopes: allScopes(), TaskID: taskIDPointer(t, scopedTaskID)}

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

func TestToolsCallAcceptsTaskScopedCredentialWithDifferentlyCasedTaskID(t *testing.T) {
	server := NewServer(fakeServices{})
	scopedTaskID := testTaskID(t)
	credential := CallerCredential{Scopes: allScopes(), TaskID: taskIDPointer(t, scopedTaskID)}

	// The same task ID, differently cased, must still be recognized as a
	// match — the check parses and normalizes both sides rather than doing a
	// raw string compare, so it doesn't spuriously reject a valid but
	// non-canonically-cased task ID.
	uppercased := strings.ToUpper(scopedTaskID)
	response := server.Handle(context.Background(), testSubject(t), credential, request(`1`, "tools/call", `{"name":"sharecrop.get_task","arguments":{"task_id":"`+uppercased+`"}}`))
	if response.Error != nil {
		t.Fatalf("expected a differently-cased but equal task id to be accepted, got %+v", response.Error)
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
	response := server.Handle(context.Background(), testSubject(t), CallerCredential{Scopes: allScopes()}, request(`1`, "tools/call", `{"name":"sharecrop.list_tasks","arguments":{"scope":"public"}}`))
	content := decodeToolText(t, response)
	if !strings.Contains(content, "\"tasks\"") {
		t.Fatalf("list tasks content missing tasks key: %s", content)
	}
}

func TestToolsCallReserveTask(t *testing.T) {
	server := NewServer(fakeServices{})
	response := server.Handle(context.Background(), testSubject(t), CallerCredential{Scopes: allScopes()}, request(`1`, "tools/call", `{"name":"sharecrop.reserve_task","arguments":{"task_id":"`+testTaskID(t)+`"}}`))
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
	response := server.Handle(context.Background(), testSubject(t), CallerCredential{Scopes: allScopes()}, request(`1`, "tools/call", `{"name":"sharecrop.reserve_task","arguments":{"task_id":"`+testTaskID(t)+`","assignee_kind":"organization_team","organization_id":"`+organizationID+`","team_id":"`+teamID+`"}}`))
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
	response := server.Handle(context.Background(), testSubject(t), CallerCredential{Scopes: allScopes()}, request(`1`, "tools/call", `{"name":"sharecrop.submit_response","arguments":{"task_id":"`+testTaskID(t)+`","response_json":"{\"answer\":\"done\"}"}}`))
	content := decodeToolText(t, response)
	if !strings.Contains(content, "receipt_token") {
		t.Fatalf("submit content missing receipt token: %s", content)
	}
}

func TestToolsCallRejectsUnknownTool(t *testing.T) {
	server := NewServer(fakeServices{})
	response := server.Handle(context.Background(), testSubject(t), CallerCredential{Scopes: allScopes()}, request(`1`, "tools/call", `{"name":"sharecrop.delete_everything","arguments":{}}`))
	if response.Error == nil || response.Error.Code != codeInvalidParams {
		t.Fatalf("expected invalid-params error, got %+v", response.Error)
	}
}

func TestToolsCallSurfacesDomainRejectionAsToolError(t *testing.T) {
	server := NewServer(fakeServices{rejectGet: true})
	response := server.Handle(context.Background(), testSubject(t), CallerCredential{Scopes: allScopes()}, request(`1`, "tools/call", `{"name":"sharecrop.get_task","arguments":{"task_id":"`+testTaskID(t)+`"}}`))
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
	rejectGet         bool
	isAdmin           bool
	invalidSubmission bool
	// listTaskCount is how many task rows exist for ListTasks; the fake
	// honors the requested page limit like a real store, so probe-based
	// pagination is observable in tests.
	listTaskCount int
	// eventFeed is the stored-event stream ListEvents serves.
	eventFeed []event.StoredEvent
	// catalogListings is what ListCollectibleCatalog serves.
	catalogListings []assets.CatalogListing
	// collectibles is what ListCollectibles serves.
	collectibles []assets.Collectible
}

func (services fakeServices) ListTasks(_ context.Context, _ auth.Subject, _ task.ListScope, _ task.ListFilters, page core.Page) task.ListResult {
	remaining := services.listTaskCount - page.Offset()
	if remaining < 0 {
		remaining = 0
	}
	if remaining > page.Limit() {
		remaining = page.Limit()
	}
	values := make([]task.ListItem, 0, remaining)
	for range remaining {
		created := core.NewTaskID().(core.TaskIDCreated)
		values = append(values, task.ListItem{
			Task:               task.Task{ID: created.Value, State: task.StateOpen},
			CreatorDisplayName: auth.NewDisplayName("mara").(auth.DisplayNameAccepted).Value,
			HolderDisplayName:  task.HolderNamed{DisplayName: auth.NewDisplayName("ada").(auth.DisplayNameAccepted).Value},
			PendingReviewCount: 2,
			Funded:             task.FundedStateRewardFunded,
		})
	}
	return task.TasksListed{Values: values, Total: int64(services.listTaskCount)}
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

func (services fakeServices) OpenTask(_ context.Context, subject auth.Subject, taskID core.TaskID) task.ChangeStateResult {
	userID := fakeUserID(subject)
	return task.TaskStateChanged{Value: task.Task{
		ID:         taskID,
		Owner:      task.UserOwner{UserID: userID},
		Visibility: task.UserVisibility{UserID: userID},
		State:      task.StateOpen,
		CreatedBy:  userID,
	}}
}

func (services fakeServices) CancelTask(_ context.Context, subject auth.Subject, taskID core.TaskID) task.ChangeStateResult {
	userID := fakeUserID(subject)
	return task.TaskStateChanged{Value: task.Task{
		ID:         taskID,
		Owner:      task.UserOwner{UserID: userID},
		Visibility: task.UserVisibility{UserID: userID},
		State:      task.StateCancelled,
		CreatedBy:  userID,
	}}
}

func (services fakeServices) FundTask(_ context.Context, funder core.UserID, taskID core.TaskID, amount ledger.CreditAmount, _ ledger.IdempotencyKey) ledger.FundResult {
	return ledger.TaskFunded{Fund: ledger.TaskFund{TaskID: taskID, CreditAmount: amount}}
}

func (services fakeServices) RefundTask(_ context.Context, _ core.UserID, taskID core.TaskID, _ ledger.IdempotencyKey) ledger.RefundResult {
	return ledger.TaskRefunded{Fund: ledger.TaskFund{TaskID: taskID, CreditAmount: ledger.NewCreditAmount(1).(ledger.CreditAmountAccepted).Value}}
}

// fakeUserID extracts a core.UserID from an auth.Subject for test-fixture
// construction; OrgSubject has no individual user, so it returns the zero
// value (fine for a fake — the fixture's identity fields aren't asserted on
// in the OrgSubject test cases).
func fakeUserID(subject auth.Subject) core.UserID {
	if userActor, isUser := subject.(auth.UserSubject); isUser {
		return userActor.ID
	}
	return core.UserID{}
}

func (services fakeServices) SubmitResponse(_ context.Context, command submission.SubmitCommand) submission.SubmitResult {
	submissionID := core.NewSubmissionID().(core.SubmissionIDCreated)
	token := submission.NewReceiptTokenPlain().(submission.ReceiptTokenPlainAccepted)
	state := submission.StateSubmitted
	validation := submission.ValidationOutcome(submission.ValidationPassed{})
	if services.invalidSubmission {
		state = submission.StateInvalid
		validation = submission.ValidationFailed{Errors: []submission.ValidationError{{Path: "$.answer", Message: "required field is missing"}}}
	}
	return submission.SubmissionCreated{
		Value: submission.Submission{
			ID:             submissionID.Value,
			TaskID:         command.TaskID,
			SubmitterID:    command.SubmitterID,
			State:          state,
			ResponseSource: command.ResponseSource,
			Attachments:    command.Attachments,
			Validation:     validation,
		},
		ReceiptToken: token.Value,
	}
}

func (services fakeServices) GetSubmissionStatus(_ context.Context, _ submission.ReceiptTokenPlain) submission.ReceiptStatusResult {
	return submission.ReceiptStatusRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "unused")}
}

func (services fakeServices) GetSubmission(_ context.Context, subject auth.Subject, submissionID core.SubmissionID) submission.GetResult {
	taskID := core.NewTaskID().(core.TaskIDCreated)
	submitterName := auth.NewDisplayName("ada").(auth.DisplayNameAccepted)
	return submission.SubmissionGot{Value: submission.Submission{
		ID:                   submissionID,
		TaskID:               taskID.Value,
		SubmitterID:          fakeUserID(subject),
		SubmitterDisplayName: submitterName.Value,
		State:                submission.StateSubmitted,
		ResponseSource:       submission.NewResponseSource(`{"answer":"done"}`).(submission.ResponseSourceAccepted).Value,
		Validation:           submission.ValidationPassed{},
		CreatedAt:            time.Date(2026, 6, 29, 0, 0, 0, 0, time.UTC),
	}}
}

func (services fakeServices) ListTaskSubmissions(_ context.Context, _ auth.Subject, _ core.TaskID, _ core.Page) submission.ListResult {
	return submission.SubmissionsListed{Values: []submission.Submission{}}
}

func (services fakeServices) ReviewAcceptSubmission(_ context.Context, _ ledger.Reviewer, taskID core.TaskID, submissionID core.SubmissionID, _ ledger.IdempotencyKey, _ ledger.CreditReviewSelection, _ ledger.TipSelection, _ ledger.CollectibleTipSelection) ledger.AcceptResult {
	return ledger.SubmissionAccepted{TaskID: taskID, SubmissionID: submissionID, Payout: ledger.NoPayout{}, Tip: ledger.NoTip{}}
}

func (services fakeServices) RequestChanges(_ context.Context, _ ledger.Reviewer, taskID core.TaskID, submissionID core.SubmissionID, _ ledger.IdempotencyKey, note submission.ReviewNote) ledger.RequestChangesResult {
	return ledger.ChangesRequested{TaskID: taskID, SubmissionID: submissionID, ReviewNote: note.String()}
}

func (services fakeServices) RejectSubmission(_ context.Context, _ ledger.Reviewer, taskID core.TaskID, submissionID core.SubmissionID, _ ledger.IdempotencyKey, _ submission.ReviewNote, _ ledger.CreditReviewSelection, _ ledger.TipSelection, _ ledger.BanSelection) ledger.RejectResult {
	return ledger.SubmissionRejected{TaskID: taskID, SubmissionID: submissionID, Payout: ledger.NoPayout{}, Tip: ledger.NoTip{}}
}

func (services fakeServices) ListSeries(_ context.Context, _ auth.UserSubject, _ core.Page) task.ListSeriesResult {
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

func (services fakeServices) UpdateSeries(_ context.Context, subject auth.UserSubject, seriesID core.TaskSeriesID, title task.SeriesTitle, description task.SeriesDescription) task.SeriesMutationResult {
	return task.SeriesMutated{Value: task.SeriesDetail{Series: task.Series{
		ID:          seriesID,
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

func (services fakeServices) ReorderSeries(_ context.Context, _ auth.UserSubject, _ core.TaskSeriesID, _ []core.TaskID) task.SeriesMutationResult {
	return task.SeriesMutationRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "unused")}
}

func (services fakeServices) AddSeriesComment(_ context.Context, subject auth.UserSubject, seriesID core.TaskSeriesID, body task.CommentBody) task.SeriesCommentResult {
	commentID := core.NewSeriesCommentID().(core.SeriesCommentIDCreated)
	return task.SeriesCommentAdded{Value: task.SeriesComment{ID: commentID.Value, SeriesID: seriesID, AuthorID: subject.ID, Body: body}}
}

func (services fakeServices) ListSeriesComments(_ context.Context, _ auth.UserSubject, _ core.TaskSeriesID, _ core.Page) task.SeriesCommentsResult {
	return task.SeriesCommentsListed{Values: nil}
}

func (services fakeServices) AddTaskComment(_ context.Context, subject auth.UserSubject, taskID core.TaskID, body task.CommentBody) task.TaskCommentResult {
	commentID := core.NewTaskCommentID().(core.TaskCommentIDCreated)
	return task.TaskCommentAdded{Value: task.TaskComment{ID: commentID.Value, TaskID: taskID, AuthorID: subject.ID, Body: body}}
}

func (services fakeServices) ListTaskComments(_ context.Context, _ auth.UserSubject, _ core.TaskID, _ core.Page) task.TaskCommentsResult {
	return task.TaskCommentsListed{Values: nil}
}

func (services fakeServices) AddSubmissionComment(_ context.Context, subject auth.UserSubject, submissionID core.SubmissionID, body task.CommentBody) submission.SubmissionCommentResult {
	commentID := core.NewSubmissionCommentID().(core.SubmissionCommentIDCreated)
	return submission.SubmissionCommentAdded{Value: submission.SubmissionComment{ID: commentID.Value, SubmissionID: submissionID, AuthorID: subject.ID, Body: body}}
}

func (services fakeServices) ListSubmissionComments(_ context.Context, _ auth.UserSubject, _ core.SubmissionID, _ core.Page) submission.SubmissionCommentsResult {
	return submission.SubmissionCommentsListed{Values: nil}
}

func (services fakeServices) UnpublishTask(_ context.Context, subject auth.Subject, taskID core.TaskID) task.ChangeStateResult {
	userID := fakeUserID(subject)
	return task.TaskStateChanged{Value: task.Task{ID: taskID, Owner: task.UserOwner{UserID: userID}, State: task.StateDraft, CreatedBy: userID}}
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

func (services fakeServices) ListReservations(_ context.Context, subject auth.Subject, taskID core.TaskID, _ core.Page) task.ReservationsListResult {
	userID := fakeUserID(subject)
	reservationID := core.NewTaskReservationID().(core.TaskReservationIDCreated)
	return task.ReservationsListed{Values: []task.Reservation{{
		ID:                reservationID.Value,
		TaskID:            taskID,
		Assignee:          task.UserAssignee{UserID: userID},
		State:             task.ReservationStateRequested,
		RequestedBy:       userID,
		HolderDisplayName: auth.NewDisplayName("ada").(auth.DisplayNameAccepted).Value,
	}}}
}

func (services fakeServices) CancelReservation(_ context.Context, subject auth.Subject, taskID core.TaskID, reservationID core.TaskReservationID) task.ReservationStateChangeResult {
	return fakeReservationStateChange(subject, taskID, reservationID, task.ReservationStateCancelledByRequester)
}

func fakeReservationStateChange(subject auth.Subject, taskID core.TaskID, reservationID core.TaskReservationID, state task.ReservationState) task.ReservationStateChangeResult {
	userID := fakeUserID(subject)
	return task.ReservationStateChanged{Value: task.Reservation{
		ID:          reservationID,
		TaskID:      taskID,
		Assignee:    task.UserAssignee{UserID: userID},
		State:       state,
		RequestedBy: userID,
	}}
}

func (services fakeServices) CreateOrganization(_ context.Context, subject auth.UserSubject, name org.OrganizationName) org.CreateOrganizationResult {
	organizationID := core.NewOrganizationID().(core.OrganizationIDCreated)
	return org.OrganizationCreated{Value: org.Organization{ID: organizationID.Value, Name: name, CreatedBy: subject.ID}}
}

func (services fakeServices) ListOrganizations(_ context.Context, _ auth.UserSubject, _ string, _ core.Page) org.ListOrganizationsResult {
	return org.OrganizationsListed{Values: []org.Organization{}}
}

func (services fakeServices) ListOrganizationMembers(_ context.Context, _ auth.UserSubject, _ core.OrganizationID, _ core.Page) org.ListMembersResult {
	return org.MembersListed{Values: []org.OrganizationMember{}}
}

func (services fakeServices) ProvisionOrganizationMember(_ context.Context, _ auth.UserSubject, organizationID core.OrganizationID, _ auth.EmailAddress, roles []org.Role) org.ProvisionMemberResult {
	membershipID := core.NewOrganizationMembershipID().(core.OrganizationMembershipIDCreated)
	userID := core.NewUserID().(core.UserIDCreated)
	return org.MemberProvisioned{Value: org.OrganizationMember{ID: membershipID.Value, OrganizationID: organizationID, UserID: userID.Value, Status: org.MembershipStatusActive, Roles: roles}}
}

func (services fakeServices) DeactivateOrganizationMember(_ context.Context, _ auth.UserSubject, _ core.OrganizationID, _ core.UserID) org.DeactivateMemberResult {
	return org.MemberDeactivationAccepted{}
}

func (services fakeServices) UpdateOrganizationMemberRoles(_ context.Context, _ auth.UserSubject, organizationID core.OrganizationID, userID core.UserID, roles []org.Role) org.UpdateMemberRolesResult {
	membershipID := core.NewOrganizationMembershipID().(core.OrganizationMembershipIDCreated)
	return org.MemberRolesUpdatedResult{Value: org.OrganizationMember{ID: membershipID.Value, OrganizationID: organizationID, UserID: userID, Status: org.MembershipStatusActive, Roles: roles}}
}

func (services fakeServices) CreateOrganizationTeam(_ context.Context, subject auth.UserSubject, organizationID core.OrganizationID, name org.TeamName) org.CreateTeamResult {
	teamID := core.NewTeamID().(core.TeamIDCreated)
	return org.TeamCreated{Value: org.Team{ID: teamID.Value, Owner: org.OrganizationOwnedTeam{OrganizationID: organizationID}, Name: name, CreatedBy: subject.ID}}
}

func (services fakeServices) ListOrganizationTeams(_ context.Context, _ auth.UserSubject, _ core.OrganizationID, _ string, _ core.Page) org.ListTeamsResult {
	return org.OrganizationTeamsListed{Values: []org.Team{}}
}

func (services fakeServices) CreateStandaloneTeam(_ context.Context, subject auth.UserSubject, name org.TeamName) org.CreateTeamResult {
	teamID := core.NewTeamID().(core.TeamIDCreated)
	return org.TeamCreated{Value: org.Team{ID: teamID.Value, Owner: org.UserOwnedTeam{OwnerUserID: subject.ID}, Name: name, CreatedBy: subject.ID}}
}

func (services fakeServices) ListStandaloneTeams(_ context.Context, _ auth.UserSubject, _ string, _ core.Page) org.ListTeamsResult {
	return org.OrganizationTeamsListed{Values: []org.Team{}}
}

func (services fakeServices) GetTeam(_ context.Context, _ auth.Subject, teamID core.TeamID) org.GetTeamResult {
	return org.TeamGot{Team: org.Team{ID: teamID, Owner: org.UserOwnedTeam{}, Name: mustTeamName("Fake team")}, Members: []core.UserID{}}
}

func (services fakeServices) GetTeamWork(_ context.Context, _ auth.UserSubject, _ core.TeamID, _ task.ListFilters, _ core.Page) task.ListResult {
	return task.TasksListed{Values: []task.ListItem{}}
}

func (services fakeServices) AddTeamMember(_ context.Context, _ auth.Subject, _ core.TeamID, _ auth.EmailAddress) org.AddTeamMemberResult {
	return org.TeamMemberAddedResult{}
}

func (services fakeServices) CheckOrganizationPermission(_ context.Context, _ core.OrganizationID, _ core.UserID, _ org.Permission) org.PermissionCheck {
	return org.PermissionGranted{}
}

func (services fakeServices) CreateOrgCredential(_ context.Context, organizationID core.OrganizationID, label agent.Label, scopes agent.ScopeSet, expiresAt *time.Time) orgcred.CreateResult {
	credentialID := core.NewOrgCredentialID().(core.OrgCredentialIDCreated)
	secret := orgcred.NewSecretPlain().(orgcred.SecretPlainAccepted)
	return orgcred.CredentialCreated{
		Value:  orgcred.Credential{ID: credentialID.Value, OrganizationID: organizationID, Label: label, Scopes: scopes, State: agent.StateActive, ExpiresAt: expiresAt},
		Secret: secret.Value,
	}
}

func (services fakeServices) ListOrgCredentials(_ context.Context, _ core.OrganizationID, _ core.Page) orgcred.ListResult {
	return orgcred.CredentialsListed{Values: []orgcred.Credential{}}
}

func (services fakeServices) RevokeOrgCredential(_ context.Context, organizationID core.OrganizationID, credentialID core.OrgCredentialID) orgcred.RevokeResult {
	return orgcred.CredentialRevoked{Value: orgcred.Credential{ID: credentialID, OrganizationID: organizationID, Label: mustLabel("Fake org credential"), Scopes: agent.NewScopeSet([]agent.Scope{agent.ScopeTasksRead}), State: agent.StateRevoked}}
}

func (services fakeServices) ListCollectibleCatalog(_ context.Context) assets.CatalogListResult {
	return assets.CatalogListed{Values: services.catalogListings}
}

func (services fakeServices) MintCollectible(_ context.Context, _ core.UserID, ownerKind string, ownerID string, organizationID string, name assets.CollectibleName, kind assets.CollectibleKind, policy assets.TransferPolicy, art string) assets.MintResult {
	collectibleID := core.NewCollectibleID().(core.CollectibleIDCreated)
	return assets.CollectibleMinted{Value: assets.Collectible{ID: collectibleID.Value, Name: name, Kind: kind, State: assets.CollectibleStateMinted, Policy: policy, OwnerKind: ownerKind, OwnerID: ownerID, OrganizationID: organizationID, Art: art}}
}

func (services fakeServices) ListCollectibles(_ context.Context, _ core.UserID, _ core.Page) assets.ListResult {
	if services.collectibles == nil {
		return assets.CollectiblesListed{Values: []assets.Collectible{}}
	}
	return assets.CollectiblesListed{Values: services.collectibles}
}

func (services fakeServices) ListCollectiblesByOwner(_ context.Context, _ string, _ string, _ core.Page) assets.ListResult {
	return assets.CollectiblesListed{Values: []assets.Collectible{}}
}

func (services fakeServices) TransferCollectible(_ context.Context, _ core.UserID, _ core.UserID, collectibleID core.CollectibleID) assets.GiftResult {
	return assets.CollectibleGifted{Value: assets.Collectible{ID: collectibleID, State: assets.CollectibleStateMinted}}
}

func (services fakeServices) TransferCollectibleToOrganization(_ context.Context, _ core.UserID, organizationID core.OrganizationID, collectibleID core.CollectibleID) assets.GiftResult {
	return assets.CollectibleGifted{Value: assets.Collectible{ID: collectibleID, State: assets.CollectibleStateMinted, OwnerKind: assets.CollectibleOwnerKindOrganization, OwnerID: organizationID.String(), OrganizationID: organizationID.String()}}
}

func (services fakeServices) TransferCollectibleFromOrganization(_ context.Context, _ core.UserID, _ core.OrganizationID, to core.UserID, collectibleID core.CollectibleID) assets.GiftResult {
	return assets.CollectibleGifted{Value: assets.Collectible{ID: collectibleID, State: assets.CollectibleStateMinted, OwnerKind: assets.CollectibleOwnerKindUser, OwnerID: to.String()}}
}

func (services fakeServices) AwardOrganizationCollectible(_ context.Context, _ core.OrganizationID, collectibleID core.CollectibleID, recipient core.UserID) assets.GiftResult {
	return assets.CollectibleGifted{Value: assets.Collectible{ID: collectibleID, State: assets.CollectibleStateMinted, OwnerKind: assets.CollectibleOwnerKindUser, OwnerID: recipient.String()}}
}

func (services fakeServices) AddCatalogEntry(_ context.Context, entry assets.CatalogEntry) assets.CatalogMutationResult {
	return assets.CatalogMutated{Value: entry}
}

func (services fakeServices) WithdrawCatalogEntry(_ context.Context, slug assets.CatalogSlug) assets.CatalogMutationResult {
	return assets.CatalogMutated{Value: assets.CatalogEntry{Slug: slug, State: assets.CatalogEntryStateWithdrawn, Cap: assets.NoEditionCap{}}}
}

func (services fakeServices) DeleteCatalogEntry(_ context.Context, slug assets.CatalogSlug) assets.CatalogMutationResult {
	return assets.CatalogMutated{Value: assets.CatalogEntry{Slug: slug, State: assets.CatalogEntryStateWithdrawn, Cap: assets.NoEditionCap{}}}
}

func (services fakeServices) WithdrawCollectible(_ context.Context, _ core.UserID, collectibleID core.CollectibleID) assets.WithdrawResult {
	return assets.CollectibleWithdrawn{Value: assets.Collectible{ID: collectibleID, State: assets.CollectibleStateWithdrawn}}
}

func (services fakeServices) ReleaseCatalogEntry(_ context.Context, slug assets.CatalogSlug) assets.CatalogMutationResult {
	return assets.CatalogMutated{Value: assets.CatalogEntry{Slug: slug, State: assets.CatalogEntryStateAvailable, Cap: assets.NoEditionCap{}}}
}

func (services fakeServices) ReleaseCollectible(_ context.Context, _ core.UserID, collectibleID core.CollectibleID) assets.ReleaseResult {
	return assets.CollectibleReleased{Value: assets.Collectible{ID: collectibleID, State: assets.CollectibleStateMinted}}
}

func (services fakeServices) DeleteWithdrawnCollectible(_ context.Context, _ core.CollectibleID) assets.DeleteCollectibleResult {
	return assets.CollectibleDeleted{}
}

func (services fakeServices) FundCollectibleReward(_ context.Context, _ core.UserID, _ core.TaskID, collectibleID core.CollectibleID) assets.FundRewardResult {
	return assets.RewardFunded{Value: assets.Collectible{ID: collectibleID, State: assets.CollectibleStateMinted}}
}

func (services fakeServices) RefundCollectibleReward(_ context.Context, _ core.UserID, _ core.TaskID) assets.RefundRewardResult {
	return assets.RewardRefunded{Values: []assets.Collectible{}}
}

func (services fakeServices) CountUnreadNotifications(_ context.Context, _ core.UserID) notification.CountResult {
	return notification.UnreadCounted{Count: 1}
}

func (services fakeServices) GetCreditBalance(_ context.Context, _ core.UserID) ledger.BalanceResult {
	return ledger.BalanceFound{Value: ledger.NewBalance(25, 5)}
}

func (services fakeServices) ListLedger(_ context.Context, _ core.UserID, _ core.Page) ledger.ListEntriesResult {
	return ledger.EntriesListed{Values: []ledger.LedgerEntry{}}
}

func (services fakeServices) GrantCredits(_ context.Context, _ core.UserID, _ ledger.GrantTarget, amount ledger.CreditAmount, _ ledger.GrantNote, _ ledger.IdempotencyKey) ledger.GrantResult {
	entryID := core.NewLedgerEntryID().(core.LedgerEntryIDCreated)
	return ledger.CreditsGranted{EntryID: entryID.Value, Amount: amount}
}

func (services fakeServices) SendCredits(_ context.Context, _ core.UserID, _ ledger.TransferSource, _ ledger.TransferTarget, amount ledger.CreditAmount, _ ledger.TransferNote, _ ledger.IdempotencyKey) ledger.SendResult {
	entryID := core.NewLedgerEntryID().(core.LedgerEntryIDCreated)
	return ledger.CreditsSent{DebitEntryID: entryID.Value, Amount: amount, Execution: ledger.FirstExecution{}}
}

func (services fakeServices) ListNotifications(_ context.Context, _ core.UserID, filter notification.StateFilter, _ core.Page) notification.ListResult {
	// The subject kind echoes the received filter so tool tests can assert the
	// state argument threads through to the service call.
	echoed := "any_state"
	if _, matched := filter.(notification.UnreadOnly); matched {
		echoed = "unread_only"
	}
	return notification.NotificationsListed{Values: []notification.Notification{{
		Kind:             notification.KindTaskFunded,
		State:            notification.StateUnread,
		Subject:          notification.Subject{Kind: echoed, ID: "echo"},
		ActorDisplayName: auth.NewDisplayName("mara").(auth.DisplayNameAccepted).Value,
		SubjectTitle:     notification.TaskSubjectTitle{Title: "Review the release"},
	}}, Total: 1}
}

func (services fakeServices) MarkNotificationRead(_ context.Context, recipient core.UserID, notificationID core.NotificationID) notification.MarkReadResult {
	return notification.NotificationRead{Value: notification.Notification{ID: notificationID, RecipientID: recipient}}
}

func (services fakeServices) ListUsers(_ context.Context, _ string, _ core.Page) auth.UserDirectoryResult {
	return auth.UsersListed{Values: []auth.UserDirectoryEntry{}}
}

func (services fakeServices) GetUserProfile(_ context.Context, _ auth.UserSubject, _ core.UserID, _ core.Page) task.ListResult {
	return task.TasksListed{Values: []task.ListItem{}}
}

func (services fakeServices) GetUserWork(_ context.Context, _ auth.UserSubject, _ core.UserID, _ core.Page) task.ListResult {
	return task.TasksListed{Values: []task.ListItem{}}
}

func (services fakeServices) GetUserSubmissions(_ context.Context, _ auth.UserSubject, _ core.UserID, _ core.Page) submission.ListResult {
	return submission.SubmissionsListed{Values: []submission.Submission{}}
}

func (services fakeServices) IsPlatformAdmin(_ context.Context, _ core.UserID) bool {
	return services.isAdmin
}

func (services fakeServices) ListPlatformAdmins(_ context.Context, _ core.Page) PlatformAdminListResult {
	return PlatformAdminsListed{Values: []PlatformAdminRecord{}}
}

func (services fakeServices) GrantPlatformAdmin(_ context.Context, userID core.UserID, _ core.UserID) PlatformAdminMutationResult {
	return PlatformAdminSaved{Value: PlatformAdminRecord{UserID: userID, Source: "granted"}}
}

func (services fakeServices) RevokePlatformAdmin(_ context.Context, userID core.UserID) PlatformAdminMutationResult {
	return PlatformAdminSaved{Value: PlatformAdminRecord{UserID: userID, Source: "granted"}}
}

func (services fakeServices) CreateModerationReport(_ context.Context, actor core.UserID, subjectKind string, subjectID string, reason string, details string) ModerationReportResult {
	return ModerationReportSaved{Value: ModerationReport{ID: "report-1", SubjectKind: subjectKind, SubjectID: subjectID, Reason: reason, Details: details, ReporterUserID: actor.String(), State: "open"}}
}

func (services fakeServices) ListAdminModerationReports(_ context.Context, _ string, _ core.Page) ModerationReportsListResult {
	return ModerationReportsListed{Values: []ModerationReport{}}
}

func (services fakeServices) TriageModerationReport(_ context.Context, actor core.UserID, reportID core.AuditEventID, state string, note string) ModerationReportResult {
	return ModerationReportSaved{Value: ModerationReport{ID: reportID.String(), State: state, ResolutionNote: note, UpdatedBy: actor.String()}}
}

func (services fakeServices) CreatePrivacyRequest(_ context.Context, actor core.UserID, kind string) PrivacyRequestResult {
	return PrivacyRequestSaved{Value: PrivacyRequestRecord{ID: "privacy-1", RequestedBy: actor, Kind: kind, State: "queued"}}
}

func (services fakeServices) ListPrivacyRequests(_ context.Context, _ core.UserID, _ core.Page) PrivacyRequestsListResult {
	return PrivacyRequestsListed{Values: []PrivacyRequestRecord{}}
}

func (services fakeServices) ListAdminPrivacyRequests(_ context.Context, _ core.Page) PrivacyRequestsListResult {
	return PrivacyRequestsListed{Values: []PrivacyRequestRecord{}}
}

func (services fakeServices) ResolveAdminPrivacyRequest(_ context.Context, requestID string, note string) PrivacyRequestResult {
	return PrivacyRequestSaved{Value: PrivacyRequestRecord{ID: requestID, State: "resolved", ResolutionNote: note}}
}

func (services fakeServices) RunPrivacyRetention(_ context.Context, _ core.UserID) PrivacyRetentionResult {
	return PrivacyRetentionRun{RedactedFieldCount: 0}
}

func (services fakeServices) ListAuditEvents(_ context.Context, _ audit.ListFilters, _ core.Page) audit.ListResult {
	return audit.EventsListed{Values: []audit.Event{}}
}

func (services fakeServices) AwardCollectible(_ context.Context, _ core.UserID, _ string, recipientKind string, recipientID string, organizationID string) assets.MintResult {
	collectibleID := core.NewCollectibleID().(core.CollectibleIDCreated)
	return assets.CollectibleMinted{Value: assets.Collectible{ID: collectibleID.Value, State: assets.CollectibleStateMinted, OwnerKind: recipientKind, OwnerID: recipientID, OrganizationID: organizationID}}
}

func mustTeamName(raw string) org.TeamName {
	accepted := org.NewTeamName(raw).(org.TeamNameAccepted)
	return accepted.Value
}

func mustLabel(raw string) agent.Label {
	accepted := agent.NewLabel(raw).(agent.LabelAccepted)
	return accepted.Value
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
