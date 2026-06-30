package wasmdemo

import (
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
	if handler.ids == nil {
		return RequestHandleRejected{Reason: "privacy request id source is required"}
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
