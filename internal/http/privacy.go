package httpserver

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/e6qu/sharecrop/internal/audit"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/submission"
)

const (
	privacyKindDataExport              = "data_export"
	privacyKindSensitiveFieldDeletion  = "sensitive_field_deletion"
	privacyRequestQueuedStatus         = "queued"
	privacyRequestResolvedStatus       = "resolved"
	privacyRequestAuditSubjectKind     = "privacy_request"
	privacyRequestMetadataEncodingFail = "privacy request metadata could not be encoded"
)

type PrivacyRequestRecord struct {
	ID                 string
	RequestedBy        core.UserID
	Kind               string
	State              string
	ExportJSON         string
	ResolutionNote     string
	CreatedAt          time.Time
	ResolvedAt         time.Time
	RedactedFieldCount int
}

type PrivacyMutationResult interface {
	privacyMutationResult()
}

type PrivacyRequestSaved struct {
	Value PrivacyRequestRecord
}

type PrivacyRequestMutationRejected struct {
	Reason core.DomainError
}

func (PrivacyRequestSaved) privacyMutationResult() {}

func (PrivacyRequestMutationRejected) privacyMutationResult() {}

type PrivacyListResult interface {
	privacyListResult()
}

type PrivacyRequestsListed struct {
	Values []PrivacyRequestRecord
}

type PrivacyRequestListRejected struct {
	Reason core.DomainError
}

func (PrivacyRequestsListed) privacyListResult() {}

func (PrivacyRequestListRejected) privacyListResult() {}

type PrivacyRetentionResult interface {
	privacyRetentionResult()
}

type PrivacyRetentionRun struct {
	RedactedFieldCount int
}

type PrivacyRetentionRejected struct {
	Reason core.DomainError
}

func (PrivacyRetentionRun) privacyRetentionResult()      {}
func (PrivacyRetentionRejected) privacyRetentionResult() {}

// SensitiveFieldRedactor performs the actual "delete on request" sensitive
// field redaction across a user's submissions - the in-memory
// memoryPrivacyService only tracks privacy *requests*, it has no submission
// data of its own, so redaction is delegated to whatever store (Postgres or,
// for the browser demo, browser storage) actually holds submissions.
type SensitiveFieldRedactor interface {
	RedactSensitiveFields(ctx context.Context, userID core.UserID) (int, error)
}

type memoryPrivacyService struct {
	mu       sync.Mutex
	requests []PrivacyRequestRecord
	redactor SensitiveFieldRedactor
}

func newMemoryPrivacyService() *memoryPrivacyService {
	return &memoryPrivacyService{requests: []PrivacyRequestRecord{}}
}

// NewMemoryPrivacyService is newMemoryPrivacyService exported for callers
// (e.g. cmd/sharecrop-wasm) that need in-memory privacy-request tracking
// wired to a real SensitiveFieldRedactor instead of DefaultRuntimeState's
// zero-redactor default.
func NewMemoryPrivacyService(redactor SensitiveFieldRedactor) PrivacyService {
	return &memoryPrivacyService{requests: []PrivacyRequestRecord{}, redactor: redactor}
}

func (service *memoryPrivacyService) Create(_ context.Context, requester core.UserID, kind string) PrivacyMutationResult {
	requestIDResult := core.NewPrivacyRequestID()
	requestID, requestIDMatched := requestIDResult.(core.PrivacyRequestIDCreated)
	if !requestIDMatched {
		return PrivacyRequestMutationRejected{Reason: requestIDResult.(core.PrivacyRequestIDRejected).Reason}
	}
	record := PrivacyRequestRecord{ID: requestID.Value.String(), RequestedBy: requester, Kind: kind, State: privacyRequestQueuedStatus, CreatedAt: time.Now().UTC()}
	service.mu.Lock()
	defer service.mu.Unlock()
	service.requests = append([]PrivacyRequestRecord{record}, service.requests...)
	return PrivacyRequestSaved{Value: record}
}

func (service *memoryPrivacyService) ListForRequester(_ context.Context, requester core.UserID, page core.Page) PrivacyListResult {
	service.mu.Lock()
	defer service.mu.Unlock()
	filtered := make([]PrivacyRequestRecord, 0)
	for index := range service.requests {
		if service.requests[index].RequestedBy == requester {
			filtered = append(filtered, service.requests[index])
		}
	}
	return PrivacyRequestsListed{Values: privacyPage(filtered, page)}
}

func (service *memoryPrivacyService) ListAll(_ context.Context, page core.Page) PrivacyListResult {
	service.mu.Lock()
	defer service.mu.Unlock()
	values := make([]PrivacyRequestRecord, len(service.requests))
	copy(values, service.requests)
	return PrivacyRequestsListed{Values: privacyPage(values, page)}
}

func (service *memoryPrivacyService) Resolve(ctx context.Context, requestID string, note string) PrivacyMutationResult {
	service.mu.Lock()
	defer service.mu.Unlock()
	for index := range service.requests {
		if service.requests[index].ID != requestID {
			continue
		}
		record := service.requests[index]
		if record.State != privacyRequestQueuedStatus {
			return PrivacyRequestMutationRejected{Reason: core.NewDomainError(core.ErrorCodeConflict, "privacy request is already resolved")}
		}
		record.State = privacyRequestResolvedStatus
		record.ResolutionNote = strings.TrimSpace(note)
		record.ResolvedAt = time.Now().UTC()
		if record.Kind == privacyKindDataExport {
			exportBytes, err := json.Marshal(map[string]string{"user_id": record.RequestedBy.String(), "export_scope": "memory"})
			if err != nil {
				return PrivacyRequestMutationRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "privacy export encoding failed")}
			}
			record.ExportJSON = string(exportBytes)
		}
		if record.Kind == privacyKindSensitiveFieldDeletion && service.redactor != nil {
			count, err := service.redactor.RedactSensitiveFields(ctx, record.RequestedBy)
			if err != nil {
				return PrivacyRequestMutationRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "redact sensitive fields failed")}
			}
			record.RedactedFieldCount = count
		}
		service.requests[index] = record
		return PrivacyRequestSaved{Value: record}
	}
	return PrivacyRequestMutationRejected{Reason: core.NewDomainError(core.ErrorCodeNotFound, "privacy request was not found")}
}

func (service *memoryPrivacyService) RecordSensitiveFieldAccess(_ context.Context, _ core.UserID, _ submission.Submission) PrivacyMutationResult {
	return PrivacyRequestSaved{Value: PrivacyRequestRecord{CreatedAt: time.Now().UTC()}}
}

// RecordSensitiveFieldAccessBatch mirrors RecordSensitiveFieldAccess for the
// in-memory runtime: the memory privacy service tracks privacy requests only
// and holds no access-event log, so the batch recording is a no-op success.
func (service *memoryPrivacyService) RecordSensitiveFieldAccessBatch(_ context.Context, _ core.UserID, _ []submission.Submission) PrivacyMutationResult {
	return PrivacyRequestSaved{Value: PrivacyRequestRecord{CreatedAt: time.Now().UTC()}}
}

func (service *memoryPrivacyService) RunRetention(ctx context.Context, actor core.UserID) PrivacyRetentionResult {
	if service.redactor != nil {
		count, err := service.redactor.RedactSensitiveFields(ctx, actor)
		if err != nil {
			return PrivacyRetentionRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "redact sensitive fields failed")}
		}
		return PrivacyRetentionRun{RedactedFieldCount: count}
	}
	service.mu.Lock()
	defer service.mu.Unlock()
	return PrivacyRetentionRun{RedactedFieldCount: 0}
}

func privacyPage(values []PrivacyRequestRecord, page core.Page) []PrivacyRequestRecord {
	start := page.Offset()
	if start > len(values) {
		start = len(values)
	}
	end := start + page.Limit()
	if end > len(values) {
		end = len(values)
	}
	return values[start:end]
}

func (server Server) createPrivacyRequest(w http.ResponseWriter, r *http.Request) {
	actorResult := server.requireUserSubject(r)
	actor, matched := actorResult.(userSubjectAccepted)
	if !matched {
		writeError(w, http.StatusUnauthorized, core.ErrorCodeUnauthenticated, actorResult.(userSubjectRejected).reason)
		return
	}

	var request privacyRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, core.ErrorCodeInvalidArgument, "request body is invalid")
		return
	}
	if !validPrivacyRequestKind(request.Kind) {
		writeError(w, http.StatusBadRequest, core.ErrorCodeInvalidArgument, "privacy request kind is invalid")
		return
	}

	result := server.privacyService.Create(r.Context(), actor.subject.ID, request.Kind)
	saved, savedMatched := result.(PrivacyRequestSaved)
	if !savedMatched {
		writeDomainError(w, result.(PrivacyRequestMutationRejected).Reason)
		return
	}

	server.recordPrivacyAuditBestEffort(r, actor.subject.ID, audit.ActionPrivacyRequestCreated, saved.Value, actor.subject.ID.String())

	writeJSON(w, http.StatusCreated, privacyRequestToResponse(saved.Value))
}

func (server Server) listPrivacyRequests(w http.ResponseWriter, r *http.Request) {
	actorResult := server.requireUserSubject(r)
	actor, matched := actorResult.(userSubjectAccepted)
	if !matched {
		writeError(w, http.StatusUnauthorized, core.ErrorCodeUnauthenticated, actorResult.(userSubjectRejected).reason)
		return
	}
	page, pageOK := parsePageOrReject(w, r)
	if !pageOK {
		return
	}
	result := server.privacyService.ListForRequester(r.Context(), actor.subject.ID, page.Probe())
	server.writePrivacyListResult(w, result, page)
}

func (server Server) listAdminPrivacyRequests(w http.ResponseWriter, r *http.Request) {
	if _, ok := server.requireAdminSubject(w, r); !ok {
		return
	}
	page, pageOK := parsePageOrReject(w, r)
	if !pageOK {
		return
	}
	result := server.privacyService.ListAll(r.Context(), page.Probe())
	server.writePrivacyListResult(w, result, page)
}

func (server Server) resolveAdminPrivacyRequest(w http.ResponseWriter, r *http.Request) {
	actor, ok := server.requireAdminSubject(w, r)
	if !ok {
		return
	}
	var request privacyResolveRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, core.ErrorCodeInvalidArgument, "request body is invalid")
		return
	}
	privacyRequestIDResult := core.ParsePrivacyRequestID(r.PathValue("privacy_request_id"))
	privacyRequestID, privacyRequestIDMatched := privacyRequestIDResult.(core.PrivacyRequestIDCreated)
	if !privacyRequestIDMatched {
		writeDomainError(w, privacyRequestIDResult.(core.PrivacyRequestIDRejected).Reason)
		return
	}
	result := server.privacyService.Resolve(r.Context(), privacyRequestID.Value.String(), request.ResolutionNote)
	saved, savedMatched := result.(PrivacyRequestSaved)
	if !savedMatched {
		writeDomainError(w, result.(PrivacyRequestMutationRejected).Reason)
		return
	}
	server.recordPrivacyAuditBestEffort(r, actor.subject.ID, audit.ActionPrivacyRequestResolved, saved.Value, saved.Value.ID)
	writeJSON(w, http.StatusOK, privacyRequestToResponse(saved.Value))
}

func (server Server) runPrivacyRetention(w http.ResponseWriter, r *http.Request) {
	actor, ok := server.requireAdminSubject(w, r)
	if !ok {
		return
	}
	result := server.privacyService.RunRetention(r.Context(), actor.subject.ID)
	run, matched := result.(PrivacyRetentionRun)
	if !matched {
		writeDomainError(w, result.(PrivacyRetentionRejected).Reason)
		return
	}
	metadataResult := encodeJSONMetadata(map[string]string{"redacted_field_count": strconv.Itoa(run.RedactedFieldCount)})
	metadata, metadataMatched := metadataResult.(jsonMetadataEncoded)
	if !metadataMatched {
		writeDomainError(w, metadataResult.(jsonMetadataRejected).reason)
		return
	}
	server.recordAuditBestEffort(r.Context(), actor.subject.ID, audit.ActionPrivacyRetentionRun, audit.Subject{Kind: "privacy_retention", ID: actor.subject.ID.String()}, audit.Metadata{JSON: metadata.value})
	writeJSON(w, http.StatusOK, privacyRetentionRunResponse{RedactedFieldCount: run.RedactedFieldCount})
}

func (server Server) writePrivacyListResult(w http.ResponseWriter, result PrivacyListResult, page core.Page) {
	listed, matched := result.(PrivacyRequestsListed)
	if !matched {
		writeDomainError(w, result.(PrivacyRequestListRejected).Reason)
		return
	}
	visible, nextOffset := probeListWindow(len(listed.Values), page)
	response := privacyRequestsResponse{Requests: make([]privacyRequestResponse, 0, visible), NextOffset: nextOffset}
	for index := range listed.Values[:visible] {
		response.Requests = append(response.Requests, privacyRequestToResponse(listed.Values[index]))
	}
	writeJSON(w, http.StatusOK, response)
}

// recordPrivacyAuditBestEffort audits a privacy-request mutation that has
// already committed; failures are logged, never surfaced (see
// recordAuditBestEffort).
func (server Server) recordPrivacyAuditBestEffort(r *http.Request, actor core.UserID, action audit.Action, record PrivacyRequestRecord, subjectID string) {
	metadataBytes, err := json.Marshal(map[string]string{"kind": record.Kind, "state": record.State})
	if err != nil {
		slog.Error(privacyRequestMetadataEncodingFail, "action", action.String(), "privacy_request_id", record.ID)
		return
	}
	server.recordAuditBestEffort(
		r.Context(),
		actor,
		action,
		audit.Subject{Kind: privacyRequestAuditSubjectKind, ID: subjectID},
		audit.Metadata{JSON: string(metadataBytes)},
	)
}

func privacyRequestToResponse(record PrivacyRequestRecord) privacyRequestResponse {
	resolvedAt := ""
	if !record.ResolvedAt.IsZero() {
		resolvedAt = record.ResolvedAt.Format(time.RFC3339)
	}
	return privacyRequestResponse{
		ID:                 record.ID,
		Kind:               record.Kind,
		Status:             record.State,
		RequestedBy:        record.RequestedBy.String(),
		ExportJSON:         record.ExportJSON,
		ResolutionNote:     record.ResolutionNote,
		CreatedAt:          record.CreatedAt.Format(time.RFC3339),
		ResolvedAt:         resolvedAt,
		RedactedFieldCount: record.RedactedFieldCount,
	}
}

func validPrivacyRequestKind(kind string) bool {
	return kind == privacyKindDataExport || kind == privacyKindSensitiveFieldDeletion
}
