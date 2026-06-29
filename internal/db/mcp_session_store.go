package db

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type MCPSessionStore struct {
	pool *pgxpool.Pool
}

func NewMCPSessionStore(pool *pgxpool.Pool) MCPSessionStore {
	return MCPSessionStore{pool: pool}
}

func (store MCPSessionStore) CreateMCPSession(ctx context.Context, id string, subject string, now time.Time) error {
	_, err := store.pool.Exec(ctx, `
		insert into mcp_http_sessions (id, subject_id, last_seen_at, created_at)
		values ($1, $2, $3, $3)
		on conflict (id) do update
		set subject_id = excluded.subject_id,
			closed_at = null,
			last_seen_at = excluded.last_seen_at
	`, id, subject, now)
	return err
}

func (store MCPSessionStore) TouchMCPSession(ctx context.Context, id string, subject string, now time.Time, cutoff time.Time) (bool, error) {
	command, err := store.pool.Exec(ctx, `
		update mcp_http_sessions
		set last_seen_at = $3
		where id = $1
		and subject_id = $2
		and closed_at is null
		and last_seen_at >= $4
	`, id, subject, now, cutoff)
	if err != nil {
		return false, err
	}
	return command.RowsAffected() == 1, nil
}

func (store MCPSessionStore) CloseMCPSession(ctx context.Context, id string, now time.Time) (bool, error) {
	command, err := store.pool.Exec(ctx, `
		update mcp_http_sessions
		set closed_at = $2
		where id = $1 and closed_at is null
	`, id, now)
	if err != nil {
		return false, err
	}
	return command.RowsAffected() == 1, nil
}

func (store MCPSessionStore) ActiveMCPSessionCount(ctx context.Context, cutoff time.Time) (int, error) {
	var count int
	err := store.pool.QueryRow(ctx, `
		select count(*)
		from mcp_http_sessions
		where closed_at is null and last_seen_at >= $1
	`, cutoff).Scan(&count)
	return count, err
}

func (store MCPSessionStore) ActiveMCPSessionCountForSubject(ctx context.Context, subject string, cutoff time.Time) (int, error) {
	var count int
	err := store.pool.QueryRow(ctx, `
		select count(*)
		from mcp_http_sessions
		where subject_id = $1
		and closed_at is null
		and last_seen_at >= $2
	`, subject, cutoff).Scan(&count)
	return count, err
}
