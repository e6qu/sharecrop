package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/e6qu/sharecrop/internal/audit"
	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/org"
)

type platformAdminSummary struct {
	UserID    string `json:"user_id"`
	Source    string `json:"source"`
	CreatedAt string `json:"created_at"`
}

type platformAdminsPayload struct {
	Admins []platformAdminSummary `json:"admins"`
}

type moderationReportSummary struct {
	ID             string `json:"id"`
	SubjectKind    string `json:"subject_kind"`
	SubjectID      string `json:"subject_id"`
	Reason         string `json:"reason"`
	Details        string `json:"details"`
	ReporterUserID string `json:"reporter_user_id"`
	State          string `json:"state"`
	ResolutionNote string `json:"resolution_note"`
	UpdatedBy      string `json:"updated_by"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
}

type moderationReportsPayload struct {
	Reports []moderationReportSummary `json:"reports"`
}

type privacyRequestSummary struct {
	ID                 string `json:"id"`
	RequestedBy        string `json:"requested_by"`
	Kind               string `json:"kind"`
	State              string `json:"state"`
	ExportJSON         string `json:"export_json"`
	ResolutionNote     string `json:"resolution_note"`
	CreatedAt          string `json:"created_at"`
	ResolvedAt         string `json:"resolved_at"`
	RedactedFieldCount int    `json:"redacted_field_count"`
}

type privacyRequestsPayload struct {
	Requests []privacyRequestSummary `json:"requests"`
}

type privacyRetentionPayload struct {
	RedactedFieldCount int `json:"redacted_field_count"`
}

type auditEventSummary struct {
	ID           string `json:"id"`
	ActorUserID  string `json:"actor_user_id"`
	Action       string `json:"action"`
	SubjectKind  string `json:"subject_kind"`
	SubjectID    string `json:"subject_id"`
	MetadataJSON string `json:"metadata_json"`
	CreatedAt    string `json:"created_at"`
}

type auditEventsPayload struct {
	Events []auditEventSummary `json:"events"`
}

func (platformAdminSummary) payloadValue()     {}
func (platformAdminsPayload) payloadValue()    {}
func (moderationReportSummary) payloadValue()  {}
func (moderationReportsPayload) payloadValue() {}
func (privacyRequestSummary) payloadValue()    {}
func (privacyRequestsPayload) payloadValue()   {}
func (privacyRetentionPayload) payloadValue()  {}
func (auditEventSummary) payloadValue()        {}
func (auditEventsPayload) payloadValue()       {}

func platformAdminToSummary(value PlatformAdminRecord) platformAdminSummary {
	return platformAdminSummary{UserID: value.UserID.String(), Source: value.Source, CreatedAt: value.CreatedAt.UTC().Format(time.RFC3339Nano)}
}

func moderationReportToSummary(value ModerationReport) moderationReportSummary {
	return moderationReportSummary{
		ID:             value.ID,
		SubjectKind:    value.SubjectKind,
		SubjectID:      value.SubjectID,
		Reason:         value.Reason,
		Details:        value.Details,
		ReporterUserID: value.ReporterUserID,
		State:          value.State,
		ResolutionNote: value.ResolutionNote,
		UpdatedBy:      value.UpdatedBy,
		CreatedAt:      value.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt:      value.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
}

func privacyRequestToSummary(value PrivacyRequestRecord) privacyRequestSummary {
	var resolvedAt string
	if !value.ResolvedAt.IsZero() {
		resolvedAt = value.ResolvedAt.UTC().Format(time.RFC3339Nano)
	}
	return privacyRequestSummary{
		ID:                 value.ID,
		RequestedBy:        value.RequestedBy.String(),
		Kind:               value.Kind,
		State:              value.State,
		ExportJSON:         value.ExportJSON,
		ResolutionNote:     value.ResolutionNote,
		CreatedAt:          value.CreatedAt.UTC().Format(time.RFC3339Nano),
		ResolvedAt:         resolvedAt,
		RedactedFieldCount: value.RedactedFieldCount,
	}
}

func auditEventToSummary(value audit.Event) auditEventSummary {
	return auditEventSummary{
		ID:           value.ID.String(),
		ActorUserID:  value.ActorUserID.String(),
		Action:       value.Action.String(),
		SubjectKind:  value.Subject.Kind,
		SubjectID:    value.Subject.ID,
		MetadataJSON: value.Metadata.JSON,
		CreatedAt:    value.CreatedAt.UTC().Format(time.RFC3339Nano),
	}
}

func (server Server) callListPlatformAdmins(ctx context.Context, subject auth.UserSubject, arguments json.RawMessage) toolResult {
	result := server.services.ListPlatformAdmins(ctx, core.DefaultPage())
	listed, matched := result.(PlatformAdminsListed)
	if !matched {
		return toolFailed{message: result.(PlatformAdminListRejected).Reason.Description()}
	}
	summaries := make([]platformAdminSummary, 0, len(listed.Values))
	for index := range listed.Values {
		summaries = append(summaries, platformAdminToSummary(listed.Values[index]))
	}
	return marshalPayload(platformAdminsPayload{Admins: summaries})
}

func (server Server) callGrantPlatformAdmin(ctx context.Context, subject auth.UserSubject, arguments json.RawMessage) toolResult {
	userID, problem := parseUserID(arguments)
	if problem != nil {
		return problem
	}
	result := server.services.GrantPlatformAdmin(ctx, userID, subject.ID)
	saved, matched := result.(PlatformAdminSaved)
	if !matched {
		return toolFailed{message: result.(PlatformAdminMutationRejected).Reason.Description()}
	}
	return marshalPayload(platformAdminToSummary(saved.Value))
}

func (server Server) callRevokePlatformAdmin(ctx context.Context, subject auth.UserSubject, arguments json.RawMessage) toolResult {
	userID, problem := parseUserID(arguments)
	if problem != nil {
		return problem
	}
	result := server.services.RevokePlatformAdmin(ctx, userID)
	saved, matched := result.(PlatformAdminSaved)
	if !matched {
		return toolFailed{message: result.(PlatformAdminMutationRejected).Reason.Description()}
	}
	return marshalPayload(platformAdminToSummary(saved.Value))
}

func (server Server) callCreateModerationReport(ctx context.Context, subject auth.UserSubject, arguments json.RawMessage) toolResult {
	var args struct {
		SubjectKind string `json:"subject_kind"`
		SubjectID   string `json:"subject_id"`
		Reason      string `json:"reason"`
		Details     string `json:"details"`
	}
	if err := json.Unmarshal(arguments, &args); err != nil {
		return invalidArguments()
	}
	result := server.services.CreateModerationReport(ctx, subject.ID, strings.TrimSpace(args.SubjectKind), strings.TrimSpace(args.SubjectID), strings.TrimSpace(args.Reason), strings.TrimSpace(args.Details))
	saved, matched := result.(ModerationReportSaved)
	if !matched {
		return toolFailed{message: result.(ModerationReportRejected).Reason.Description()}
	}
	return marshalPayload(moderationReportToSummary(saved.Value))
}

func (server Server) callListAdminModerationReports(ctx context.Context, subject auth.UserSubject, arguments json.RawMessage) toolResult {
	var args struct {
		State string `json:"state"`
	}
	if err := json.Unmarshal(arguments, &args); err != nil {
		return invalidArguments()
	}
	result := server.services.ListAdminModerationReports(ctx, strings.TrimSpace(args.State), core.DefaultPage())
	listed, matched := result.(ModerationReportsListed)
	if !matched {
		return toolFailed{message: result.(ModerationReportsListRejected).Reason.Description()}
	}
	summaries := make([]moderationReportSummary, 0, len(listed.Values))
	for index := range listed.Values {
		summaries = append(summaries, moderationReportToSummary(listed.Values[index]))
	}
	return marshalPayload(moderationReportsPayload{Reports: summaries})
}

func (server Server) callTriageModerationReport(ctx context.Context, subject auth.UserSubject, arguments json.RawMessage) toolResult {
	var args struct {
		ReportID       string `json:"report_id"`
		State          string `json:"state"`
		ResolutionNote string `json:"resolution_note"`
	}
	if err := json.Unmarshal(arguments, &args); err != nil {
		return invalidArguments()
	}
	reportIDResult := core.ParseAuditEventID(args.ReportID)
	reportID, reportIDMatched := reportIDResult.(core.AuditEventIDCreated)
	if !reportIDMatched {
		return toolProtocolError{code: codeInvalidParams, message: reportIDResult.(core.AuditEventIDRejected).Reason.Description()}
	}
	result := server.services.TriageModerationReport(ctx, subject.ID, reportID.Value, strings.TrimSpace(args.State), args.ResolutionNote)
	saved, matched := result.(ModerationReportSaved)
	if !matched {
		return toolFailed{message: result.(ModerationReportRejected).Reason.Description()}
	}
	return marshalPayload(moderationReportToSummary(saved.Value))
}

func (server Server) callCreatePrivacyRequest(ctx context.Context, subject auth.UserSubject, arguments json.RawMessage) toolResult {
	var args struct {
		Kind string `json:"kind"`
	}
	if err := json.Unmarshal(arguments, &args); err != nil {
		return invalidArguments()
	}
	result := server.services.CreatePrivacyRequest(ctx, subject.ID, strings.TrimSpace(args.Kind))
	saved, matched := result.(PrivacyRequestSaved)
	if !matched {
		return toolFailed{message: result.(PrivacyRequestRejected).Reason.Description()}
	}
	return marshalPayload(privacyRequestToSummary(saved.Value))
}

func (server Server) callListPrivacyRequests(ctx context.Context, subject auth.UserSubject, arguments json.RawMessage) toolResult {
	result := server.services.ListPrivacyRequests(ctx, subject.ID, core.DefaultPage())
	return privacyRequestsListResult(result)
}

func (server Server) callListAdminPrivacyRequests(ctx context.Context, subject auth.UserSubject, arguments json.RawMessage) toolResult {
	result := server.services.ListAdminPrivacyRequests(ctx, core.DefaultPage())
	return privacyRequestsListResult(result)
}

func privacyRequestsListResult(result PrivacyRequestsListResult) toolResult {
	listed, matched := result.(PrivacyRequestsListed)
	if !matched {
		return toolFailed{message: result.(PrivacyRequestsListRejected).Reason.Description()}
	}
	summaries := make([]privacyRequestSummary, 0, len(listed.Values))
	for index := range listed.Values {
		summaries = append(summaries, privacyRequestToSummary(listed.Values[index]))
	}
	return marshalPayload(privacyRequestsPayload{Requests: summaries})
}

func (server Server) callResolveAdminPrivacyRequest(ctx context.Context, subject auth.UserSubject, arguments json.RawMessage) toolResult {
	var args struct {
		PrivacyRequestID string `json:"privacy_request_id"`
		ResolutionNote   string `json:"resolution_note"`
	}
	if err := json.Unmarshal(arguments, &args); err != nil {
		return invalidArguments()
	}
	result := server.services.ResolveAdminPrivacyRequest(ctx, args.PrivacyRequestID, args.ResolutionNote)
	saved, matched := result.(PrivacyRequestSaved)
	if !matched {
		return toolFailed{message: result.(PrivacyRequestRejected).Reason.Description()}
	}
	return marshalPayload(privacyRequestToSummary(saved.Value))
}

func (server Server) callRunPrivacyRetention(ctx context.Context, subject auth.UserSubject, arguments json.RawMessage) toolResult {
	result := server.services.RunPrivacyRetention(ctx, subject.ID)
	run, matched := result.(PrivacyRetentionRun)
	if !matched {
		return toolFailed{message: result.(PrivacyRetentionRejected).Reason.Description()}
	}
	return marshalPayload(privacyRetentionPayload{RedactedFieldCount: run.RedactedFieldCount})
}

func (server Server) callListOrganizationAuditEvents(ctx context.Context, subject auth.UserSubject, arguments json.RawMessage) toolResult {
	organizationID, problem := parseOrganizationID(arguments)
	if problem != nil {
		return problem
	}
	permissionCheck := server.services.CheckOrganizationPermission(ctx, organizationID, subject.ID, org.PermissionManageMembers)
	if _, denied := permissionCheck.(org.PermissionDenied); denied {
		return toolFailed{message: "organization audit access denied"}
	}
	filters := audit.NoListFilters()
	filters.SubjectID = audit.SubjectIDEquals{Value: organizationID.String()}
	result := server.services.ListAuditEvents(ctx, filters, core.DefaultPage())
	return auditEventsListResult(result)
}

func (server Server) callListAdminAuditEvents(ctx context.Context, subject auth.UserSubject, arguments json.RawMessage) toolResult {
	var args struct {
		Action      string `json:"action"`
		SubjectKind string `json:"subject_kind"`
		SubjectID   string `json:"subject_id"`
	}
	if err := json.Unmarshal(arguments, &args); err != nil {
		return invalidArguments()
	}
	filters := audit.NoListFilters()
	if strings.TrimSpace(args.Action) != "" {
		filters.Action = audit.ActionEquals{Value: audit.ActionFromString(strings.TrimSpace(args.Action))}
	}
	if strings.TrimSpace(args.SubjectKind) != "" {
		filters.SubjectKind = audit.SubjectKindEquals{Value: strings.TrimSpace(args.SubjectKind)}
	}
	if strings.TrimSpace(args.SubjectID) != "" {
		filters.SubjectID = audit.SubjectIDEquals{Value: strings.TrimSpace(args.SubjectID)}
	}
	result := server.services.ListAuditEvents(ctx, filters, core.DefaultPage())
	return auditEventsListResult(result)
}

func auditEventsListResult(result audit.ListResult) toolResult {
	listed, matched := result.(audit.EventsListed)
	if !matched {
		return toolFailed{message: result.(audit.ListRejected).Reason.Description()}
	}
	summaries := make([]auditEventSummary, 0, len(listed.Values))
	for index := range listed.Values {
		summaries = append(summaries, auditEventToSummary(listed.Values[index]))
	}
	return marshalPayload(auditEventsPayload{Events: summaries})
}
