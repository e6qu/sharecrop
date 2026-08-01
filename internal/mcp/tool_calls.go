package mcp

import (
	"context"
	"encoding/json"
	"time"

	"github.com/e6qu/sharecrop/internal/attachment"
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
	// ExpiresAt is the task's expiration instant in RFC3339, empty when the
	// task has no expiration policy (matching the REST task response).
	ExpiresAt string `json:"expires_at"`
}

// taskListRow is one task-listing row: the task summary plus the read-model
// enrichments listings resolve (mirroring REST's taskListItemResponse): the
// creator's display name, the active reservation holder's display name (empty
// when no user-assigned reservation is active), the credit-reward escrow state
// (reward_funded, reward_unfunded, or no_credit_reward), and the count of
// submissions still awaiting review (populated only on the caller's own
// tasks; 0 elsewhere).
type taskListRow struct {
	taskSummary
	CreatorDisplayName string `json:"creator_display_name"`
	HolderDisplayName  string `json:"holder_display_name"`
	Funded             string `json:"funded"`
	PendingReviewCount int64  `json:"pending_review_count"`
}

type tasksPayload struct {
	Tasks      []taskListRow `json:"tasks"`
	NextOffset int           `json:"next_offset"`
	// Total counts every row matching the filter, ignoring limit/offset.
	Total int64 `json:"total"`
}

type schemaPayload struct {
	TaskID             string `json:"task_id"`
	ResponseSchemaJSON string `json:"response_schema_json"`
}

type validationErrorPayload struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}

// validationErrorsToPayload flattens a validation outcome into the wire
// error array: empty when validation passed, one row per error otherwise.
func validationErrorsToPayload(outcome submission.ValidationOutcome) []validationErrorPayload {
	failed, matched := outcome.(submission.ValidationFailed)
	if !matched {
		return []validationErrorPayload{}
	}
	errors := make([]validationErrorPayload, 0, len(failed.Errors))
	for index := range failed.Errors {
		errors = append(errors, validationErrorPayload{Path: failed.Errors[index].Path, Message: failed.Errors[index].Message})
	}
	return errors
}

type attachmentPayload struct {
	Name        string `json:"name"`
	ContentType string `json:"content_type"`
	SizeBytes   int    `json:"size_bytes"`
	DataURL     string `json:"data_url"`
}

func attachmentsToPayload(values []attachment.Attachment) []attachmentPayload {
	payloads := make([]attachmentPayload, 0, len(values))
	for index := range values {
		payloads = append(payloads, attachmentPayload{
			Name:        values[index].Name.String(),
			ContentType: values[index].ContentType.String(),
			SizeBytes:   values[index].Content.SizeBytes(),
			DataURL:     values[index].DataURL(),
		})
	}
	return payloads
}

// invalidSubmissionGuidance tells the submitting agent why state is
// "invalid" and that it can retry at once: an invalid submission does not
// consume an active reservation.
const invalidSubmissionGuidance = "response_json failed schema validation; see validation_errors. An active reservation is kept for an invalid submission, so fix the errors and call submit_response again immediately."

type submitPayload struct {
	SubmissionID     string                   `json:"submission_id"`
	State            string                   `json:"state"`
	ReceiptToken     string                   `json:"receipt_token"`
	ValidationErrors []validationErrorPayload `json:"validation_errors"`
	// Guidance is non-empty only for an invalid submission, explaining the
	// state and the immediate-resubmit path.
	Guidance string `json:"guidance,omitempty"`
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
	ID                   string `json:"id"`
	TaskID               string `json:"task_id"`
	SubmitterID          string `json:"submitter_id"`
	SubmitterDisplayName string `json:"submitter_display_name"`
	State                string `json:"state"`
	CreatedAt            string `json:"created_at"`
}

type submissionsPayload struct {
	Submissions []submissionSummary `json:"submissions"`
	NextOffset  int                 `json:"next_offset"`
	// Total counts every row matching the filter, ignoring limit/offset.
	Total int64 `json:"total"`
}

// submissionDetail is get_submission's full read: everything a reviewer
// needs to judge the work, mirroring the REST submission response fields.
type submissionDetail struct {
	ID                   string                   `json:"id"`
	TaskID               string                   `json:"task_id"`
	SubmitterID          string                   `json:"submitter_id"`
	SubmitterDisplayName string                   `json:"submitter_display_name"`
	State                string                   `json:"state"`
	ResponseJSON         string                   `json:"response_json"`
	ReviewNote           string                   `json:"review_note"`
	Attachments          []attachmentPayload      `json:"attachments"`
	ValidationErrors     []validationErrorPayload `json:"validation_errors"`
	CreatedAt            string                   `json:"created_at"`
}

type acceptPayload struct {
	TaskID         string   `json:"task_id"`
	SubmissionID   string   `json:"submission_id"`
	PayoutKind     string   `json:"payout_kind"`
	PayoutAmount   int64    `json:"payout_amount"`
	WorkerUserID   string   `json:"worker_user_id"`
	CollectibleIDs []string `json:"collectible_ids"`
	TipAmount      int64    `json:"tip_amount"`
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
	// HolderDisplayName names the user who requested the reservation,
	// mirroring REST's reservation row enrichment.
	HolderDisplayName string `json:"holder_display_name"`
	// IssuedWorkerCredential is a one-time plaintext secret for a new
	// task-scoped agent credential, present only immediately after this
	// reservation was created into an active state.
	IssuedWorkerCredential string `json:"issued_worker_credential"`
}

type reservationPayload struct {
	Reservation reservationSummary `json:"reservation"`
}

type reservationsPayload struct {
	Reservations []reservationSummary `json:"reservations"`
	NextOffset   int                  `json:"next_offset"`
}

func (server Server) callListTasks(ctx context.Context, subject auth.Subject, arguments json.RawMessage) toolResult {
	var args struct {
		Scope string `json:"scope"`
		// State is the deprecated single-state alias kept for existing
		// callers; States is the REST-parity repeated filter.
		State               string   `json:"state"`
		States              []string `json:"states"`
		ParticipationPolicy string   `json:"participation_policy"`
		Query               string   `json:"query"`
		TaskType            string   `json:"task_type"`
		CreatedAfter        string   `json:"created_after"`
		Funded              string   `json:"funded"`
		Sort                string   `json:"sort"`
		Limit               int      `json:"limit"`
		Offset              int      `json:"offset"`
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
			return toolFailed{code: core.ErrorCodePermissionDenied, message: "scope \"user\" requires a personal agent credential, not an organization credential"}
		}
		scope = task.UserListScope{UserID: userActor.ID}
	default:
		return toolProtocolError{code: codeInvalidParams, message: "scope must be public or user"}
	}

	// Mirrors REST's parseTaskListFilters (internal/http/tasks.go): repeated
	// states, participation_policy, query, task_type, and sort.
	rawStates := args.States
	if args.State != "" {
		rawStates = append(rawStates, args.State)
	}
	filters := task.NoListFilters()
	if len(rawStates) > 0 {
		states := make([]task.State, len(rawStates))
		for index, rawState := range rawStates {
			stateResult := task.ParseState(rawState)
			stateAccepted, stateMatched := stateResult.(task.StateAccepted)
			if !stateMatched {
				return toolProtocolError{code: codeInvalidParams, message: stateResult.(task.StateRejected).Reason.Description()}
			}
			states[index] = stateAccepted.Value
		}
		if len(states) == 1 {
			filters.State = task.StateEquals{Value: states[0]}
		} else {
			filters.State = task.StateIn{Values: states}
		}
	}
	if args.ParticipationPolicy != "" {
		policyResult := task.ParseParticipationPolicy(args.ParticipationPolicy)
		policyAccepted, policyMatched := policyResult.(task.ParticipationPolicyAccepted)
		if !policyMatched {
			return toolProtocolError{code: codeInvalidParams, message: policyResult.(task.ParticipationPolicyRejected).Reason.Description()}
		}
		filters.Participation = task.ParticipationPolicyEquals{Value: policyAccepted.Value}
	}
	if args.Query != "" {
		searchResult := task.NewSearchText(args.Query)
		searchAccepted, searchMatched := searchResult.(task.SearchTextAccepted)
		if !searchMatched {
			return toolProtocolError{code: codeInvalidParams, message: searchResult.(task.SearchTextRejected).Reason.Description()}
		}
		filters.Search = task.SearchContains{Value: searchAccepted.Value}
	}
	if args.TaskType != "" {
		typeResult := task.ParseTaskType(args.TaskType)
		typeAccepted, typeMatched := typeResult.(task.TaskTypeAccepted)
		if !typeMatched {
			return toolProtocolError{code: codeInvalidParams, message: typeResult.(task.TaskTypeRejected).Reason.Description()}
		}
		filters.Type = task.TypeEquals{Value: typeAccepted.Value}
	}
	if args.Sort != "" {
		sortResult := task.ParseSortOrder(args.Sort)
		sortAccepted, sortMatched := sortResult.(task.SortOrderAccepted)
		if !sortMatched {
			return toolProtocolError{code: codeInvalidParams, message: sortResult.(task.SortOrderRejected).Reason.Description()}
		}
		filters.Sort = sortAccepted.Value
	}
	if args.CreatedAfter != "" {
		instant, err := time.Parse(time.RFC3339, args.CreatedAfter)
		if err != nil {
			return toolProtocolError{code: codeInvalidParams, message: "created_after must be an RFC3339 timestamp"}
		}
		filters.Created = task.CreatedAfter{Instant: instant.UTC()}
	}
	if args.Funded != "" {
		fundedResult := task.ParseFundedState(args.Funded)
		fundedParsed, fundedMatched := fundedResult.(task.FundedStateParsed)
		if !fundedMatched {
			return toolProtocolError{code: codeInvalidParams, message: fundedResult.(task.FundedStateRejected).Reason.Description()}
		}
		filters.Funded = task.FundedEquals{Value: fundedParsed.Value}
	}

	page, pageProblem := parseMCPPage(args.Limit, args.Offset)
	if pageProblem != nil {
		return pageProblem
	}

	result := server.services.ListTasks(ctx, subject, scope, filters, page.Probe())
	listed, matched := result.(task.TasksListed)
	if !matched {
		return toolFailed{code: result.(task.ListRejected).Reason.Code(), message: result.(task.ListRejected).Reason.Description()}
	}

	visible, nextOffset := core.ProbeListWindow(len(listed.Values), page)
	rows := make([]taskListRow, 0, visible)
	for index := range listed.Values[:visible] {
		rows = append(rows, listItemToRow(listed.Values[index]))
	}
	return marshalPayload(tasksPayload{Tasks: rows, NextOffset: nextOffset, Total: listed.Total})
}

// parseMCPPage maps optional limit/offset tool arguments onto core.Page.
// Both absent (zero) means the default page. A zero limit with an offset
// keeps the default limit; negatives are rejected like REST's page parsing.
func parseMCPPage(limit int, offset int) (core.Page, toolResult) {
	if limit == 0 && offset == 0 {
		return core.DefaultPage(), nil
	}
	effectiveLimit := limit
	if effectiveLimit == 0 {
		effectiveLimit = core.DefaultPage().Limit()
	}
	pageResult := core.NewPage(effectiveLimit, offset)
	page, matched := pageResult.(core.PageAccepted)
	if !matched {
		return core.Page{}, toolProtocolError{code: codeInvalidParams, message: pageResult.(core.PageRejected).Reason.Description()}
	}
	return page.Value, nil
}

// mcpPageArguments is the optional limit/offset pair shared by the list
// tools that take no other arguments beyond their subject id.
type mcpPageArguments struct {
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

// parseMCPPageArguments extracts the optional limit/offset arguments from a
// tool call whose other arguments are parsed separately.
func parseMCPPageArguments(arguments json.RawMessage) (core.Page, toolResult) {
	var args mcpPageArguments
	if err := json.Unmarshal(arguments, &args); err != nil {
		return core.Page{}, invalidArguments()
	}
	return parseMCPPage(args.Limit, args.Offset)
}

func (server Server) callGetTask(ctx context.Context, subject auth.UserSubject, arguments json.RawMessage) toolResult {
	taskID, problem := parseTaskID(arguments)
	if problem != nil {
		return problem
	}
	result := server.services.GetTask(ctx, subject, taskID)
	got, matched := result.(task.TaskGot)
	if !matched {
		return toolFailed{code: result.(task.GetRejected).Reason.Code(), message: result.(task.GetRejected).Reason.Description()}
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
		return toolFailed{code: result.(task.GetRejected).Reason.Code(), message: result.(task.GetRejected).Reason.Description()}
	}
	return marshalPayload(schemaPayload{TaskID: got.Value.ID.String(), ResponseSchemaJSON: got.Value.ResponseSchema.String()})
}

// mcpTaskOwnerArgs mirrors REST's taskOwnerRequest DTO field names.
type mcpTaskOwnerArgs struct {
	Kind           string `json:"kind"`
	UserID         string `json:"user_id"`
	TeamID         string `json:"team_id"`
	OrganizationID string `json:"organization_id"`
}

// mcpAttachmentArgs mirrors REST's attachmentRequest DTO field names.
type mcpAttachmentArgs struct {
	Name        string `json:"name"`
	ContentType string `json:"content_type"`
	DataURL     string `json:"data_url"`
}

func (server Server) callCreateTask(ctx context.Context, subject auth.UserSubject, arguments json.RawMessage) toolResult {
	var args struct {
		Title                    string              `json:"title"`
		Description              string              `json:"description"`
		ResponseSchemaJSON       string              `json:"response_schema_json"`
		Owner                    mcpTaskOwnerArgs    `json:"owner"`
		VisibilityKind           string              `json:"visibility_kind"`
		VisibilityUserID         string              `json:"visibility_user_id"`
		VisibilityTeamID         string              `json:"visibility_team_id"`
		VisibilityOrganizationID string              `json:"visibility_organization_id"`
		RewardKind               string              `json:"reward_kind"`
		RewardCreditAmount       int64               `json:"reward_credit_amount"`
		RewardCollectibleIDs     []string            `json:"reward_collectible_ids"`
		ParticipationPolicy      string              `json:"participation_policy"`
		AssigneeScope            string              `json:"assignee_scope"`
		ReservationExpiryHours   int                 `json:"reservation_expiry_hours"`
		TaskType                 string              `json:"task_type"`
		ReferenceURL             string              `json:"reference_url"`
		SeriesID                 string              `json:"series_id"`
		SeriesPosition           int                 `json:"series_position"`
		PayloadJSON              string              `json:"payload_json"`
		Attachments              []mcpAttachmentArgs `json:"attachments"`
		ExpiresAt                string              `json:"expires_at"`
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

	owner, ownerProblem := parseMCPTaskOwner(args.Owner, subject)
	if ownerProblem != nil {
		return ownerProblem
	}
	visibility, visibilityProblem := parseMCPTaskVisibility(args.VisibilityKind, args.VisibilityUserID, args.VisibilityTeamID, args.VisibilityOrganizationID, subject)
	if visibilityProblem != nil {
		return visibilityProblem
	}
	rewardResult := parseMCPReward(args.RewardKind, args.RewardCreditAmount, args.RewardCollectibleIDs)
	reward, rewardMatched := rewardResult.(mcpRewardAccepted)
	if !rewardMatched {
		return toolProtocolError{code: codeInvalidParams, message: rewardResult.(mcpRewardRejected).reason}
	}

	// Participation mirrors REST's parseTaskParticipationRequest: policy
	// defaults to open, assignee scope to user, and the reservation expiry
	// to the default TTL.
	participationRaw := args.ParticipationPolicy
	if participationRaw == "" {
		participationRaw = task.ParticipationPolicyOpen.String()
	}
	participationResult := task.ParseParticipationPolicy(participationRaw)
	participationAccepted, participationMatched := participationResult.(task.ParticipationPolicyAccepted)
	if !participationMatched {
		return toolProtocolError{code: codeInvalidParams, message: participationResult.(task.ParticipationPolicyRejected).Reason.Description()}
	}
	assigneeScopeRaw := args.AssigneeScope
	if assigneeScopeRaw == "" {
		assigneeScopeRaw = task.AssigneeScopeUser.String()
	}
	assigneeScopeResult := task.ParseAssigneeScope(assigneeScopeRaw)
	assigneeScopeAccepted, assigneeScopeMatched := assigneeScopeResult.(task.AssigneeScopeAccepted)
	if !assigneeScopeMatched {
		return toolProtocolError{code: codeInvalidParams, message: assigneeScopeResult.(task.AssigneeScopeRejected).Reason.Description()}
	}
	reservationTTL := task.DefaultReservationTTL()
	if args.ReservationExpiryHours != 0 {
		ttlResult := task.NewReservationTTL(args.ReservationExpiryHours)
		ttlAccepted, ttlMatched := ttlResult.(task.ReservationTTLAccepted)
		if !ttlMatched {
			return toolProtocolError{code: codeInvalidParams, message: ttlResult.(task.ReservationTTLRejected).Reason.Description()}
		}
		reservationTTL = ttlAccepted.Value
	}

	placement, placementProblem := parseMCPPlacement(args.SeriesID, args.SeriesPosition)
	if placementProblem != nil {
		return placementProblem
	}
	payload, payloadProblem := parseMCPPayload(args.PayloadJSON)
	if payloadProblem != nil {
		return payloadProblem
	}
	attachments, attachmentsProblem := parseMCPAttachments(args.Attachments)
	if attachmentsProblem != nil {
		return attachmentsProblem
	}
	expirationResult := task.ParseExpirationPolicy(args.ExpiresAt, time.Now().UTC())
	expirationAccepted, expirationMatched := expirationResult.(task.ExpirationPolicyAccepted)
	if !expirationMatched {
		return toolProtocolError{code: codeInvalidParams, message: expirationResult.(task.ExpirationPolicyRejected).Reason.Description()}
	}

	command := task.CreateCommand{
		Actor:              subject,
		Owner:              owner,
		Title:              titleAccepted.Value,
		Description:        descriptionAccepted.Value,
		Type:               taskTypeAccepted.Value,
		Reference:          referenceAccepted.Value,
		Reward:             reward.value,
		Participation:      participationAccepted.Value,
		AssigneeScope:      assigneeScopeAccepted.Value,
		ReservationTTL:     reservationTTL,
		Visibility:         visibility,
		Placement:          placement,
		ResponseSchema:     schemaSourceAccepted.Value,
		Payload:            payload,
		Attachments:        attachments,
		Expiration:         expirationAccepted.Value,
		FundCollectibleIDs: reward.collectibleIDs,
	}
	result := server.services.CreateTask(ctx, command)
	created, matched := result.(task.TaskCreated)
	if !matched {
		return toolFailed{code: result.(task.CreateRejected).Reason.Code(), message: result.(task.CreateRejected).Reason.Description()}
	}
	return marshalPayload(taskToDetail(created.Value))
}

// parseMCPTaskOwner mirrors REST's parseTaskOwnerRequest with one MCP
// ergonomic default: an absent owner object (or absent user_id on a
// user-kind owner) means the calling agent's own user.
func parseMCPTaskOwner(args mcpTaskOwnerArgs, subject auth.UserSubject) (task.Owner, toolResult) {
	switch args.Kind {
	case "", task.OwnerKindUser.String():
		if args.UserID == "" {
			return task.UserOwner{UserID: subject.ID}, nil
		}
		userIDResult := core.ParseUserID(args.UserID)
		userID, matched := userIDResult.(core.UserIDCreated)
		if !matched {
			return nil, invalidIDArgument("owner.user_id")
		}
		return task.UserOwner{UserID: userID.Value}, nil
	case task.OwnerKindTeam.String():
		teamIDResult := core.ParseTeamID(args.TeamID)
		teamID, matched := teamIDResult.(core.TeamIDCreated)
		if !matched {
			return nil, invalidIDArgument("owner.team_id")
		}
		return task.TeamOwner{TeamID: teamID.Value}, nil
	case task.OwnerKindOrganization.String():
		organizationIDResult := core.ParseOrganizationID(args.OrganizationID)
		organizationID, matched := organizationIDResult.(core.OrganizationIDCreated)
		if !matched {
			return nil, invalidIDArgument("owner.organization_id")
		}
		return task.OrganizationOwner{OrganizationID: organizationID.Value}, nil
	case task.OwnerKindOrganizationTeam.String():
		organizationIDResult := core.ParseOrganizationID(args.OrganizationID)
		organizationID, organizationMatched := organizationIDResult.(core.OrganizationIDCreated)
		if !organizationMatched {
			return nil, invalidIDArgument("owner.organization_id")
		}
		teamIDResult := core.ParseTeamID(args.TeamID)
		teamID, teamMatched := teamIDResult.(core.TeamIDCreated)
		if !teamMatched {
			return nil, invalidIDArgument("owner.team_id")
		}
		return task.OrganizationTeamOwner{OrganizationID: organizationID.Value, TeamID: teamID.Value}, nil
	default:
		return nil, toolProtocolError{code: codeInvalidParams, message: "task owner kind is invalid"}
	}
}

// mcpVisibilityKindRequiredMessage names every valid visibility_kind so a
// caller that omitted it can correct the call without guessing. An implicit
// default is deliberately not offered: it silently produced private
// ("invisible") tasks that never appeared in the marketplace.
const mcpVisibilityKindRequiredMessage = "visibility_kind is required: \"public\" (marketplace-visible to everyone), \"user\" (private to you/the named user), \"team\", \"organization\", or \"organization_team\""

// parseMCPTaskVisibility mirrors REST's parseTaskVisibilityRequest, except
// that the kind is required: there is no implicit owner-derived default. A
// user-kind visibility with no explicit id means the calling agent's user.
func parseMCPTaskVisibility(kind string, rawUserID string, rawTeamID string, rawOrganizationID string, subject auth.UserSubject) (task.Visibility, toolResult) {
	switch kind {
	case "":
		return nil, toolProtocolError{code: codeInvalidParams, message: mcpVisibilityKindRequiredMessage}
	case task.VisibilityKindPublic.String():
		return task.PublicVisibility{}, nil
	case task.VisibilityKindUser.String():
		if rawUserID == "" {
			return task.UserVisibility{UserID: subject.ID}, nil
		}
		userIDResult := core.ParseUserID(rawUserID)
		userID, matched := userIDResult.(core.UserIDCreated)
		if !matched {
			return nil, invalidIDArgument("visibility_user_id")
		}
		return task.UserVisibility{UserID: userID.Value}, nil
	case task.VisibilityKindTeam.String():
		teamIDResult := core.ParseTeamID(rawTeamID)
		teamID, matched := teamIDResult.(core.TeamIDCreated)
		if !matched {
			return nil, invalidIDArgument("visibility_team_id")
		}
		return task.TeamVisibility{TeamID: teamID.Value}, nil
	case task.VisibilityKindOrganization.String():
		organizationIDResult := core.ParseOrganizationID(rawOrganizationID)
		organizationID, matched := organizationIDResult.(core.OrganizationIDCreated)
		if !matched {
			return nil, invalidIDArgument("visibility_organization_id")
		}
		return task.OrganizationVisibility{OrganizationID: organizationID.Value}, nil
	case task.VisibilityKindOrganizationTeam.String():
		organizationIDResult := core.ParseOrganizationID(rawOrganizationID)
		organizationID, organizationMatched := organizationIDResult.(core.OrganizationIDCreated)
		if !organizationMatched {
			return nil, invalidIDArgument("visibility_organization_id")
		}
		teamIDResult := core.ParseTeamID(rawTeamID)
		teamID, teamMatched := teamIDResult.(core.TeamIDCreated)
		if !teamMatched {
			return nil, invalidIDArgument("visibility_team_id")
		}
		return task.OrganizationTeamVisibility{OrganizationID: organizationID.Value, TeamID: teamID.Value}, nil
	default:
		return nil, toolProtocolError{code: codeInvalidParams, message: mcpVisibilityKindRequiredMessage}
	}
}

// parseMCPPlacement maps the optional series arguments onto the placement
// sum: no series_id means a standalone task; a series_id targets an existing
// series at series_position (default 1), like REST's "existing_series"
// placement kind.
func parseMCPPlacement(rawSeriesID string, rawPosition int) (task.SeriesPlacement, toolResult) {
	if rawSeriesID == "" {
		return task.StandalonePlacement{}, nil
	}
	seriesIDResult := core.ParseTaskSeriesID(rawSeriesID)
	seriesID, seriesMatched := seriesIDResult.(core.TaskSeriesIDCreated)
	if !seriesMatched {
		return nil, invalidIDArgument("series_id")
	}
	position := rawPosition
	if position == 0 {
		position = 1
	}
	positionResult := task.NewSeriesPosition(position)
	positionAccepted, positionMatched := positionResult.(task.SeriesPositionAccepted)
	if !positionMatched {
		return nil, toolProtocolError{code: codeInvalidParams, message: positionResult.(task.SeriesPositionRejected).Reason.Description()}
	}
	return task.ExistingSeriesPlacement{SeriesID: seriesID.Value, Position: positionAccepted.Value}, nil
}

// parseMCPPayload maps the optional payload_json argument onto the payload
// sum, applying the same JSON validity check as REST's "json" payload kind.
func parseMCPPayload(rawPayloadJSON string) (task.DataPayload, toolResult) {
	if rawPayloadJSON == "" {
		return task.NoDataPayload{}, nil
	}
	if !json.Valid([]byte(rawPayloadJSON)) {
		return nil, toolProtocolError{code: codeInvalidParams, message: "task payload JSON is invalid"}
	}
	sourceResult := task.NewPayloadSource(rawPayloadJSON)
	source, matched := sourceResult.(task.PayloadSourceAccepted)
	if !matched {
		return nil, toolProtocolError{code: codeInvalidParams, message: sourceResult.(task.PayloadSourceRejected).Reason.Description()}
	}
	return task.JSONDataPayload{Source: source.Value}, nil
}

// parseMCPAttachments mirrors REST's attachmentsFromRequest: same DTO field
// names, same count bound, same per-attachment constructor.
func parseMCPAttachments(values []mcpAttachmentArgs) ([]attachment.Attachment, toolResult) {
	if len(values) > attachment.MaxCount {
		return nil, toolProtocolError{code: codeInvalidParams, message: "too many attachments"}
	}
	attachments := make([]attachment.Attachment, 0, len(values))
	for index := range values {
		result := attachment.NewAttachment(values[index].Name, values[index].ContentType, values[index].DataURL)
		accepted, matched := result.(attachment.AttachmentAccepted)
		if !matched {
			return nil, toolProtocolError{code: codeInvalidParams, message: result.(attachment.AttachmentRejected).Reason.Description()}
		}
		attachments = append(attachments, accepted.Value)
	}
	return attachments, nil
}

func (server Server) callOpenTask(ctx context.Context, subject auth.Subject, arguments json.RawMessage) toolResult {
	taskID, problem := parseTaskID(arguments)
	if problem != nil {
		return problem
	}
	result := server.services.OpenTask(ctx, subject, taskID)
	changed, matched := result.(task.TaskStateChanged)
	if !matched {
		return toolFailed{code: result.(task.ChangeStateRejected).Reason.Code(), message: result.(task.ChangeStateRejected).Reason.Description()}
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
		return toolFailed{code: result.(task.ChangeStateRejected).Reason.Code(), message: result.(task.ChangeStateRejected).Reason.Description()}
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
		return invalidIDArgument("task_id")
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
		return toolFailed{code: result.(ledger.FundRejected).Reason.Code(), message: result.(ledger.FundRejected).Reason.Description()}
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
		return invalidIDArgument("task_id")
	}
	keyResult := ledger.NewIdempotencyKey(args.IdempotencyKey)
	key, keyMatched := keyResult.(ledger.IdempotencyKeyAccepted)
	if !keyMatched {
		return toolProtocolError{code: codeInvalidParams, message: keyResult.(ledger.IdempotencyKeyRejected).Reason.Description()}
	}
	result := server.services.RefundTask(ctx, subject.ID, taskID.Value, key.Value)
	refunded, matched := result.(ledger.TaskRefunded)
	if !matched {
		return toolFailed{code: result.(ledger.RefundRejected).Reason.Code(), message: result.(ledger.RefundRejected).Reason.Description()}
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
	// collectibleIDs are the reward collectibles to escrow inside the create
	// transaction (CreateCommand.FundCollectibleIDs).
	collectibleIDs []core.CollectibleID
}

type mcpRewardRejected struct {
	reason string
}

func (mcpRewardAccepted) mcpRewardResult() {}

func (mcpRewardRejected) mcpRewardResult() {}

// parseMCPReward mirrors REST's parseTaskRewardRequest exactly: a
// collectible reward requires at least one collectible id and its count is
// len(ids); a bundle takes the credit amount plus optional ids (count
// defaults to 1 when none are named, matching the declare-now/fund-later
// flow REST supports).
func parseMCPReward(kind string, creditAmount int64, rawCollectibleIDs []string) mcpRewardResult {
	switch kind {
	case task.RewardKindNone.String():
		return mcpRewardAccepted{value: task.NoRewardSpec{}, collectibleIDs: []core.CollectibleID{}}
	case task.RewardKindCredit.String():
		amountResult := task.NewCreditRewardAmount(creditAmount)
		amount, matched := amountResult.(task.CreditRewardAmountAccepted)
		if !matched {
			return mcpRewardRejected{reason: amountResult.(task.CreditRewardAmountRejected).Reason.Description()}
		}
		return mcpRewardAccepted{value: task.CreditRewardSpec{Amount: amount.Value}, collectibleIDs: []core.CollectibleID{}}
	case task.RewardKindCollectible.String():
		collectibleIDs, idsProblem := parseMCPRewardCollectibleIDs(rawCollectibleIDs)
		if idsProblem != "" {
			return mcpRewardRejected{reason: idsProblem}
		}
		countResult := task.NewCollectibleRewardCount(len(collectibleIDs))
		count, countMatched := countResult.(task.CollectibleRewardCountAccepted)
		if !countMatched {
			return mcpRewardRejected{reason: countResult.(task.CollectibleRewardCountRejected).Reason.Description()}
		}
		return mcpRewardAccepted{value: task.CollectibleRewardSpec{Count: count.Value}, collectibleIDs: collectibleIDs}
	case task.RewardKindBundle.String():
		amountResult := task.NewCreditRewardAmount(creditAmount)
		amount, matched := amountResult.(task.CreditRewardAmountAccepted)
		if !matched {
			return mcpRewardRejected{reason: amountResult.(task.CreditRewardAmountRejected).Reason.Description()}
		}
		collectibleIDs := []core.CollectibleID{}
		countValue := 1
		if len(rawCollectibleIDs) > 0 {
			parsedIDs, idsProblem := parseMCPRewardCollectibleIDs(rawCollectibleIDs)
			if idsProblem != "" {
				return mcpRewardRejected{reason: idsProblem}
			}
			collectibleIDs = parsedIDs
			countValue = len(parsedIDs)
		}
		countResult := task.NewCollectibleRewardCount(countValue)
		count, countMatched := countResult.(task.CollectibleRewardCountAccepted)
		if !countMatched {
			return mcpRewardRejected{reason: countResult.(task.CollectibleRewardCountRejected).Reason.Description()}
		}
		return mcpRewardAccepted{value: task.BundleRewardSpec{Credit: amount.Value, Collectible: count.Value}, collectibleIDs: collectibleIDs}
	default:
		return mcpRewardRejected{reason: "reward_kind must be none, credit, collectible, or bundle"}
	}
}

// parseMCPRewardCollectibleIDs mirrors REST's parseRewardCollectibleIDs:
// at least one id, no duplicates, every id valid. A non-empty report string
// is the rejection reason.
func parseMCPRewardCollectibleIDs(rawIDs []string) ([]core.CollectibleID, string) {
	if len(rawIDs) == 0 {
		return nil, "at least one collectible is required for this reward"
	}
	values := make([]core.CollectibleID, 0, len(rawIDs))
	seen := make(map[string]bool, len(rawIDs))
	for _, rawID := range rawIDs {
		if seen[rawID] {
			return nil, "collectible reward ids must be unique"
		}
		idResult := core.ParseCollectibleID(rawID)
		id, matched := idResult.(core.CollectibleIDCreated)
		if !matched {
			return nil, "reward_collectible_ids must contain valid collectible ids"
		}
		seen[rawID] = true
		values = append(values, id.Value)
	}
	return values, ""
}

func (server Server) callSubmitResponse(ctx context.Context, subject auth.UserSubject, arguments json.RawMessage) toolResult {
	var args struct {
		TaskID       string              `json:"task_id"`
		ResponseJSON string              `json:"response_json"`
		Attachments  []mcpAttachmentArgs `json:"attachments"`
	}
	if err := json.Unmarshal(arguments, &args); err != nil {
		return invalidArguments()
	}
	taskIDResult := core.ParseTaskID(args.TaskID)
	taskID, taskMatched := taskIDResult.(core.TaskIDCreated)
	if !taskMatched {
		return invalidIDArgument("task_id")
	}
	sourceResult := submission.NewResponseSource(args.ResponseJSON)
	source, sourceMatched := sourceResult.(submission.ResponseSourceAccepted)
	if !sourceMatched {
		return toolProtocolError{code: codeInvalidParams, message: sourceResult.(submission.ResponseSourceRejected).Reason.Description()}
	}
	attachments, attachmentsProblem := parseMCPAttachments(args.Attachments)
	if attachmentsProblem != nil {
		return attachmentsProblem
	}

	command := submission.SubmitCommand{
		TaskID:         taskID.Value,
		SubmitterID:    subject.ID,
		ResponseSource: source.Value,
		Attachments:    attachments,
	}
	result := server.services.SubmitResponse(ctx, command)
	created, matched := result.(submission.SubmissionCreated)
	if !matched {
		return toolFailed{code: result.(submission.SubmitRejected).Reason.Code(), message: result.(submission.SubmitRejected).Reason.Description()}
	}
	payload := submitPayload{
		SubmissionID:     created.Value.ID.String(),
		State:            created.Value.State.String(),
		ReceiptToken:     created.ReceiptToken.String(),
		ValidationErrors: validationErrorsToPayload(created.Value.Validation),
	}
	if created.Value.State == submission.StateInvalid {
		payload.Guidance = invalidSubmissionGuidance
	}
	return marshalPayload(payload)
}

func (server Server) callGetSubmission(ctx context.Context, subject auth.Subject, arguments json.RawMessage) toolResult {
	var args struct {
		SubmissionID string `json:"submission_id"`
	}
	if err := json.Unmarshal(arguments, &args); err != nil {
		return invalidArguments()
	}
	submissionIDResult := core.ParseSubmissionID(args.SubmissionID)
	submissionID, submissionMatched := submissionIDResult.(core.SubmissionIDCreated)
	if !submissionMatched {
		return invalidIDArgument("submission_id")
	}
	result := server.services.GetSubmission(ctx, subject, submissionID.Value)
	got, matched := result.(submission.SubmissionGot)
	if !matched {
		return toolFailed{code: result.(submission.GetRejected).Reason.Code(), message: result.(submission.GetRejected).Reason.Description()}
	}
	return marshalPayload(submissionToDetail(got.Value))
}

func submissionToDetail(value submission.Submission) submissionDetail {
	return submissionDetail{
		ID:                   value.ID.String(),
		TaskID:               value.TaskID.String(),
		SubmitterID:          value.SubmitterID.String(),
		SubmitterDisplayName: value.SubmitterDisplayName.String(),
		State:                value.State.String(),
		ResponseJSON:         value.ResponseSource.String(),
		ReviewNote:           value.ReviewNote.String(),
		Attachments:          attachmentsToPayload(value.Attachments),
		ValidationErrors:     validationErrorsToPayload(value.Validation),
		CreatedAt:            submissionCreatedAtString(value),
	}
}

// submissionCreatedAtString renders the submission instant, empty for a
// value that never passed through a store read (fakes, pre-read models).
func submissionCreatedAtString(value submission.Submission) string {
	if value.CreatedAt.IsZero() {
		return ""
	}
	return value.CreatedAt.UTC().Format(time.RFC3339)
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
		return toolFailed{code: result.(submission.ReceiptStatusRejected).Reason.Code(), message: result.(submission.ReceiptStatusRejected).Reason.Description()}
	}
	return marshalPayload(statusPayload{
		SubmissionID: found.Value.ID.String(),
		TaskID:       found.Value.TaskID.String(),
		State:        found.Value.State.String(),
		ResponseJSON: found.Value.ResponseSource.String(),
		ReviewNote:   found.Value.ReviewNote.String(),
	})
}

func (server Server) callListTaskSubmissions(ctx context.Context, subject auth.Subject, arguments json.RawMessage) toolResult {
	taskID, problem := parseTaskID(arguments)
	if problem != nil {
		return problem
	}
	page, pageProblem := parseMCPPageArguments(arguments)
	if pageProblem != nil {
		return pageProblem
	}
	result := server.services.ListTaskSubmissions(ctx, subject, taskID, page.Probe())
	listed, matched := result.(submission.SubmissionsListed)
	if !matched {
		return toolFailed{code: result.(submission.ListRejected).Reason.Code(), message: result.(submission.ListRejected).Reason.Description()}
	}
	visible, nextOffset := core.ProbeListWindow(len(listed.Values), page)
	summaries := make([]submissionSummary, 0, visible)
	for index := range listed.Values[:visible] {
		summaries = append(summaries, submissionToSummary(listed.Values[index]))
	}
	return marshalPayload(submissionsPayload{Submissions: summaries, NextOffset: nextOffset, Total: listed.Total})
}

func (server Server) callAcceptSubmission(ctx context.Context, subject auth.Subject, arguments json.RawMessage) toolResult {
	var args struct {
		TaskID           string `json:"task_id"`
		SubmissionID     string `json:"submission_id"`
		IdempotencyKey   string `json:"idempotency_key"`
		PayoutAmount     int64  `json:"payout_amount"`
		TipAmount        int64  `json:"tip_amount"`
		TipCollectibleID string `json:"tip_collectible_id"`
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
	// The optional collectible tip mirrors REST's acceptSubmission handler
	// (internal/http/reviews.go): an absent id means no collectible tip.
	collectibleTip := ledger.CollectibleTipSelection(ledger.NoCollectibleTipSelection{})
	if args.TipCollectibleID != "" {
		collectibleIDResult := core.ParseCollectibleID(args.TipCollectibleID)
		collectibleIDAccepted, collectibleIDMatched := collectibleIDResult.(core.CollectibleIDCreated)
		if !collectibleIDMatched {
			return invalidIDArgument("tip_collectible_id")
		}
		collectibleTip = ledger.CollectibleTipSelected{ID: collectibleIDAccepted.Value}
	}

	reviewer, reviewerProblem := reviewerForSubject(subject)
	if reviewerProblem != nil {
		return reviewerProblem
	}
	result := server.services.ReviewAcceptSubmission(ctx, reviewer, ids.taskID, ids.submissionID, key.Value, creditSelection.value, tipSelection.value, collectibleTip)
	accepted, matched := result.(ledger.SubmissionAccepted)
	if !matched {
		return toolFailed{code: result.(ledger.AcceptRejected).Reason.Code(), message: result.(ledger.AcceptRejected).Reason.Description()}
	}
	// The payout/tip rendering mirrors REST's acceptToResponse
	// (internal/http/reviews.go), including the collectible outcomes.
	payload := acceptPayload{
		TaskID:         accepted.TaskID.String(),
		SubmissionID:   accepted.SubmissionID.String(),
		PayoutKind:     "none",
		CollectibleIDs: []string{},
	}
	switch payout := accepted.Payout.(type) {
	case ledger.CreditPayout:
		payload.PayoutKind = "credit"
		payload.PayoutAmount = payout.Amount.Int64()
		payload.WorkerUserID = payout.WorkerUserID.String()
	case ledger.CollectiblePayout:
		payload.PayoutKind = "collectible"
		payload.CollectibleIDs = collectibleIDsToStrings(payout.CollectibleIDs)
		payload.WorkerUserID = payout.WorkerUserID.String()
	case ledger.BundlePayout:
		payload.PayoutKind = "bundle"
		payload.PayoutAmount = payout.Amount.Int64()
		payload.CollectibleIDs = collectibleIDsToStrings(payout.CollectibleIDs)
		payload.WorkerUserID = payout.WorkerUserID.String()
	}
	switch tip := accepted.Tip.(type) {
	case ledger.CreditTip:
		payload.TipAmount = tip.Amount.Int64()
		if payload.WorkerUserID == "" {
			payload.WorkerUserID = tip.WorkerUserID.String()
		}
	case ledger.CollectibleTip:
		payload.CollectibleIDs = append(payload.CollectibleIDs, tip.CollectibleID.String())
		if payload.WorkerUserID == "" {
			payload.WorkerUserID = tip.WorkerUserID.String()
		}
		payload.PayoutKind = appendMCPCollectiblePayoutKind(payload.PayoutKind)
	case ledger.BundleTip:
		payload.TipAmount = tip.Amount.Int64()
		payload.CollectibleIDs = append(payload.CollectibleIDs, tip.CollectibleID.String())
		if payload.WorkerUserID == "" {
			payload.WorkerUserID = tip.WorkerUserID.String()
		}
		payload.PayoutKind = appendMCPCollectiblePayoutKind(payload.PayoutKind)
	}
	return marshalPayload(payload)
}

func collectibleIDsToStrings(ids []core.CollectibleID) []string {
	values := make([]string, 0, len(ids))
	for index := range ids {
		values = append(values, ids[index].String())
	}
	return values
}

// appendMCPCollectiblePayoutKind matches REST's appendCollectiblePayoutKind:
// a collectible tip upgrades the reported payout kind.
func appendMCPCollectiblePayoutKind(current string) string {
	if current == "none" {
		return "collectible"
	}
	if current == "credit" {
		return "bundle"
	}
	return current
}

// keyedReviewArguments is the (ids, idempotency key, required note) triple
// shared by the keyed review tools (request changes, reject).
type keyedReviewArguments struct {
	ids     parsedTaskSubmissionIDs
	key     ledger.IdempotencyKey
	note    submission.ReviewNote
	problem toolResult
}

// reviewerForSubject maps the authenticated MCP caller onto the ledger
// reviewer it acts as, mirroring REST's reviewerForSubject
// (internal/http/reviews.go): a personal agent credential reviews as its
// user, an organization credential reviews as the organization — full parity
// with an org-admin member on the organization's own tasks, except that tips
// and bans move personal value and are refused by the ledger service with a
// clear message.
func reviewerForSubject(subject auth.Subject) (ledger.Reviewer, toolResult) {
	switch typed := subject.(type) {
	case auth.UserSubject:
		return ledger.UserReviewer{ID: typed.ID}, nil
	case auth.OrgSubject:
		return ledger.OrganizationReviewer{ID: typed.ID}, nil
	default:
		return nil, toolFailed{code: core.ErrorCodePermissionDenied, message: "submission review requires a personal agent credential or an organization credential"}
	}
}

func parseKeyedReviewArguments(taskID string, submissionID string, rawKey string, rawNote string) keyedReviewArguments {
	ids := parseTaskSubmissionIDs(taskID, submissionID)
	if ids.problem != nil {
		return keyedReviewArguments{problem: ids.problem}
	}
	keyResult := ledger.NewIdempotencyKey(rawKey)
	key, keyMatched := keyResult.(ledger.IdempotencyKeyAccepted)
	if !keyMatched {
		return keyedReviewArguments{problem: toolProtocolError{code: codeInvalidParams, message: keyResult.(ledger.IdempotencyKeyRejected).Reason.Description()}}
	}
	noteResult := submission.NewRequiredReviewNote(rawNote)
	note, noteMatched := noteResult.(submission.ReviewNoteAccepted)
	if !noteMatched {
		return keyedReviewArguments{problem: toolProtocolError{code: codeInvalidParams, message: noteResult.(submission.ReviewNoteRejected).Reason.Description()}}
	}
	return keyedReviewArguments{ids: ids, key: key.Value, note: note.Value}
}

func (server Server) callRequestChanges(ctx context.Context, subject auth.Subject, arguments json.RawMessage) toolResult {
	var args struct {
		TaskID         string `json:"task_id"`
		SubmissionID   string `json:"submission_id"`
		IdempotencyKey string `json:"idempotency_key"`
		ReviewNote     string `json:"review_note"`
	}
	if err := json.Unmarshal(arguments, &args); err != nil {
		return invalidArguments()
	}
	parsed := parseKeyedReviewArguments(args.TaskID, args.SubmissionID, args.IdempotencyKey, args.ReviewNote)
	if parsed.problem != nil {
		return parsed.problem
	}
	reviewer, reviewerProblem := reviewerForSubject(subject)
	if reviewerProblem != nil {
		return reviewerProblem
	}
	result := server.services.RequestChanges(ctx, reviewer, parsed.ids.taskID, parsed.ids.submissionID, parsed.key, parsed.note)
	changed, matched := result.(ledger.ChangesRequested)
	if !matched {
		return toolFailed{code: result.(ledger.RequestChangesRejected).Reason.Code(), message: result.(ledger.RequestChangesRejected).Reason.Description()}
	}
	return marshalPayload(reviewPayload{
		TaskID:       changed.TaskID.String(),
		SubmissionID: changed.SubmissionID.String(),
		State:        "changes_requested",
		ReviewNote:   changed.ReviewNote,
		PayoutKind:   "none",
	})
}

func (server Server) callRejectSubmission(ctx context.Context, subject auth.Subject, arguments json.RawMessage) toolResult {
	var args struct {
		TaskID              string `json:"task_id"`
		SubmissionID        string `json:"submission_id"`
		IdempotencyKey      string `json:"idempotency_key"`
		ReviewNote          string `json:"review_note"`
		PartialCreditAmount int64  `json:"partial_credit_amount"`
		TipAmount           int64  `json:"tip_amount"`
		BanSelection        string `json:"ban_selection"`
	}
	if err := json.Unmarshal(arguments, &args); err != nil {
		return invalidArguments()
	}
	parsed := parseKeyedReviewArguments(args.TaskID, args.SubmissionID, args.IdempotencyKey, args.ReviewNote)
	if parsed.problem != nil {
		return parsed.problem
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
	var banSelection ledger.BanSelection
	switch args.BanSelection {
	case "", "none":
		banSelection = ledger.NoBanSelection{}
	case "ban_implementor":
		banSelection = ledger.BanImplementorSelection{}
	default:
		return toolProtocolError{code: codeInvalidParams, message: "ban_selection must be none or ban_implementor"}
	}
	reviewer, reviewerProblem := reviewerForSubject(subject)
	if reviewerProblem != nil {
		return reviewerProblem
	}
	result := server.services.RejectSubmission(ctx, reviewer, parsed.ids.taskID, parsed.ids.submissionID, parsed.key, parsed.note, creditSelection.value, tipSelection.value, banSelection)
	rejected, matched := result.(ledger.SubmissionRejected)
	if !matched {
		return toolFailed{code: result.(ledger.RejectRejected).Reason.Code(), message: result.(ledger.RejectRejected).Reason.Description()}
	}
	payload := reviewPayload{
		TaskID:       rejected.TaskID.String(),
		SubmissionID: rejected.SubmissionID.String(),
		State:        "rejected",
		ReviewNote:   parsed.note.String(),
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
	Comments   []seriesCommentSummary `json:"comments"`
	NextOffset int                    `json:"next_offset"`
}

type submissionCommentSummary struct {
	ID           string `json:"id"`
	SubmissionID string `json:"submission_id"`
	AuthorID     string `json:"author_user_id"`
	Body         string `json:"body"`
	CreatedAt    string `json:"created_at"`
}

type submissionCommentsPayload struct {
	Comments   []submissionCommentSummary `json:"comments"`
	NextOffset int                        `json:"next_offset"`
}

type seriesListPayload struct {
	Series     []seriesSummary `json:"series"`
	NextOffset int             `json:"next_offset"`
}

type seriesDetailPayload struct {
	Series seriesSummary `json:"series"`
	Tasks  []taskSummary `json:"tasks"`
}

func (server Server) callListTaskSeries(ctx context.Context, subject auth.UserSubject, arguments json.RawMessage) toolResult {
	page, pageProblem := parseMCPPageArguments(arguments)
	if pageProblem != nil {
		return pageProblem
	}
	result := server.services.ListSeries(ctx, subject, page.Probe())
	listed, matched := result.(task.SeriesListed)
	if !matched {
		return toolFailed{code: result.(task.ListSeriesRejected).Reason.Code(), message: result.(task.ListSeriesRejected).Reason.Description()}
	}
	visible, nextOffset := core.ProbeListWindow(len(listed.Values), page)
	summaries := make([]seriesSummary, 0, visible)
	for index := range listed.Values[:visible] {
		summaries = append(summaries, seriesToSummary(listed.Values[index]))
	}
	return marshalPayload(seriesListPayload{Series: summaries, NextOffset: nextOffset})
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
		return invalidIDArgument("series_id")
	}
	result := server.services.GetSeries(ctx, subject, seriesID.Value)
	got, matched := result.(task.SeriesGot)
	if !matched {
		return toolFailed{code: result.(task.GetSeriesRejected).Reason.Code(), message: result.(task.GetSeriesRejected).Reason.Description()}
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
		return toolFailed{code: result.(task.ReservationRejected).Reason.Code(), message: result.(task.ReservationRejected).Reason.Description()}
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
		return parsedReserveTaskArguments{problem: invalidIDArgument("task_id")}
	}

	switch args.AssigneeKind {
	case "", task.AssigneeScopeUser.String():
		return parsedReserveTaskArguments{taskID: taskID.Value, assigneeKind: task.AssigneeScopeUser.String()}
	case task.AssigneeScopeOrganizationTeam.String():
		organizationIDResult := core.ParseOrganizationID(args.OrganizationID)
		organizationID, organizationIDMatched := organizationIDResult.(core.OrganizationIDCreated)
		if !organizationIDMatched {
			return parsedReserveTaskArguments{problem: invalidIDArgument("organization_id")}
		}
		teamIDResult := core.ParseTeamID(args.TeamID)
		teamID, teamIDMatched := teamIDResult.(core.TeamIDCreated)
		if !teamIDMatched {
			return parsedReserveTaskArguments{problem: invalidIDArgument("team_id")}
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
	page, pageProblem := parseMCPPageArguments(arguments)
	if pageProblem != nil {
		return pageProblem
	}
	result := server.services.ListReservations(ctx, subject, taskID, page.Probe())
	listed, matched := result.(task.ReservationsListed)
	if !matched {
		return toolFailed{code: result.(task.ReservationsListRejected).Reason.Code(), message: result.(task.ReservationsListRejected).Reason.Description()}
	}
	visible, nextOffset := core.ProbeListWindow(len(listed.Values), page)
	reservations := make([]reservationSummary, 0, visible)
	for index := range listed.Values[:visible] {
		reservations = append(reservations, reservationToSummary(listed.Values[index], ""))
	}
	return marshalPayload(reservationsPayload{Reservations: reservations, NextOffset: nextOffset})
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
		return toolFailed{code: result.(task.ReservationStateChangeRejected).Reason.Code(), message: result.(task.ReservationStateChangeRejected).Reason.Description()}
	}
	return marshalPayload(reservationPayload{Reservation: reservationToSummary(changed.Value, "")})
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
		return parsedTaskReservationIDs{problem: invalidIDArgument("task_id")}
	}
	reservationIDResult := core.ParseTaskReservationID(args.ReservationID)
	reservationID, reservationMatched := reservationIDResult.(core.TaskReservationIDCreated)
	if !reservationMatched {
		return parsedTaskReservationIDs{problem: invalidIDArgument("reservation_id")}
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
		return core.TaskID{}, invalidIDArgument("task_id")
	}
	return taskID.Value, nil
}

func invalidArguments() toolResult {
	return toolProtocolError{code: codeInvalidParams, message: "tool arguments are invalid"}
}

// invalidIDArgument phrases a malformed id argument with the same
// "<argument> must be ..." shape the other input-shape rejections use. The
// domain id rejection carries the raw UUID library error (e.g. "invalid UUID
// length: 4"), which is not an agent-facing message, so every tool routes
// malformed ids through this instead.
func invalidIDArgument(argument string) toolResult {
	return toolProtocolError{code: codeInvalidParams, message: argument + " must be a valid id (a UUID)"}
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

func (submissionDetail) payloadValue() {}

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
		return invalidIDArgument("series_id")
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
		return invalidIDArgument("series_id")
	}
	order := make([]core.TaskID, 0, len(args.TaskIDs))
	for index := range args.TaskIDs {
		taskIDResult := core.ParseTaskID(args.TaskIDs[index])
		taskID, taskMatched := taskIDResult.(core.TaskIDCreated)
		if !taskMatched {
			return invalidIDArgument("task_id")
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
		return invalidIDArgument("series_id")
	}
	bodyResult := task.NewCommentBody(args.Body)
	body, bodyMatched := bodyResult.(task.CommentBodyAccepted)
	if !bodyMatched {
		return toolProtocolError{code: codeInvalidParams, message: bodyResult.(task.CommentBodyRejected).Reason.Description()}
	}
	result := server.services.AddSeriesComment(ctx, subject, seriesID.Value, body.Value)
	added, matched := result.(task.SeriesCommentAdded)
	if !matched {
		return toolFailed{code: result.(task.SeriesCommentRejected).Reason.Code(), message: result.(task.SeriesCommentRejected).Reason.Description()}
	}
	return marshalPayload(commentToSummary(added.Value))
}

func (server Server) callListSeriesComments(ctx context.Context, subject auth.UserSubject, arguments json.RawMessage) toolResult {
	seriesID, problem := parseSeriesID(arguments)
	if problem != nil {
		return problem
	}
	page, pageProblem := parseMCPPageArguments(arguments)
	if pageProblem != nil {
		return pageProblem
	}
	result := server.services.ListSeriesComments(ctx, subject, seriesID, page.Probe())
	listed, matched := result.(task.SeriesCommentsListed)
	if !matched {
		return toolFailed{code: result.(task.SeriesCommentsListRejected).Reason.Code(), message: result.(task.SeriesCommentsListRejected).Reason.Description()}
	}
	visible, nextOffset := core.ProbeListWindow(len(listed.Values), page)
	comments := make([]seriesCommentSummary, 0, visible)
	for index := range listed.Values[:visible] {
		comments = append(comments, commentToSummary(listed.Values[index]))
	}
	return marshalPayload(seriesCommentsPayload{Comments: comments, NextOffset: nextOffset})
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
		return invalidIDArgument("task_id")
	}
	bodyResult := task.NewCommentBody(args.Body)
	body, bodyMatched := bodyResult.(task.CommentBodyAccepted)
	if !bodyMatched {
		return toolProtocolError{code: codeInvalidParams, message: bodyResult.(task.CommentBodyRejected).Reason.Description()}
	}
	result := server.services.AddTaskComment(ctx, subject, taskID.Value, body.Value)
	added, matched := result.(task.TaskCommentAdded)
	if !matched {
		return toolFailed{code: result.(task.TaskCommentRejected).Reason.Code(), message: result.(task.TaskCommentRejected).Reason.Description()}
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
	page, pageProblem := parseMCPPageArguments(arguments)
	if pageProblem != nil {
		return pageProblem
	}
	result := server.services.ListTaskComments(ctx, subject, taskID, page.Probe())
	listed, matched := result.(task.TaskCommentsListed)
	if !matched {
		return toolFailed{code: result.(task.TaskCommentsListRejected).Reason.Code(), message: result.(task.TaskCommentsListRejected).Reason.Description()}
	}
	visible, nextOffset := core.ProbeListWindow(len(listed.Values), page)
	comments := make([]seriesCommentSummary, 0, visible)
	for index := range listed.Values[:visible] {
		comments = append(comments, seriesCommentSummary{
			ID:        listed.Values[index].ID.String(),
			AuthorID:  listed.Values[index].AuthorID.String(),
			Body:      listed.Values[index].Body.String(),
			CreatedAt: listed.Values[index].CreatedAt.UTC().Format(time.RFC3339),
		})
	}
	return marshalPayload(seriesCommentsPayload{Comments: comments, NextOffset: nextOffset})
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
		return invalidIDArgument("submission_id")
	}
	bodyResult := task.NewCommentBody(args.Body)
	body, bodyMatched := bodyResult.(task.CommentBodyAccepted)
	if !bodyMatched {
		return toolProtocolError{code: codeInvalidParams, message: bodyResult.(task.CommentBodyRejected).Reason.Description()}
	}
	result := server.services.AddSubmissionComment(ctx, subject, submissionID.Value, body.Value)
	added, matched := result.(submission.SubmissionCommentAdded)
	if !matched {
		return toolFailed{code: result.(submission.SubmissionCommentRejected).Reason.Code(), message: result.(submission.SubmissionCommentRejected).Reason.Description()}
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
		return invalidIDArgument("submission_id")
	}
	page, pageProblem := parseMCPPageArguments(arguments)
	if pageProblem != nil {
		return pageProblem
	}
	result := server.services.ListSubmissionComments(ctx, subject, submissionID.Value, page.Probe())
	listed, matched := result.(submission.SubmissionCommentsListed)
	if !matched {
		return toolFailed{code: result.(submission.SubmissionCommentsListRejected).Reason.Code(), message: result.(submission.SubmissionCommentsListRejected).Reason.Description()}
	}
	visible, nextOffset := core.ProbeListWindow(len(listed.Values), page)
	comments := make([]submissionCommentSummary, 0, visible)
	for index := range listed.Values[:visible] {
		comments = append(comments, submissionCommentToSummary(listed.Values[index]))
	}
	return marshalPayload(submissionCommentsPayload{Comments: comments, NextOffset: nextOffset})
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
		return toolFailed{code: result.(task.ChangeStateRejected).Reason.Code(), message: result.(task.ChangeStateRejected).Reason.Description()}
	}
	return marshalPayload(taskToDetail(changed.Value))
}

func (server Server) seriesMutationResult(result task.SeriesMutationResult) toolResult {
	mutated, matched := result.(task.SeriesMutated)
	if !matched {
		return toolFailed{code: result.(task.SeriesMutationRejected).Reason.Code(), message: result.(task.SeriesMutationRejected).Reason.Description()}
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
		return core.TaskSeriesID{}, invalidIDArgument("series_id")
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
		return core.TaskSeriesID{}, core.TaskID{}, invalidIDArgument("series_id")
	}
	taskResult := core.ParseTaskID(args.TaskID)
	taskID, taskMatched := taskResult.(core.TaskIDCreated)
	if !taskMatched {
		return core.TaskSeriesID{}, core.TaskID{}, invalidIDArgument("task_id")
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
		return parsedTaskSubmissionIDs{problem: invalidIDArgument("task_id")}
	}
	submissionIDResult := core.ParseSubmissionID(rawSubmissionID)
	submissionID, submissionMatched := submissionIDResult.(core.SubmissionIDCreated)
	if !submissionMatched {
		return parsedTaskSubmissionIDs{problem: invalidIDArgument("submission_id")}
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

// listItemToRow flattens a task-listing read model onto the wire row,
// mirroring REST's taskListItemToResponse enrichment fields.
func listItemToRow(item task.ListItem) taskListRow {
	row := taskListRow{
		taskSummary:        taskToSummary(item.Task),
		CreatorDisplayName: item.CreatorDisplayName.String(),
		Funded:             item.Funded.String(),
		PendingReviewCount: item.PendingReviewCount,
	}
	if named, matched := item.HolderDisplayName.(task.HolderNamed); matched {
		row.HolderDisplayName = named.DisplayName.String()
	}
	return row
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
		ExpiresAt:          task.ExpirationInstantString(value.Expiration),
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
		ID:                   value.ID.String(),
		TaskID:               value.TaskID.String(),
		SubmitterID:          value.SubmitterID.String(),
		SubmitterDisplayName: value.SubmitterDisplayName.String(),
		State:                value.State.String(),
		CreatedAt:            submissionCreatedAtString(value),
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
		HolderDisplayName:      value.HolderDisplayName.String(),
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
