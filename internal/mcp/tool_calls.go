package mcp

import (
	"context"
	"encoding/json"
	"time"

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
	TaskType           string `json:"task_type"`
	ReferenceURL       string `json:"reference_url"`
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

type fundPayload struct {
	TaskID       string `json:"task_id"`
	CreditAmount int64  `json:"credit_amount"`
}

type statusPayload struct {
	SubmissionID string `json:"submission_id"`
	TaskID       string `json:"task_id"`
	State        string `json:"state"`
	ResponseJSON string `json:"response_json"`
	ReviewNote   string `json:"review_note"`
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

type reservationSummary struct {
	ID           string `json:"id"`
	TaskID       string `json:"task_id"`
	AssigneeKind string `json:"assignee_kind"`
	AssigneeID   string `json:"assignee_id"`
	State        string `json:"state"`
	RequestedBy  string `json:"requested_by"`
	// IssuedWorkerCredential is a one-time plaintext secret for a new
	// task-scoped agent credential, present only immediately after this
	// reservation was created or approved into an active state.
	IssuedWorkerCredential string `json:"issued_worker_credential"`
}

type reservationPayload struct {
	Reservation reservationSummary `json:"reservation"`
}

type reservationsPayload struct {
	Reservations []reservationSummary `json:"reservations"`
}

func (server Server) callListTasks(ctx context.Context, subject auth.Subject, arguments json.RawMessage) toolResult {
	var args struct {
		Scope string `json:"scope"`
		State string `json:"state"`
	}
	if err := json.Unmarshal(arguments, &args); err != nil {
		return invalidArguments()
	}

	var scope task.ListScope
	switch args.Scope {
	case "public":
		scope = task.PublicListScope{}
	case "user":
		userActor, isUser := subject.(auth.UserSubject)
		if !isUser {
			return toolFailed{message: "scope \"user\" requires a personal agent credential, not an organization credential"}
		}
		scope = task.UserListScope{UserID: userActor.ID}
	default:
		return toolProtocolError{code: codeInvalidParams, message: "scope must be public or user"}
	}

	filters := task.NoListFilters()
	if args.State != "" {
		stateResult := task.ParseState(args.State)
		stateAccepted, stateMatched := stateResult.(task.StateAccepted)
		if !stateMatched {
			return toolProtocolError{code: codeInvalidParams, message: stateResult.(task.StateRejected).Reason.Description()}
		}
		filters.State = task.StateEquals{Value: stateAccepted.Value}
	}

	result := server.services.ListTasks(ctx, subject, scope, filters)
	listed, matched := result.(task.TasksListed)
	if !matched {
		return toolFailed{message: result.(task.ListRejected).Reason.Description()}
	}

	summaries := make([]taskSummary, 0, len(listed.Values))
	for index := range listed.Values {
		summaries = append(summaries, taskToSummary(listed.Values[index].Task))
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
		Title               string `json:"title"`
		Description         string `json:"description"`
		ResponseSchemaJSON  string `json:"response_schema_json"`
		Visibility          string `json:"visibility"`
		RewardKind          string `json:"reward_kind"`
		RewardCreditAmount  int64  `json:"reward_credit_amount"`
		ParticipationPolicy string `json:"participation_policy"`
		TaskType            string `json:"task_type"`
		ReferenceURL        string `json:"reference_url"`
	}
	if err := json.Unmarshal(arguments, &args); err != nil {
		return invalidArguments()
	}

	taskTypeResult := task.ParseTaskType(args.TaskType)
	taskTypeAccepted, taskTypeMatched := taskTypeResult.(task.TaskTypeAccepted)
	if !taskTypeMatched {
		return toolProtocolError{code: codeInvalidParams, message: taskTypeResult.(task.TaskTypeRejected).Reason.Description()}
	}
	referenceResult := task.NewReferenceURL(args.ReferenceURL)
	referenceAccepted, referenceMatched := referenceResult.(task.ReferenceURLAccepted)
	if !referenceMatched {
		return toolProtocolError{code: codeInvalidParams, message: referenceResult.(task.ReferenceURLRejected).Reason.Description()}
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

	participationRaw := args.ParticipationPolicy
	if participationRaw == "" {
		participationRaw = task.ParticipationPolicyOpen.String()
	}
	participationResult := task.ParseParticipationPolicy(participationRaw)
	participationAccepted, participationMatched := participationResult.(task.ParticipationPolicyAccepted)
	if !participationMatched {
		return toolProtocolError{code: codeInvalidParams, message: participationResult.(task.ParticipationPolicyRejected).Reason.Description()}
	}

	command := task.CreateCommand{
		Actor:          subject,
		Owner:          task.UserOwner{UserID: subject.ID},
		Title:          titleAccepted.Value,
		Description:    descriptionAccepted.Value,
		Type:           taskTypeAccepted.Value,
		Reference:      referenceAccepted.Value,
		Reward:         reward.value,
		Participation:  participationAccepted.Value,
		AssigneeScope:  task.AssigneeScopeUser,
		ReservationTTL: task.DefaultReservationTTL(),
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

func (server Server) callOpenTask(ctx context.Context, subject auth.Subject, arguments json.RawMessage) toolResult {
	taskID, problem := parseTaskID(arguments)
	if problem != nil {
		return problem
	}
	result := server.services.OpenTask(ctx, subject, taskID)
	changed, matched := result.(task.TaskStateChanged)
	if !matched {
		return toolFailed{message: result.(task.ChangeStateRejected).Reason.Description()}
	}
	return marshalPayload(taskToDetail(changed.Value))
}

func (server Server) callCancelTask(ctx context.Context, subject auth.Subject, arguments json.RawMessage) toolResult {
	taskID, problem := parseTaskID(arguments)
	if problem != nil {
		return problem
	}
	result := server.services.CancelTask(ctx, subject, taskID)
	changed, matched := result.(task.TaskStateChanged)
	if !matched {
		return toolFailed{message: result.(task.ChangeStateRejected).Reason.Description()}
	}
	return marshalPayload(taskToDetail(changed.Value))
}

func (server Server) callFundTask(ctx context.Context, subject auth.UserSubject, arguments json.RawMessage) toolResult {
	var args struct {
		TaskID         string `json:"task_id"`
		Amount         int64  `json:"amount"`
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
	amountResult := ledger.NewCreditAmount(args.Amount)
	amount, amountMatched := amountResult.(ledger.CreditAmountAccepted)
	if !amountMatched {
		return toolProtocolError{code: codeInvalidParams, message: amountResult.(ledger.CreditAmountRejected).Reason.Description()}
	}
	keyResult := ledger.NewIdempotencyKey(args.IdempotencyKey)
	key, keyMatched := keyResult.(ledger.IdempotencyKeyAccepted)
	if !keyMatched {
		return toolProtocolError{code: codeInvalidParams, message: keyResult.(ledger.IdempotencyKeyRejected).Reason.Description()}
	}
	result := server.services.FundTask(ctx, subject.ID, taskID.Value, amount.Value, key.Value)
	funded, matched := result.(ledger.TaskFunded)
	if !matched {
		return toolFailed{message: result.(ledger.FundRejected).Reason.Description()}
	}
	return marshalPayload(fundPayload{
		TaskID:       funded.Fund.TaskID.String(),
		CreditAmount: funded.Fund.CreditAmount.Int64(),
	})
}

func (server Server) callRefundTask(ctx context.Context, subject auth.UserSubject, arguments json.RawMessage) toolResult {
	var args struct {
		TaskID         string `json:"task_id"`
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
	keyResult := ledger.NewIdempotencyKey(args.IdempotencyKey)
	key, keyMatched := keyResult.(ledger.IdempotencyKeyAccepted)
	if !keyMatched {
		return toolProtocolError{code: codeInvalidParams, message: keyResult.(ledger.IdempotencyKeyRejected).Reason.Description()}
	}
	result := server.services.RefundTask(ctx, subject.ID, taskID.Value, key.Value)
	refunded, matched := result.(ledger.TaskRefunded)
	if !matched {
		return toolFailed{message: result.(ledger.RefundRejected).Reason.Description()}
	}
	return marshalPayload(fundPayload{
		TaskID:       refunded.Fund.TaskID.String(),
		CreditAmount: refunded.Fund.CreditAmount.Int64(),
	})
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
		ReviewNote:   found.Value.ReviewNote.String(),
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

	result := server.services.ReviewAcceptSubmission(ctx, subject.ID, ids.taskID, ids.submissionID, key.Value, creditSelection.value, tipSelection.value, ledger.NoCollectibleTipSelection{})
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
	ID          string `json:"id"`
	OwnerKind   string `json:"owner_kind"`
	Title       string `json:"title"`
	Description string `json:"description"`
	State       string `json:"state"`
	CreatedBy   string `json:"created_by"`
}

type seriesCommentSummary struct {
	ID        string `json:"id"`
	AuthorID  string `json:"author_user_id"`
	Body      string `json:"body"`
	CreatedAt string `json:"created_at"`
}

type seriesCommentsPayload struct {
	Comments []seriesCommentSummary `json:"comments"`
}

type submissionCommentSummary struct {
	ID           string `json:"id"`
	SubmissionID string `json:"submission_id"`
	AuthorID     string `json:"author_user_id"`
	Body         string `json:"body"`
	CreatedAt    string `json:"created_at"`
}

type submissionCommentsPayload struct {
	Comments []submissionCommentSummary `json:"comments"`
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

func (server Server) callReserveTask(ctx context.Context, subject auth.UserSubject, arguments json.RawMessage) toolResult {
	parsed := parseReserveTaskArguments(arguments)
	if parsed.problem != nil {
		return parsed.problem
	}
	var result task.ReservationResult
	if parsed.assigneeKind == task.AssigneeScopeOrganizationTeam.String() {
		result = server.services.ReserveTaskForOrganizationTeam(ctx, subject, parsed.taskID, parsed.organizationID, parsed.teamID)
	} else {
		result = server.services.ReserveTask(ctx, subject, parsed.taskID)
	}
	created, matched := result.(task.ReservationCreated)
	if !matched {
		return toolFailed{message: result.(task.ReservationRejected).Reason.Description()}
	}
	return marshalPayload(reservationPayload{Reservation: reservationToSummary(created.Value, created.IssuedWorkerCredentialSecret)})
}

type parsedReserveTaskArguments struct {
	taskID         core.TaskID
	assigneeKind   string
	organizationID core.OrganizationID
	teamID         core.TeamID
	problem        toolResult
}

func parseReserveTaskArguments(arguments json.RawMessage) parsedReserveTaskArguments {
	var args struct {
		TaskID         string `json:"task_id"`
		AssigneeKind   string `json:"assignee_kind"`
		OrganizationID string `json:"organization_id"`
		TeamID         string `json:"team_id"`
	}
	if err := json.Unmarshal(arguments, &args); err != nil {
		return parsedReserveTaskArguments{problem: invalidArguments()}
	}

	taskIDResult := core.ParseTaskID(args.TaskID)
	taskID, taskIDMatched := taskIDResult.(core.TaskIDCreated)
	if !taskIDMatched {
		return parsedReserveTaskArguments{problem: toolProtocolError{code: codeInvalidParams, message: taskIDResult.(core.TaskIDRejected).Reason.Description()}}
	}

	switch args.AssigneeKind {
	case "", task.AssigneeScopeUser.String():
		return parsedReserveTaskArguments{taskID: taskID.Value, assigneeKind: task.AssigneeScopeUser.String()}
	case task.AssigneeScopeOrganizationTeam.String():
		organizationIDResult := core.ParseOrganizationID(args.OrganizationID)
		organizationID, organizationIDMatched := organizationIDResult.(core.OrganizationIDCreated)
		if !organizationIDMatched {
			return parsedReserveTaskArguments{problem: toolProtocolError{code: codeInvalidParams, message: organizationIDResult.(core.OrganizationIDRejected).Reason.Description()}}
		}
		teamIDResult := core.ParseTeamID(args.TeamID)
		teamID, teamIDMatched := teamIDResult.(core.TeamIDCreated)
		if !teamIDMatched {
			return parsedReserveTaskArguments{problem: toolProtocolError{code: codeInvalidParams, message: teamIDResult.(core.TeamIDRejected).Reason.Description()}}
		}
		return parsedReserveTaskArguments{taskID: taskID.Value, assigneeKind: args.AssigneeKind, organizationID: organizationID.Value, teamID: teamID.Value}
	default:
		return parsedReserveTaskArguments{problem: toolProtocolError{code: codeInvalidParams, message: "reservation assignee kind is invalid"}}
	}
}

func (server Server) callListReservations(ctx context.Context, subject auth.Subject, arguments json.RawMessage) toolResult {
	taskID, problem := parseTaskID(arguments)
	if problem != nil {
		return problem
	}
	result := server.services.ListReservations(ctx, subject, taskID)
	listed, matched := result.(task.ReservationsListed)
	if !matched {
		return toolFailed{message: result.(task.ReservationsListRejected).Reason.Description()}
	}
	reservations := make([]reservationSummary, 0, len(listed.Values))
	for index := range listed.Values {
		reservations = append(reservations, reservationToSummary(listed.Values[index], ""))
	}
	return marshalPayload(reservationsPayload{Reservations: reservations})
}

type mcpReservationChanger func(context.Context, auth.Subject, core.TaskID, core.TaskReservationID) task.ReservationStateChangeResult

func (server Server) callChangeReservation(ctx context.Context, subject auth.Subject, arguments json.RawMessage, changer mcpReservationChanger) toolResult {
	ids := parseTaskReservationIDs(arguments)
	if ids.problem != nil {
		return ids.problem
	}
	result := changer(ctx, subject, ids.taskID, ids.reservationID)
	changed, matched := result.(task.ReservationStateChanged)
	if !matched {
		return toolFailed{message: result.(task.ReservationStateChangeRejected).Reason.Description()}
	}
	return marshalPayload(reservationPayload{Reservation: reservationToSummary(changed.Value, changed.IssuedWorkerCredentialSecret)})
}

func seriesToSummary(value task.Series) seriesSummary {
	return seriesSummary{
		ID:          value.ID.String(),
		OwnerKind:   ownerKind(value.Owner),
		Title:       value.Title.String(),
		Description: value.Description.String(),
		State:       value.State.String(),
		CreatedBy:   value.CreatedBy.String(),
	}
}

func seriesDetailToPayload(detail task.SeriesDetail) seriesDetailPayload {
	tasks := make([]taskSummary, 0, len(detail.Tasks))
	for index := range detail.Tasks {
		tasks = append(tasks, taskToSummary(detail.Tasks[index]))
	}
	return seriesDetailPayload{Series: seriesToSummary(detail.Series), Tasks: tasks}
}

type parsedTaskReservationIDs struct {
	taskID        core.TaskID
	reservationID core.TaskReservationID
	problem       toolResult
}

func parseTaskReservationIDs(arguments json.RawMessage) parsedTaskReservationIDs {
	var args struct {
		TaskID        string `json:"task_id"`
		ReservationID string `json:"reservation_id"`
	}
	if err := json.Unmarshal(arguments, &args); err != nil {
		return parsedTaskReservationIDs{problem: invalidArguments()}
	}
	taskIDResult := core.ParseTaskID(args.TaskID)
	taskID, taskMatched := taskIDResult.(core.TaskIDCreated)
	if !taskMatched {
		return parsedTaskReservationIDs{problem: toolProtocolError{code: codeInvalidParams, message: taskIDResult.(core.TaskIDRejected).Reason.Description()}}
	}
	reservationIDResult := core.ParseTaskReservationID(args.ReservationID)
	reservationID, reservationMatched := reservationIDResult.(core.TaskReservationIDCreated)
	if !reservationMatched {
		return parsedTaskReservationIDs{problem: toolProtocolError{code: codeInvalidParams, message: reservationIDResult.(core.TaskReservationIDRejected).Reason.Description()}}
	}
	return parsedTaskReservationIDs{taskID: taskID.Value, reservationID: reservationID.Value}
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

func (fundPayload) payloadValue() {}

func (submitPayload) payloadValue() {}

func (statusPayload) payloadValue() {}

func (submissionsPayload) payloadValue() {}

func (acceptPayload) payloadValue() {}

func (reviewPayload) payloadValue() {}

func (reservationPayload) payloadValue() {}

func (reservationsPayload) payloadValue() {}

func (seriesListPayload) payloadValue() {}

func (seriesDetailPayload) payloadValue() {}

func (seriesCommentsPayload) payloadValue() {}

func (seriesCommentSummary) payloadValue() {}

func (submissionCommentsPayload) payloadValue() {}

func (submissionCommentSummary) payloadValue() {}

func (server Server) callCreateSeries(ctx context.Context, subject auth.UserSubject, arguments json.RawMessage) toolResult {
	var args struct {
		Title       string `json:"title"`
		Description string `json:"description"`
	}
	if err := json.Unmarshal(arguments, &args); err != nil {
		return invalidArguments()
	}
	titleResult := task.NewSeriesTitle(args.Title)
	title, titleMatched := titleResult.(task.SeriesTitleAccepted)
	if !titleMatched {
		return toolProtocolError{code: codeInvalidParams, message: titleResult.(task.SeriesTitleRejected).Reason.Description()}
	}
	descriptionResult := task.NewSeriesDescription(args.Description)
	description, descriptionMatched := descriptionResult.(task.SeriesDescriptionAccepted)
	if !descriptionMatched {
		return toolProtocolError{code: codeInvalidParams, message: descriptionResult.(task.SeriesDescriptionRejected).Reason.Description()}
	}
	return server.seriesMutationResult(server.services.CreateSeries(ctx, subject, title.Value, description.Value))
}

func (server Server) callUpdateSeries(ctx context.Context, subject auth.UserSubject, arguments json.RawMessage) toolResult {
	var args struct {
		SeriesID    string `json:"series_id"`
		Title       string `json:"title"`
		Description string `json:"description"`
	}
	if err := json.Unmarshal(arguments, &args); err != nil {
		return invalidArguments()
	}
	seriesResult := core.ParseTaskSeriesID(args.SeriesID)
	seriesID, seriesMatched := seriesResult.(core.TaskSeriesIDCreated)
	if !seriesMatched {
		return toolProtocolError{code: codeInvalidParams, message: seriesResult.(core.TaskSeriesIDRejected).Reason.Description()}
	}
	titleResult := task.NewSeriesTitle(args.Title)
	title, titleMatched := titleResult.(task.SeriesTitleAccepted)
	if !titleMatched {
		return toolProtocolError{code: codeInvalidParams, message: titleResult.(task.SeriesTitleRejected).Reason.Description()}
	}
	descriptionResult := task.NewSeriesDescription(args.Description)
	description, descriptionMatched := descriptionResult.(task.SeriesDescriptionAccepted)
	if !descriptionMatched {
		return toolProtocolError{code: codeInvalidParams, message: descriptionResult.(task.SeriesDescriptionRejected).Reason.Description()}
	}
	return server.seriesMutationResult(server.services.UpdateSeries(ctx, subject, seriesID.Value, title.Value, description.Value))
}

func (server Server) callReorderSeries(ctx context.Context, subject auth.UserSubject, arguments json.RawMessage) toolResult {
	var args struct {
		SeriesID string   `json:"series_id"`
		TaskIDs  []string `json:"task_ids"`
	}
	if err := json.Unmarshal(arguments, &args); err != nil {
		return invalidArguments()
	}
	seriesResult := core.ParseTaskSeriesID(args.SeriesID)
	seriesID, seriesMatched := seriesResult.(core.TaskSeriesIDCreated)
	if !seriesMatched {
		return toolProtocolError{code: codeInvalidParams, message: seriesResult.(core.TaskSeriesIDRejected).Reason.Description()}
	}
	order := make([]core.TaskID, 0, len(args.TaskIDs))
	for index := range args.TaskIDs {
		taskIDResult := core.ParseTaskID(args.TaskIDs[index])
		taskID, taskMatched := taskIDResult.(core.TaskIDCreated)
		if !taskMatched {
			return toolProtocolError{code: codeInvalidParams, message: taskIDResult.(core.TaskIDRejected).Reason.Description()}
		}
		order = append(order, taskID.Value)
	}
	return server.seriesMutationResult(server.services.ReorderSeries(ctx, subject, seriesID.Value, order))
}

func (server Server) callAddTaskToSeries(ctx context.Context, subject auth.UserSubject, arguments json.RawMessage) toolResult {
	seriesID, taskID, problem := parseSeriesAndTaskID(arguments)
	if problem != nil {
		return problem
	}
	return server.seriesMutationResult(server.services.AddTaskToSeries(ctx, subject, seriesID, taskID))
}

func (server Server) callRemoveTaskFromSeries(ctx context.Context, subject auth.UserSubject, arguments json.RawMessage) toolResult {
	seriesID, taskID, problem := parseSeriesAndTaskID(arguments)
	if problem != nil {
		return problem
	}
	return server.seriesMutationResult(server.services.RemoveTaskFromSeries(ctx, subject, seriesID, taskID))
}

func (server Server) callChangeSeriesState(ctx context.Context, subject auth.UserSubject, arguments json.RawMessage, transition task.SeriesStateTransition) toolResult {
	seriesID, problem := parseSeriesID(arguments)
	if problem != nil {
		return problem
	}
	return server.seriesMutationResult(server.services.ChangeSeriesState(ctx, subject, seriesID, transition))
}

func (server Server) callAddSeriesComment(ctx context.Context, subject auth.UserSubject, arguments json.RawMessage) toolResult {
	var args struct {
		SeriesID string `json:"series_id"`
		Body     string `json:"body"`
	}
	if err := json.Unmarshal(arguments, &args); err != nil {
		return invalidArguments()
	}
	seriesResult := core.ParseTaskSeriesID(args.SeriesID)
	seriesID, seriesMatched := seriesResult.(core.TaskSeriesIDCreated)
	if !seriesMatched {
		return toolProtocolError{code: codeInvalidParams, message: seriesResult.(core.TaskSeriesIDRejected).Reason.Description()}
	}
	bodyResult := task.NewCommentBody(args.Body)
	body, bodyMatched := bodyResult.(task.CommentBodyAccepted)
	if !bodyMatched {
		return toolProtocolError{code: codeInvalidParams, message: bodyResult.(task.CommentBodyRejected).Reason.Description()}
	}
	result := server.services.AddSeriesComment(ctx, subject, seriesID.Value, body.Value)
	added, matched := result.(task.SeriesCommentAdded)
	if !matched {
		return toolFailed{message: result.(task.SeriesCommentRejected).Reason.Description()}
	}
	return marshalPayload(commentToSummary(added.Value))
}

func (server Server) callListSeriesComments(ctx context.Context, subject auth.UserSubject, arguments json.RawMessage) toolResult {
	seriesID, problem := parseSeriesID(arguments)
	if problem != nil {
		return problem
	}
	result := server.services.ListSeriesComments(ctx, subject, seriesID)
	listed, matched := result.(task.SeriesCommentsListed)
	if !matched {
		return toolFailed{message: result.(task.SeriesCommentsListRejected).Reason.Description()}
	}
	comments := make([]seriesCommentSummary, 0, len(listed.Values))
	for index := range listed.Values {
		comments = append(comments, commentToSummary(listed.Values[index]))
	}
	return marshalPayload(seriesCommentsPayload{Comments: comments})
}

func (server Server) callAddTaskComment(ctx context.Context, subject auth.UserSubject, arguments json.RawMessage) toolResult {
	var args struct {
		TaskID string `json:"task_id"`
		Body   string `json:"body"`
	}
	if err := json.Unmarshal(arguments, &args); err != nil {
		return invalidArguments()
	}
	taskResult := core.ParseTaskID(args.TaskID)
	taskID, taskMatched := taskResult.(core.TaskIDCreated)
	if !taskMatched {
		return toolProtocolError{code: codeInvalidParams, message: taskResult.(core.TaskIDRejected).Reason.Description()}
	}
	bodyResult := task.NewCommentBody(args.Body)
	body, bodyMatched := bodyResult.(task.CommentBodyAccepted)
	if !bodyMatched {
		return toolProtocolError{code: codeInvalidParams, message: bodyResult.(task.CommentBodyRejected).Reason.Description()}
	}
	result := server.services.AddTaskComment(ctx, subject, taskID.Value, body.Value)
	added, matched := result.(task.TaskCommentAdded)
	if !matched {
		return toolFailed{message: result.(task.TaskCommentRejected).Reason.Description()}
	}
	return marshalPayload(seriesCommentSummary{
		ID:        added.Value.ID.String(),
		AuthorID:  added.Value.AuthorID.String(),
		Body:      added.Value.Body.String(),
		CreatedAt: added.Value.CreatedAt.UTC().Format(time.RFC3339),
	})
}

func (server Server) callListTaskComments(ctx context.Context, subject auth.UserSubject, arguments json.RawMessage) toolResult {
	taskID, problem := parseTaskID(arguments)
	if problem != nil {
		return problem
	}
	result := server.services.ListTaskComments(ctx, subject, taskID)
	listed, matched := result.(task.TaskCommentsListed)
	if !matched {
		return toolFailed{message: result.(task.TaskCommentsListRejected).Reason.Description()}
	}
	comments := make([]seriesCommentSummary, 0, len(listed.Values))
	for index := range listed.Values {
		comments = append(comments, seriesCommentSummary{
			ID:        listed.Values[index].ID.String(),
			AuthorID:  listed.Values[index].AuthorID.String(),
			Body:      listed.Values[index].Body.String(),
			CreatedAt: listed.Values[index].CreatedAt.UTC().Format(time.RFC3339),
		})
	}
	return marshalPayload(seriesCommentsPayload{Comments: comments})
}

func (server Server) callAddSubmissionComment(ctx context.Context, subject auth.UserSubject, arguments json.RawMessage) toolResult {
	var args struct {
		SubmissionID string `json:"submission_id"`
		Body         string `json:"body"`
	}
	if err := json.Unmarshal(arguments, &args); err != nil {
		return invalidArguments()
	}
	submissionResult := core.ParseSubmissionID(args.SubmissionID)
	submissionID, submissionMatched := submissionResult.(core.SubmissionIDCreated)
	if !submissionMatched {
		return toolProtocolError{code: codeInvalidParams, message: submissionResult.(core.SubmissionIDRejected).Reason.Description()}
	}
	bodyResult := task.NewCommentBody(args.Body)
	body, bodyMatched := bodyResult.(task.CommentBodyAccepted)
	if !bodyMatched {
		return toolProtocolError{code: codeInvalidParams, message: bodyResult.(task.CommentBodyRejected).Reason.Description()}
	}
	result := server.services.AddSubmissionComment(ctx, subject, submissionID.Value, body.Value)
	added, matched := result.(submission.SubmissionCommentAdded)
	if !matched {
		return toolFailed{message: result.(submission.SubmissionCommentRejected).Reason.Description()}
	}
	return marshalPayload(submissionCommentToSummary(added.Value))
}

func (server Server) callListSubmissionComments(ctx context.Context, subject auth.UserSubject, arguments json.RawMessage) toolResult {
	var args struct {
		SubmissionID string `json:"submission_id"`
	}
	if err := json.Unmarshal(arguments, &args); err != nil {
		return invalidArguments()
	}
	submissionResult := core.ParseSubmissionID(args.SubmissionID)
	submissionID, submissionMatched := submissionResult.(core.SubmissionIDCreated)
	if !submissionMatched {
		return toolProtocolError{code: codeInvalidParams, message: submissionResult.(core.SubmissionIDRejected).Reason.Description()}
	}
	result := server.services.ListSubmissionComments(ctx, subject, submissionID.Value)
	listed, matched := result.(submission.SubmissionCommentsListed)
	if !matched {
		return toolFailed{message: result.(submission.SubmissionCommentsListRejected).Reason.Description()}
	}
	comments := make([]submissionCommentSummary, 0, len(listed.Values))
	for index := range listed.Values {
		comments = append(comments, submissionCommentToSummary(listed.Values[index]))
	}
	return marshalPayload(submissionCommentsPayload{Comments: comments})
}

func submissionCommentToSummary(value submission.SubmissionComment) submissionCommentSummary {
	return submissionCommentSummary{
		ID:           value.ID.String(),
		SubmissionID: value.SubmissionID.String(),
		AuthorID:     value.AuthorID.String(),
		Body:         value.Body.String(),
		CreatedAt:    value.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func (server Server) callUnpublishTask(ctx context.Context, subject auth.Subject, arguments json.RawMessage) toolResult {
	taskID, problem := parseTaskID(arguments)
	if problem != nil {
		return problem
	}
	result := server.services.UnpublishTask(ctx, subject, taskID)
	changed, matched := result.(task.TaskStateChanged)
	if !matched {
		return toolFailed{message: result.(task.ChangeStateRejected).Reason.Description()}
	}
	return marshalPayload(taskToDetail(changed.Value))
}

func (server Server) seriesMutationResult(result task.SeriesMutationResult) toolResult {
	mutated, matched := result.(task.SeriesMutated)
	if !matched {
		return toolFailed{message: result.(task.SeriesMutationRejected).Reason.Description()}
	}
	return marshalPayload(seriesDetailToPayload(mutated.Value))
}

func commentToSummary(value task.SeriesComment) seriesCommentSummary {
	return seriesCommentSummary{
		ID:        value.ID.String(),
		AuthorID:  value.AuthorID.String(),
		Body:      value.Body.String(),
		CreatedAt: value.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func parseSeriesID(arguments json.RawMessage) (core.TaskSeriesID, toolResult) {
	var args struct {
		SeriesID string `json:"series_id"`
	}
	if err := json.Unmarshal(arguments, &args); err != nil {
		return core.TaskSeriesID{}, invalidArguments()
	}
	result := core.ParseTaskSeriesID(args.SeriesID)
	seriesID, matched := result.(core.TaskSeriesIDCreated)
	if !matched {
		return core.TaskSeriesID{}, toolProtocolError{code: codeInvalidParams, message: result.(core.TaskSeriesIDRejected).Reason.Description()}
	}
	return seriesID.Value, nil
}

func parseSeriesAndTaskID(arguments json.RawMessage) (core.TaskSeriesID, core.TaskID, toolResult) {
	var args struct {
		SeriesID string `json:"series_id"`
		TaskID   string `json:"task_id"`
	}
	if err := json.Unmarshal(arguments, &args); err != nil {
		return core.TaskSeriesID{}, core.TaskID{}, invalidArguments()
	}
	seriesResult := core.ParseTaskSeriesID(args.SeriesID)
	seriesID, seriesMatched := seriesResult.(core.TaskSeriesIDCreated)
	if !seriesMatched {
		return core.TaskSeriesID{}, core.TaskID{}, toolProtocolError{code: codeInvalidParams, message: seriesResult.(core.TaskSeriesIDRejected).Reason.Description()}
	}
	taskResult := core.ParseTaskID(args.TaskID)
	taskID, taskMatched := taskResult.(core.TaskIDCreated)
	if !taskMatched {
		return core.TaskSeriesID{}, core.TaskID{}, toolProtocolError{code: codeInvalidParams, message: taskResult.(core.TaskIDRejected).Reason.Description()}
	}
	return seriesID.Value, taskID.Value, nil
}

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
		TaskType:           value.Type.String(),
		ReferenceURL:       value.Reference.String(),
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

func reservationToSummary(value task.Reservation, issuedWorkerCredential string) reservationSummary {
	assigneeKind, assigneeID := reservationAssigneeParts(value.Assignee)
	return reservationSummary{
		ID:                     value.ID.String(),
		TaskID:                 value.TaskID.String(),
		AssigneeKind:           assigneeKind,
		AssigneeID:             assigneeID,
		State:                  value.State.String(),
		RequestedBy:            value.RequestedBy.String(),
		IssuedWorkerCredential: issuedWorkerCredential,
	}
}

func reservationAssigneeParts(assignee task.Assignee) (string, string) {
	switch typed := assignee.(type) {
	case task.UserAssignee:
		return task.AssigneeScopeUser.String(), typed.UserID.String()
	case task.OrganizationTeamAssignee:
		return task.AssigneeScopeOrganizationTeam.String(), typed.TeamID.String()
	default:
		return "", ""
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
