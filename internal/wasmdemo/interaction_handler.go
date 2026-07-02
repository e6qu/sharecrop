package wasmdemo

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"time"
)

type InteractionIDSource interface {
	NextSubmissionID() string
	NextCommentID() string
	NextReservationID() string
	NextLedgerEntryID() string
}

type InteractionHandler struct {
	storage BrowserStorage
	clock   HandlerClock
	actor   HandlerActor
	ids     InteractionIDSource
}

func NewInteractionHandler(storage BrowserStorage, clock HandlerClock, actor HandlerActor, ids InteractionIDSource) InteractionHandler {
	return InteractionHandler{storage: storage, clock: clock, actor: actor, ids: ids}
}

func (handler InteractionHandler) Handle(request Request) HandleResult {
	if handler.storage == nil {
		return RequestHandleRejected{Reason: "browser storage is required"}
	}
	if handler.clock == nil {
		return RequestHandleRejected{Reason: "handler clock is required"}
	}
	if handler.actor == nil {
		return RequestHandleRejected{Reason: "handler actor is required"}
	}
	switch {
	case taskCommentsPathID(request.Path) != "":
		return handler.handleTaskComments(request, taskCommentsPathID(request.Path))
	case submissionCommentsPathID(request.Path) != "":
		return handler.handleSubmissionComments(request, submissionCommentsPathID(request.Path))
	case taskReservationsPath(request.Path).taskID != "":
		return handler.handleReservations(request, taskReservationsPath(request.Path))
	case taskSubmissionsPath(request.Path).taskID != "":
		return handler.handleTaskSubmissions(request, taskSubmissionsPath(request.Path))
	case userSubmissionsPath(request.Path) != "":
		return handler.handleUserSubmissions(request, userSubmissionsPath(request.Path))
	case creditsPathOnly(request.Path) == "/api/credits/balance":
		return handler.handleLedgerBalance(request, "user", handler.actor.UserID())
	case creditsPathOnly(request.Path) == "/api/credits/ledger":
		return handler.handleLedgerEntries(request, "user", handler.actor.UserID())
	case organizationCreditsPath(request.Path).organizationID != "":
		route := organizationCreditsPath(request.Path)
		if route.kind == "balance" {
			return handler.handleLedgerBalance(request, "organization", route.organizationID)
		}
		if route.kind == "ledger" {
			return handler.handleLedgerEntries(request, "organization", route.organizationID)
		}
	}
	return RequestHandleRejected{Reason: "request route is not implemented by the WASM demo handler"}
}

func (handler InteractionHandler) handleTaskComments(request Request, taskID string) HandleResult {
	switch request.Method.String() {
	case MethodPost.String():
		if handler.ids == nil {
			return RequestHandleRejected{Reason: "interaction id source is required"}
		}
		var body commentBody
		if err := json.Unmarshal([]byte(request.Body), &body); err != nil {
			return RequestHandleRejected{Reason: "task comment body is invalid"}
		}
		comment := StoredComment{
			ID:           strings.TrimSpace(handler.ids.NextCommentID()),
			ParentKind:   "task",
			ParentID:     strings.TrimSpace(taskID),
			AuthorUserID: strings.TrimSpace(handler.actor.UserID()),
			Body:         strings.TrimSpace(body.Body),
			CreatedAt:    handler.clock.Now().UTC().Format(time.RFC3339),
		}
		saveResult := SaveComment(handler.storage, comment)
		saved, savedMatched := saveResult.(CommentStored)
		if !savedMatched {
			return RequestHandleRejected{Reason: saveResult.(CommentStorageRejected).Reason}
		}
		return taskCommentResponseResult(saved.Value, 201)
	case MethodGet.String():
		listResult := ListComments(handler.storage, "task", taskID)
		listed, listedMatched := listResult.(CommentsStored)
		if !listedMatched {
			return RequestHandleRejected{Reason: listResult.(CommentStorageRejected).Reason}
		}
		return taskCommentsResponseResult(listed.Values)
	default:
		return RequestHandleRejected{Reason: "request method is unsupported for task comments"}
	}
}

func (handler InteractionHandler) handleSubmissionComments(request Request, submissionID string) HandleResult {
	switch request.Method.String() {
	case MethodPost.String():
		if handler.ids == nil {
			return RequestHandleRejected{Reason: "interaction id source is required"}
		}
		if _, matched := LoadSubmission(handler.storage, submissionID).(SubmissionStored); !matched {
			return RequestHandleRejected{Reason: "submission was not found"}
		}
		var body commentBody
		if err := json.Unmarshal([]byte(request.Body), &body); err != nil {
			return RequestHandleRejected{Reason: "submission comment body is invalid"}
		}
		comment := StoredComment{
			ID:           strings.TrimSpace(handler.ids.NextCommentID()),
			ParentKind:   "submission",
			ParentID:     strings.TrimSpace(submissionID),
			AuthorUserID: strings.TrimSpace(handler.actor.UserID()),
			Body:         strings.TrimSpace(body.Body),
			CreatedAt:    handler.clock.Now().UTC().Format(time.RFC3339),
		}
		saveResult := SaveComment(handler.storage, comment)
		saved, savedMatched := saveResult.(CommentStored)
		if !savedMatched {
			return RequestHandleRejected{Reason: saveResult.(CommentStorageRejected).Reason}
		}
		if err := handler.notifySubmissionComment(submissionID); err != nil {
			return RequestHandleRejected{Reason: err.Error()}
		}
		return submissionCommentResponseResult(saved.Value, 201)
	case MethodGet.String():
		listResult := ListComments(handler.storage, "submission", submissionID)
		listed, listedMatched := listResult.(CommentsStored)
		if !listedMatched {
			return RequestHandleRejected{Reason: listResult.(CommentStorageRejected).Reason}
		}
		return submissionCommentsResponseResult(listed.Values)
	default:
		return RequestHandleRejected{Reason: "request method is unsupported for submission comments"}
	}
}

func (handler InteractionHandler) notifySubmissionComment(submissionID string) error {
	notificationIDs, matched := handler.ids.(interface{ NextNotificationID() string })
	if !matched {
		return errString("notification id source is required")
	}
	submissionResult := LoadSubmission(handler.storage, submissionID)
	submission, submissionMatched := submissionResult.(SubmissionStored)
	if !submissionMatched {
		return errString(submissionResult.(SubmissionStorageRejected).Reason)
	}
	taskResult := LoadTask(handler.storage, submission.Value.TaskID)
	task, taskMatched := taskResult.(TaskStored)
	if !taskMatched {
		return errString(taskResult.(TaskStorageRejected).Reason)
	}
	actorID := strings.TrimSpace(handler.actor.UserID())
	recipientID := submission.Value.SubmitterID
	if recipientID == actorID {
		recipientID = task.Value.CreatedBy
	}
	return saveInteractionNotification(
		handler.storage,
		notificationIDs.NextNotificationID(),
		recipientID,
		actorID,
		"submission_commented",
		"submission",
		submissionID,
		`{"task_id":"`+submission.Value.TaskID+`"}`,
		handler.clock.Now().UTC().Format(time.RFC3339),
	)
}

func saveInteractionNotification(storage BrowserStorage, id string, recipientID string, actorID string, kind string, subjectKind string, subjectID string, metadataJSON string, createdAt string) error {
	result := SaveNotification(storage, StoredNotification{
		ID:              id,
		RecipientUserID: recipientID,
		ActorUserID:     actorID,
		Kind:            kind,
		SubjectKind:     subjectKind,
		SubjectID:       subjectID,
		State:           "unread",
		MetadataJSON:    metadataJSON,
		CreatedAt:       createdAt,
	})
	if _, matched := result.(NotificationStored); !matched {
		return errString(result.(NotificationStorageRejected).Reason)
	}
	return nil
}

func (handler InteractionHandler) handleReservations(request Request, route taskReservationsRoute) HandleResult {
	if route.reservationID == "" {
		switch request.Method.String() {
		case MethodPost.String():
			return handler.handleCreateReservation(request, route.taskID)
		case MethodGet.String():
			pageResult := storedListPageFromPath(request.Path, "reservation")
			page, pageMatched := pageResult.(storedListPageFromPathAccepted)
			if !pageMatched {
				return RequestHandleRejected{Reason: pageResult.(storedListPageFromPathRejected).reason}
			}
			listResult := ListTaskReservations(handler.storage, route.taskID, page.value)
			listed, listedMatched := listResult.(ReservationsStored)
			if !listedMatched {
				return RequestHandleRejected{Reason: listResult.(ReservationStorageRejected).Reason}
			}
			return reservationsResponseResult(listed.Values)
		default:
			return RequestHandleRejected{Reason: "request method is unsupported for reservations"}
		}
	}
	if request.Method.String() != MethodPost.String() {
		return RequestHandleRejected{Reason: "request method is unsupported for reservation transition"}
	}
	nextState := reservationActionState(route.action)
	if nextState == "" {
		return RequestHandleRejected{Reason: "reservation action is unsupported"}
	}
	transitionResult := TransitionReservation(handler.storage, route.taskID, route.reservationID, nextState)
	transitioned, transitionedMatched := transitionResult.(ReservationStored)
	if !transitionedMatched {
		return RequestHandleRejected{Reason: transitionResult.(ReservationStorageRejected).Reason}
	}
	return reservationResponseResult(transitioned.Value, 200)
}

func (handler InteractionHandler) handleCreateReservation(request Request, taskID string) HandleResult {
	if handler.ids == nil {
		return RequestHandleRejected{Reason: "interaction id source is required"}
	}
	loadResult := LoadTask(handler.storage, taskID)
	loaded, loadedMatched := loadResult.(TaskStored)
	if !loadedMatched {
		return RequestHandleRejected{Reason: loadResult.(TaskStorageRejected).Reason}
	}
	var body reservationBody
	if err := json.Unmarshal([]byte(request.Body), &body); err != nil {
		return RequestHandleRejected{Reason: "reservation body is invalid"}
	}
	assignee := reservationAssigneeFromBody(body, handler.actor.UserID())
	state := "active"
	if loaded.Value.ParticipationPolicy == "approval_required" {
		state = "requested"
	}
	reservation := StoredReservation{
		ID:           strings.TrimSpace(handler.ids.NextReservationID()),
		TaskID:       strings.TrimSpace(taskID),
		AssigneeKind: assignee.kind,
		AssigneeID:   assignee.id,
		State:        state,
		RequestedBy:  strings.TrimSpace(handler.actor.UserID()),
	}
	saveResult := SaveReservation(handler.storage, reservation)
	saved, savedMatched := saveResult.(ReservationStored)
	if !savedMatched {
		return RequestHandleRejected{Reason: saveResult.(ReservationStorageRejected).Reason}
	}
	return reservationResponseResult(saved.Value, 201)
}

func (handler InteractionHandler) handleTaskSubmissions(request Request, route taskSubmissionsRoute) HandleResult {
	if route.submissionID != "" {
		if route.action == "accept" {
			if request.Method.String() != MethodPost.String() {
				return RequestHandleRejected{Reason: "request method is unsupported for submission acceptance"}
			}
			return handler.handleAcceptSubmission(request, route.taskID, route.submissionID)
		}
		return RequestHandleRejected{Reason: "submission action is unsupported"}
	}
	switch request.Method.String() {
	case MethodPost.String():
		return handler.handleCreateSubmission(request, route.taskID)
	case MethodGet.String():
		pageResult := storedListPageFromPath(request.Path, "submission")
		page, pageMatched := pageResult.(storedListPageFromPathAccepted)
		if !pageMatched {
			return RequestHandleRejected{Reason: pageResult.(storedListPageFromPathRejected).reason}
		}
		listResult := ListTaskSubmissions(handler.storage, route.taskID, page.value)
		listed, listedMatched := listResult.(SubmissionsStored)
		if !listedMatched {
			return RequestHandleRejected{Reason: listResult.(SubmissionStorageRejected).Reason}
		}
		return submissionsResponseResult(listed.Values)
	default:
		return RequestHandleRejected{Reason: "request method is unsupported for submissions"}
	}
}

func (handler InteractionHandler) handleCreateSubmission(request Request, taskID string) HandleResult {
	if handler.ids == nil {
		return RequestHandleRejected{Reason: "interaction id source is required"}
	}
	taskResult := LoadTask(handler.storage, taskID)
	taskLoaded, taskMatched := taskResult.(TaskStored)
	if !taskMatched {
		return RequestHandleRejected{Reason: taskResult.(TaskStorageRejected).Reason}
	}
	var body submissionBody
	if err := json.Unmarshal([]byte(request.Body), &body); err != nil {
		return RequestHandleRejected{Reason: "submission body is invalid"}
	}
	submissionID := strings.TrimSpace(handler.ids.NextSubmissionID())
	attachmentsResult := attachmentsFromSubmissionBody(body.Attachments, submissionID)
	attachments, attachmentsMatched := attachmentsResult.(submissionAttachmentsAccepted)
	if !attachmentsMatched {
		return RequestHandleRejected{Reason: attachmentsResult.(submissionAttachmentsRejected).reason}
	}
	submission := StoredSubmission{
		ID:               submissionID,
		TaskID:           strings.TrimSpace(taskID),
		SubmitterID:      strings.TrimSpace(handler.actor.UserID()),
		State:            submissionStateForResponse(taskLoaded.Value, body.ResponseJSON),
		ResponseJSON:     strings.TrimSpace(body.ResponseJSON),
		ReviewNote:       "",
		Attachments:      attachments.values,
		ValidationErrors: validationErrorsForResponse(taskLoaded.Value, body.ResponseJSON),
		SensitiveFields:  sensitiveFieldsForTask(taskLoaded.Value),
	}
	saveResult := SaveSubmission(handler.storage, submission)
	saved, savedMatched := saveResult.(SubmissionStored)
	if !savedMatched {
		return RequestHandleRejected{Reason: saveResult.(SubmissionStorageRejected).Reason}
	}
	saveAttachmentsResult := SaveAttachments(handler.storage, "submission", saved.Value.ID, attachments.values)
	if _, matched := saveAttachmentsResult.(AttachmentsStored); !matched {
		return RequestHandleRejected{Reason: saveAttachmentsResult.(AttachmentStorageRejected).Reason}
	}
	if err := handler.notifySubmissionCreated(saved.Value); err != nil {
		return RequestHandleRejected{Reason: err.Error()}
	}
	encoded, err := json.Marshal(submissionCreatedBody{Submission: saved.Value, ReceiptToken: "wasm-" + saved.Value.ID})
	if err != nil {
		return RequestHandleRejected{Reason: "submission response encoding failed"}
	}
	return RequestHandled{Value: Response{Status: 201, Body: string(encoded)}}
}

func submissionStateForResponse(task StoredTask, responseJSON string) string {
	if len(validationErrorsForResponse(task, responseJSON)) > 0 {
		return "invalid"
	}
	return "submitted"
}

func validationErrorsForResponse(task StoredTask, responseJSON string) []StoredSubmissionValidationError {
	var schema sensitiveObjectSchema
	if err := json.Unmarshal([]byte(task.ResponseSchemaJSON), &schema); err != nil || schema.Kind != "object" {
		return []StoredSubmissionValidationError{}
	}
	var response map[string]json.RawMessage
	if err := json.Unmarshal([]byte(strings.TrimSpace(responseJSON)), &response); err != nil {
		return []StoredSubmissionValidationError{{Path: "response", Message: "response must be valid JSON"}}
	}
	errors := []StoredSubmissionValidationError{}
	for index := range schema.Fields {
		if strings.TrimSpace(schema.Fields[index].Name) == "" {
			continue
		}
		if _, ok := response[schema.Fields[index].Name]; !ok {
			errors = append(errors, StoredSubmissionValidationError{Path: schema.Fields[index].Name, Message: "field is required"})
		}
	}
	return errors
}

func (handler InteractionHandler) notifySubmissionCreated(submission StoredSubmission) error {
	notificationIDs, matched := handler.ids.(interface{ NextNotificationID() string })
	if !matched {
		return errString("notification id source is required")
	}
	taskResult := LoadTask(handler.storage, submission.TaskID)
	task, taskMatched := taskResult.(TaskStored)
	if !taskMatched {
		return errString(taskResult.(TaskStorageRejected).Reason)
	}
	if task.Value.CreatedBy == submission.SubmitterID {
		return nil
	}
	return saveInteractionNotification(
		handler.storage,
		notificationIDs.NextNotificationID(),
		task.Value.CreatedBy,
		submission.SubmitterID,
		"submission_created",
		"submission",
		submission.ID,
		`{"task_id":"`+submission.TaskID+`"}`,
		handler.clock.Now().UTC().Format(time.RFC3339),
	)
}

func sensitiveFieldsForTask(task StoredTask) []StoredSubmissionSensitiveField {
	var schema sensitiveObjectSchema
	if err := json.Unmarshal([]byte(task.ResponseSchemaJSON), &schema); err != nil {
		return []StoredSubmissionSensitiveField{}
	}
	if schema.Kind != "object" {
		return []StoredSubmissionSensitiveField{}
	}
	fields := make([]StoredSubmissionSensitiveField, 0, len(schema.Fields))
	for index := range schema.Fields {
		sensitivity := schema.Fields[index].Schema.Sensitivity
		if sensitivity.Category == "" {
			continue
		}
		fields = append(fields, StoredSubmissionSensitiveField{
			Path:       schema.Fields[index].Name,
			Category:   sensitivity.Category,
			Retention:  sensitivity.Retention,
			Redaction:  sensitivity.Redaction,
			State:      "active",
			RedactedAt: "",
		})
	}
	return fields
}

func (handler InteractionHandler) handleAcceptSubmission(request Request, taskID string, submissionID string) HandleResult {
	if handler.ids == nil {
		return RequestHandleRejected{Reason: "interaction id source is required"}
	}
	loadResult := LoadSubmission(handler.storage, submissionID)
	loaded, loadedMatched := loadResult.(SubmissionStored)
	if !loadedMatched {
		return RequestHandleRejected{Reason: loadResult.(SubmissionStorageRejected).Reason}
	}
	if loaded.Value.TaskID != strings.TrimSpace(taskID) {
		return RequestHandleRejected{Reason: "submission does not belong to task"}
	}
	var body acceptSubmissionBody
	if err := json.Unmarshal([]byte(request.Body), &body); err != nil {
		return RequestHandleRejected{Reason: "submission acceptance body is invalid"}
	}
	taskResult := LoadTask(handler.storage, taskID)
	taskLoaded, taskMatched := taskResult.(TaskStored)
	if !taskMatched {
		return RequestHandleRejected{Reason: taskResult.(TaskStorageRejected).Reason}
	}
	payout := body.PayoutAmount
	if payout == 0 {
		payout = taskLoaded.Value.RewardCreditAmount
	}
	loaded.Value.State = "accepted"
	saveResult := SaveSubmission(handler.storage, loaded.Value)
	if _, savedMatched := saveResult.(SubmissionStored); !savedMatched {
		return RequestHandleRejected{Reason: saveResult.(SubmissionStorageRejected).Reason}
	}
	total := payout + body.TipAmount
	if total > 0 {
		ledgerResult := SaveLedgerEntry(handler.storage, StoredLedgerEntry{
			ID:        strings.TrimSpace(handler.ids.NextLedgerEntryID()),
			OwnerKind: "user",
			OwnerID:   loaded.Value.SubmitterID,
			Kind:      "task_payout",
			Amount:    total,
			TaskID:    taskID,
		})
		if _, ledgerMatched := ledgerResult.(LedgerEntryStored); !ledgerMatched {
			return RequestHandleRejected{Reason: ledgerResult.(LedgerStorageRejected).Reason}
		}
	}
	if err := handler.notifySubmissionAccepted(loaded.Value); err != nil {
		return RequestHandleRejected{Reason: err.Error()}
	}
	encoded, err := json.Marshal(acceptSubmissionResultBody{
		TaskID:         taskID,
		SubmissionID:   submissionID,
		PayoutKind:     payoutKind(total),
		PayoutAmount:   payout,
		WorkerUserID:   loaded.Value.SubmitterID,
		CollectibleIDs: []string{},
		TipAmount:      body.TipAmount,
	})
	if err != nil {
		return RequestHandleRejected{Reason: "submission acceptance response encoding failed"}
	}
	return RequestHandled{Value: Response{Status: 200, Body: string(encoded)}}
}

func (handler InteractionHandler) notifySubmissionAccepted(submission StoredSubmission) error {
	notificationIDs, matched := handler.ids.(interface{ NextNotificationID() string })
	if !matched {
		return errString("notification id source is required")
	}
	return saveInteractionNotification(
		handler.storage,
		notificationIDs.NextNotificationID(),
		submission.SubmitterID,
		handler.actor.UserID(),
		"submission_accepted",
		"submission",
		submission.ID,
		`{"task_id":"`+submission.TaskID+`"}`,
		handler.clock.Now().UTC().Format(time.RFC3339),
	)
}

func (handler InteractionHandler) handleUserSubmissions(request Request, userID string) HandleResult {
	if request.Method.String() != MethodGet.String() {
		return RequestHandleRejected{Reason: "request method is unsupported for user submissions"}
	}
	pageResult := storedListPageFromPath(request.Path, "submission")
	page, pageMatched := pageResult.(storedListPageFromPathAccepted)
	if !pageMatched {
		return RequestHandleRejected{Reason: pageResult.(storedListPageFromPathRejected).reason}
	}
	listResult := ListUserSubmissions(handler.storage, userID, page.value)
	listed, listedMatched := listResult.(SubmissionsStored)
	if !listedMatched {
		return RequestHandleRejected{Reason: listResult.(SubmissionStorageRejected).Reason}
	}
	return submissionsResponseResult(listed.Values)
}

func (handler InteractionHandler) handleLedgerBalance(request Request, ownerKind string, ownerID string) HandleResult {
	if request.Method.String() != MethodGet.String() {
		return RequestHandleRejected{Reason: "request method is unsupported for ledger balance"}
	}
	balanceResult := LedgerBalance(handler.storage, ownerKind, ownerID)
	balance, balanceMatched := balanceResult.(LedgerBalanceStored)
	if !balanceMatched {
		return RequestHandleRejected{Reason: balanceResult.(LedgerStorageRejected).Reason}
	}
	encoded, err := json.Marshal(balanceBody{Amount: balance.Amount})
	if err != nil {
		return RequestHandleRejected{Reason: "ledger balance response encoding failed"}
	}
	return RequestHandled{Value: Response{Status: 200, Body: string(encoded)}}
}

func (handler InteractionHandler) handleLedgerEntries(request Request, ownerKind string, ownerID string) HandleResult {
	if request.Method.String() != MethodGet.String() {
		return RequestHandleRejected{Reason: "request method is unsupported for ledger entries"}
	}
	pageResult := storedListPageFromPath(request.Path, "ledger")
	page, pageMatched := pageResult.(storedListPageFromPathAccepted)
	if !pageMatched {
		return RequestHandleRejected{Reason: pageResult.(storedListPageFromPathRejected).reason}
	}
	listResult := ListLedgerEntries(handler.storage, ownerKind, ownerID, page.value)
	listed, listedMatched := listResult.(LedgerEntriesStored)
	if !listedMatched {
		return RequestHandleRejected{Reason: listResult.(LedgerStorageRejected).Reason}
	}
	encoded, err := json.Marshal(ledgerEntriesBody{Entries: listed.Values})
	if err != nil {
		return RequestHandleRejected{Reason: "ledger entries response encoding failed"}
	}
	return RequestHandled{Value: Response{Status: 200, Body: string(encoded)}}
}

type commentBody struct {
	Body string `json:"body"`
}

type reservationBody struct {
	AssigneeKind   string `json:"assignee_kind"`
	AssigneeID     string `json:"assignee_id"`
	OrganizationID string `json:"organization_id"`
	TeamID         string `json:"team_id"`
}

type submissionBody struct {
	ResponseJSON string                      `json:"response_json"`
	Attachments  []taskAttachmentRequestBody `json:"attachments"`
}

type sensitiveObjectSchema struct {
	Kind   string                 `json:"kind"`
	Fields []sensitiveSchemaField `json:"fields"`
}

type sensitiveSchemaField struct {
	Name   string                `json:"name"`
	Schema sensitiveSchemaNested `json:"schema"`
}

type sensitiveSchemaNested struct {
	Sensitivity sensitiveSchemaMetadata `json:"sensitivity"`
}

type sensitiveSchemaMetadata struct {
	Category  string `json:"category"`
	Retention string `json:"retention"`
	Redaction string `json:"redaction"`
}

type submissionCreatedBody struct {
	Submission   StoredSubmission `json:"submission"`
	ReceiptToken string           `json:"receipt_token"`
}

type acceptSubmissionBody struct {
	IdempotencyKey   string `json:"idempotency_key"`
	PayoutAmount     int64  `json:"payout_amount"`
	TipAmount        int64  `json:"tip_amount"`
	TipCollectibleID string `json:"tip_collectible_id"`
}

type acceptSubmissionResultBody struct {
	TaskID         string   `json:"task_id"`
	SubmissionID   string   `json:"submission_id"`
	PayoutKind     string   `json:"payout_kind"`
	PayoutAmount   int64    `json:"payout_amount"`
	WorkerUserID   string   `json:"worker_user_id"`
	CollectibleIDs []string `json:"collectible_ids"`
	TipAmount      int64    `json:"tip_amount"`
}

type balanceBody struct {
	Amount int64 `json:"amount"`
}

type ledgerEntriesBody struct {
	Entries []StoredLedgerEntry `json:"entries"`
}

type reservationAssignee struct {
	kind string
	id   string
}

func reservationAssigneeFromBody(body reservationBody, actorID string) reservationAssignee {
	kind := strings.TrimSpace(body.AssigneeKind)
	switch kind {
	case "team":
		return reservationAssignee{kind: kind, id: strings.TrimSpace(body.TeamID)}
	case "organization_team":
		return reservationAssignee{kind: kind, id: strings.TrimSpace(body.OrganizationID) + ":" + strings.TrimSpace(body.TeamID)}
	case "user":
		id := strings.TrimSpace(body.AssigneeID)
		if id == "" {
			id = strings.TrimSpace(actorID)
		}
		return reservationAssignee{kind: kind, id: id}
	default:
		return reservationAssignee{kind: "user", id: strings.TrimSpace(actorID)}
	}
}

func reservationActionState(action string) string {
	switch action {
	case "approve":
		return "active"
	case "decline":
		return "declined"
	case "cancel":
		return "cancelled"
	default:
		return ""
	}
}

func payoutKind(amount int64) string {
	if amount > 0 {
		return "credit"
	}
	return "none"
}

type submissionAttachmentsResult interface {
	submissionAttachmentsResult()
}

type submissionAttachmentsAccepted struct {
	values []StoredAttachment
}

type submissionAttachmentsRejected struct {
	reason string
}

func (submissionAttachmentsAccepted) submissionAttachmentsResult() {}
func (submissionAttachmentsRejected) submissionAttachmentsResult() {}

func attachmentsFromSubmissionBody(values []taskAttachmentRequestBody, submissionID string) submissionAttachmentsResult {
	if len(values) > maxStoredAttachments {
		return submissionAttachmentsRejected{reason: "too many attachments"}
	}
	attachments := make([]StoredAttachment, 0, len(values))
	for index := range values {
		contentType := strings.ToLower(strings.TrimSpace(values[index].ContentType))
		prefix := "data:" + contentType + ";base64,"
		dataURL := strings.TrimSpace(values[index].DataURL)
		if !strings.HasPrefix(dataURL, prefix) {
			return submissionAttachmentsRejected{reason: "attachment data URL is invalid"}
		}
		decoded, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(dataURL, prefix))
		if err != nil {
			return submissionAttachmentsRejected{reason: "attachment content is invalid"}
		}
		attachments = append(attachments, StoredAttachment{
			ParentKind:  "submission",
			ParentID:    strings.TrimSpace(submissionID),
			Name:        strings.TrimSpace(values[index].Name),
			ContentType: contentType,
			SizeBytes:   len(decoded),
			DataURL:     dataURL,
		})
	}
	return submissionAttachmentsAccepted{values: attachments}
}

func taskCommentResponseResult(comment StoredComment, status int) HandleResult {
	encoded, err := json.Marshal(taskCommentBody{
		ID:           comment.ID,
		TaskID:       comment.ParentID,
		AuthorUserID: comment.AuthorUserID,
		Body:         comment.Body,
		CreatedAt:    comment.CreatedAt,
	})
	if err != nil {
		return RequestHandleRejected{Reason: "task comment response encoding failed"}
	}
	return RequestHandled{Value: Response{Status: status, Body: string(encoded)}}
}

func taskCommentsResponseResult(comments []StoredComment) HandleResult {
	values := make([]taskCommentBody, 0, len(comments))
	for index := range comments {
		values = append(values, taskCommentBody{
			ID:           comments[index].ID,
			TaskID:       comments[index].ParentID,
			AuthorUserID: comments[index].AuthorUserID,
			Body:         comments[index].Body,
			CreatedAt:    comments[index].CreatedAt,
		})
	}
	encoded, err := json.Marshal(taskCommentsBody{Comments: values})
	if err != nil {
		return RequestHandleRejected{Reason: "task comments response encoding failed"}
	}
	return RequestHandled{Value: Response{Status: 200, Body: string(encoded)}}
}

func submissionCommentResponseResult(comment StoredComment, status int) HandleResult {
	encoded, err := json.Marshal(submissionCommentBody{
		ID:           comment.ID,
		SubmissionID: comment.ParentID,
		AuthorUserID: comment.AuthorUserID,
		Body:         comment.Body,
		CreatedAt:    comment.CreatedAt,
	})
	if err != nil {
		return RequestHandleRejected{Reason: "submission comment response encoding failed"}
	}
	return RequestHandled{Value: Response{Status: status, Body: string(encoded)}}
}

func submissionCommentsResponseResult(comments []StoredComment) HandleResult {
	values := make([]submissionCommentBody, 0, len(comments))
	for index := range comments {
		values = append(values, submissionCommentBody{
			ID:           comments[index].ID,
			SubmissionID: comments[index].ParentID,
			AuthorUserID: comments[index].AuthorUserID,
			Body:         comments[index].Body,
			CreatedAt:    comments[index].CreatedAt,
		})
	}
	encoded, err := json.Marshal(submissionCommentsBody{Comments: values})
	if err != nil {
		return RequestHandleRejected{Reason: "submission comments response encoding failed"}
	}
	return RequestHandled{Value: Response{Status: 200, Body: string(encoded)}}
}

func reservationResponseResult(reservation StoredReservation, status int) HandleResult {
	encoded, err := json.Marshal(reservation)
	if err != nil {
		return RequestHandleRejected{Reason: "reservation response encoding failed"}
	}
	return RequestHandled{Value: Response{Status: status, Body: string(encoded)}}
}

func reservationsResponseResult(reservations []StoredReservation) HandleResult {
	encoded, err := json.Marshal(reservationsBody{Reservations: reservations})
	if err != nil {
		return RequestHandleRejected{Reason: "reservations response encoding failed"}
	}
	return RequestHandled{Value: Response{Status: 200, Body: string(encoded)}}
}

func submissionsResponseResult(submissions []StoredSubmission) HandleResult {
	encoded, err := json.Marshal(submissionsBody{Submissions: submissions})
	if err != nil {
		return RequestHandleRejected{Reason: "submissions response encoding failed"}
	}
	return RequestHandled{Value: Response{Status: 200, Body: string(encoded)}}
}

type taskCommentBody struct {
	ID           string `json:"id"`
	TaskID       string `json:"task_id"`
	AuthorUserID string `json:"author_user_id"`
	Body         string `json:"body"`
	CreatedAt    string `json:"created_at"`
}

type taskCommentsBody struct {
	Comments []taskCommentBody `json:"comments"`
}

type submissionCommentBody struct {
	ID           string `json:"id"`
	SubmissionID string `json:"submission_id"`
	AuthorUserID string `json:"author_user_id"`
	Body         string `json:"body"`
	CreatedAt    string `json:"created_at"`
}

type submissionCommentsBody struct {
	Comments []submissionCommentBody `json:"comments"`
}

type reservationsBody struct {
	Reservations []StoredReservation `json:"reservations"`
}

type submissionsBody struct {
	Submissions []StoredSubmission `json:"submissions"`
}

type taskReservationsRoute struct {
	taskID        string
	reservationID string
	action        string
}

type taskSubmissionsRoute struct {
	taskID       string
	submissionID string
	action       string
}

type organizationCreditsRoute struct {
	organizationID string
	kind           string
}

func taskCommentsPathID(path string) string {
	parts := strings.Split(strings.Trim(strings.SplitN(path, "?", 2)[0], "/"), "/")
	if len(parts) == 4 && parts[0] == "api" && parts[1] == "tasks" && parts[3] == "comments" {
		return strings.TrimSpace(parts[2])
	}
	return ""
}

func submissionCommentsPathID(path string) string {
	parts := strings.Split(strings.Trim(strings.SplitN(path, "?", 2)[0], "/"), "/")
	if len(parts) == 4 && parts[0] == "api" && parts[1] == "submissions" && parts[3] == "comments" {
		return strings.TrimSpace(parts[2])
	}
	return ""
}

func taskReservationsPath(path string) taskReservationsRoute {
	parts := strings.Split(strings.Trim(strings.SplitN(path, "?", 2)[0], "/"), "/")
	if len(parts) == 4 && parts[0] == "api" && parts[1] == "tasks" && parts[3] == "reservations" {
		return taskReservationsRoute{taskID: strings.TrimSpace(parts[2])}
	}
	if len(parts) == 6 && parts[0] == "api" && parts[1] == "tasks" && parts[3] == "reservations" {
		return taskReservationsRoute{taskID: strings.TrimSpace(parts[2]), reservationID: strings.TrimSpace(parts[4]), action: strings.TrimSpace(parts[5])}
	}
	return taskReservationsRoute{}
}

func taskSubmissionsPath(path string) taskSubmissionsRoute {
	parts := strings.Split(strings.Trim(strings.SplitN(path, "?", 2)[0], "/"), "/")
	if len(parts) == 4 && parts[0] == "api" && parts[1] == "tasks" && parts[3] == "submissions" {
		return taskSubmissionsRoute{taskID: strings.TrimSpace(parts[2])}
	}
	if len(parts) == 6 && parts[0] == "api" && parts[1] == "tasks" && parts[3] == "submissions" {
		return taskSubmissionsRoute{taskID: strings.TrimSpace(parts[2]), submissionID: strings.TrimSpace(parts[4]), action: strings.TrimSpace(parts[5])}
	}
	return taskSubmissionsRoute{}
}

func userSubmissionsPath(path string) string {
	parts := strings.Split(strings.Trim(strings.SplitN(path, "?", 2)[0], "/"), "/")
	if len(parts) == 4 && parts[0] == "api" && parts[1] == "users" && parts[3] == "submissions" {
		return strings.TrimSpace(parts[2])
	}
	return ""
}

func creditsPathOnly(path string) string {
	return strings.SplitN(path, "?", 2)[0]
}

func organizationCreditsPath(path string) organizationCreditsRoute {
	parts := strings.Split(strings.Trim(strings.SplitN(path, "?", 2)[0], "/"), "/")
	if len(parts) == 5 && parts[0] == "api" && parts[1] == "organizations" && parts[3] == "credits" {
		return organizationCreditsRoute{organizationID: strings.TrimSpace(parts[2]), kind: strings.TrimSpace(parts[4])}
	}
	return organizationCreditsRoute{}
}
