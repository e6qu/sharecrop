package httpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/e6qu/sharecrop/internal/audit"
	"github.com/e6qu/sharecrop/internal/core"
)

const moderationDetailsMaxLength = 2000

type moderationMetadata struct {
	Reason  string `json:"reason"`
	Details string `json:"details"`
}

type rawModerationMetadata struct {
	Reason  string  `json:"reason"`
	Details *string `json:"details"`
}

type ModerationTriageRecord struct {
	ReportID       core.AuditEventID
	State          string
	ResolutionNote string
	UpdatedBy      string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type ModerationTriageMutationResult interface{ moderationTriageMutationResult() }
type ModerationTriageSaved struct{ Value ModerationTriageRecord }
type ModerationTriageMutationRejected struct{ Reason core.DomainError }

func (ModerationTriageSaved) moderationTriageMutationResult()            {}
func (ModerationTriageMutationRejected) moderationTriageMutationResult() {}

type ModerationTriageListResult interface{ moderationTriageListResult() }
type ModerationTriageListed struct{ Values []ModerationTriageRecord }
type ModerationTriageListRejected struct{ Reason core.DomainError }

func (ModerationTriageListed) moderationTriageListResult()       {}
func (ModerationTriageListRejected) moderationTriageListResult() {}

type memoryModerationTriageService struct {
	mu      sync.Mutex
	records map[string]ModerationTriageRecord
}

func newMemoryModerationTriageService() *memoryModerationTriageService {
	return &memoryModerationTriageService{records: map[string]ModerationTriageRecord{}}
}

func (service *memoryModerationTriageService) RecordOpen(_ context.Context, event audit.Event) ModerationTriageMutationResult {
	record := ModerationTriageRecord{ReportID: event.ID, State: "open", ResolutionNote: "", CreatedAt: event.CreatedAt, UpdatedAt: event.CreatedAt}
	service.mu.Lock()
	defer service.mu.Unlock()
	service.records[event.ID.String()] = record
	return ModerationTriageSaved{Value: record}
}

func (service *memoryModerationTriageService) List(_ context.Context, ids []core.AuditEventID) ModerationTriageListResult {
	service.mu.Lock()
	defer service.mu.Unlock()
	values := make([]ModerationTriageRecord, 0, len(ids))
	for _, id := range ids {
		record, ok := service.records[id.String()]
		if !ok {
			return ModerationTriageListRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "moderation report triage state is missing")}
		}
		values = append(values, record)
	}
	return ModerationTriageListed{Values: values}
}

func (service *memoryModerationTriageService) Update(_ context.Context, actor core.UserID, reportID core.AuditEventID, state string, note string) ModerationTriageMutationResult {
	if !validModerationTriageState(state) {
		return ModerationTriageMutationRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidEnum, "moderation triage state is invalid")}
	}
	service.mu.Lock()
	defer service.mu.Unlock()
	record, ok := service.records[reportID.String()]
	if !ok {
		return ModerationTriageMutationRejected{Reason: core.NewDomainError(core.ErrorCodeNotFound, "moderation report was not found")}
	}
	record.State = state
	record.ResolutionNote = strings.TrimSpace(note)
	record.UpdatedBy = actor.String()
	record.UpdatedAt = time.Now().UTC()
	service.records[reportID.String()] = record
	return ModerationTriageSaved{Value: record}
}

func (server Server) createModerationReport(w http.ResponseWriter, r *http.Request) {
	actorResult := server.requireUserSubject(r)
	actor, matched := actorResult.(userSubjectAccepted)
	if !matched {
		writeError(w, http.StatusUnauthorized, actorResult.(userSubjectRejected).reason)
		return
	}

	var request moderationReportRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "request body is invalid")
		return
	}

	subjectKind := strings.TrimSpace(request.SubjectKind)
	if !validModerationSubjectKind(subjectKind) {
		writeDomainError(w, core.NewDomainError(core.ErrorCodeInvalidEnum, "moderation subject kind is invalid"))
		return
	}
	subjectID := strings.TrimSpace(request.SubjectID)
	if subjectID == "" {
		writeDomainError(w, core.NewDomainError(core.ErrorCodeInvalidArgument, "moderation subject id is required"))
		return
	}
	reason := strings.TrimSpace(request.Reason)
	if !validModerationReason(reason) {
		writeDomainError(w, core.NewDomainError(core.ErrorCodeInvalidEnum, "moderation reason is invalid"))
		return
	}
	details := strings.TrimSpace(request.Details)
	if len(details) > moderationDetailsMaxLength {
		writeDomainError(w, core.NewDomainError(core.ErrorCodeInvalidArgument, "moderation details are too long"))
		return
	}

	metadataResult := encodeModerationMetadata(reason, details)
	metadata, metadataMatched := metadataResult.(moderationMetadataEncoded)
	if !metadataMatched {
		writeDomainError(w, metadataResult.(moderationMetadataRejected).reason)
		return
	}

	result := server.auditService.Record(
		r.Context(),
		actor.subject.ID,
		audit.ActionModerationReportCreated,
		audit.Subject{Kind: subjectKind, ID: subjectID},
		audit.Metadata{JSON: metadata.value},
	)
	if rejected, rejectedMatched := result.(audit.RecordRejected); rejectedMatched {
		writeDomainError(w, rejected.Reason)
		return
	}
	recorded, recordedMatched := result.(audit.EventRecorded)
	if !recordedMatched {
		writeError(w, http.StatusInternalServerError, "moderation report was not recorded")
		return
	}
	responseResult := moderationReportFromAuditEvent(recorded.Value)
	response, responseMatched := responseResult.(moderationReportConverted)
	if !responseMatched {
		writeDomainError(w, responseResult.(moderationReportConversionRejected).reason)
		return
	}
	triageResult := server.moderationTriage.RecordOpen(r.Context(), recorded.Value)
	triage, triageMatched := triageResult.(ModerationTriageSaved)
	if !triageMatched {
		writeDomainError(w, triageResult.(ModerationTriageMutationRejected).Reason)
		return
	}
	response.value = applyModerationTriage(response.value, triage.Value)
	writeJSON(w, http.StatusCreated, response.value)
}

func (server Server) listAdminModerationReports(w http.ResponseWriter, r *http.Request) {
	if _, ok := server.requireAdminSubject(w, r); !ok {
		return
	}

	filters := audit.NoListFilters()
	filters.Action = audit.ActionEquals{Value: audit.ActionModerationReportCreated}
	result := server.auditService.List(r.Context(), filters, parsePage(r))
	listed, listedMatched := result.(audit.EventsListed)
	if !listedMatched {
		writeDomainError(w, result.(audit.ListRejected).Reason)
		return
	}

	reportIDs := make([]core.AuditEventID, 0, len(listed.Values))
	for _, event := range listed.Values {
		reportIDs = append(reportIDs, event.ID)
	}
	triageResult := server.moderationTriage.List(r.Context(), reportIDs)
	triageListed, triageMatched := triageResult.(ModerationTriageListed)
	if !triageMatched {
		writeDomainError(w, triageResult.(ModerationTriageListRejected).Reason)
		return
	}
	triageByID := map[string]ModerationTriageRecord{}
	for _, record := range triageListed.Values {
		triageByID[record.ReportID.String()] = record
	}
	stateFilter := strings.TrimSpace(r.URL.Query().Get("state"))
	response := moderationReportsResponse{Reports: make([]moderationReportResponse, 0, len(listed.Values))}
	for _, event := range listed.Values {
		converted := moderationReportFromAuditEvent(event)
		report, convertedMatched := converted.(moderationReportConverted)
		if !convertedMatched {
			writeDomainError(w, converted.(moderationReportConversionRejected).reason)
			return
		}
		triage, ok := triageByID[event.ID.String()]
		if !ok {
			writeDomainError(w, core.NewDomainError(core.ErrorCodeInvalidState, "moderation report triage state is missing"))
			return
		}
		withTriage := applyModerationTriage(report.value, triage)
		if stateFilter != "" && withTriage.State != stateFilter {
			continue
		}
		response.Reports = append(response.Reports, withTriage)
	}
	writeJSON(w, http.StatusOK, response)
}

func (server Server) triageModerationReport(w http.ResponseWriter, r *http.Request) {
	actor, ok := server.requireAdminSubject(w, r)
	if !ok {
		return
	}
	reportIDResult := core.ParseAuditEventID(r.PathValue("report_id"))
	reportID, reportIDMatched := reportIDResult.(core.AuditEventIDCreated)
	if !reportIDMatched {
		writeDomainError(w, reportIDResult.(core.AuditEventIDRejected).Reason)
		return
	}
	var request moderationTriageRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "request body is invalid")
		return
	}
	result := server.moderationTriage.Update(r.Context(), actor.subject.ID, reportID.Value, strings.TrimSpace(request.State), request.ResolutionNote)
	saved, matched := result.(ModerationTriageSaved)
	if !matched {
		writeDomainError(w, result.(ModerationTriageMutationRejected).Reason)
		return
	}
	metadataResult := encodeJSONMetadata(map[string]string{"state": saved.Value.State, "resolution_note": saved.Value.ResolutionNote})
	metadata, metadataMatched := metadataResult.(jsonMetadataEncoded)
	if !metadataMatched {
		writeDomainError(w, metadataResult.(jsonMetadataRejected).reason)
		return
	}
	if !server.recordAudit(w, r.Context(), actor.subject.ID, audit.ActionFromString("moderation_report_triaged"), audit.Subject{Kind: "moderation_report", ID: saved.Value.ReportID.String()}, audit.Metadata{JSON: metadata.value}) {
		return
	}
	getResult := server.auditService.Get(r.Context(), saved.Value.ReportID)
	found, foundMatched := getResult.(audit.EventFound)
	if !foundMatched {
		writeDomainError(w, getResult.(audit.GetRejected).Reason)
		return
	}
	converted := moderationReportFromAuditEvent(found.Value)
	report, reportMatched := converted.(moderationReportConverted)
	if !reportMatched {
		writeDomainError(w, converted.(moderationReportConversionRejected).reason)
		return
	}
	writeJSON(w, http.StatusOK, applyModerationTriage(report.value, saved.Value))
}

func validModerationSubjectKind(value string) bool {
	switch value {
	case "task", "submission", "task_comment", "submission_comment", "task_series_comment", "user", "organization", "team", "collectible":
		return true
	default:
		return false
	}
}

func validModerationReason(value string) bool {
	switch value {
	case "spam", "abuse", "pii", "policy", "other":
		return true
	default:
		return false
	}
}

func validModerationTriageState(value string) bool {
	switch value {
	case "open", "resolved", "dismissed":
		return true
	default:
		return false
	}
}

type moderationMetadataResult interface {
	moderationMetadataResult()
}

type moderationMetadataEncoded struct {
	value string
}

type moderationMetadataRejected struct {
	reason core.DomainError
}

func (moderationMetadataEncoded) moderationMetadataResult()  {}
func (moderationMetadataRejected) moderationMetadataResult() {}

func encodeModerationMetadata(reason string, details string) moderationMetadataResult {
	encoded, err := json.Marshal(moderationMetadata{Reason: reason, Details: details})
	if err != nil {
		return moderationMetadataRejected{reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "moderation metadata is invalid")}
	}
	return moderationMetadataEncoded{value: string(encoded)}
}

type moderationReportConversionResult interface {
	moderationReportConversionResult()
}

type moderationReportConverted struct {
	value moderationReportResponse
}

type moderationReportConversionRejected struct {
	reason core.DomainError
}

func (moderationReportConverted) moderationReportConversionResult()          {}
func (moderationReportConversionRejected) moderationReportConversionResult() {}

func moderationReportFromAuditEvent(event audit.Event) moderationReportConversionResult {
	if event.Action.String() != audit.ActionModerationReportCreated.String() {
		return moderationReportConversionRejected{reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "audit event is not a moderation report")}
	}
	if !validModerationSubjectKind(event.Subject.Kind) {
		return moderationReportConversionRejected{reason: core.NewDomainError(core.ErrorCodeInvalidEnum, "moderation subject kind is invalid")}
	}
	var metadata moderationMetadata
	var rawMetadata rawModerationMetadata
	if err := json.Unmarshal([]byte(event.Metadata.JSON), &rawMetadata); err != nil {
		return moderationReportConversionRejected{reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "moderation report metadata is invalid")}
	}
	if rawMetadata.Details == nil {
		return moderationReportConversionRejected{reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "moderation report metadata is invalid")}
	}
	metadata = moderationMetadata{Reason: rawMetadata.Reason, Details: *rawMetadata.Details}
	if !validModerationReason(metadata.Reason) {
		return moderationReportConversionRejected{reason: core.NewDomainError(core.ErrorCodeInvalidEnum, "moderation reason is invalid")}
	}
	return moderationReportConverted{value: moderationReportResponse{
		ID:             event.ID.String(),
		SubjectKind:    event.Subject.Kind,
		SubjectID:      event.Subject.ID,
		SubjectHref:    moderationSubjectHref(event.Subject.Kind, event.Subject.ID),
		Reason:         metadata.Reason,
		Details:        metadata.Details,
		ReporterUserID: event.ActorUserID.String(),
		CreatedAt:      event.CreatedAt.UTC().Format(time.RFC3339Nano),
	}}
}

func applyModerationTriage(response moderationReportResponse, triage ModerationTriageRecord) moderationReportResponse {
	response.State = triage.State
	response.ResolutionNote = triage.ResolutionNote
	response.UpdatedBy = triage.UpdatedBy
	response.UpdatedAt = triage.UpdatedAt.UTC().Format(time.RFC3339Nano)
	return response
}

func moderationSubjectHref(kind string, id string) string {
	switch kind {
	case "task":
		return "#/tasks/" + id
	case "user":
		return "#/users/" + id
	case "organization":
		return "#/organizations/" + id
	case "team":
		return "#/teams/" + id
	case "collectible":
		return "#/collectibles/" + id
	default:
		return ""
	}
}
