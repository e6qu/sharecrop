package httpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"
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

// ModerationTriageState is the sealed triage lifecycle of a moderation report,
// matching the moderation_report_triage state check constraint
// (migrations/000030_admin_moderation_retention.sql).
type ModerationTriageState struct {
	value string
}

var (
	ModerationTriageStateOpen      = ModerationTriageState{value: "open"}
	ModerationTriageStateResolved  = ModerationTriageState{value: "resolved"}
	ModerationTriageStateDismissed = ModerationTriageState{value: "dismissed"}
)

func (state ModerationTriageState) String() string {
	return state.value
}

type ModerationTriageStateResult interface {
	moderationTriageStateResult()
}

type ModerationTriageStateAccepted struct {
	Value ModerationTriageState
}

type ModerationTriageStateRejected struct {
	Reason core.DomainError
}

func (ModerationTriageStateAccepted) moderationTriageStateResult() {}

func (ModerationTriageStateRejected) moderationTriageStateResult() {}

func ParseModerationTriageState(raw string) ModerationTriageStateResult {
	switch raw {
	case ModerationTriageStateOpen.value:
		return ModerationTriageStateAccepted{Value: ModerationTriageStateOpen}
	case ModerationTriageStateResolved.value:
		return ModerationTriageStateAccepted{Value: ModerationTriageStateResolved}
	case ModerationTriageStateDismissed.value:
		return ModerationTriageStateAccepted{Value: ModerationTriageStateDismissed}
	default:
		return ModerationTriageStateRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidEnum, "moderation triage state is invalid")}
	}
}

// ModerationTriageStateFilter is the optional triage-state restriction on a
// moderation report listing: AnyTriageState lists every report, while
// TriageStateEquals restricts the listing to one triage state (an untriaged
// report counts as open). The filter is applied in the store query, before
// pagination.
type ModerationTriageStateFilter interface {
	moderationTriageStateFilter()
}

type AnyTriageState struct{}

type TriageStateEquals struct {
	State ModerationTriageState
}

func (AnyTriageState) moderationTriageStateFilter() {}

func (TriageStateEquals) moderationTriageStateFilter() {}

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
	record := ModerationTriageRecord{ReportID: event.ID, State: ModerationTriageStateOpen.String(), ResolutionNote: "", CreatedAt: event.CreatedAt, UpdatedAt: event.CreatedAt}
	service.mu.Lock()
	defer service.mu.Unlock()
	service.records[event.ID.String()] = record
	return ModerationTriageSaved{Value: record}
}

// List returns the newest-first page of triage records matching the state
// filter. Every report recorded through RecordOpen has a triage record, so
// this in-memory listing is complete for the memory runtime.
func (service *memoryModerationTriageService) List(_ context.Context, filter ModerationTriageStateFilter, page core.Page) ModerationTriageListResult {
	service.mu.Lock()
	defer service.mu.Unlock()
	values := make([]ModerationTriageRecord, 0, len(service.records))
	for _, record := range service.records {
		if equals, restricted := filter.(TriageStateEquals); restricted && record.State != equals.State.String() {
			continue
		}
		values = append(values, record)
	}
	sort.Slice(values, func(left int, right int) bool {
		if !values[left].CreatedAt.Equal(values[right].CreatedAt) {
			return values[left].CreatedAt.After(values[right].CreatedAt)
		}
		return values[left].ReportID.String() > values[right].ReportID.String()
	})
	start := page.Offset()
	if start > len(values) {
		start = len(values)
	}
	end := start + page.Limit()
	if end > len(values) {
		end = len(values)
	}
	return ModerationTriageListed{Values: values[start:end]}
}

func (service *memoryModerationTriageService) Update(_ context.Context, actor core.UserID, reportID core.AuditEventID, state ModerationTriageState, note string) ModerationTriageMutationResult {
	service.mu.Lock()
	defer service.mu.Unlock()
	record, ok := service.records[reportID.String()]
	if !ok {
		return ModerationTriageMutationRejected{Reason: core.NewDomainError(core.ErrorCodeNotFound, "moderation report was not found")}
	}
	record.State = state.String()
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
		writeError(w, http.StatusUnauthorized, core.ErrorCodeUnauthenticated, actorResult.(userSubjectRejected).reason)
		return
	}

	if !server.allowBySubject(w, actor.subject.ID.String()) {
		return
	}

	var request moderationReportRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, core.ErrorCodeInvalidArgument, "request body is invalid")
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
		writeError(w, http.StatusInternalServerError, core.ErrorCodeUnavailable, "moderation report was not recorded")
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

	page, pageOK := parsePageOrReject(w, r)
	if !pageOK {
		return
	}
	filter := ModerationTriageStateFilter(AnyTriageState{})
	if rawState := strings.TrimSpace(r.URL.Query().Get("state")); rawState != "" {
		stateResult := ParseModerationTriageState(rawState)
		state, stateMatched := stateResult.(ModerationTriageStateAccepted)
		if !stateMatched {
			writeDomainError(w, stateResult.(ModerationTriageStateRejected).Reason)
			return
		}
		filter = TriageStateEquals{State: state.Value}
	}

	// The triage service applies the state filter and the pagination in one
	// query over all moderation reports (untriaged reports count as open), so
	// a page is full whenever enough matching reports exist - it is never
	// shortened by post-filtering.
	triageResult := server.moderationTriage.List(r.Context(), filter, page.Probe())
	triageListed, triageMatched := triageResult.(ModerationTriageListed)
	if !triageMatched {
		writeDomainError(w, triageResult.(ModerationTriageListRejected).Reason)
		return
	}
	visible, nextOffset := probeListWindow(len(triageListed.Values), page)
	response := moderationReportsResponse{Reports: make([]moderationReportResponse, 0, visible), NextOffset: nextOffset}
	for _, triage := range triageListed.Values[:visible] {
		eventResult := server.auditService.Get(r.Context(), triage.ReportID)
		found, foundMatched := eventResult.(audit.EventFound)
		if !foundMatched {
			writeDomainError(w, eventResult.(audit.GetRejected).Reason)
			return
		}
		converted := moderationReportFromAuditEvent(found.Value)
		report, convertedMatched := converted.(moderationReportConverted)
		if !convertedMatched {
			writeDomainError(w, converted.(moderationReportConversionRejected).reason)
			return
		}
		response.Reports = append(response.Reports, applyModerationTriage(report.value, triage))
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
		writeError(w, http.StatusBadRequest, core.ErrorCodeInvalidArgument, "request body is invalid")
		return
	}
	stateResult := ParseModerationTriageState(strings.TrimSpace(request.State))
	state, stateMatched := stateResult.(ModerationTriageStateAccepted)
	if !stateMatched {
		writeDomainError(w, stateResult.(ModerationTriageStateRejected).Reason)
		return
	}
	result := server.moderationTriage.Update(r.Context(), actor.subject.ID, reportID.Value, state.Value, request.ResolutionNote)
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
	server.recordAuditBestEffort(r.Context(), actor.subject.ID, audit.ActionModerationReportTriaged, audit.Subject{Kind: "moderation_report", ID: saved.Value.ReportID.String()}, audit.Metadata{JSON: metadata.value})
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
