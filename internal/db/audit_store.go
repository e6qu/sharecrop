package db

import (
	"context"
	"time"

	"github.com/e6qu/sharecrop/internal/audit"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AuditStore struct {
	db Querier
}

func NewAuditStore(pool *pgxpool.Pool) AuditStore {
	return AuditStore{db: NewPGX(pool)}
}

func (store AuditStore) Record(ctx context.Context, event audit.Event) audit.RecordResult {
	_, err := store.db.Exec(ctx, `
		insert into audit_events (id, actor_user_id, action, subject_kind, subject_id, metadata_json, created_at)
		values ($1, $2, $3, $4, $5, $6::jsonb, $7)
	`, event.ID.String(), event.ActorUserID.String(), event.Action.String(), event.Subject.Kind, event.Subject.ID, event.Metadata.JSON, event.CreatedAt)
	if err != nil {
		return audit.RecordRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "record audit event failed")}
	}
	return audit.EventRecorded{Value: event}
}

func (store AuditStore) Get(ctx context.Context, id core.AuditEventID) audit.GetResult {
	var rawID string
	var rawActorID string
	var action string
	var subjectKind string
	var subjectID string
	var metadataJSON string
	var createdAt time.Time
	if err := store.db.QueryRow(ctx, `
		select id::text, actor_user_id::text, action, subject_kind, subject_id, metadata_json::text, created_at
		from audit_events
		where id = $1
	`, id.String()).Scan(&rawID, &rawActorID, &action, &subjectKind, &subjectID, &metadataJSON, &createdAt); err != nil {
		if err == ErrNoRows {
			return audit.GetRejected{Reason: core.NewDomainError(core.ErrorCodeNotFound, "audit event was not found")}
		}
		return audit.GetRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "get audit event failed")}
	}
	result := parseAuditEvent(rawID, rawActorID, action, subjectKind, subjectID, metadataJSON, createdAt)
	event, matched := result.(auditEventAccepted)
	if !matched {
		return audit.GetRejected{Reason: result.(auditEventRejected).reason}
	}
	return audit.EventFound{Value: event.value}
}

func (store AuditStore) List(ctx context.Context, filters audit.ListFilters, page core.Page) audit.ListResult {
	where := ""
	arguments := NamedArgs{"limit": page.Limit(), "offset": page.Offset()}

	switch filter := filters.Action.(type) {
	case audit.ActionEquals:
		arguments["action"] = filter.Value.String()
		where += " where action = @action"
	case audit.AnyAction:
	default:
		return audit.ListRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "audit action filter is invalid")}
	}
	switch filter := filters.SubjectKind.(type) {
	case audit.SubjectKindEquals:
		arguments["subject_kind"] = filter.Value
		where += auditFilterPrefix(where) + " subject_kind = @subject_kind"
	case audit.AnySubjectKind:
	default:
		return audit.ListRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "audit subject kind filter is invalid")}
	}
	switch filter := filters.SubjectID.(type) {
	case audit.SubjectIDEquals:
		arguments["subject_id"] = filter.Value
		where += auditFilterPrefix(where) + " subject_id = @subject_id"
	case audit.AnySubjectID:
	default:
		return audit.ListRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "audit subject id filter is invalid")}
	}

	rows, err := store.db.Query(ctx, `
		select id::text, actor_user_id::text, action, subject_kind, subject_id, metadata_json::text, created_at
		from audit_events
		`+where+`
		order by created_at desc, id desc
		limit @limit offset @offset
	`, arguments)
	if err != nil {
		return audit.ListRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "list audit events failed")}
	}
	defer rows.Close()

	events := make([]audit.Event, 0)
	for rows.Next() {
		result := scanAuditEvent(rows)
		event, matched := result.(auditEventAccepted)
		if !matched {
			return audit.ListRejected{Reason: result.(auditEventRejected).reason}
		}
		events = append(events, event.value)
	}
	if err := rows.Err(); err != nil {
		return audit.ListRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "read audit events failed")}
	}
	return audit.EventsListed{Values: events}
}

func auditFilterPrefix(where string) string {
	if where == "" {
		return " where"
	}
	return " and"
}

type auditEventResult interface {
	auditEventResult()
}

type auditEventAccepted struct {
	value audit.Event
}

type auditEventRejected struct {
	reason core.DomainError
}

func (auditEventAccepted) auditEventResult() {}

func (auditEventRejected) auditEventResult() {}

func scanAuditEvent(rows Rows) auditEventResult {
	var rawID string
	var rawActorID string
	var action string
	var subjectKind string
	var subjectID string
	var metadataJSON string
	var createdAt time.Time
	if err := rows.Scan(&rawID, &rawActorID, &action, &subjectKind, &subjectID, &metadataJSON, &createdAt); err != nil {
		return auditEventRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "scan audit event failed")}
	}
	return parseAuditEvent(rawID, rawActorID, action, subjectKind, subjectID, metadataJSON, createdAt)
}

func parseAuditEvent(rawID string, rawActorID string, action string, subjectKind string, subjectID string, metadataJSON string, createdAt time.Time) auditEventResult {
	idResult := core.ParseAuditEventID(rawID)
	id, idMatched := idResult.(core.AuditEventIDCreated)
	if !idMatched {
		return auditEventRejected{reason: idResult.(core.AuditEventIDRejected).Reason}
	}
	actorResult := core.ParseUserID(rawActorID)
	actor, actorMatched := actorResult.(core.UserIDCreated)
	if !actorMatched {
		return auditEventRejected{reason: actorResult.(core.UserIDRejected).Reason}
	}
	return auditEventAccepted{value: audit.Event{
		ID:          id.Value,
		ActorUserID: actor.Value,
		Action:      audit.ActionFromString(action),
		Subject:     audit.Subject{Kind: subjectKind, ID: subjectID},
		Metadata:    audit.Metadata{JSON: metadataJSON},
		CreatedAt:   createdAt,
	}}
}
