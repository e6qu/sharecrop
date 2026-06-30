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
