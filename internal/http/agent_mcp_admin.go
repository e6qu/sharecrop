package httpserver

import (
	"context"
	"encoding/json"

	"github.com/e6qu/sharecrop/internal/assets"
	"github.com/e6qu/sharecrop/internal/audit"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/mcp"
)

// moderationReportFromEventAndTriage builds an mcp.ModerationReport directly
// from the raw audit event and triage record (time.Time fields kept as-is,
// unlike moderationReportFromAuditEvent/applyModerationTriage which produce
// an already-stringified HTTP response) so internal/mcp's own summary
// helpers do the JSON formatting, matching every other domain type's pattern.
func moderationReportFromEventAndTriage(event audit.Event, triage ModerationTriageRecord) (mcp.ModerationReport, bool) {
	if event.Action.String() != audit.ActionModerationReportCreated.String() {
		return mcp.ModerationReport{}, false
	}
	var rawMetadata rawModerationMetadata
	if err := json.Unmarshal([]byte(event.Metadata.JSON), &rawMetadata); err != nil || rawMetadata.Details == nil {
		return mcp.ModerationReport{}, false
	}
	return mcp.ModerationReport{
		ID:             event.ID.String(),
		SubjectKind:    event.Subject.Kind,
		SubjectID:      event.Subject.ID,
		Reason:         rawMetadata.Reason,
		Details:        *rawMetadata.Details,
		ReporterUserID: event.ActorUserID.String(),
		State:          triage.State,
		ResolutionNote: triage.ResolutionNote,
		UpdatedBy:      triage.UpdatedBy,
		CreatedAt:      event.CreatedAt,
		UpdatedAt:      triage.UpdatedAt,
	}, true
}

func (services mcpServices) CreateModerationReport(ctx context.Context, actor core.UserID, subjectKind string, subjectID string, reason string, details string) mcp.ModerationReportResult {
	if !validModerationSubjectKind(subjectKind) {
		return mcp.ModerationReportRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidEnum, "moderation subject kind is invalid")}
	}
	if subjectID == "" {
		return mcp.ModerationReportRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "moderation subject id is required")}
	}
	if !validModerationReason(reason) {
		return mcp.ModerationReportRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidEnum, "moderation reason is invalid")}
	}
	if len(details) > moderationDetailsMaxLength {
		return mcp.ModerationReportRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "moderation details are too long")}
	}
	metadataResult := encodeModerationMetadata(reason, details)
	metadata, metadataMatched := metadataResult.(moderationMetadataEncoded)
	if !metadataMatched {
		return mcp.ModerationReportRejected{Reason: metadataResult.(moderationMetadataRejected).reason}
	}
	result := services.auditService.Record(ctx, actor, audit.ActionModerationReportCreated, audit.Subject{Kind: subjectKind, ID: subjectID}, audit.Metadata{JSON: metadata.value})
	recorded, recordedMatched := result.(audit.EventRecorded)
	if !recordedMatched {
		return mcp.ModerationReportRejected{Reason: result.(audit.RecordRejected).Reason}
	}
	triageResult := services.moderationTriage.RecordOpen(ctx, recorded.Value)
	triage, triageMatched := triageResult.(ModerationTriageSaved)
	if !triageMatched {
		return mcp.ModerationReportRejected{Reason: triageResult.(ModerationTriageMutationRejected).Reason}
	}
	report, ok := moderationReportFromEventAndTriage(recorded.Value, triage.Value)
	if !ok {
		return mcp.ModerationReportRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "moderation report could not be converted")}
	}
	return mcp.ModerationReportSaved{Value: report}
}

func (services mcpServices) ListAdminModerationReports(ctx context.Context, stateFilter string, page core.Page) mcp.ModerationReportsListResult {
	filters := audit.NoListFilters()
	filters.Action = audit.ActionEquals{Value: audit.ActionModerationReportCreated}
	result := services.auditService.List(ctx, filters, page)
	listed, listedMatched := result.(audit.EventsListed)
	if !listedMatched {
		return mcp.ModerationReportsListRejected{Reason: result.(audit.ListRejected).Reason}
	}
	reportIDs := make([]core.AuditEventID, 0, len(listed.Values))
	for _, event := range listed.Values {
		reportIDs = append(reportIDs, event.ID)
	}
	triageResult := services.moderationTriage.List(ctx, reportIDs)
	triageListed, triageMatched := triageResult.(ModerationTriageListed)
	if !triageMatched {
		return mcp.ModerationReportsListRejected{Reason: triageResult.(ModerationTriageListRejected).Reason}
	}
	triageByID := map[string]ModerationTriageRecord{}
	for _, record := range triageListed.Values {
		triageByID[record.ReportID.String()] = record
	}
	reports := make([]mcp.ModerationReport, 0, len(listed.Values))
	for _, event := range listed.Values {
		triage, ok := triageByID[event.ID.String()]
		if !ok {
			return mcp.ModerationReportsListRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "moderation report triage state is missing")}
		}
		report, ok := moderationReportFromEventAndTriage(event, triage)
		if !ok {
			return mcp.ModerationReportsListRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "moderation report could not be converted")}
		}
		if stateFilter != "" && report.State != stateFilter {
			continue
		}
		reports = append(reports, report)
	}
	return mcp.ModerationReportsListed{Values: reports}
}

func (services mcpServices) TriageModerationReport(ctx context.Context, actor core.UserID, reportID core.AuditEventID, state string, note string) mcp.ModerationReportResult {
	result := services.moderationTriage.Update(ctx, actor, reportID, state, note)
	saved, matched := result.(ModerationTriageSaved)
	if !matched {
		return mcp.ModerationReportRejected{Reason: result.(ModerationTriageMutationRejected).Reason}
	}
	getResult := services.auditService.Get(ctx, saved.Value.ReportID)
	found, foundMatched := getResult.(audit.EventFound)
	if !foundMatched {
		return mcp.ModerationReportRejected{Reason: getResult.(audit.GetRejected).Reason}
	}
	report, ok := moderationReportFromEventAndTriage(found.Value, saved.Value)
	if !ok {
		return mcp.ModerationReportRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "moderation report could not be converted")}
	}
	return mcp.ModerationReportSaved{Value: report}
}

func (services mcpServices) IsPlatformAdmin(ctx context.Context, userID core.UserID) bool {
	_, allowed := services.platformAdmins.IsAdmin(ctx, userID).(PlatformAdminAllowed)
	return allowed
}

func (services mcpServices) ListPlatformAdmins(ctx context.Context, page core.Page) mcp.PlatformAdminListResult {
	result := services.platformAdmins.List(ctx, page)
	listed, matched := result.(PlatformAdminsListed)
	if !matched {
		return mcp.PlatformAdminListRejected{Reason: result.(PlatformAdminListRejected).Reason}
	}
	records := make([]mcp.PlatformAdminRecord, 0, len(listed.Values))
	for _, record := range listed.Values {
		records = append(records, platformAdminRecordToMCP(record))
	}
	return mcp.PlatformAdminsListed{Values: records}
}

func (services mcpServices) GrantPlatformAdmin(ctx context.Context, userID core.UserID, granter core.UserID) mcp.PlatformAdminMutationResult {
	result := services.platformAdmins.Grant(ctx, userID, granter)
	saved, matched := result.(PlatformAdminSaved)
	if !matched {
		return mcp.PlatformAdminMutationRejected{Reason: result.(PlatformAdminMutationRejected).Reason}
	}
	return mcp.PlatformAdminSaved{Value: platformAdminRecordToMCP(saved.Value)}
}

func (services mcpServices) RevokePlatformAdmin(ctx context.Context, userID core.UserID) mcp.PlatformAdminMutationResult {
	result := services.platformAdmins.Revoke(ctx, userID)
	saved, matched := result.(PlatformAdminSaved)
	if !matched {
		return mcp.PlatformAdminMutationRejected{Reason: result.(PlatformAdminMutationRejected).Reason}
	}
	return mcp.PlatformAdminSaved{Value: platformAdminRecordToMCP(saved.Value)}
}

func platformAdminRecordToMCP(record PlatformAdminRecord) mcp.PlatformAdminRecord {
	return mcp.PlatformAdminRecord{UserID: record.UserID, Source: record.Source, CreatedAt: record.CreatedAt}
}

func (services mcpServices) CreatePrivacyRequest(ctx context.Context, actor core.UserID, kind string) mcp.PrivacyRequestResult {
	if !validPrivacyRequestKind(kind) {
		return mcp.PrivacyRequestRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidEnum, "privacy request kind is invalid")}
	}
	result := services.privacyService.Create(ctx, actor, kind)
	saved, matched := result.(PrivacyRequestSaved)
	if !matched {
		return mcp.PrivacyRequestRejected{Reason: result.(PrivacyRequestMutationRejected).Reason}
	}
	return mcp.PrivacyRequestSaved{Value: privacyRequestRecordToMCP(saved.Value)}
}

func (services mcpServices) ListPrivacyRequests(ctx context.Context, actor core.UserID, page core.Page) mcp.PrivacyRequestsListResult {
	return privacyListResultToMCP(services.privacyService.ListForRequester(ctx, actor, page))
}

func (services mcpServices) ListAdminPrivacyRequests(ctx context.Context, page core.Page) mcp.PrivacyRequestsListResult {
	return privacyListResultToMCP(services.privacyService.ListAll(ctx, page))
}

func privacyListResultToMCP(result PrivacyListResult) mcp.PrivacyRequestsListResult {
	listed, matched := result.(PrivacyRequestsListed)
	if !matched {
		return mcp.PrivacyRequestsListRejected{Reason: result.(PrivacyRequestListRejected).Reason}
	}
	records := make([]mcp.PrivacyRequestRecord, 0, len(listed.Values))
	for index := range listed.Values {
		records = append(records, privacyRequestRecordToMCP(listed.Values[index]))
	}
	return mcp.PrivacyRequestsListed{Values: records}
}

func privacyRequestRecordToMCP(record PrivacyRequestRecord) mcp.PrivacyRequestRecord {
	return mcp.PrivacyRequestRecord{
		ID:                 record.ID,
		RequestedBy:        record.RequestedBy,
		Kind:               record.Kind,
		State:              record.State,
		ExportJSON:         record.ExportJSON,
		ResolutionNote:     record.ResolutionNote,
		CreatedAt:          record.CreatedAt,
		ResolvedAt:         record.ResolvedAt,
		RedactedFieldCount: record.RedactedFieldCount,
	}
}

func (services mcpServices) ResolveAdminPrivacyRequest(ctx context.Context, requestID string, note string) mcp.PrivacyRequestResult {
	result := services.privacyService.Resolve(ctx, requestID, note)
	saved, matched := result.(PrivacyRequestSaved)
	if !matched {
		return mcp.PrivacyRequestRejected{Reason: result.(PrivacyRequestMutationRejected).Reason}
	}
	return mcp.PrivacyRequestSaved{Value: privacyRequestRecordToMCP(saved.Value)}
}

func (services mcpServices) RunPrivacyRetention(ctx context.Context, actor core.UserID) mcp.PrivacyRetentionResult {
	result := services.privacyService.RunRetention(ctx, actor)
	run, matched := result.(PrivacyRetentionRun)
	if !matched {
		return mcp.PrivacyRetentionRejected{Reason: result.(PrivacyRetentionRejected).Reason}
	}
	return mcp.PrivacyRetentionRun{RedactedFieldCount: run.RedactedFieldCount}
}

func (services mcpServices) ListAuditEvents(ctx context.Context, filters audit.ListFilters, page core.Page) audit.ListResult {
	return services.auditService.List(ctx, filters, page)
}

func (services mcpServices) AwardCollectible(ctx context.Context, slug string, recipientKind string, recipientID string, organizationID string) assets.MintResult {
	entry, found := assets.CatalogBySlug(slug)
	if !found {
		return assets.MintRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "unknown default collectible")}
	}
	nameResult := assets.NewCollectibleName(entry.Name)
	name, nameMatched := nameResult.(assets.CollectibleNameAccepted)
	if !nameMatched {
		return assets.MintRejected{Reason: nameResult.(assets.CollectibleNameRejected).Reason}
	}
	return services.assetService.Mint(ctx, recipientKind, recipientID, organizationID, name.Value, entry.Kind, entry.Policy, entry.Art)
}
