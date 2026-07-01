package wasmdemo

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"time"
)

type Response struct {
	Status int
	Body   string
}

type HandleResult interface {
	handleResult()
}

type RequestHandled struct {
	Value Response
}

type RequestHandleRejected struct {
	Reason string
}

func (RequestHandled) handleResult()        {}
func (RequestHandleRejected) handleResult() {}

type ModerationTriageHandler struct {
	storage BrowserStorage
	clock   HandlerClock
}

type HandlerClock interface {
	Now() time.Time
}

type HandlerActor interface {
	UserID() string
}

type PrivacyRequestIDSource interface {
	NextPrivacyRequestID() string
}

type SavedQueueViewIDSource interface {
	NextSavedQueueViewID() string
}

type TaskIDSource interface {
	NextTaskID() string
}

func NewModerationTriageHandler(storage BrowserStorage, clock HandlerClock) ModerationTriageHandler {
	return ModerationTriageHandler{storage: storage, clock: clock}
}

func (handler ModerationTriageHandler) Handle(request Request) HandleResult {
	if handler.storage == nil {
		return RequestHandleRejected{Reason: "browser storage is required"}
	}
	if handler.clock == nil {
		return RequestHandleRejected{Reason: "handler clock is required"}
	}
	reportID, matched := moderationTriagePathReportID(request.Path)
	if !matched {
		return RequestHandleRejected{Reason: "request route is not implemented by the WASM demo handler"}
	}
	if request.Method.String() != MethodPost.String() {
		return RequestHandleRejected{Reason: "request method is unsupported for moderation triage"}
	}
	var body moderationTriageBody
	if err := json.Unmarshal([]byte(request.Body), &body); err != nil {
		return RequestHandleRejected{Reason: "moderation triage request body is invalid"}
	}
	triage := StoredModerationTriage{
		ReportID:       reportID,
		State:          strings.TrimSpace(body.State),
		ResolutionNote: strings.TrimSpace(body.ResolutionNote),
		UpdatedBy:      strings.TrimSpace(body.UpdatedBy),
		UpdatedAt:      handler.clock.Now().Format(time.RFC3339Nano),
	}
	result := SaveModerationTriage(handler.storage, triage)
	stored, storedMatched := result.(ModerationTriageStored)
	if !storedMatched {
		return RequestHandleRejected{Reason: result.(ModerationTriageStorageRejected).Reason}
	}
	encoded, err := json.Marshal(stored.Value)
	if err != nil {
		return RequestHandleRejected{Reason: "moderation triage response encoding failed"}
	}
	return RequestHandled{Value: Response{Status: 200, Body: string(encoded)}}
}

type moderationTriageBody struct {
	State          string `json:"state"`
	ResolutionNote string `json:"resolution_note"`
	UpdatedBy      string `json:"updated_by"`
}

func moderationTriagePathReportID(path string) (string, bool) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 6 {
		return "", false
	}
	if parts[0] != "api" || parts[1] != "admin" || parts[2] != "moderation" || parts[3] != "reports" || parts[5] != "triage" {
		return "", false
	}
	reportID := strings.TrimSpace(parts[4])
	if reportID == "" {
		return "", false
	}
	return reportID, true
}

type PrivacyRequestHandler struct {
	storage BrowserStorage
	clock   HandlerClock
	actor   HandlerActor
	ids     PrivacyRequestIDSource
}

func NewPrivacyRequestHandler(storage BrowserStorage, clock HandlerClock, actor HandlerActor, ids PrivacyRequestIDSource) PrivacyRequestHandler {
	return PrivacyRequestHandler{storage: storage, clock: clock, actor: actor, ids: ids}
}

func (handler PrivacyRequestHandler) Handle(request Request) HandleResult {
	if handler.storage == nil {
		return RequestHandleRejected{Reason: "browser storage is required"}
	}
	if handler.clock == nil {
		return RequestHandleRejected{Reason: "handler clock is required"}
	}
	if handler.actor == nil {
		return RequestHandleRejected{Reason: "handler actor is required"}
	}
	if request.Path == "/api/privacy-requests" {
		return handler.handleCreate(request)
	}
	if request.Path == "/api/admin/privacy-requests" {
		return handler.handleList(request)
	}
	return RequestHandleRejected{Reason: "request route is not implemented by the WASM demo handler"}
}

func (handler PrivacyRequestHandler) handleCreate(request Request) HandleResult {
	if request.Method.String() != MethodPost.String() {
		return RequestHandleRejected{Reason: "request method is unsupported for privacy request creation"}
	}
	if handler.ids == nil {
		return RequestHandleRejected{Reason: "privacy request id source is required"}
	}
	var body privacyRequestBody
	if err := json.Unmarshal([]byte(request.Body), &body); err != nil {
		return RequestHandleRejected{Reason: "privacy request body is invalid"}
	}
	stored := StoredPrivacyRequest{
		ID:                 strings.TrimSpace(handler.ids.NextPrivacyRequestID()),
		Kind:               strings.TrimSpace(body.Kind),
		Status:             "queued",
		RequestedBy:        strings.TrimSpace(handler.actor.UserID()),
		ExportJSON:         "",
		ResolutionNote:     "",
		CreatedAt:          handler.clock.Now().Format(time.RFC3339Nano),
		ResolvedAt:         "",
		RedactedFieldCount: 0,
	}
	saveResult := SavePrivacyRequest(handler.storage, stored)
	saved, savedMatched := saveResult.(PrivacyRequestStored)
	if !savedMatched {
		return RequestHandleRejected{Reason: saveResult.(PrivacyRequestStorageRejected).Reason}
	}
	encoded, err := json.Marshal(saved.Value)
	if err != nil {
		return RequestHandleRejected{Reason: "privacy request response encoding failed"}
	}
	return RequestHandled{Value: Response{Status: 201, Body: string(encoded)}}
}

func (handler PrivacyRequestHandler) handleList(request Request) HandleResult {
	if request.Method.String() != MethodGet.String() {
		return RequestHandleRejected{Reason: "request method is unsupported for privacy request listing"}
	}
	listResult := ListPrivacyRequests(handler.storage)
	listed, listedMatched := listResult.(PrivacyRequestsStored)
	if !listedMatched {
		return RequestHandleRejected{Reason: listResult.(PrivacyRequestStorageRejected).Reason}
	}
	encoded, err := json.Marshal(privacyRequestsBody{Requests: listed.Values})
	if err != nil {
		return RequestHandleRejected{Reason: "privacy requests response encoding failed"}
	}
	return RequestHandled{Value: Response{Status: 200, Body: string(encoded)}}
}

type privacyRequestBody struct {
	Kind string `json:"kind"`
}

type privacyRequestsBody struct {
	Requests []StoredPrivacyRequest `json:"requests"`
}

type SavedQueueViewHandler struct {
	storage BrowserStorage
	actor   HandlerActor
	ids     SavedQueueViewIDSource
}

func NewSavedQueueViewHandler(storage BrowserStorage, actor HandlerActor, ids SavedQueueViewIDSource) SavedQueueViewHandler {
	return SavedQueueViewHandler{storage: storage, actor: actor, ids: ids}
}

func (handler SavedQueueViewHandler) Handle(request Request) HandleResult {
	if handler.storage == nil {
		return RequestHandleRejected{Reason: "browser storage is required"}
	}
	if handler.actor == nil {
		return RequestHandleRejected{Reason: "handler actor is required"}
	}
	if savedQueueViewPathOnly(request.Path) != "/api/saved-queue-views" {
		return RequestHandleRejected{Reason: "request route is not implemented by the WASM demo handler"}
	}
	switch request.Method.String() {
	case MethodPost.String():
		if handler.ids == nil {
			return RequestHandleRejected{Reason: "saved queue view id source is required"}
		}
		return handler.handleUpsert(request)
	case MethodGet.String():
		return handler.handleList(request)
	default:
		return RequestHandleRejected{Reason: "request method is unsupported for saved queue views"}
	}
}

func (handler SavedQueueViewHandler) handleUpsert(request Request) HandleResult {
	var body savedQueueViewBody
	if err := json.Unmarshal([]byte(request.Body), &body); err != nil {
		return RequestHandleRejected{Reason: "saved queue view body is invalid"}
	}
	view := StoredSavedQueueView{
		ID:          strings.TrimSpace(handler.ids.NextSavedQueueViewID()),
		UserID:      strings.TrimSpace(handler.actor.UserID()),
		Scope:       strings.TrimSpace(body.Scope),
		Name:        strings.TrimSpace(body.Name),
		Query:       strings.TrimSpace(body.Query),
		StateFilter: strings.TrimSpace(body.StateFilter),
		TypeFilter:  strings.TrimSpace(body.TypeFilter),
		Sort:        strings.TrimSpace(body.Sort),
	}
	saveResult := SaveSavedQueueView(handler.storage, view)
	saved, savedMatched := saveResult.(SavedQueueViewStored)
	if !savedMatched {
		return RequestHandleRejected{Reason: saveResult.(SavedQueueViewStorageRejected).Reason}
	}
	encoded, err := json.Marshal(saved.Value)
	if err != nil {
		return RequestHandleRejected{Reason: "saved queue view response encoding failed"}
	}
	return RequestHandled{Value: Response{Status: 200, Body: string(encoded)}}
}

func (handler SavedQueueViewHandler) handleList(request Request) HandleResult {
	scope := savedQueueViewScopeFromPath(request.Path)
	listResult := ListSavedQueueViews(handler.storage, handler.actor.UserID(), scope)
	listed, listedMatched := listResult.(SavedQueueViewsStored)
	if !listedMatched {
		return RequestHandleRejected{Reason: listResult.(SavedQueueViewStorageRejected).Reason}
	}
	encoded, err := json.Marshal(savedQueueViewsBody{Views: listed.Values})
	if err != nil {
		return RequestHandleRejected{Reason: "saved queue views response encoding failed"}
	}
	return RequestHandled{Value: Response{Status: 200, Body: string(encoded)}}
}

type savedQueueViewBody struct {
	Scope       string `json:"scope"`
	Name        string `json:"name"`
	Query       string `json:"query"`
	StateFilter string `json:"state_filter"`
	TypeFilter  string `json:"type_filter"`
	Sort        string `json:"sort"`
}

type savedQueueViewsBody struct {
	Views []StoredSavedQueueView `json:"views"`
}

func savedQueueViewScopeFromPath(path string) string {
	parts := strings.SplitN(path, "?", 2)
	if len(parts) != 2 {
		return ""
	}
	for _, part := range strings.Split(parts[1], "&") {
		keyValue := strings.SplitN(part, "=", 2)
		if len(keyValue) == 2 && keyValue[0] == "scope" {
			return strings.TrimSpace(keyValue[1])
		}
	}
	return ""
}

func savedQueueViewPathOnly(path string) string {
	return strings.SplitN(path, "?", 2)[0]
}

type TaskHandler struct {
	storage BrowserStorage
	actor   HandlerActor
	ids     TaskIDSource
}

func NewTaskHandler(storage BrowserStorage, actor HandlerActor, ids TaskIDSource) TaskHandler {
	return TaskHandler{storage: storage, actor: actor, ids: ids}
}

func (handler TaskHandler) Handle(request Request) HandleResult {
	if handler.storage == nil {
		return RequestHandleRejected{Reason: "browser storage is required"}
	}
	if handler.actor == nil {
		return RequestHandleRejected{Reason: "handler actor is required"}
	}
	if request.Path == "/api/tasks" {
		if request.Method.String() != MethodPost.String() {
			return RequestHandleRejected{Reason: "request method is unsupported for task creation"}
		}
		if handler.ids == nil {
			return RequestHandleRejected{Reason: "task id source is required"}
		}
		return handler.handleCreateTask(request)
	}
	taskID := taskDetailPathID(request.Path)
	if taskID == "" {
		return RequestHandleRejected{Reason: "request route is not implemented by the WASM demo handler"}
	}
	if request.Method.String() != MethodGet.String() {
		return RequestHandleRejected{Reason: "request method is unsupported for task detail"}
	}
	return handler.handleGetTask(taskID)
}

func (handler TaskHandler) handleCreateTask(request Request) HandleResult {
	var body taskBody
	if err := json.Unmarshal([]byte(request.Body), &body); err != nil {
		return RequestHandleRejected{Reason: "task request body is invalid"}
	}
	taskID := strings.TrimSpace(handler.ids.NextTaskID())
	owner := taskOwnerFromBody(body.Owner, handler.actor.UserID())
	visibility := taskVisibilityFromBody(body.Visibility, owner)
	reward := taskRewardFromBody(body.Reward)
	participation := taskParticipationFromBody(body.Participation)
	placement := taskPlacementFromBody(body.Placement)
	payload := taskPayloadFromBody(body.Payload)
	task := StoredTask{
		ID:                     taskID,
		OwnerKind:              owner.kind,
		OwnerID:                owner.id,
		Title:                  strings.TrimSpace(body.Title),
		Description:            strings.TrimSpace(body.Description),
		TaskType:               strings.TrimSpace(body.TaskType),
		ReferenceURL:           strings.TrimSpace(body.ReferenceURL),
		RewardKind:             reward.kind,
		RewardCreditAmount:     reward.creditAmount,
		RewardCollectibleCount: len(reward.collectibleIDs),
		ParticipationPolicy:    participation.policy,
		AssigneeScope:          participation.assigneeScope,
		ReservationExpiryHours: participation.reservationHours,
		State:                  "draft",
		VisibilityKind:         visibility.kind,
		VisibilityID:           visibility.id,
		SeriesKind:             placement.kind,
		SeriesID:               placement.id,
		SeriesPosition:         placement.position,
		ResponseSchemaJSON:     strings.TrimSpace(body.ResponseSchemaJSON),
		PayloadKind:            payload.kind,
		PayloadJSON:            payload.source,
		CreatedBy:              strings.TrimSpace(handler.actor.UserID()),
	}
	if task.TaskType == "" {
		task.TaskType = "general"
	}
	if task.ResponseSchemaJSON == "" {
		task.ResponseSchemaJSON = `{"kind":"freeform"}`
	}
	attachmentsResult := attachmentsFromTaskBody(body.Attachments, task.ID)
	attachments, attachmentsMatched := attachmentsResult.(taskAttachmentsAccepted)
	if !attachmentsMatched {
		return RequestHandleRejected{Reason: attachmentsResult.(taskAttachmentsRejected).reason}
	}
	saveResult := SaveTask(handler.storage, task)
	saved, savedMatched := saveResult.(TaskStored)
	if !savedMatched {
		return RequestHandleRejected{Reason: saveResult.(TaskStorageRejected).Reason}
	}
	saveAttachmentsResult := SaveAttachments(handler.storage, "task", saved.Value.ID, attachments.values)
	savedAttachments, savedAttachmentsMatched := saveAttachmentsResult.(AttachmentsStored)
	if !savedAttachmentsMatched {
		return RequestHandleRejected{Reason: saveAttachmentsResult.(AttachmentStorageRejected).Reason}
	}
	return taskResponseResult(saved.Value, savedAttachments.Values, 201)
}

func (handler TaskHandler) handleGetTask(taskID string) HandleResult {
	loadResult := LoadTask(handler.storage, taskID)
	loaded, loadedMatched := loadResult.(TaskStored)
	if !loadedMatched {
		return RequestHandleRejected{Reason: loadResult.(TaskStorageRejected).Reason}
	}
	attachmentsResult := ListAttachments(handler.storage, "task", taskID)
	attachments, attachmentsMatched := attachmentsResult.(AttachmentsStored)
	if !attachmentsMatched {
		return RequestHandleRejected{Reason: attachmentsResult.(AttachmentStorageRejected).Reason}
	}
	return taskResponseResult(loaded.Value, attachments.Values, 200)
}

func taskResponseResult(task StoredTask, attachments []StoredAttachment, status int) HandleResult {
	encoded, err := json.Marshal(taskResponseBody{
		StoredTask:       task,
		Attachments:      attachments,
		AvailabilityKind: "available",
		ViewerAction:     "submit",
		ReviewerAction:   "none",
	})
	if err != nil {
		return RequestHandleRejected{Reason: "task response encoding failed"}
	}
	return RequestHandled{Value: Response{Status: status, Body: string(encoded)}}
}

type taskBody struct {
	Owner              taskOwnerBody               `json:"owner"`
	Title              string                      `json:"title"`
	Description        string                      `json:"description"`
	TaskType           string                      `json:"task_type"`
	ReferenceURL       string                      `json:"reference_url"`
	Reward             taskRewardBody              `json:"reward"`
	Participation      taskParticipationBody       `json:"participation"`
	Visibility         taskVisibilityBody          `json:"visibility"`
	Placement          taskPlacementBody           `json:"placement"`
	ResponseSchemaJSON string                      `json:"response_schema_json"`
	Payload            taskPayloadBody             `json:"payload"`
	Attachments        []taskAttachmentRequestBody `json:"attachments"`
}

type taskOwnerBody struct {
	Kind           string `json:"kind"`
	UserID         string `json:"user_id"`
	TeamID         string `json:"team_id"`
	OrganizationID string `json:"organization_id"`
}

type taskRewardBody struct {
	Kind           string   `json:"kind"`
	CreditAmount   int64    `json:"credit_amount"`
	CollectibleIDs []string `json:"collectible_ids"`
}

type taskParticipationBody struct {
	Policy                 string `json:"policy"`
	AssigneeScope          string `json:"assignee_scope"`
	ReservationExpiryHours int    `json:"reservation_expiry_hours"`
}

type taskVisibilityBody struct {
	Kind           string `json:"kind"`
	UserID         string `json:"user_id"`
	TeamID         string `json:"team_id"`
	OrganizationID string `json:"organization_id"`
}

type taskPlacementBody struct {
	Kind           string `json:"kind"`
	SeriesID       string `json:"series_id"`
	SeriesPosition int    `json:"series_position"`
}

type taskPayloadBody struct {
	Kind string `json:"kind"`
	JSON string `json:"json"`
}

type taskAttachmentRequestBody struct {
	Name        string `json:"name"`
	ContentType string `json:"content_type"`
	DataURL     string `json:"data_url"`
}

type taskResponseBody struct {
	StoredTask
	Attachments      []StoredAttachment `json:"attachments"`
	AvailabilityKind string             `json:"availability_kind"`
	ViewerAction     string             `json:"viewer_action"`
	ReviewerAction   string             `json:"reviewer_action"`
}

type taskOwnerParts struct {
	kind string
	id   string
}

func taskOwnerFromBody(body taskOwnerBody, actorID string) taskOwnerParts {
	kind := strings.TrimSpace(body.Kind)
	switch kind {
	case "team":
		return taskOwnerParts{kind: kind, id: strings.TrimSpace(body.TeamID)}
	case "organization":
		return taskOwnerParts{kind: kind, id: strings.TrimSpace(body.OrganizationID)}
	case "organization_team":
		return taskOwnerParts{kind: kind, id: strings.TrimSpace(body.OrganizationID)}
	default:
		return taskOwnerParts{kind: "user", id: strings.TrimSpace(actorID)}
	}
}

type taskRewardParts struct {
	kind           string
	creditAmount   int64
	collectibleIDs []string
}

func taskRewardFromBody(body taskRewardBody) taskRewardParts {
	kind := strings.TrimSpace(body.Kind)
	if kind == "" {
		kind = "none"
	}
	if len(body.CollectibleIDs) > 0 && body.CreditAmount > 0 {
		kind = "bundle"
	} else if len(body.CollectibleIDs) > 0 {
		kind = "collectible"
	}
	return taskRewardParts{kind: kind, creditAmount: body.CreditAmount, collectibleIDs: body.CollectibleIDs}
}

type taskParticipationParts struct {
	policy           string
	assigneeScope    string
	reservationHours int
}

func taskParticipationFromBody(body taskParticipationBody) taskParticipationParts {
	policy := strings.TrimSpace(body.Policy)
	if policy == "" {
		policy = "open"
	}
	scope := strings.TrimSpace(body.AssigneeScope)
	if scope == "" {
		scope = "user"
	}
	hours := body.ReservationExpiryHours
	if hours == 0 {
		hours = 48
	}
	return taskParticipationParts{policy: policy, assigneeScope: scope, reservationHours: hours}
}

type taskVisibilityParts struct {
	kind string
	id   string
}

func taskVisibilityFromBody(body taskVisibilityBody, owner taskOwnerParts) taskVisibilityParts {
	kind := strings.TrimSpace(body.Kind)
	if kind == "" || kind == "default" {
		if owner.kind == "organization" || owner.kind == "organization_team" {
			return taskVisibilityParts{kind: "organization", id: owner.id}
		}
		return taskVisibilityParts{kind: "public", id: ""}
	}
	switch kind {
	case "user":
		return taskVisibilityParts{kind: kind, id: strings.TrimSpace(body.UserID)}
	case "team":
		return taskVisibilityParts{kind: kind, id: strings.TrimSpace(body.TeamID)}
	case "organization":
		return taskVisibilityParts{kind: kind, id: strings.TrimSpace(body.OrganizationID)}
	default:
		return taskVisibilityParts{kind: kind, id: ""}
	}
}

type taskPlacementParts struct {
	kind     string
	id       string
	position int
}

func taskPlacementFromBody(body taskPlacementBody) taskPlacementParts {
	kind := strings.TrimSpace(body.Kind)
	if kind == "" {
		kind = "standalone"
	}
	return taskPlacementParts{kind: kind, id: strings.TrimSpace(body.SeriesID), position: body.SeriesPosition}
}

type taskPayloadParts struct {
	kind   string
	source string
}

func taskPayloadFromBody(body taskPayloadBody) taskPayloadParts {
	if strings.TrimSpace(body.Kind) == "json" {
		return taskPayloadParts{kind: "json", source: strings.TrimSpace(body.JSON)}
	}
	return taskPayloadParts{kind: "none", source: ""}
}

type taskAttachmentsResult interface {
	taskAttachmentsResult()
}

type taskAttachmentsAccepted struct {
	values []StoredAttachment
}

type taskAttachmentsRejected struct {
	reason string
}

func (taskAttachmentsAccepted) taskAttachmentsResult() {}
func (taskAttachmentsRejected) taskAttachmentsResult() {}

func attachmentsFromTaskBody(values []taskAttachmentRequestBody, taskID string) taskAttachmentsResult {
	if len(values) > maxStoredAttachments {
		return taskAttachmentsRejected{reason: "too many attachments"}
	}
	attachments := make([]StoredAttachment, 0, len(values))
	for index := range values {
		contentType := strings.ToLower(strings.TrimSpace(values[index].ContentType))
		prefix := "data:" + contentType + ";base64,"
		dataURL := strings.TrimSpace(values[index].DataURL)
		if !strings.HasPrefix(dataURL, prefix) {
			return taskAttachmentsRejected{reason: "attachment data URL is invalid"}
		}
		decoded, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(dataURL, prefix))
		if err != nil {
			return taskAttachmentsRejected{reason: "attachment content is invalid"}
		}
		attachments = append(attachments, StoredAttachment{
			ParentKind:  "task",
			ParentID:    taskID,
			Name:        strings.TrimSpace(values[index].Name),
			ContentType: contentType,
			SizeBytes:   len(decoded),
			DataURL:     dataURL,
		})
	}
	return taskAttachmentsAccepted{values: attachments}
}
