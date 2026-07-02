package wasmdemo

import (
	"encoding/base64"
	"encoding/json"
	"net/url"
	"strconv"
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

type OrganizationIDSource interface {
	NextOrganizationID() string
	NextOrganizationMemberID() string
	NextTeamID() string
}

type OrganizationUserResolver interface {
	UserIDForEmail(string) (string, bool)
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
		if request.Method.String() == MethodPost.String() {
			return handler.handleCreate(request)
		}
		if request.Method.String() == MethodGet.String() {
			return handler.handleListUser(request)
		}
		return RequestHandleRejected{Reason: "request method is unsupported for privacy requests"}
	}
	if strings.SplitN(request.Path, "?", 2)[0] == "/api/admin/privacy-requests" {
		return handler.handleList(request)
	}
	if requestID := adminPrivacyResolvePathID(request.Path); requestID != "" {
		return handler.handleResolve(request, requestID)
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
	if runtimeIDs, matched := handler.ids.(RuntimeIDSource); matched {
		if err := SaveAuditEvent(handler.storage, StoredAuditEvent{
			ID:           strings.TrimSpace(runtimeIDs.NextAuditEventID()),
			ActorID:      strings.TrimSpace(handler.actor.UserID()),
			Action:       "privacy_request_created",
			SubjectKind:  "privacy_request",
			SubjectID:    strings.TrimSpace(handler.actor.UserID()),
			MetadataJSON: `{"kind":"` + stored.Kind + `"}`,
			CreatedAt:    handler.clock.Now().UTC().Format(time.RFC3339),
		}); err != nil {
			return RequestHandleRejected{Reason: err.Error()}
		}
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

func (handler PrivacyRequestHandler) handleListUser(request Request) HandleResult {
	listResult := ListPrivacyRequests(handler.storage)
	listed, listedMatched := listResult.(PrivacyRequestsStored)
	if !listedMatched {
		return RequestHandleRejected{Reason: listResult.(PrivacyRequestStorageRejected).Reason}
	}
	userID := strings.TrimSpace(handler.actor.UserID())
	values := make([]StoredPrivacyRequest, 0, len(listed.Values))
	for index := range listed.Values {
		if listed.Values[index].RequestedBy == userID {
			values = append(values, listed.Values[index])
		}
	}
	encoded, err := json.Marshal(privacyRequestsBody{Requests: values})
	if err != nil {
		return RequestHandleRejected{Reason: "privacy requests response encoding failed"}
	}
	return RequestHandled{Value: Response{Status: 200, Body: string(encoded)}}
}

func (handler PrivacyRequestHandler) handleResolve(request Request, requestID string) HandleResult {
	if request.Method.String() != MethodPost.String() {
		return RequestHandleRejected{Reason: "request method is unsupported for privacy request resolution"}
	}
	listResult := ListPrivacyRequests(handler.storage)
	listed, listedMatched := listResult.(PrivacyRequestsStored)
	if !listedMatched {
		return RequestHandleRejected{Reason: listResult.(PrivacyRequestStorageRejected).Reason}
	}
	var body privacyResolutionBody
	if err := json.Unmarshal([]byte(request.Body), &body); err != nil {
		return RequestHandleRejected{Reason: "privacy resolution body is invalid"}
	}
	for index := range listed.Values {
		if listed.Values[index].ID == strings.TrimSpace(requestID) {
			privacy := listed.Values[index]
			if privacy.Status != "queued" {
				return RequestHandleRejected{Reason: "privacy request is already resolved"}
			}
			privacy.Status = "resolved"
			privacy.ResolutionNote = strings.TrimSpace(body.ResolutionNote)
			privacy.ResolvedAt = handler.clock.Now().UTC().Format(time.RFC3339)
			if privacy.Kind == "data_export" {
				privacy.ExportJSON = `{"user_id":"` + privacy.RequestedBy + `","generated_at":"` + privacy.ResolvedAt + `","submissions":[],"sensitive_fields":[]}`
			}
			if privacy.Kind == "sensitive_field_deletion" {
				count, err := redactSensitiveFieldsForUser(handler.storage, privacy.RequestedBy, privacy.ResolvedAt)
				if err != nil {
					return RequestHandleRejected{Reason: err.Error()}
				}
				privacy.RedactedFieldCount = count
			}
			saveResult := SavePrivacyRequest(handler.storage, privacy)
			saved, savedMatched := saveResult.(PrivacyRequestStored)
			if !savedMatched {
				return RequestHandleRejected{Reason: saveResult.(PrivacyRequestStorageRejected).Reason}
			}
			encoded, err := json.Marshal(saved.Value)
			if err != nil {
				return RequestHandleRejected{Reason: "privacy resolution response encoding failed"}
			}
			return RequestHandled{Value: Response{Status: 200, Body: string(encoded)}}
		}
	}
	return RequestHandleRejected{Reason: "privacy request was not found"}
}

func redactSensitiveFieldsForUser(storage BrowserStorage, userID string, redactedAt string) (int, error) {
	listResult := ListAllSubmissions(storage, DefaultStoredListPage())
	listed, matched := listResult.(SubmissionsStored)
	if !matched {
		return 0, errString(listResult.(SubmissionStorageRejected).Reason)
	}
	count := 0
	for index := range listed.Values {
		submission := listed.Values[index]
		if submission.SubmitterID != strings.TrimSpace(userID) {
			continue
		}
		changed := false
		for fieldIndex := range submission.SensitiveFields {
			if submission.SensitiveFields[fieldIndex].State == "active" && submission.SensitiveFields[fieldIndex].Retention == "delete_on_request" {
				submission.SensitiveFields[fieldIndex].State = "redacted"
				submission.SensitiveFields[fieldIndex].RedactedAt = redactedAt
				count++
				changed = true
			}
		}
		if changed {
			saveResult := SaveSubmission(storage, submission)
			if _, saved := saveResult.(SubmissionStored); !saved {
				return 0, errString(saveResult.(SubmissionStorageRejected).Reason)
			}
		}
	}
	return count, nil
}

type privacyRequestBody struct {
	Kind string `json:"kind"`
}

type privacyResolutionBody struct {
	ResolutionNote string `json:"resolution_note"`
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

func queryStringFromPath(path string) string {
	parts := strings.SplitN(path, "?", 2)
	if len(parts) != 2 {
		return ""
	}
	return parts[1]
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
	if tasksPathOnly(request.Path) == "/api/tasks" {
		switch request.Method.String() {
		case MethodPost.String():
			if handler.ids == nil {
				return RequestHandleRejected{Reason: "task id source is required"}
			}
			return handler.handleCreateTask(request)
		case MethodGet.String():
			return handler.handleListTasks(request)
		default:
			return RequestHandleRejected{Reason: "request method is unsupported for tasks"}
		}
	}
	if teamID := teamWorkPathID(request.Path); teamID != "" {
		return handler.handleTeamWork(request, teamID)
	}
	if action := taskActionPath(request.Path); action.taskID != "" {
		return handler.handleTaskAction(request, action)
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

func (handler TaskHandler) handleTeamWork(request Request, teamID string) HandleResult {
	if request.Method.String() != MethodGet.String() {
		return RequestHandleRejected{Reason: "request method is unsupported for team work"}
	}
	pageResult := storedListPageFromPath(request.Path, "team work")
	page, pageMatched := pageResult.(storedListPageFromPathAccepted)
	if !pageMatched {
		return RequestHandleRejected{Reason: pageResult.(storedListPageFromPathRejected).reason}
	}
	values, err := url.ParseQuery(queryStringFromPath(request.Path))
	if err != nil {
		return RequestHandleRejected{Reason: "team work query is invalid"}
	}
	listResult := ListTasks(handler.storage, values.Get("query"), "", handler.actor.UserID(), "", "", DefaultStoredListPage())
	listed, listedMatched := listResult.(TasksStored)
	if !listedMatched {
		return RequestHandleRejected{Reason: listResult.(TaskStorageRejected).Reason}
	}
	tasks := make([]StoredTask, 0, len(listed.Values))
	for index := range listed.Values {
		task := listed.Values[index]
		if task.VisibilityID == strings.TrimSpace(teamID) || task.OwnerID == strings.TrimSpace(teamID) {
			tasks = append(tasks, task)
		}
	}
	start, end := pageBounds(len(tasks), page.value)
	encoded, err := json.Marshal(tasksResponseBody{Tasks: taskSummaries(tasks[start:end])})
	if err != nil {
		return RequestHandleRejected{Reason: "team work response encoding failed"}
	}
	return RequestHandled{Value: Response{Status: 200, Body: string(encoded)}}
}

func (handler TaskHandler) handleListTasks(request Request) HandleResult {
	pageResult := storedListPageFromPath(request.Path, "task")
	page, pageMatched := pageResult.(storedListPageFromPathAccepted)
	if !pageMatched {
		return RequestHandleRejected{Reason: pageResult.(storedListPageFromPathRejected).reason}
	}
	values, err := url.ParseQuery(queryStringFromPath(request.Path))
	if err != nil {
		return RequestHandleRejected{Reason: "task query is invalid"}
	}
	listResult := ListTasks(
		handler.storage,
		values.Get("query"),
		values.Get("scope"),
		handler.actor.UserID(),
		values.Get("organization_id"),
		values.Get("state"),
		page.value,
	)
	listed, listedMatched := listResult.(TasksStored)
	if !listedMatched {
		return RequestHandleRejected{Reason: listResult.(TaskStorageRejected).Reason}
	}
	encoded, err := json.Marshal(tasksResponseBody{Tasks: taskSummaries(listed.Values)})
	if err != nil {
		return RequestHandleRejected{Reason: "tasks response encoding failed"}
	}
	return RequestHandled{Value: Response{Status: 200, Body: string(encoded)}}
}

func (handler TaskHandler) handleTaskAction(request Request, action taskActionRoute) HandleResult {
	if request.Method.String() != MethodPost.String() {
		return RequestHandleRejected{Reason: "request method is unsupported for task action"}
	}
	loadResult := LoadTask(handler.storage, action.taskID)
	loaded, loadedMatched := loadResult.(TaskStored)
	if !loadedMatched {
		return RequestHandleRejected{Reason: loadResult.(TaskStorageRejected).Reason}
	}
	task := loaded.Value
	switch action.action {
	case "open":
		if task.State != "draft" {
			return RequestHandleRejected{Reason: "only draft tasks can be opened"}
		}
		task.State = "open"
	case "cancel":
		if task.State != "draft" && task.State != "open" {
			return RequestHandleRejected{Reason: "only draft or open tasks can be cancelled"}
		}
		if task.EscrowAmount > 0 || task.RewardCollectibleCount > 0 {
			return RequestHandleRejected{Reason: "refund the task's held escrow before cancelling"}
		}
		task.State = "cancelled"
	case "unpublish":
		if task.State != "open" {
			return RequestHandleRejected{Reason: "only open tasks can be unpublished"}
		}
		task.State = "draft"
	case "funding":
		var body taskFundingBody
		if err := json.Unmarshal([]byte(request.Body), &body); err != nil {
			return RequestHandleRejected{Reason: "task funding body is invalid"}
		}
		if body.Amount < 1 {
			return RequestHandleRejected{Reason: "task funding amount is invalid"}
		}
		if task.EscrowAmount > 0 {
			return RequestHandleRejected{Reason: "task is already funded"}
		}
		task.EscrowAmount = body.Amount
		task.FundedOrganizationID = strings.TrimSpace(body.OrganizationID)
	case "refund":
		if task.State != "draft" && task.State != "open" {
			return RequestHandleRejected{Reason: "only draft or open tasks can be refunded"}
		}
		if task.EscrowAmount == 0 && task.RewardCollectibleCount == 0 {
			return RequestHandleRejected{Reason: "task has no escrow to refund"}
		}
		amount := task.EscrowAmount
		task.EscrowAmount = 0
		task.RewardCollectibleCount = 0
		if task.RewardCreditAmount == 0 {
			task.RewardKind = "none"
		} else {
			task.RewardKind = "credit"
		}
		task.State = "cancelled"
		saveResult := SaveTask(handler.storage, task)
		if _, savedMatched := saveResult.(TaskStored); !savedMatched {
			return RequestHandleRejected{Reason: saveResult.(TaskStorageRejected).Reason}
		}
		encoded, err := json.Marshal(taskRefundBody{TaskID: task.ID, Amount: amount, State: "refunded"})
		if err != nil {
			return RequestHandleRejected{Reason: "task refund response encoding failed"}
		}
		return RequestHandled{Value: Response{Status: 200, Body: string(encoded)}}
	case "collectible-refund":
		if task.RewardCollectibleCount == 0 {
			return RequestHandleRejected{Reason: "task has no escrowed collectibles to refund"}
		}
		refunded := make([]StoredCollectible, 0, len(task.RewardCollectibleIDs))
		for index := range task.RewardCollectibleIDs {
			collectible, err := LoadCollectible(handler.storage, task.RewardCollectibleIDs[index])
			if err != nil {
				return RequestHandleRejected{Reason: err.Error()}
			}
			collectible.State = "minted"
			if err := SaveCollectible(handler.storage, collectible); err != nil {
				return RequestHandleRejected{Reason: err.Error()}
			}
			refunded = append(refunded, collectible)
		}
		task.RewardCollectibleCount = 0
		task.RewardCollectibleIDs = []string{}
		if task.RewardCreditAmount == 0 {
			task.RewardKind = "none"
		} else {
			task.RewardKind = "credit"
		}
		saveResult := SaveTask(handler.storage, task)
		if _, savedMatched := saveResult.(TaskStored); !savedMatched {
			return RequestHandleRejected{Reason: saveResult.(TaskStorageRejected).Reason}
		}
		encoded, err := json.Marshal(collectibleRefundBody{Collectibles: refunded})
		if err != nil {
			return RequestHandleRejected{Reason: "collectible refund response encoding failed"}
		}
		return RequestHandled{Value: Response{Status: 200, Body: string(encoded)}}
	case "collectible-reward":
		task.RewardCollectibleCount = task.RewardCollectibleCount + 1
		if task.RewardCreditAmount > 0 {
			task.RewardKind = "bundle"
		} else {
			task.RewardKind = "collectible"
		}
	default:
		return RequestHandleRejected{Reason: "task action is unsupported"}
	}
	saveResult := SaveTask(handler.storage, task)
	saved, savedMatched := saveResult.(TaskStored)
	if !savedMatched {
		return RequestHandleRejected{Reason: saveResult.(TaskStorageRejected).Reason}
	}
	if action.action == "funding" {
		encoded, err := json.Marshal(taskFundingResponseBody{TaskID: task.ID, Amount: task.EscrowAmount, State: "held"})
		if err != nil {
			return RequestHandleRejected{Reason: "task funding response encoding failed"}
		}
		return RequestHandled{Value: Response{Status: 201, Body: string(encoded)}}
	}
	return taskResponseResult(saved.Value, []StoredAttachment{}, 200)
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
		RewardCollectibleIDs:   reward.collectibleIDs,
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
	for index := range saved.Value.RewardCollectibleIDs {
		collectible, err := LoadCollectible(handler.storage, saved.Value.RewardCollectibleIDs[index])
		if err != nil {
			return RequestHandleRejected{Reason: err.Error()}
		}
		collectible.State = "escrowed"
		if err := SaveCollectible(handler.storage, collectible); err != nil {
			return RequestHandleRejected{Reason: err.Error()}
		}
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
		StoredTask:         task,
		Attachments:        attachments,
		AvailabilityKind:   "available",
		ViewerAction:       taskViewerAction(task),
		ReviewerAction:     "none",
		ActiveAssigneeKind: "none",
		ActiveAssigneeID:   "",
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
	Attachments        []StoredAttachment `json:"attachments"`
	AvailabilityKind   string             `json:"availability_kind"`
	ViewerAction       string             `json:"viewer_action"`
	ReviewerAction     string             `json:"reviewer_action"`
	ActiveAssigneeKind string             `json:"active_assignee_kind"`
	ActiveAssigneeID   string             `json:"active_assignee_id"`
}

type tasksResponseBody struct {
	Tasks []taskResponseBody `json:"tasks"`
}

func taskSummaries(tasks []StoredTask) []taskResponseBody {
	summaries := make([]taskResponseBody, 0, len(tasks))
	for index := range tasks {
		summaries = append(summaries, taskResponseBody{
			StoredTask:         tasks[index],
			Attachments:        []StoredAttachment{},
			AvailabilityKind:   taskAvailabilityKind(tasks[index]),
			ViewerAction:       taskViewerAction(tasks[index]),
			ReviewerAction:     "none",
			ActiveAssigneeKind: "none",
			ActiveAssigneeID:   "",
		})
	}
	return summaries
}

func taskViewerAction(task StoredTask) string {
	switch task.ParticipationPolicy {
	case "reservation_required":
		return "reserve"
	case "approval_required":
		return "request_approval"
	default:
		return "submit"
	}
}

func taskAvailabilityKind(task StoredTask) string {
	if task.State == "closed" || task.State == "cancelled" || task.State == "refunded" {
		return "closed"
	}
	return "available"
}

type taskFundingBody struct {
	Amount         int64  `json:"amount"`
	IdempotencyKey string `json:"idempotency_key"`
	OrganizationID string `json:"organization_id"`
}

type taskFundingResponseBody struct {
	TaskID string `json:"task_id"`
	Amount int64  `json:"amount"`
	State  string `json:"state"`
}

type taskRefundBody struct {
	TaskID string `json:"task_id"`
	Amount int64  `json:"amount"`
	State  string `json:"state"`
}

type collectibleRefundBody struct {
	Collectibles []StoredCollectible `json:"collectibles"`
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
	case "organization_team":
		return taskVisibilityParts{kind: kind, id: strings.TrimSpace(body.TeamID)}
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

type NotificationHandler struct {
	storage BrowserStorage
	actor   HandlerActor
}

func NewNotificationHandler(storage BrowserStorage, actor HandlerActor) NotificationHandler {
	return NotificationHandler{storage: storage, actor: actor}
}

func (handler NotificationHandler) Handle(request Request) HandleResult {
	if handler.storage == nil {
		return RequestHandleRejected{Reason: "browser storage is required"}
	}
	if handler.actor == nil {
		return RequestHandleRejected{Reason: "handler actor is required"}
	}
	if notificationsPathOnly(request.Path) == "/api/notifications" {
		if request.Method.String() != MethodGet.String() {
			return RequestHandleRejected{Reason: "request method is unsupported for notification listing"}
		}
		return handler.handleList(request)
	}
	notificationID := notificationReadPathID(request.Path)
	if notificationID == "" {
		return RequestHandleRejected{Reason: "request route is not implemented by the WASM demo handler"}
	}
	if request.Method.String() != MethodPost.String() {
		return RequestHandleRejected{Reason: "request method is unsupported for notification mark-read"}
	}
	return handler.handleMarkRead(notificationID)
}

func (handler NotificationHandler) handleList(request Request) HandleResult {
	pageResult := notificationPageFromPath(request.Path)
	page, pageMatched := pageResult.(notificationPageAccepted)
	if !pageMatched {
		return RequestHandleRejected{Reason: pageResult.(notificationPageRejected).reason}
	}
	listResult := ListNotifications(handler.storage, handler.actor.UserID(), page.value)
	listed, listedMatched := listResult.(NotificationsStored)
	if !listedMatched {
		return RequestHandleRejected{Reason: listResult.(NotificationStorageRejected).Reason}
	}
	encoded, err := json.Marshal(notificationsBody{Notifications: listed.Values})
	if err != nil {
		return RequestHandleRejected{Reason: "notifications response encoding failed"}
	}
	return RequestHandled{Value: Response{Status: 200, Body: string(encoded)}}
}

func (handler NotificationHandler) handleMarkRead(notificationID string) HandleResult {
	markResult := MarkNotificationRead(handler.storage, notificationID, handler.actor.UserID())
	marked, markedMatched := markResult.(NotificationStored)
	if !markedMatched {
		return RequestHandleRejected{Reason: markResult.(NotificationStorageRejected).Reason}
	}
	encoded, err := json.Marshal(marked.Value)
	if err != nil {
		return RequestHandleRejected{Reason: "notification response encoding failed"}
	}
	return RequestHandled{Value: Response{Status: 200, Body: string(encoded)}}
}

type notificationsBody struct {
	Notifications []StoredNotification `json:"notifications"`
}

type notificationPageResult interface {
	notificationPageResult()
}

type notificationPageAccepted struct {
	value NotificationPage
}

type notificationPageRejected struct {
	reason string
}

func (notificationPageAccepted) notificationPageResult() {}
func (notificationPageRejected) notificationPageResult() {}

func notificationPageFromPath(path string) notificationPageResult {
	parts := strings.SplitN(path, "?", 2)
	if len(parts) != 2 {
		return notificationPageAccepted{value: DefaultNotificationPage()}
	}
	values, err := url.ParseQuery(parts[1])
	if err != nil {
		return notificationPageRejected{reason: "notification pagination query is invalid"}
	}
	limit, limitMatched := notificationQueryInt(values, "limit", 20)
	if !limitMatched {
		return notificationPageRejected{reason: "notification limit is invalid"}
	}
	offset, offsetMatched := notificationQueryInt(values, "offset", 0)
	if !offsetMatched {
		return notificationPageRejected{reason: "notification offset is invalid"}
	}
	pageResult := NewNotificationPage(limit, offset)
	page, pageMatched := pageResult.(NotificationPageAccepted)
	if !pageMatched {
		return notificationPageRejected{reason: pageResult.(NotificationPageRejected).Reason}
	}
	return notificationPageAccepted{value: page.Value}
}

func notificationQueryInt(values url.Values, key string, defaultValue int) (int, bool) {
	raw := strings.TrimSpace(values.Get(key))
	if raw == "" {
		return defaultValue, true
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil {
		return 0, false
	}
	return parsed, true
}

type OrganizationHandler struct {
	storage  BrowserStorage
	actor    HandlerActor
	ids      OrganizationIDSource
	resolver OrganizationUserResolver
}

func NewOrganizationHandler(storage BrowserStorage, actor HandlerActor, ids OrganizationIDSource, resolver OrganizationUserResolver) OrganizationHandler {
	return OrganizationHandler{storage: storage, actor: actor, ids: ids, resolver: resolver}
}

func (handler OrganizationHandler) Handle(request Request) HandleResult {
	if handler.storage == nil {
		return RequestHandleRejected{Reason: "browser storage is required"}
	}
	if handler.actor == nil {
		return RequestHandleRejected{Reason: "handler actor is required"}
	}
	switch {
	case organizationCollectionPathOnly(request.Path) == "/api/organizations":
		return handler.handleOrganizations(request)
	case organizationTeamsRoute(request.Path) != "":
		return handler.handleOrganizationTeams(request, organizationTeamsRoute(request.Path))
	case standaloneTeamsPathOnly(request.Path) == "/api/teams":
		return handler.handleStandaloneTeams(request)
	case organizationMemberRoute(request.Path) != "":
		return handler.handleOrganizationMembers(request)
	default:
		return RequestHandleRejected{Reason: "request route is not implemented by the WASM demo handler"}
	}
}

func (handler OrganizationHandler) handleOrganizations(request Request) HandleResult {
	switch request.Method.String() {
	case MethodPost.String():
		if handler.ids == nil {
			return RequestHandleRejected{Reason: "organization id source is required"}
		}
		var body organizationBody
		if err := json.Unmarshal([]byte(request.Body), &body); err != nil {
			return RequestHandleRejected{Reason: "organization request body is invalid"}
		}
		organization := StoredOrganization{
			ID:        strings.TrimSpace(handler.ids.NextOrganizationID()),
			Name:      strings.TrimSpace(body.Name),
			CreatedBy: strings.TrimSpace(handler.actor.UserID()),
		}
		saveResult := SaveOrganization(handler.storage, organization)
		saved, savedMatched := saveResult.(OrganizationStored)
		if !savedMatched {
			return RequestHandleRejected{Reason: saveResult.(OrganizationStorageRejected).Reason}
		}
		memberResult := SaveOrganizationMember(handler.storage, StoredOrganizationMember{
			ID:             strings.TrimSpace(handler.ids.NextOrganizationMemberID()),
			OrganizationID: saved.Value.ID,
			UserID:         strings.TrimSpace(handler.actor.UserID()),
			Status:         "active",
			Roles:          []string{"owner"},
		})
		if _, matched := memberResult.(OrganizationMemberStored); !matched {
			return RequestHandleRejected{Reason: memberResult.(OrganizationMemberStorageRejected).Reason}
		}
		return organizationResponseResult(saved.Value, 201)
	case MethodGet.String():
		pageResult := storedListPageFromPath(request.Path, "organization")
		page, pageMatched := pageResult.(storedListPageFromPathAccepted)
		if !pageMatched {
			return RequestHandleRejected{Reason: pageResult.(storedListPageFromPathRejected).reason}
		}
		listResult := ListOrganizations(handler.storage, queryValueFromPath(request.Path), page.value)
		listed, listedMatched := listResult.(OrganizationsStored)
		if !listedMatched {
			return RequestHandleRejected{Reason: listResult.(OrganizationStorageRejected).Reason}
		}
		encoded, err := json.Marshal(organizationsBody{Organizations: listed.Values})
		if err != nil {
			return RequestHandleRejected{Reason: "organizations response encoding failed"}
		}
		return RequestHandled{Value: Response{Status: 200, Body: string(encoded)}}
	default:
		return RequestHandleRejected{Reason: "request method is unsupported for organizations"}
	}
}

func (handler OrganizationHandler) handleOrganizationTeams(request Request, organizationID string) HandleResult {
	if strings.TrimSpace(organizationID) == "" {
		return RequestHandleRejected{Reason: "organization id is required"}
	}
	switch request.Method.String() {
	case MethodPost.String():
		if handler.ids == nil {
			return RequestHandleRejected{Reason: "organization id source is required"}
		}
		var body teamBody
		if err := json.Unmarshal([]byte(request.Body), &body); err != nil {
			return RequestHandleRejected{Reason: "team request body is invalid"}
		}
		team := StoredTeam{
			ID:             strings.TrimSpace(handler.ids.NextTeamID()),
			OwnerKind:      "organization",
			OrganizationID: strings.TrimSpace(organizationID),
			OwnerUserID:    "",
			Name:           strings.TrimSpace(body.Name),
			CreatedBy:      strings.TrimSpace(handler.actor.UserID()),
		}
		saveResult := SaveTeam(handler.storage, team)
		saved, savedMatched := saveResult.(TeamStored)
		if !savedMatched {
			return RequestHandleRejected{Reason: saveResult.(TeamStorageRejected).Reason}
		}
		return teamResponseResult(saved.Value, 201)
	case MethodGet.String():
		pageResult := storedListPageFromPath(request.Path, "organization team")
		page, pageMatched := pageResult.(storedListPageFromPathAccepted)
		if !pageMatched {
			return RequestHandleRejected{Reason: pageResult.(storedListPageFromPathRejected).reason}
		}
		listResult := ListOrganizationTeams(handler.storage, organizationID, queryValueFromPath(request.Path), page.value)
		listed, listedMatched := listResult.(TeamsStored)
		if !listedMatched {
			return RequestHandleRejected{Reason: listResult.(TeamStorageRejected).Reason}
		}
		return teamsResponseResult(listed.Values)
	default:
		return RequestHandleRejected{Reason: "request method is unsupported for organization teams"}
	}
}

func (handler OrganizationHandler) handleStandaloneTeams(request Request) HandleResult {
	switch request.Method.String() {
	case MethodPost.String():
		if handler.ids == nil {
			return RequestHandleRejected{Reason: "organization id source is required"}
		}
		var body teamBody
		if err := json.Unmarshal([]byte(request.Body), &body); err != nil {
			return RequestHandleRejected{Reason: "team request body is invalid"}
		}
		team := StoredTeam{
			ID:             strings.TrimSpace(handler.ids.NextTeamID()),
			OwnerKind:      "user",
			OrganizationID: "",
			OwnerUserID:    strings.TrimSpace(handler.actor.UserID()),
			Name:           strings.TrimSpace(body.Name),
			CreatedBy:      strings.TrimSpace(handler.actor.UserID()),
		}
		saveResult := SaveTeam(handler.storage, team)
		saved, savedMatched := saveResult.(TeamStored)
		if !savedMatched {
			return RequestHandleRejected{Reason: saveResult.(TeamStorageRejected).Reason}
		}
		return teamResponseResult(saved.Value, 201)
	case MethodGet.String():
		pageResult := storedListPageFromPath(request.Path, "standalone team")
		page, pageMatched := pageResult.(storedListPageFromPathAccepted)
		if !pageMatched {
			return RequestHandleRejected{Reason: pageResult.(storedListPageFromPathRejected).reason}
		}
		listResult := ListStandaloneTeams(handler.storage, handler.actor.UserID(), queryValueFromPath(request.Path), page.value)
		listed, listedMatched := listResult.(TeamsStored)
		if !listedMatched {
			return RequestHandleRejected{Reason: listResult.(TeamStorageRejected).Reason}
		}
		return teamsResponseResult(listed.Values)
	default:
		return RequestHandleRejected{Reason: "request method is unsupported for standalone teams"}
	}
}

func (handler OrganizationHandler) handleOrganizationMembers(request Request) HandleResult {
	route := parseOrganizationMemberRoute(request.Path)
	if route.organizationID == "" {
		return RequestHandleRejected{Reason: "request route is not implemented by the WASM demo handler"}
	}
	if route.userID == "" {
		switch request.Method.String() {
		case MethodPost.String():
			return handler.handleProvisionOrganizationMember(request, route.organizationID)
		case MethodGet.String():
			pageResult := storedListPageFromPath(request.Path, "organization member")
			page, pageMatched := pageResult.(storedListPageFromPathAccepted)
			if !pageMatched {
				return RequestHandleRejected{Reason: pageResult.(storedListPageFromPathRejected).reason}
			}
			listResult := ListOrganizationMembers(handler.storage, route.organizationID, page.value)
			listed, listedMatched := listResult.(OrganizationMembersStored)
			if !listedMatched {
				return RequestHandleRejected{Reason: listResult.(OrganizationMemberStorageRejected).Reason}
			}
			return organizationMembersResponseResult(listed.Values, 200)
		default:
			return RequestHandleRejected{Reason: "request method is unsupported for organization members"}
		}
	}
	if route.action == "roles" {
		if request.Method.String() != MethodPatch.String() {
			return RequestHandleRejected{Reason: "request method is unsupported for organization member roles"}
		}
		var body organizationRolesBody
		if err := json.Unmarshal([]byte(request.Body), &body); err != nil {
			return RequestHandleRejected{Reason: "organization member roles body is invalid"}
		}
		updateResult := UpdateOrganizationMemberRoles(handler.storage, route.organizationID, route.userID, body.Roles)
		updated, updatedMatched := updateResult.(OrganizationMemberStored)
		if !updatedMatched {
			return RequestHandleRejected{Reason: updateResult.(OrganizationMemberStorageRejected).Reason}
		}
		return organizationMemberResponseResult(updated.Value, 200)
	}
	if route.action == "deactivate" {
		if request.Method.String() != MethodPatch.String() {
			return RequestHandleRejected{Reason: "request method is unsupported for organization member deactivation"}
		}
		deactivateResult := DeactivateOrganizationMember(handler.storage, route.organizationID, route.userID)
		if _, deactivated := deactivateResult.(OrganizationMemberStored); !deactivated {
			return RequestHandleRejected{Reason: deactivateResult.(OrganizationMemberStorageRejected).Reason}
		}
		encoded, err := json.Marshal(statusBody{Status: "deactivated"})
		if err != nil {
			return RequestHandleRejected{Reason: "organization member deactivation response encoding failed"}
		}
		return RequestHandled{Value: Response{Status: 200, Body: string(encoded)}}
	}
	return RequestHandleRejected{Reason: "request route is not implemented by the WASM demo handler"}
}

func (handler OrganizationHandler) handleProvisionOrganizationMember(request Request, organizationID string) HandleResult {
	if handler.ids == nil {
		return RequestHandleRejected{Reason: "organization id source is required"}
	}
	if handler.resolver == nil {
		return RequestHandleRejected{Reason: "organization user resolver is required"}
	}
	var body provisionOrganizationMemberBody
	if err := json.Unmarshal([]byte(request.Body), &body); err != nil {
		return RequestHandleRejected{Reason: "organization member request body is invalid"}
	}
	email := strings.TrimSpace(body.Email)
	userID, resolved := handler.resolver.UserIDForEmail(email)
	if !resolved || strings.TrimSpace(userID) == "" {
		return RequestHandleRejected{Reason: "organization member user was not found"}
	}
	member := StoredOrganizationMember{
		ID:             strings.TrimSpace(handler.ids.NextOrganizationMemberID()),
		OrganizationID: strings.TrimSpace(organizationID),
		UserID:         strings.TrimSpace(userID),
		Status:         "active",
		Roles:          body.Roles,
	}
	saveResult := SaveOrganizationMember(handler.storage, member)
	saved, savedMatched := saveResult.(OrganizationMemberStored)
	if !savedMatched {
		return RequestHandleRejected{Reason: saveResult.(OrganizationMemberStorageRejected).Reason}
	}
	return organizationMemberResponseResult(saved.Value, 201)
}

type organizationBody struct {
	Name string `json:"name"`
}

type teamBody struct {
	Name string `json:"name"`
}

type provisionOrganizationMemberBody struct {
	Email string   `json:"email"`
	Roles []string `json:"roles"`
}

type organizationRolesBody struct {
	Roles []string `json:"roles"`
}

type organizationsBody struct {
	Organizations []StoredOrganization `json:"organizations"`
}

type organizationMemberBody struct {
	ID             string   `json:"id"`
	OrganizationID string   `json:"organization_id"`
	UserID         string   `json:"user_id"`
	Status         string   `json:"status"`
	Roles          []string `json:"roles"`
}

type organizationMembersBody struct {
	Members []organizationMemberBody `json:"members"`
}

type teamsBody struct {
	Teams []StoredTeam `json:"teams"`
}

type statusBody struct {
	Status string `json:"status"`
}

type organizationMemberRouteParts struct {
	organizationID string
	userID         string
	action         string
}

func organizationResponseResult(organization StoredOrganization, status int) HandleResult {
	encoded, err := json.Marshal(organization)
	if err != nil {
		return RequestHandleRejected{Reason: "organization response encoding failed"}
	}
	return RequestHandled{Value: Response{Status: status, Body: string(encoded)}}
}

func teamResponseResult(team StoredTeam, status int) HandleResult {
	encoded, err := json.Marshal(team)
	if err != nil {
		return RequestHandleRejected{Reason: "team response encoding failed"}
	}
	return RequestHandled{Value: Response{Status: status, Body: string(encoded)}}
}

func teamsResponseResult(teams []StoredTeam) HandleResult {
	encoded, err := json.Marshal(teamsBody{Teams: teams})
	if err != nil {
		return RequestHandleRejected{Reason: "teams response encoding failed"}
	}
	return RequestHandled{Value: Response{Status: 200, Body: string(encoded)}}
}

func organizationMemberResponseResult(member StoredOrganizationMember, status int) HandleResult {
	encoded, err := json.Marshal(organizationMemberToBody(member))
	if err != nil {
		return RequestHandleRejected{Reason: "organization member response encoding failed"}
	}
	return RequestHandled{Value: Response{Status: status, Body: string(encoded)}}
}

func organizationMembersResponseResult(members []StoredOrganizationMember, status int) HandleResult {
	values := make([]organizationMemberBody, 0, len(members))
	for index := range members {
		values = append(values, organizationMemberToBody(members[index]))
	}
	encoded, err := json.Marshal(organizationMembersBody{Members: values})
	if err != nil {
		return RequestHandleRejected{Reason: "organization members response encoding failed"}
	}
	return RequestHandled{Value: Response{Status: status, Body: string(encoded)}}
}

func organizationMemberToBody(member StoredOrganizationMember) organizationMemberBody {
	return organizationMemberBody{
		ID:             member.ID,
		OrganizationID: member.OrganizationID,
		UserID:         member.UserID,
		Status:         member.Status,
		Roles:          member.Roles,
	}
}

func parseOrganizationMemberRoute(path string) organizationMemberRouteParts {
	parts := strings.Split(strings.Trim(strings.SplitN(path, "?", 2)[0], "/"), "/")
	if len(parts) == 4 && parts[0] == "api" && parts[1] == "organizations" && parts[3] == "members" {
		return organizationMemberRouteParts{organizationID: strings.TrimSpace(parts[2])}
	}
	if len(parts) == 6 && parts[0] == "api" && parts[1] == "organizations" && parts[3] == "members" {
		return organizationMemberRouteParts{
			organizationID: strings.TrimSpace(parts[2]),
			userID:         strings.TrimSpace(parts[4]),
			action:         strings.TrimSpace(parts[5]),
		}
	}
	return organizationMemberRouteParts{}
}

type storedListPageFromPathResult interface {
	storedListPageFromPathResult()
}

type storedListPageFromPathAccepted struct {
	value StoredListPage
}

type storedListPageFromPathRejected struct {
	reason string
}

func (storedListPageFromPathAccepted) storedListPageFromPathResult() {}
func (storedListPageFromPathRejected) storedListPageFromPathResult() {}

func storedListPageFromPath(path string, label string) storedListPageFromPathResult {
	parts := strings.SplitN(path, "?", 2)
	if len(parts) != 2 {
		return storedListPageFromPathAccepted{value: DefaultStoredListPage()}
	}
	values, err := url.ParseQuery(parts[1])
	if err != nil {
		return storedListPageFromPathRejected{reason: label + " pagination query is invalid"}
	}
	limit, limitMatched := notificationQueryInt(values, "limit", 20)
	if !limitMatched {
		return storedListPageFromPathRejected{reason: label + " limit is invalid"}
	}
	offset, offsetMatched := notificationQueryInt(values, "offset", 0)
	if !offsetMatched {
		return storedListPageFromPathRejected{reason: label + " offset is invalid"}
	}
	pageResult := NewStoredListPage(limit, offset)
	page, pageMatched := pageResult.(StoredListPageAccepted)
	if !pageMatched {
		return storedListPageFromPathRejected{reason: pageResult.(StoredListPageRejected).Reason}
	}
	return storedListPageFromPathAccepted{value: page.Value}
}

func queryValueFromPath(path string) string {
	parts := strings.SplitN(path, "?", 2)
	if len(parts) != 2 {
		return ""
	}
	values, err := url.ParseQuery(parts[1])
	if err != nil {
		return ""
	}
	return strings.TrimSpace(values.Get("query"))
}
