package httpserver

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/e6qu/sharecrop/internal/audit"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/org"
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
	return audit.EventRecorded{Value: service.events[len(service.events)-1]}
}

func (service *memoryAuditService) Get(_ context.Context, id core.AuditEventID) audit.GetResult {
	service.mu.Lock()
	defer service.mu.Unlock()
	for _, event := range service.events {
		if event.ID.String() == id.String() {
			return audit.EventFound{Value: event}
		}
	}
	return audit.GetRejected{Reason: core.NewDomainError(core.ErrorCodeNotFound, "audit event was not found")}
}

func (service *memoryAuditService) List(_ context.Context, filters audit.ListFilters, page core.Page) audit.ListResult {
	service.mu.Lock()
	defer service.mu.Unlock()
	filtered := make([]audit.Event, 0, len(service.events))
	for eventIndex := range service.events {
		event := service.events[eventIndex]
		if !auditEventMatchesFilters(event, filters) {
			continue
		}
		filtered = append(filtered, event)
	}
	start := page.Offset()
	if start > len(filtered) {
		start = len(filtered)
	}
	end := start + page.Limit()
	if end > len(filtered) {
		end = len(filtered)
	}
	values := make([]audit.Event, 0, end-start)
	for index := end - 1; index >= start; index-- {
		values = append(values, filtered[index])
	}
	return audit.EventsListed{Values: values}
}

func auditEventMatchesFilters(event audit.Event, filters audit.ListFilters) bool {
	switch filter := filters.Action.(type) {
	case audit.ActionEquals:
		if event.Action.String() != filter.Value.String() {
			return false
		}
	case audit.AnyAction:
	default:
		return false
	}
	switch filter := filters.SubjectKind.(type) {
	case audit.SubjectKindEquals:
		if event.Subject.Kind != filter.Value {
			return false
		}
	case audit.AnySubjectKind:
	default:
		return false
	}
	switch filter := filters.SubjectID.(type) {
	case audit.SubjectIDEquals:
		if event.Subject.ID != filter.Value {
			return false
		}
	case audit.AnySubjectID:
	default:
		return false
	}
	return true
}

func (server Server) listAuditEvents(w http.ResponseWriter, r *http.Request) {
	if _, ok := server.requireAdminSubject(w, r); !ok {
		return
	}

	filters := parseAuditListFilters(r)
	result := server.auditService.List(r.Context(), filters, parsePage(r))
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

func (server Server) listOrganizationAuditEvents(w http.ResponseWriter, r *http.Request) {
	actorResult := server.requireUserSubject(r)
	actor, matched := actorResult.(userSubjectAccepted)
	if !matched {
		writeError(w, http.StatusUnauthorized, actorResult.(userSubjectRejected).reason)
		return
	}

	organizationIDResult := parseOrganizationPathValue(r)
	organizationIDAccepted, organizationIDMatched := organizationIDResult.(organizationIDAccepted)
	if !organizationIDMatched {
		writeError(w, http.StatusBadRequest, organizationIDResult.(organizationIDRejected).reason)
		return
	}

	check := server.organizationService.CheckOrganizationPermission(r.Context(), organizationIDAccepted.value, actor.subject.ID, org.PermissionManageMembers)
	if rejected, denied := check.(org.PermissionDenied); denied {
		writeDomainError(w, rejected.Reason)
		return
	}

	filters := audit.NoListFilters()
	filters.SubjectID = audit.SubjectIDEquals{Value: organizationIDAccepted.value.String()}
	result := server.auditService.List(r.Context(), filters, parsePage(r))
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

func parseAuditListFilters(r *http.Request) audit.ListFilters {
	filters := audit.NoListFilters()
	if rawAction := strings.TrimSpace(r.URL.Query().Get("action")); rawAction != "" {
		filters.Action = audit.ActionEquals{Value: audit.ActionFromString(rawAction)}
	}
	if rawSubjectKind := strings.TrimSpace(r.URL.Query().Get("subject_kind")); rawSubjectKind != "" {
		filters.SubjectKind = audit.SubjectKindEquals{Value: rawSubjectKind}
	}
	if rawSubjectID := strings.TrimSpace(r.URL.Query().Get("subject_id")); rawSubjectID != "" {
		filters.SubjectID = audit.SubjectIDEquals{Value: rawSubjectID}
	}
	return filters
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
