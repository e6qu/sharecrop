package db

import (
	"context"
	"strings"
	"time"

	"github.com/e6qu/sharecrop/internal/audit"
	"github.com/e6qu/sharecrop/internal/core"
	httpserver "github.com/e6qu/sharecrop/internal/http"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ModerationTriageStore struct {
	db Querier
}

func NewModerationTriageStore(pool *pgxpool.Pool) ModerationTriageStore {
	return NewModerationTriageStoreFromHandle(NewPGX(pool))
}

func NewModerationTriageStoreFromHandle(handle Beginner) ModerationTriageStore {
	return ModerationTriageStore{db: handle}
}

func (store ModerationTriageStore) RecordOpen(ctx context.Context, event audit.Event) httpserver.ModerationTriageMutationResult {
	_, err := store.db.Exec(ctx, `
		insert into moderation_report_triage (report_audit_event_id, state, resolution_note, created_at, updated_at)
		values ($1, 'open', '', $2, $2)
	`, event.ID.String(), event.CreatedAt)
	if err != nil {
		return httpserver.ModerationTriageMutationRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "record moderation triage state failed")}
	}
	return httpserver.ModerationTriageSaved{Value: httpserver.ModerationTriageRecord{ReportID: event.ID, State: "open", CreatedAt: event.CreatedAt, UpdatedAt: event.CreatedAt}}
}

// List pages moderation reports newest-first with the triage state filter
// applied in SQL, before pagination: every moderation_report_created audit
// event joins its triage row, and a report without one counts as open. A page
// is therefore full whenever enough matching reports exist.
func (store ModerationTriageStore) List(ctx context.Context, filter httpserver.ModerationTriageStateFilter, page core.Page) httpserver.ModerationTriageListResult {
	query := `
		select audit_events.id::text,
			coalesce(moderation_report_triage.state, 'open'),
			coalesce(moderation_report_triage.resolution_note, ''),
			coalesce(moderation_report_triage.updated_by_user_id::text, ''),
			coalesce(moderation_report_triage.created_at, audit_events.created_at),
			coalesce(moderation_report_triage.updated_at, audit_events.created_at)
		from audit_events
		left join moderation_report_triage
			on moderation_report_triage.report_audit_event_id = audit_events.id
		where audit_events.action = @action
	`
	arguments := NamedArgs{
		"action": audit.ActionModerationReportCreated.String(),
		"limit":  page.Limit(),
		"offset": page.Offset(),
	}
	switch typed := filter.(type) {
	case httpserver.AnyTriageState:
	case httpserver.TriageStateEquals:
		arguments["state"] = typed.State.String()
		query += " and coalesce(moderation_report_triage.state, 'open') = @state"
	default:
		return httpserver.ModerationTriageListRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "moderation triage state filter is invalid")}
	}
	query += " order by audit_events.created_at desc, audit_events.id desc limit @limit offset @offset"

	rows, err := store.db.Query(ctx, query, arguments)
	if err != nil {
		return httpserver.ModerationTriageListRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "list moderation report triage failed")}
	}
	defer rows.Close()

	values := make([]httpserver.ModerationTriageRecord, 0)
	for rows.Next() {
		var rawID string
		var state string
		var note string
		var updatedBy string
		var createdAt time.Time
		var updatedAt time.Time
		if err := rows.Scan(&rawID, &state, &note, &updatedBy, &createdAt, &updatedAt); err != nil {
			return httpserver.ModerationTriageListRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "scan moderation report triage failed")}
		}
		scanned := parseModerationTriage(rawID, state, note, updatedBy, createdAt, updatedAt)
		accepted, matched := scanned.(moderationTriageAccepted)
		if !matched {
			return httpserver.ModerationTriageListRejected{Reason: scanned.(moderationTriageRejected).reason}
		}
		values = append(values, accepted.value)
	}
	if err := rows.Err(); err != nil {
		return httpserver.ModerationTriageListRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "read moderation report triage failed")}
	}
	return httpserver.ModerationTriageListed{Values: values}
}

func (store ModerationTriageStore) Update(ctx context.Context, actor core.UserID, reportID core.AuditEventID, state httpserver.ModerationTriageState, note string) httpserver.ModerationTriageMutationResult {
	var rawID string
	var storedState string
	var storedNote string
	var updatedBy string
	var createdAt time.Time
	var updatedAt time.Time
	if err := store.db.QueryRow(ctx, `
		update moderation_report_triage
		set state = $2, resolution_note = $3, updated_by_user_id = $4, updated_at = now()
		where report_audit_event_id = $1
		returning report_audit_event_id::text, state, resolution_note, coalesce(updated_by_user_id::text, ''), created_at, updated_at
	`, reportID.String(), state.String(), strings.TrimSpace(note), actor.String()).Scan(&rawID, &storedState, &storedNote, &updatedBy, &createdAt, &updatedAt); err != nil {
		if err == ErrNoRows {
			return httpserver.ModerationTriageMutationRejected{Reason: core.NewDomainError(core.ErrorCodeNotFound, "moderation report was not found")}
		}
		return httpserver.ModerationTriageMutationRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "update moderation report triage failed")}
	}
	scanned := parseModerationTriage(rawID, storedState, storedNote, updatedBy, createdAt, updatedAt)
	accepted, matched := scanned.(moderationTriageAccepted)
	if !matched {
		return httpserver.ModerationTriageMutationRejected{Reason: scanned.(moderationTriageRejected).reason}
	}
	return httpserver.ModerationTriageSaved{Value: accepted.value}
}

type moderationTriageResult interface{ moderationTriageResult() }
type moderationTriageAccepted struct {
	value httpserver.ModerationTriageRecord
}
type moderationTriageRejected struct{ reason core.DomainError }

func (moderationTriageAccepted) moderationTriageResult() {}
func (moderationTriageRejected) moderationTriageResult() {}

func parseModerationTriage(rawID string, state string, note string, updatedBy string, createdAt time.Time, updatedAt time.Time) moderationTriageResult {
	idResult := core.ParseAuditEventID(rawID)
	id, matched := idResult.(core.AuditEventIDCreated)
	if !matched {
		return moderationTriageRejected{reason: idResult.(core.AuditEventIDRejected).Reason}
	}
	return moderationTriageAccepted{value: httpserver.ModerationTriageRecord{ReportID: id.Value, State: state, ResolutionNote: note, UpdatedBy: updatedBy, CreatedAt: createdAt, UpdatedAt: updatedAt}}
}
