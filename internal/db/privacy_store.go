package db

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/core/id"
	httpserver "github.com/e6qu/sharecrop/internal/http"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PrivacyStore struct {
	pool *pgxpool.Pool
}

func NewPrivacyStore(pool *pgxpool.Pool) PrivacyStore {
	return PrivacyStore{pool: pool}
}

func (store PrivacyStore) Create(ctx context.Context, requester core.UserID, kind string) httpserver.PrivacyMutationResult {
	requestIDResult := id.New()
	requestID, requestIDMatched := requestIDResult.(id.IDCreated)
	if !requestIDMatched {
		return httpserver.PrivacyRequestMutationRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidID, requestIDResult.(id.IDRejected).Description)}
	}
	record := httpserver.PrivacyRequestRecord{ID: requestID.Value.String(), RequestedBy: requester, Kind: kind, State: "queued", CreatedAt: time.Now().UTC()}
	_, err := store.pool.Exec(ctx, `
		insert into privacy_requests (id, requested_by_user_id, kind, state, export_json, resolution_note, created_at)
		values ($1, $2, $3, $4, '', '', $5)
	`, record.ID, requester.String(), kind, record.State, record.CreatedAt)
	if err != nil {
		return httpserver.PrivacyRequestMutationRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "create privacy request failed")}
	}
	return httpserver.PrivacyRequestSaved{Value: record}
}

func (store PrivacyStore) ListForRequester(ctx context.Context, requester core.UserID, page core.Page) httpserver.PrivacyListResult {
	rows, err := store.pool.Query(ctx, `
		select id::text, requested_by_user_id::text, kind, state, export_json, resolution_note, created_at, coalesce(resolved_at, '0001-01-01T00:00:00Z'::timestamptz)
		from privacy_requests
		where requested_by_user_id = $1
		order by created_at desc, id desc
		limit $2 offset $3
	`, requester.String(), page.Limit(), page.Offset())
	if err != nil {
		return httpserver.PrivacyRequestListRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "list privacy requests failed")}
	}
	return scanPrivacyRequests(rows)
}

func (store PrivacyStore) ListAll(ctx context.Context, page core.Page) httpserver.PrivacyListResult {
	rows, err := store.pool.Query(ctx, `
		select id::text, requested_by_user_id::text, kind, state, export_json, resolution_note, created_at, coalesce(resolved_at, '0001-01-01T00:00:00Z'::timestamptz)
		from privacy_requests
		order by created_at desc, id desc
		limit $1 offset $2
	`, page.Limit(), page.Offset())
	if err != nil {
		return httpserver.PrivacyRequestListRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "list admin privacy requests failed")}
	}
	return scanPrivacyRequests(rows)
}

func (store PrivacyStore) Resolve(ctx context.Context, requestID string, note string) httpserver.PrivacyMutationResult {
	tx, err := store.pool.Begin(ctx)
	if err != nil {
		return httpserver.PrivacyRequestMutationRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "begin privacy request resolution failed")}
	}
	defer tx.Rollback(ctx)

	row := tx.QueryRow(ctx, `
		select id::text, requested_by_user_id::text, kind, state, export_json, resolution_note, created_at, coalesce(resolved_at, '0001-01-01T00:00:00Z'::timestamptz)
		from privacy_requests
		where id = $1
		for update
	`, requestID)
	var rawID string
	var rawRequesterID string
	var kind string
	var state string
	var exportJSON string
	var existingNote string
	var createdAt time.Time
	var resolvedAt time.Time
	if err := row.Scan(&rawID, &rawRequesterID, &kind, &state, &exportJSON, &existingNote, &createdAt, &resolvedAt); err != nil {
		return httpserver.PrivacyRequestMutationRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "scan privacy request failed")}
	}
	recordResult := parsePrivacyRequestRecord(rawID, rawRequesterID, kind, state, exportJSON, existingNote, createdAt, resolvedAt)
	record, matched := recordResult.(privacyRequestRowAccepted)
	if !matched {
		return httpserver.PrivacyRequestMutationRejected{Reason: recordResult.(privacyRequestRowRejected).reason}
	}
	if record.value.State != "queued" {
		return httpserver.PrivacyRequestMutationRejected{Reason: core.NewDomainError(core.ErrorCodeConflict, "privacy request is already resolved")}
	}

	resolved := record.value
	resolved.State = "resolved"
	resolved.ResolutionNote = strings.TrimSpace(note)
	resolved.ResolvedAt = time.Now().UTC()
	if resolved.Kind == "data_export" {
		exportBytes, err := json.Marshal(map[string]string{"user_id": resolved.RequestedBy.String()})
		if err != nil {
			return httpserver.PrivacyRequestMutationRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "privacy export encoding failed")}
		}
		resolved.ExportJSON = string(exportBytes)
	}
	if resolved.Kind == "sensitive_field_deletion" {
		if _, err := tx.Exec(ctx, `
			update submission_sensitive_fields
			set state = 'redacted', redacted_at = now()
			where state = 'active'
			and retention = 'delete_on_request'
			and submission_id in (select id from submissions where user_id = $1)
		`, resolved.RequestedBy.String()); err != nil {
			return httpserver.PrivacyRequestMutationRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "redact sensitive fields failed")}
		}
	}
	if _, err := tx.Exec(ctx, `
		update privacy_requests
		set state = 'resolved', export_json = $2, resolution_note = $3, resolved_at = $4
		where id = $1
	`, resolved.ID, resolved.ExportJSON, resolved.ResolutionNote, resolved.ResolvedAt); err != nil {
		return httpserver.PrivacyRequestMutationRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "resolve privacy request failed")}
	}
	if err := tx.Commit(ctx); err != nil {
		return httpserver.PrivacyRequestMutationRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "commit privacy request resolution failed")}
	}
	return httpserver.PrivacyRequestSaved{Value: resolved}
}

type privacyRequestRowResult interface {
	privacyRequestRowResult()
}

type privacyRequestRowAccepted struct {
	value httpserver.PrivacyRequestRecord
}

type privacyRequestRowRejected struct {
	reason core.DomainError
}

func (privacyRequestRowAccepted) privacyRequestRowResult() {}

func (privacyRequestRowRejected) privacyRequestRowResult() {}

func scanPrivacyRequests(rows pgx.Rows) httpserver.PrivacyListResult {
	defer rows.Close()
	requests := make([]httpserver.PrivacyRequestRecord, 0)
	for rows.Next() {
		var rawID string
		var rawRequesterID string
		var kind string
		var state string
		var exportJSON string
		var note string
		var createdAt time.Time
		var resolvedAt time.Time
		if err := rows.Scan(&rawID, &rawRequesterID, &kind, &state, &exportJSON, &note, &createdAt, &resolvedAt); err != nil {
			return httpserver.PrivacyRequestListRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "scan privacy request failed")}
		}
		result := parsePrivacyRequestRecord(rawID, rawRequesterID, kind, state, exportJSON, note, createdAt, resolvedAt)
		accepted, matched := result.(privacyRequestRowAccepted)
		if !matched {
			return httpserver.PrivacyRequestListRejected{Reason: result.(privacyRequestRowRejected).reason}
		}
		requests = append(requests, accepted.value)
	}
	if err := rows.Err(); err != nil {
		return httpserver.PrivacyRequestListRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "read privacy requests failed")}
	}
	return httpserver.PrivacyRequestsListed{Values: requests}
}

func parsePrivacyRequestRecord(rawID string, rawRequesterID string, kind string, state string, exportJSON string, note string, createdAt time.Time, resolvedAt time.Time) privacyRequestRowResult {
	requesterResult := core.ParseUserID(rawRequesterID)
	requester, matched := requesterResult.(core.UserIDCreated)
	if !matched {
		return privacyRequestRowRejected{reason: requesterResult.(core.UserIDRejected).Reason}
	}
	return privacyRequestRowAccepted{value: httpserver.PrivacyRequestRecord{
		ID:             rawID,
		RequestedBy:    requester.Value,
		Kind:           kind,
		State:          state,
		ExportJSON:     exportJSON,
		ResolutionNote: note,
		CreatedAt:      createdAt,
		ResolvedAt:     resolvedAt,
	}}
}
