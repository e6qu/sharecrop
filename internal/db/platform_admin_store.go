package db

import (
	"context"
	"sort"
	"time"

	"github.com/e6qu/sharecrop/internal/core"
	httpserver "github.com/e6qu/sharecrop/internal/http"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PlatformAdminStore struct {
	pool      *pgxpool.Pool
	bootstrap map[string]httpserver.PlatformAdminRecord
}

func NewPlatformAdminStore(pool *pgxpool.Pool, bootstrap map[string]bool) PlatformAdminStore {
	records := map[string]httpserver.PlatformAdminRecord{}
	for rawID := range bootstrap {
		parsed := core.ParseUserID(rawID)
		created, matched := parsed.(core.UserIDCreated)
		if matched {
			records[rawID] = httpserver.PlatformAdminRecord{UserID: created.Value, Source: "bootstrap", CreatedAt: time.Now().UTC()}
		}
	}
	return PlatformAdminStore{pool: pool, bootstrap: records}
}

func (store PlatformAdminStore) IsAdmin(ctx context.Context, userID core.UserID) httpserver.PlatformAdminCheckResult {
	if _, ok := store.bootstrap[userID.String()]; ok {
		return httpserver.PlatformAdminAllowed{}
	}
	var exists bool
	if err := store.pool.QueryRow(ctx, `select exists(select 1 from platform_admins where user_id = $1 and state = 'active')`, userID.String()).Scan(&exists); err != nil {
		return httpserver.PlatformAdminDenied{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "check platform admin failed")}
	}
	if exists {
		return httpserver.PlatformAdminAllowed{}
	}
	return httpserver.PlatformAdminDenied{Reason: core.NewDomainError(core.ErrorCodePermissionDenied, "platform admin access is required")}
}

func (store PlatformAdminStore) List(ctx context.Context, page core.Page) httpserver.PlatformAdminListResult {
	records := make([]httpserver.PlatformAdminRecord, 0, len(store.bootstrap))
	for _, record := range store.bootstrap {
		records = append(records, record)
	}
	rows, err := store.pool.Query(ctx, `
		select user_id::text, source, created_at
		from platform_admins
		where state = 'active'
		order by created_at desc, user_id desc
	`)
	if err != nil {
		return httpserver.PlatformAdminListRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "list platform admins failed")}
	}
	defer rows.Close()
	for rows.Next() {
		scanned := scanPlatformAdmin(rows)
		record, ok := scanned.(platformAdminAccepted)
		if !ok {
			return httpserver.PlatformAdminListRejected{Reason: scanned.(platformAdminRejected).reason}
		}
		if _, bootstrap := store.bootstrap[record.value.UserID.String()]; !bootstrap {
			records = append(records, record.value)
		}
	}
	if err := rows.Err(); err != nil {
		return httpserver.PlatformAdminListRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "read platform admins failed")}
	}
	sort.Slice(records, func(left int, right int) bool {
		return records[left].CreatedAt.After(records[right].CreatedAt)
	})
	return httpserver.PlatformAdminsListed{Values: pagePlatformAdminRecords(records, page)}
}

func (store PlatformAdminStore) Grant(ctx context.Context, userID core.UserID, actor core.UserID) httpserver.PlatformAdminMutationResult {
	if record, ok := store.bootstrap[userID.String()]; ok {
		return httpserver.PlatformAdminSaved{Value: record}
	}
	var createdAt time.Time
	if err := store.pool.QueryRow(ctx, `
		insert into platform_admins (user_id, source, state, granted_by_user_id)
		values ($1, 'granted', 'active', $2)
		on conflict (user_id) do update set source = 'granted', state = 'active', granted_by_user_id = $2, updated_at = now()
		returning created_at
	`, userID.String(), actor.String()).Scan(&createdAt); err != nil {
		return httpserver.PlatformAdminMutationRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "grant platform admin failed")}
	}
	return httpserver.PlatformAdminSaved{Value: httpserver.PlatformAdminRecord{UserID: userID, Source: "granted", CreatedAt: createdAt}}
}

func (store PlatformAdminStore) Revoke(ctx context.Context, userID core.UserID) httpserver.PlatformAdminMutationResult {
	if _, ok := store.bootstrap[userID.String()]; ok {
		return httpserver.PlatformAdminMutationRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "bootstrap platform admins cannot be revoked")}
	}
	var rawID string
	var source string
	var createdAt time.Time
	if err := store.pool.QueryRow(ctx, `
		update platform_admins
		set state = 'revoked', updated_at = now()
		where user_id = $1 and state = 'active'
		returning user_id::text, source, created_at
	`, userID.String()).Scan(&rawID, &source, &createdAt); err != nil {
		if err == pgx.ErrNoRows {
			return httpserver.PlatformAdminMutationRejected{Reason: core.NewDomainError(core.ErrorCodeNotFound, "platform admin was not found")}
		}
		return httpserver.PlatformAdminMutationRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "revoke platform admin failed")}
	}
	return httpserver.PlatformAdminSaved{Value: httpserver.PlatformAdminRecord{UserID: userID, Source: source, CreatedAt: createdAt}}
}

type platformAdminResult interface{ platformAdminResult() }
type platformAdminAccepted struct {
	value httpserver.PlatformAdminRecord
}
type platformAdminRejected struct{ reason core.DomainError }

func (platformAdminAccepted) platformAdminResult() {}
func (platformAdminRejected) platformAdminResult() {}

func scanPlatformAdmin(rows pgx.Rows) platformAdminResult {
	var rawID string
	var source string
	var createdAt time.Time
	if err := rows.Scan(&rawID, &source, &createdAt); err != nil {
		return platformAdminRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "scan platform admin failed")}
	}
	idResult := core.ParseUserID(rawID)
	idCreated, matched := idResult.(core.UserIDCreated)
	if !matched {
		return platformAdminRejected{reason: idResult.(core.UserIDRejected).Reason}
	}
	return platformAdminAccepted{value: httpserver.PlatformAdminRecord{UserID: idCreated.Value, Source: source, CreatedAt: createdAt}}
}

func pagePlatformAdminRecords(records []httpserver.PlatformAdminRecord, page core.Page) []httpserver.PlatformAdminRecord {
	start := page.Offset()
	if start > len(records) {
		start = len(records)
	}
	end := start + page.Limit()
	if end > len(records) {
		end = len(records)
	}
	return records[start:end]
}
