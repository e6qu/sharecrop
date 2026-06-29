package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"
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
	writeJSON(w, http.StatusCreated, response.value)
}

func (server Server) listAdminModerationReports(w http.ResponseWriter, r *http.Request) {
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

	filters := audit.NoListFilters()
	filters.Action = audit.ActionEquals{Value: audit.ActionModerationReportCreated}
	result := server.auditService.List(r.Context(), filters, parsePage(r))
	listed, listedMatched := result.(audit.EventsListed)
	if !listedMatched {
		writeDomainError(w, result.(audit.ListRejected).Reason)
		return
	}

	response := moderationReportsResponse{Reports: make([]moderationReportResponse, 0, len(listed.Values))}
	for _, event := range listed.Values {
		converted := moderationReportFromAuditEvent(event)
		report, convertedMatched := converted.(moderationReportConverted)
		if !convertedMatched {
			writeDomainError(w, converted.(moderationReportConversionRejected).reason)
			return
		}
		response.Reports = append(response.Reports, report.value)
	}
	writeJSON(w, http.StatusOK, response)
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
		Reason:         metadata.Reason,
		Details:        metadata.Details,
		ReporterUserID: event.ActorUserID.String(),
		CreatedAt:      event.CreatedAt.UTC().Format(time.RFC3339Nano),
	}}
}
