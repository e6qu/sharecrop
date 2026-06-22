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
	RewardKind     string `json:"reward_kind"`
	RewardAmount   int64  `json:"reward_credit_amount"`
	Collectibles   int    `json:"reward_collectible_count"`
	State          string `json:"state"`
	VisibilityKind string `json:"visibility_kind"`
	CreatedBy      string `json:"created_by"`
}

type taskDetail struct {
	ID                 string `json:"id"`
	OwnerKind          string `json:"owner_kind"`
	Title              string `json:"title"`
	Description        string `json:"description"`
	RewardKind         string `json:"reward_kind"`
	RewardAmount       int64  `json:"reward_credit_amount"`
	Collectibles       int    `json:"reward_collectible_count"`
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
	ID          string `json:"id"`
	TaskID      string `json:"task_id"`
	SubmitterID string `json:"submitter_id"`
	State       string `json:"state"`
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
	TipAmount    int64  `json:"tip_amount"`
}

type reviewPayload struct {
	TaskID       string `json:"task_id"`
	SubmissionID string `json:"submission_id"`
	State        string `json:"state"`
	ReviewNote   string `json:"review_note"`
	PayoutKind   string `json:"payout_kind"`
	PayoutAmount int64  `json:"payout_amount"`
	WorkerUserID string `json:"worker_user_id"`
	TipAmount    int64  `json:"tip_amount"`
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
		RewardKind         string `json:"reward_kind"`
		RewardCreditAmount int64  `json:"reward_credit_amount"`
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
	case "user":
		visibility = task.UserVisibility{UserID: subject.ID}
	case "public":
		visibility = task.PublicVisibility{}
	default:
		return toolProtocolError{code: codeInvalidParams, message: "visibility must be user or public"}
	}
	rewardResult := parseMCPReward(args.RewardKind, args.RewardCreditAmount)
	reward, rewardMatched := rewardResult.(mcpRewardAccepted)
	if !rewardMatched {
		return toolProtocolError{code: codeInvalidParams, message: rewardResult.(mcpRewardRejected).reason}
	}

	command := task.CreateCommand{
		Actor:          subject,
		Owner:          task.UserOwner{UserID: subject.ID},
		Title:          titleAccepted.Value,
		Description:    descriptionAccepted.Value,
		Reward:         reward.value,
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

type mcpRewardResult interface {
	mcpRewardResult()
}

type mcpRewardAccepted struct {
	value task.RewardSpec
}

type mcpRewardRejected struct {
	reason string
}

func (mcpRewardAccepted) mcpRewardResult() {}

func (mcpRewardRejected) mcpRewardResult() {}

func parseMCPReward(kind string, creditAmount int64) mcpRewardResult {
	switch kind {
	case task.RewardKindNone.String():
		return mcpRewardAccepted{value: task.NoRewardSpec{}}
	case task.RewardKindCredit.String():
		amountResult := task.NewCreditRewardAmount(creditAmount)
		amount, matched := amountResult.(task.CreditRewardAmountAccepted)
		if !matched {
			return mcpRewardRejected{reason: amountResult.(task.CreditRewardAmountRejected).Reason.Description()}
		}
		return mcpRewardAccepted{value: task.CreditRewardSpec{Amount: amount.Value}}
	case task.RewardKindCollectible.String():
		count := task.NewCollectibleRewardCount(1).(task.CollectibleRewardCountAccepted)
		return mcpRewardAccepted{value: task.CollectibleRewardSpec{Count: count.Value}}
	case task.RewardKindBundle.String():
		amountResult := task.NewCreditRewardAmount(creditAmount)
		amount, matched := amountResult.(task.CreditRewardAmountAccepted)
		if !matched {
			return mcpRewardRejected{reason: amountResult.(task.CreditRewardAmountRejected).Reason.Description()}
		}
		count := task.NewCollectibleRewardCount(1).(task.CollectibleRewardCountAccepted)
		return mcpRewardAccepted{value: task.BundleRewardSpec{Credit: amount.Value, Collectible: count.Value}}
	default:
		return mcpRewardRejected{reason: "reward_kind must be none, credit, collectible, or bundle"}
	}
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
		SubmitterID:    subject.ID,
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
		PayoutAmount   int64  `json:"payout_amount"`
		TipAmount      int64  `json:"tip_amount"`
	}
	if err := json.Unmarshal(arguments, &args); err != nil {
		return invalidArguments()
	}
	ids := parseTaskSubmissionIDs(args.TaskID, args.SubmissionID)
	if ids.problem != nil {
		return ids.problem
	}
	keyResult := ledger.NewIdempotencyKey(args.IdempotencyKey)
	key, keyMatched := keyResult.(ledger.IdempotencyKeyAccepted)
	if !keyMatched {
		return toolProtocolError{code: codeInvalidParams, message: keyResult.(ledger.IdempotencyKeyRejected).Reason.Description()}
	}
	creditSelectionResult := acceptCreditSelection(args.PayoutAmount)
	creditSelection, creditSelectionMatched := creditSelectionResult.(mcpCreditSelectionAccepted)
	if !creditSelectionMatched {
		return toolProtocolError{code: codeInvalidParams, message: creditSelectionResult.(mcpCreditSelectionRejected).message}
	}
	tipSelectionResult := mcpTipSelection(args.TipAmount)
	tipSelection, tipSelectionMatched := tipSelectionResult.(mcpTipSelectionAccepted)
	if !tipSelectionMatched {
		return toolProtocolError{code: codeInvalidParams, message: tipSelectionResult.(mcpTipSelectionRejected).message}
	}

	result := server.services.ReviewAcceptSubmission(ctx, subject.ID, ids.taskID, ids.submissionID, key.Value, creditSelection.value, tipSelection.value)
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
	if payout, payoutMatched := accepted.Payout.(ledger.BundlePayout); payoutMatched {
		payload.PayoutKind = "bundle"
		payload.PayoutAmount = payout.Amount.Int64()
		payload.WorkerUserID = payout.WorkerUserID.String()
	}
	if tip, tipMatched := accepted.Tip.(ledger.CreditTip); tipMatched {
		payload.TipAmount = tip.Amount.Int64()
		if payload.WorkerUserID == "" {
			payload.WorkerUserID = tip.WorkerUserID.String()
		}
	}
	return marshalPayload(payload)
}

func (server Server) callRequestChanges(ctx context.Context, subject auth.UserSubject, arguments json.RawMessage) toolResult {
	var args struct {
		TaskID       string `json:"task_id"`
		SubmissionID string `json:"submission_id"`
		ReviewNote   string `json:"review_note"`
	}
	if err := json.Unmarshal(arguments, &args); err != nil {
		return invalidArguments()
	}
	ids := parseTaskSubmissionIDs(args.TaskID, args.SubmissionID)
	if ids.problem != nil {
		return ids.problem
	}
	noteResult := submission.NewRequiredReviewNote(args.ReviewNote)
	note, noteMatched := noteResult.(submission.ReviewNoteAccepted)
	if !noteMatched {
		return toolProtocolError{code: codeInvalidParams, message: noteResult.(submission.ReviewNoteRejected).Reason.Description()}
	}
	result := server.services.RequestChanges(ctx, subject.ID, ids.taskID, ids.submissionID, note.Value)
	changed, matched := result.(ledger.ChangesRequested)
	if !matched {
		return toolFailed{message: result.(ledger.RequestChangesRejected).Reason.Description()}
	}
	return marshalPayload(reviewPayload{
		TaskID:       changed.TaskID.String(),
		SubmissionID: changed.SubmissionID.String(),
		State:        "changes_requested",
		ReviewNote:   changed.ReviewNote,
		PayoutKind:   "none",
	})
}

func (server Server) callRejectSubmission(ctx context.Context, subject auth.UserSubject, arguments json.RawMessage) toolResult {
	var args struct {
		TaskID              string `json:"task_id"`
		SubmissionID        string `json:"submission_id"`
		IdempotencyKey      string `json:"idempotency_key"`
		ReviewNote          string `json:"review_note"`
		PartialCreditAmount int64  `json:"partial_credit_amount"`
		TipAmount           int64  `json:"tip_amount"`
		BanImplementor      bool   `json:"ban_implementor"`
	}
	if err := json.Unmarshal(arguments, &args); err != nil {
		return invalidArguments()
	}
	ids := parseTaskSubmissionIDs(args.TaskID, args.SubmissionID)
	if ids.problem != nil {
		return ids.problem
	}
	keyResult := ledger.NewIdempotencyKey(args.IdempotencyKey)
	key, keyMatched := keyResult.(ledger.IdempotencyKeyAccepted)
	if !keyMatched {
		return toolProtocolError{code: codeInvalidParams, message: keyResult.(ledger.IdempotencyKeyRejected).Reason.Description()}
	}
	noteResult := submission.NewRequiredReviewNote(args.ReviewNote)
	note, noteMatched := noteResult.(submission.ReviewNoteAccepted)
	if !noteMatched {
		return toolProtocolError{code: codeInvalidParams, message: noteResult.(submission.ReviewNoteRejected).Reason.Description()}
	}
	creditSelectionResult := rejectCreditSelection(args.PartialCreditAmount)
	creditSelection, creditSelectionMatched := creditSelectionResult.(mcpCreditSelectionAccepted)
	if !creditSelectionMatched {
		return toolProtocolError{code: codeInvalidParams, message: creditSelectionResult.(mcpCreditSelectionRejected).message}
	}
	tipSelectionResult := mcpTipSelection(args.TipAmount)
	tipSelection, tipSelectionMatched := tipSelectionResult.(mcpTipSelectionAccepted)
	if !tipSelectionMatched {
		return toolProtocolError{code: codeInvalidParams, message: tipSelectionResult.(mcpTipSelectionRejected).message}
	}
	banSelection := ledger.BanSelection(ledger.NoBanSelection{})
	if args.BanImplementor {
		banSelection = ledger.BanImplementorSelection{}
	}
	result := server.services.RejectSubmission(ctx, subject.ID, ids.taskID, ids.submissionID, key.Value, note.Value, creditSelection.value, tipSelection.value, banSelection)
	rejected, matched := result.(ledger.SubmissionRejected)
	if !matched {
		return toolFailed{message: result.(ledger.RejectRejected).Reason.Description()}
	}
	payload := reviewPayload{
		TaskID:       rejected.TaskID.String(),
		SubmissionID: rejected.SubmissionID.String(),
		State:        "rejected",
		ReviewNote:   note.Value.String(),
		PayoutKind:   "none",
	}
	if payout, payoutMatched := rejected.Payout.(ledger.CreditPayout); payoutMatched {
		payload.PayoutKind = "credit"
		payload.PayoutAmount = payout.Amount.Int64()
		payload.WorkerUserID = payout.WorkerUserID.String()
	}
	if tip, tipMatched := rejected.Tip.(ledger.CreditTip); tipMatched {
		payload.TipAmount = tip.Amount.Int64()
		if payload.WorkerUserID == "" {
			payload.WorkerUserID = tip.WorkerUserID.String()
		}
	}
	return marshalPayload(payload)
}

type seriesSummary struct {
	ID        string `json:"id"`
	OwnerKind string `json:"owner_kind"`
	Title     string `json:"title"`
	CreatedBy string `json:"created_by"`
}

type seriesListPayload struct {
	Series []seriesSummary `json:"series"`
}

type seriesDetailPayload struct {
	Series seriesSummary `json:"series"`
	Tasks  []taskSummary `json:"tasks"`
}

func (server Server) callListTaskSeries(ctx context.Context, subject auth.UserSubject) toolResult {
	result := server.services.ListSeries(ctx, subject)
	listed, matched := result.(task.SeriesListed)
	if !matched {
		return toolFailed{message: result.(task.ListSeriesRejected).Reason.Description()}
	}
	summaries := make([]seriesSummary, 0, len(listed.Values))
	for index := range listed.Values {
		summaries = append(summaries, seriesToSummary(listed.Values[index]))
	}
	return marshalPayload(seriesListPayload{Series: summaries})
}

func (server Server) callGetTaskSeries(ctx context.Context, subject auth.UserSubject, arguments json.RawMessage) toolResult {
	var args struct {
		SeriesID string `json:"series_id"`
	}
	if err := json.Unmarshal(arguments, &args); err != nil {
		return invalidArguments()
	}
	seriesIDResult := core.ParseTaskSeriesID(args.SeriesID)
	seriesID, idMatched := seriesIDResult.(core.TaskSeriesIDCreated)
	if !idMatched {
		return toolProtocolError{code: codeInvalidParams, message: seriesIDResult.(core.TaskSeriesIDRejected).Reason.Description()}
	}
	result := server.services.GetSeries(ctx, subject, seriesID.Value)
	got, matched := result.(task.SeriesGot)
	if !matched {
		return toolFailed{message: result.(task.GetSeriesRejected).Reason.Description()}
	}
	tasks := make([]taskSummary, 0, len(got.Value.Tasks))
	for index := range got.Value.Tasks {
		tasks = append(tasks, taskToSummary(got.Value.Tasks[index]))
	}
	return marshalPayload(seriesDetailPayload{Series: seriesToSummary(got.Value.Series), Tasks: tasks})
}

func seriesToSummary(value task.Series) seriesSummary {
	return seriesSummary{
		ID:        value.ID.String(),
		OwnerKind: ownerKind(value.Owner),
		Title:     value.Title.String(),
		CreatedBy: value.CreatedBy.String(),
	}
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

func (reviewPayload) payloadValue() {}

func (seriesListPayload) payloadValue() {}

func (seriesDetailPayload) payloadValue() {}

type parsedTaskSubmissionIDs struct {
	taskID       core.TaskID
	submissionID core.SubmissionID
	problem      toolResult
}

func parseTaskSubmissionIDs(rawTaskID string, rawSubmissionID string) parsedTaskSubmissionIDs {
	taskIDResult := core.ParseTaskID(rawTaskID)
	taskID, taskMatched := taskIDResult.(core.TaskIDCreated)
	if !taskMatched {
		return parsedTaskSubmissionIDs{problem: toolProtocolError{code: codeInvalidParams, message: taskIDResult.(core.TaskIDRejected).Reason.Description()}}
	}
	submissionIDResult := core.ParseSubmissionID(rawSubmissionID)
	submissionID, submissionMatched := submissionIDResult.(core.SubmissionIDCreated)
	if !submissionMatched {
		return parsedTaskSubmissionIDs{problem: toolProtocolError{code: codeInvalidParams, message: submissionIDResult.(core.SubmissionIDRejected).Reason.Description()}}
	}
	return parsedTaskSubmissionIDs{taskID: taskID.Value, submissionID: submissionID.Value}
}

func taskToSummary(value task.Task) taskSummary {
	rewardKind, rewardAmount, collectibleCount := rewardParts(value.Reward)
	return taskSummary{
		ID:             value.ID.String(),
		OwnerKind:      ownerKind(value.Owner),
		Title:          value.Title.String(),
		RewardKind:     rewardKind,
		RewardAmount:   rewardAmount,
		Collectibles:   collectibleCount,
		State:          value.State.String(),
		VisibilityKind: visibilityKind(value.Visibility),
		CreatedBy:      value.CreatedBy.String(),
	}
}

func taskToDetail(value task.Task) taskDetail {
	payloadKind, payloadJSON := payloadParts(value.Payload)
	rewardKind, rewardAmount, collectibleCount := rewardParts(value.Reward)
	return taskDetail{
		ID:                 value.ID.String(),
		OwnerKind:          ownerKind(value.Owner),
		Title:              value.Title.String(),
		Description:        value.Description.String(),
		RewardKind:         rewardKind,
		RewardAmount:       rewardAmount,
		Collectibles:       collectibleCount,
		State:              value.State.String(),
		VisibilityKind:     visibilityKind(value.Visibility),
		ResponseSchemaJSON: value.ResponseSchema.String(),
		PayloadKind:        payloadKind,
		PayloadJSON:        payloadJSON,
		CreatedBy:          value.CreatedBy.String(),
	}
}

func rewardParts(reward task.RewardSpec) (string, int64, int) {
	switch typed := reward.(type) {
	case task.NoRewardSpec:
		return task.RewardKindNone.String(), 0, 0
	case task.CreditRewardSpec:
		return task.RewardKindCredit.String(), typed.Amount.Int64(), 0
	case task.CollectibleRewardSpec:
		return task.RewardKindCollectible.String(), 0, typed.Count.Int()
	case task.BundleRewardSpec:
		return task.RewardKindBundle.String(), typed.Credit.Int64(), typed.Collectible.Int()
	default:
		return "", 0, 0
	}
}

func submissionToSummary(value submission.Submission) submissionSummary {
	return submissionSummary{
		ID:          value.ID.String(),
		TaskID:      value.TaskID.String(),
		SubmitterID: value.SubmitterID.String(),
		State:       value.State.String(),
	}
}

type mcpCreditSelectionResult interface {
	mcpCreditSelectionResult()
}

type mcpCreditSelectionAccepted struct {
	value ledger.CreditReviewSelection
}

type mcpCreditSelectionRejected struct {
	message string
}

func (mcpCreditSelectionAccepted) mcpCreditSelectionResult() {}

func (mcpCreditSelectionRejected) mcpCreditSelectionResult() {}

func acceptCreditSelection(amount int64) mcpCreditSelectionResult {
	if amount < 0 {
		return mcpCreditSelectionRejected{message: "payout amount cannot be negative"}
	}
	if amount == 0 {
		return mcpCreditSelectionAccepted{value: ledger.FullCreditReviewSelection{}}
	}
	creditResult := ledger.NewCreditAmount(amount)
	credit, matched := creditResult.(ledger.CreditAmountAccepted)
	if !matched {
		return mcpCreditSelectionRejected{message: creditResult.(ledger.CreditAmountRejected).Reason.Description()}
	}
	return mcpCreditSelectionAccepted{value: ledger.PartialCreditReviewSelection{Amount: credit.Value}}
}

func rejectCreditSelection(amount int64) mcpCreditSelectionResult {
	if amount < 0 {
		return mcpCreditSelectionRejected{message: "partial credit amount cannot be negative"}
	}
	if amount == 0 {
		return mcpCreditSelectionAccepted{value: ledger.NoCreditReviewSelection{}}
	}
	creditResult := ledger.NewCreditAmount(amount)
	credit, matched := creditResult.(ledger.CreditAmountAccepted)
	if !matched {
		return mcpCreditSelectionRejected{message: creditResult.(ledger.CreditAmountRejected).Reason.Description()}
	}
	return mcpCreditSelectionAccepted{value: ledger.PartialCreditReviewSelection{Amount: credit.Value}}
}

type mcpTipSelectionResult interface {
	mcpTipSelectionResult()
}

type mcpTipSelectionAccepted struct {
	value ledger.TipSelection
}

type mcpTipSelectionRejected struct {
	message string
}

func (mcpTipSelectionAccepted) mcpTipSelectionResult() {}

func (mcpTipSelectionRejected) mcpTipSelectionResult() {}

func mcpTipSelection(amount int64) mcpTipSelectionResult {
	if amount < 0 {
		return mcpTipSelectionRejected{message: "tip amount cannot be negative"}
	}
	if amount == 0 {
		return mcpTipSelectionAccepted{value: ledger.NoTipSelection{}}
	}
	creditResult := ledger.NewCreditAmount(amount)
	credit, matched := creditResult.(ledger.CreditAmountAccepted)
	if !matched {
		return mcpTipSelectionRejected{message: creditResult.(ledger.CreditAmountRejected).Reason.Description()}
	}
	return mcpTipSelectionAccepted{value: ledger.CreditTipSelection{Amount: credit.Value}}
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
