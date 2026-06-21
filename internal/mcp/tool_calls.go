package mcp

import (
	"context"
	"encoding/json"

	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/ledger"
	"github.com/e6qu/sharecrop/internal/schema"
	"github.com/e6qu/sharecrop/internal/submission"
	"github.com/e6qu/sharecrop/internal/task"
)

type taskSummary struct {
	ID             string `json:"id"`
	OwnerKind      string `json:"owner_kind"`
	Title          string `json:"title"`
	State          string `json:"state"`
	VisibilityKind string `json:"visibility_kind"`
	CreatedBy      string `json:"created_by"`
}

type taskDetail struct {
	ID                 string `json:"id"`
	OwnerKind          string `json:"owner_kind"`
	Title              string `json:"title"`
	Description        string `json:"description"`
	State              string `json:"state"`
	VisibilityKind     string `json:"visibility_kind"`
	ResponseSchemaJSON string `json:"response_schema_json"`
	PayloadKind        string `json:"payload_kind"`
	PayloadJSON        string `json:"payload_json"`
	CreatedBy          string `json:"created_by"`
}

type tasksPayload struct {
	Tasks []taskSummary `json:"tasks"`
}

type schemaPayload struct {
	TaskID             string `json:"task_id"`
	ResponseSchemaJSON string `json:"response_schema_json"`
}

type submitPayload struct {
	SubmissionID string `json:"submission_id"`
	State        string `json:"state"`
	ReceiptToken string `json:"receipt_token"`
}

type statusPayload struct {
	SubmissionID string `json:"submission_id"`
	TaskID       string `json:"task_id"`
	State        string `json:"state"`
	ResponseJSON string `json:"response_json"`
}

type submissionSummary struct {
	ID            string `json:"id"`
	TaskID        string `json:"task_id"`
	SubmitterKind string `json:"submitter_kind"`
	State         string `json:"state"`
}

type submissionsPayload struct {
	Submissions []submissionSummary `json:"submissions"`
}

type acceptPayload struct {
	TaskID       string `json:"task_id"`
	SubmissionID string `json:"submission_id"`
	PayoutKind   string `json:"payout_kind"`
	PayoutAmount int64  `json:"payout_amount"`
	WorkerUserID string `json:"worker_user_id"`
}

func (server Server) callListTasks(ctx context.Context, subject auth.UserSubject, arguments json.RawMessage) toolResult {
	var args struct {
		Scope string `json:"scope"`
	}
	if err := json.Unmarshal(arguments, &args); err != nil {
		return invalidArguments()
	}

	var scope task.ListScope
	switch args.Scope {
	case "public":
		scope = task.PublicListScope{}
	case "user":
		scope = task.UserListScope{UserID: subject.ID}
	default:
		return toolProtocolError{code: codeInvalidParams, message: "scope must be public or user"}
	}

	result := server.services.ListTasks(ctx, subject, scope)
	listed, matched := result.(task.TasksListed)
	if !matched {
		return toolFailed{message: result.(task.ListRejected).Reason.Description()}
	}

	summaries := make([]taskSummary, 0, len(listed.Values))
	for index := range listed.Values {
		summaries = append(summaries, taskToSummary(listed.Values[index]))
	}
	return marshalPayload(tasksPayload{Tasks: summaries})
}

func (server Server) callGetTask(ctx context.Context, subject auth.UserSubject, arguments json.RawMessage) toolResult {
	taskID, problem := parseTaskID(arguments)
	if problem != nil {
		return problem
	}
	result := server.services.GetTask(ctx, subject, taskID)
	got, matched := result.(task.TaskGot)
	if !matched {
		return toolFailed{message: result.(task.GetRejected).Reason.Description()}
	}
	return marshalPayload(taskToDetail(got.Value))
}

func (server Server) callGetTaskSchema(ctx context.Context, subject auth.UserSubject, arguments json.RawMessage) toolResult {
	taskID, problem := parseTaskID(arguments)
	if problem != nil {
		return problem
	}
	result := server.services.GetTask(ctx, subject, taskID)
	got, matched := result.(task.TaskGot)
	if !matched {
		return toolFailed{message: result.(task.GetRejected).Reason.Description()}
	}
	return marshalPayload(schemaPayload{TaskID: got.Value.ID.String(), ResponseSchemaJSON: got.Value.ResponseSchema.String()})
}

func (server Server) callCreateTask(ctx context.Context, subject auth.UserSubject, arguments json.RawMessage) toolResult {
	var args struct {
		Title              string `json:"title"`
		Description        string `json:"description"`
		ResponseSchemaJSON string `json:"response_schema_json"`
		Visibility         string `json:"visibility"`
	}
	if err := json.Unmarshal(arguments, &args); err != nil {
		return invalidArguments()
	}

	titleResult := task.NewTitle(args.Title)
	titleAccepted, titleMatched := titleResult.(task.TitleAccepted)
	if !titleMatched {
		return toolProtocolError{code: codeInvalidParams, message: titleResult.(task.TitleRejected).Reason.Description()}
	}
	descriptionResult := task.NewDescription(args.Description)
	descriptionAccepted, descriptionMatched := descriptionResult.(task.DescriptionAccepted)
	if !descriptionMatched {
		return toolProtocolError{code: codeInvalidParams, message: descriptionResult.(task.DescriptionRejected).Reason.Description()}
	}
	if _, schemaMatched := schema.ParseSchemaJSON([]byte(args.ResponseSchemaJSON)).(schema.SchemaParsed); !schemaMatched {
		return toolProtocolError{code: codeInvalidParams, message: "response schema JSON is invalid"}
	}
	schemaSourceResult := task.NewResponseSchemaSource(args.ResponseSchemaJSON)
	schemaSourceAccepted, schemaSourceMatched := schemaSourceResult.(task.ResponseSchemaSourceAccepted)
	if !schemaSourceMatched {
		return toolProtocolError{code: codeInvalidParams, message: schemaSourceResult.(task.ResponseSchemaSourceRejected).Reason.Description()}
	}

	var visibility task.Visibility
	switch args.Visibility {
	case "", "user":
		visibility = task.UserVisibility{UserID: subject.ID}
	case "public":
		visibility = task.PublicVisibility{}
	default:
		return toolProtocolError{code: codeInvalidParams, message: "visibility must be user or public"}
	}

	command := task.CreateCommand{
		Actor:          subject,
		Owner:          task.UserOwner{UserID: subject.ID},
		Title:          titleAccepted.Value,
		Description:    descriptionAccepted.Value,
		Visibility:     visibility,
		Placement:      task.StandalonePlacement{},
		ResponseSchema: schemaSourceAccepted.Value,
		Payload:        task.NoDataPayload{},
	}
	result := server.services.CreateTask(ctx, command)
	created, matched := result.(task.TaskCreated)
	if !matched {
		return toolFailed{message: result.(task.CreateRejected).Reason.Description()}
	}
	return marshalPayload(taskToDetail(created.Value))
}

func (server Server) callSubmitResponse(ctx context.Context, subject auth.UserSubject, arguments json.RawMessage) toolResult {
	var args struct {
		TaskID       string `json:"task_id"`
		ResponseJSON string `json:"response_json"`
	}
	if err := json.Unmarshal(arguments, &args); err != nil {
		return invalidArguments()
	}
	taskIDResult := core.ParseTaskID(args.TaskID)
	taskID, taskMatched := taskIDResult.(core.TaskIDCreated)
	if !taskMatched {
		return toolProtocolError{code: codeInvalidParams, message: taskIDResult.(core.TaskIDRejected).Reason.Description()}
	}
	sourceResult := submission.NewResponseSource(args.ResponseJSON)
	source, sourceMatched := sourceResult.(submission.ResponseSourceAccepted)
	if !sourceMatched {
		return toolProtocolError{code: codeInvalidParams, message: sourceResult.(submission.ResponseSourceRejected).Reason.Description()}
	}

	command := submission.SubmitCommand{
		TaskID:         taskID.Value,
		Submitter:      submission.AuthenticatedSubmitter{UserID: subject.ID},
		ResponseSource: source.Value,
	}
	result := server.services.SubmitResponse(ctx, command)
	created, matched := result.(submission.SubmissionCreated)
	if !matched {
		return toolFailed{message: result.(submission.SubmitRejected).Reason.Description()}
	}
	return marshalPayload(submitPayload{
		SubmissionID: created.Value.ID.String(),
		State:        created.Value.State.String(),
		ReceiptToken: created.ReceiptToken.String(),
	})
}

func (server Server) callGetSubmissionStatus(ctx context.Context, arguments json.RawMessage) toolResult {
	var args struct {
		ReceiptToken string `json:"receipt_token"`
	}
	if err := json.Unmarshal(arguments, &args); err != nil {
		return invalidArguments()
	}
	tokenResult := submission.ParseReceiptTokenPlain(args.ReceiptToken)
	token, tokenMatched := tokenResult.(submission.ReceiptTokenPlainAccepted)
	if !tokenMatched {
		return toolProtocolError{code: codeInvalidParams, message: tokenResult.(submission.ReceiptTokenPlainRejected).Reason.Description()}
	}
	result := server.services.GetSubmissionStatus(ctx, token.Value)
	found, matched := result.(submission.ReceiptStatusFound)
	if !matched {
		return toolFailed{message: result.(submission.ReceiptStatusRejected).Reason.Description()}
	}
	return marshalPayload(statusPayload{
		SubmissionID: found.Value.ID.String(),
		TaskID:       found.Value.TaskID.String(),
		State:        found.Value.State.String(),
		ResponseJSON: found.Value.ResponseSource.String(),
	})
}

func (server Server) callListTaskSubmissions(ctx context.Context, subject auth.UserSubject, arguments json.RawMessage) toolResult {
	taskID, problem := parseTaskID(arguments)
	if problem != nil {
		return problem
	}
	result := server.services.ListTaskSubmissions(ctx, subject, taskID)
	listed, matched := result.(submission.SubmissionsListed)
	if !matched {
		return toolFailed{message: result.(submission.ListRejected).Reason.Description()}
	}
	summaries := make([]submissionSummary, 0, len(listed.Values))
	for index := range listed.Values {
		summaries = append(summaries, submissionToSummary(listed.Values[index]))
	}
	return marshalPayload(submissionsPayload{Submissions: summaries})
}

func (server Server) callAcceptSubmission(ctx context.Context, subject auth.UserSubject, arguments json.RawMessage) toolResult {
	var args struct {
		TaskID         string `json:"task_id"`
		SubmissionID   string `json:"submission_id"`
		IdempotencyKey string `json:"idempotency_key"`
	}
	if err := json.Unmarshal(arguments, &args); err != nil {
		return invalidArguments()
	}
	taskIDResult := core.ParseTaskID(args.TaskID)
	taskID, taskMatched := taskIDResult.(core.TaskIDCreated)
	if !taskMatched {
		return toolProtocolError{code: codeInvalidParams, message: taskIDResult.(core.TaskIDRejected).Reason.Description()}
	}
	submissionIDResult := core.ParseSubmissionID(args.SubmissionID)
	submissionID, submissionMatched := submissionIDResult.(core.SubmissionIDCreated)
	if !submissionMatched {
		return toolProtocolError{code: codeInvalidParams, message: submissionIDResult.(core.SubmissionIDRejected).Reason.Description()}
	}
	keyResult := ledger.NewIdempotencyKey(args.IdempotencyKey)
	key, keyMatched := keyResult.(ledger.IdempotencyKeyAccepted)
	if !keyMatched {
		return toolProtocolError{code: codeInvalidParams, message: keyResult.(ledger.IdempotencyKeyRejected).Reason.Description()}
	}

	result := server.services.AcceptSubmission(ctx, subject.ID, taskID.Value, submissionID.Value, key.Value)
	accepted, matched := result.(ledger.SubmissionAccepted)
	if !matched {
		return toolFailed{message: result.(ledger.AcceptRejected).Reason.Description()}
	}
	payload := acceptPayload{
		TaskID:       accepted.TaskID.String(),
		SubmissionID: accepted.SubmissionID.String(),
		PayoutKind:   "none",
	}
	if payout, payoutMatched := accepted.Payout.(ledger.CreditPayout); payoutMatched {
		payload.PayoutKind = "credit"
		payload.PayoutAmount = payout.Amount.Int64()
		payload.WorkerUserID = payout.WorkerUserID.String()
	}
	return marshalPayload(payload)
}

func parseTaskID(arguments json.RawMessage) (core.TaskID, toolResult) {
	var args struct {
		TaskID string `json:"task_id"`
	}
	if err := json.Unmarshal(arguments, &args); err != nil {
		return core.TaskID{}, invalidArguments()
	}
	result := core.ParseTaskID(args.TaskID)
	taskID, matched := result.(core.TaskIDCreated)
	if !matched {
		return core.TaskID{}, toolProtocolError{code: codeInvalidParams, message: result.(core.TaskIDRejected).Reason.Description()}
	}
	return taskID.Value, nil
}

func invalidArguments() toolResult {
	return toolProtocolError{code: codeInvalidParams, message: "tool arguments are invalid"}
}

func marshalPayload(value payloadValue) toolResult {
	encoded, err := json.Marshal(value)
	if err != nil {
		return toolProtocolError{code: codeInternalError, message: "failed to encode tool result"}
	}
	return toolSucceeded{payload: encoded}
}

type payloadValue interface {
	payloadValue()
}

func (tasksPayload) payloadValue() {}

func (taskDetail) payloadValue() {}

func (schemaPayload) payloadValue() {}

func (submitPayload) payloadValue() {}

func (statusPayload) payloadValue() {}

func (submissionsPayload) payloadValue() {}

func (acceptPayload) payloadValue() {}

func taskToSummary(value task.Task) taskSummary {
	return taskSummary{
		ID:             value.ID.String(),
		OwnerKind:      ownerKind(value.Owner),
		Title:          value.Title.String(),
		State:          value.State.String(),
		VisibilityKind: visibilityKind(value.Visibility),
		CreatedBy:      value.CreatedBy.String(),
	}
}

func taskToDetail(value task.Task) taskDetail {
	payloadKind, payloadJSON := payloadParts(value.Payload)
	return taskDetail{
		ID:                 value.ID.String(),
		OwnerKind:          ownerKind(value.Owner),
		Title:              value.Title.String(),
		Description:        value.Description.String(),
		State:              value.State.String(),
		VisibilityKind:     visibilityKind(value.Visibility),
		ResponseSchemaJSON: value.ResponseSchema.String(),
		PayloadKind:        payloadKind,
		PayloadJSON:        payloadJSON,
		CreatedBy:          value.CreatedBy.String(),
	}
}

func submissionToSummary(value submission.Submission) submissionSummary {
	return submissionSummary{
		ID:            value.ID.String(),
		TaskID:        value.TaskID.String(),
		SubmitterKind: submitterKind(value.Submitter),
		State:         value.State.String(),
	}
}

func ownerKind(owner task.Owner) string {
	switch owner.(type) {
	case task.UserOwner:
		return task.OwnerKindUser.String()
	case task.TeamOwner:
		return task.OwnerKindTeam.String()
	case task.OrganizationOwner:
		return task.OwnerKindOrganization.String()
	case task.OrganizationTeamOwner:
		return task.OwnerKindOrganizationTeam.String()
	default:
		return ""
	}
}

func visibilityKind(visibility task.Visibility) string {
	switch visibility.(type) {
	case task.PublicVisibility:
		return task.VisibilityKindPublic.String()
	case task.UserVisibility:
		return task.VisibilityKindUser.String()
	case task.TeamVisibility:
		return task.VisibilityKindTeam.String()
	case task.OrganizationVisibility:
		return task.VisibilityKindOrganization.String()
	case task.OrganizationTeamVisibility:
		return task.VisibilityKindOrganizationTeam.String()
	default:
		return ""
	}
}

func payloadParts(payload task.DataPayload) (string, string) {
	switch typed := payload.(type) {
	case task.NoDataPayload:
		return "none", ""
	case task.JSONDataPayload:
		return "json", typed.Source.String()
	default:
		return "", ""
	}
}

func submitterKind(submitter submission.Submitter) string {
	switch submitter.(type) {
	case submission.AuthenticatedSubmitter:
		return submission.SubmitterKindAuthenticated.String()
	case submission.AnonymousSubmitter:
		return submission.SubmitterKindAnonymous.String()
	default:
		return ""
	}
}
