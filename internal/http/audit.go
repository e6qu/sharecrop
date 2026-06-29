package httpserver

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/e6qu/sharecrop/internal/audit"
	"github.com/e6qu/sharecrop/internal/core"
)

type auditEventResponse struct {
	ID           string `json:"id"`
	ActorUserID  string `json:"actor_user_id"`
	Action       string `json:"action"`
	SubjectKind  string `json:"subject_kind"`
	SubjectID    string `json:"subject_id"`
	MetadataJSON string `json:"metadata_json"`
	CreatedAt    string `json:"created_at"`
}

type auditEventsResponse struct {
	Events []auditEventResponse `json:"events"`
}

func (auditEventsResponse) writableResponse() {}

type memoryAuditService struct {
	mu     sync.Mutex
	events []audit.Event
}

func newMemoryAuditService() *memoryAuditService {
	return &memoryAuditService{events: []audit.Event{}}
}

func (service *memoryAuditService) Record(_ context.Context, actor core.UserID, action audit.Action, subject audit.Subject, metadata audit.Metadata) audit.RecordResult {
	idResult := core.NewAuditEventID()
	id, matched := idResult.(core.AuditEventIDCreated)
	if !matched {
		return audit.RecordRejected{Reason: idResult.(core.AuditEventIDRejected).Reason}
	}
	service.mu.Lock()
	defer service.mu.Unlock()
	service.events = append(service.events, audit.Event{
		ID:          id.Value,
		ActorUserID: actor,
		Action:      action,
		Subject:     subject,
		Metadata:    metadata,
		CreatedAt:   time.Now().UTC(),
	})
	return audit.EventRecorded{}
}

func (service *memoryAuditService) List(_ context.Context, page core.Page) audit.ListResult {
	service.mu.Lock()
	defer service.mu.Unlock()
	start := page.Offset()
	if start > len(service.events) {
		start = len(service.events)
	}
	end := start + page.Limit()
	if end > len(service.events) {
		end = len(service.events)
	}
	values := make([]audit.Event, 0, end-start)
	for index := end - 1; index >= start; index-- {
		values = append(values, service.events[index])
	}
	return audit.EventsListed{Values: values}
}

func (server Server) listAuditEvents(w http.ResponseWriter, r *http.Request) {
	actorResult := server.requireUserSubject(r)
	actor, matched := actorResult.(userSubjectAccepted)
	if !matched {
		writeError(w, http.StatusUnauthorized, actorResult.(userSubjectRejected).reason)
		return
	}
	if !server.adminUserIDs[actor.subject.ID.String()] {
		writeError(w, http.StatusForbidden, "platform admin access is required")
		return
	}

	result := server.auditService.List(r.Context(), parsePage(r))
	listed, listedMatched := result.(audit.EventsListed)
	if !listedMatched {
		writeDomainError(w, result.(audit.ListRejected).Reason)
		return
	}

	response := auditEventsResponse{Events: make([]auditEventResponse, 0, len(listed.Values))}
	for _, event := range listed.Values {
		response.Events = append(response.Events, auditEventToResponse(event))
	}
	writeJSON(w, http.StatusOK, response)
}

func (server Server) recordAudit(w http.ResponseWriter, ctx context.Context, actor core.UserID, action audit.Action, subject audit.Subject, metadata audit.Metadata) bool {
	result := server.auditService.Record(ctx, actor, action, subject, metadata)
	if rejected, matched := result.(audit.RecordRejected); matched {
		writeDomainError(w, rejected.Reason)
		return false
	}
	return true
}

func auditEventToResponse(event audit.Event) auditEventResponse {
	return auditEventResponse{
		ID:           event.ID.String(),
		ActorUserID:  event.ActorUserID.String(),
		Action:       event.Action.String(),
		SubjectKind:  event.Subject.Kind,
		SubjectID:    event.Subject.ID,
		MetadataJSON: event.Metadata.JSON,
		CreatedAt:    event.CreatedAt.UTC().Format(time.RFC3339Nano),
	}
}
