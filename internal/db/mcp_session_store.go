package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type MCPSessionStore struct {
	db Beginner
}

func NewMCPSessionStore(pool *pgxpool.Pool) MCPSessionStore {
	return MCPSessionStore{db: NewPGX(pool)}
}

func (store MCPSessionStore) CreateMCPSession(ctx context.Context, id string, subject string, now time.Time) error {
	_, err := store.db.Exec(ctx, `
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
	command, err := store.db.Exec(ctx, `
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
	return command == 1, nil
}

func (store MCPSessionStore) CloseMCPSession(ctx context.Context, id string, now time.Time) (bool, error) {
	command, err := store.db.Exec(ctx, `
		update mcp_http_sessions
		set closed_at = $2
		where id = $1 and closed_at is null
	`, id, now)
	if err != nil {
		return false, err
	}
	return command == 1, nil
}

func (store MCPSessionStore) ActiveMCPSessionCount(ctx context.Context, cutoff time.Time) (int, error) {
	var count int
	err := store.db.QueryRow(ctx, `
		select count(*)
		from mcp_http_sessions
		where closed_at is null and last_seen_at >= $1
	`, cutoff).Scan(&count)
	return count, err
}

func (store MCPSessionStore) ActiveMCPSessionCountForSubject(ctx context.Context, subject string, cutoff time.Time) (int, error) {
	var count int
	err := store.db.QueryRow(ctx, `
		select count(*)
		from mcp_http_sessions
		where subject_id = $1
		and closed_at is null
		and last_seen_at >= $2
	`, subject, cutoff).Scan(&count)
	return count, err
}

func (store MCPSessionStore) AppendMCPEvent(ctx context.Context, sessionID string, payload []byte, now time.Time) (string, []byte, error) {
	tx, err := store.db.Begin(ctx)
	if err != nil {
		return "", nil, err
	}
	defer func() {
		rollbackErr := tx.Rollback(ctx)
		_ = rollbackErr
	}()

	var exists bool
	if err := tx.QueryRow(ctx, `
		select true
		from mcp_http_sessions
		where id = $1 and closed_at is null
		for update
	`, sessionID).Scan(&exists); err != nil {
		return "", nil, err
	}
	var sequence int64
	if err := tx.QueryRow(ctx, `
		select coalesce(max(sequence), 0) + 1
		from mcp_http_events
		where session_id = $1
	`, sessionID).Scan(&sequence); err != nil {
		return "", nil, err
	}
	eventID := fmt.Sprintf("%s-%d", sessionID, sequence)
	copied := make([]byte, len(payload))
	copy(copied, payload)
	if _, err := tx.Exec(ctx, `
		insert into mcp_http_events (session_id, sequence, event_id, payload, created_at)
		values ($1, $2, $3, $4, $5)
	`, sessionID, sequence, eventID, copied, now); err != nil {
		return "", nil, err
	}
	if _, err := tx.Exec(ctx, `
		delete from mcp_http_events
		where session_id = $1
		and sequence not in (
			select sequence
			from mcp_http_events
			where session_id = $1
			order by sequence desc
			limit 100
		)
	`, sessionID); err != nil {
		return "", nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return "", nil, err
	}
	return eventID, copied, nil
}

func (store MCPSessionStore) ListMCPEvents(ctx context.Context, sessionID string, lastEventID string, limit int) ([]string, [][]byte, error) {
	var afterSequence int64
	if lastEventID != "" {
		err := store.db.QueryRow(ctx, `
			select sequence
			from mcp_http_events
			where session_id = $1 and event_id = $2
		`, sessionID, lastEventID).Scan(&afterSequence)
		if err != nil && err != ErrNoRows {
			return nil, nil, err
		}
		if err == ErrNoRows {
			afterSequence = 1<<63 - 1
		}
	}
	rows, err := store.db.Query(ctx, `
		select event_id, payload
		from mcp_http_events
		where session_id = $1 and sequence > $2
		order by sequence asc
		limit $3
	`, sessionID, afterSequence, limit)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	eventIDs := make([]string, 0)
	payloads := make([][]byte, 0)
	for rows.Next() {
		var eventID string
		var payload []byte
		if err := rows.Scan(&eventID, &payload); err != nil {
			return nil, nil, err
		}
		copied := make([]byte, len(payload))
		copy(copied, payload)
		eventIDs = append(eventIDs, eventID)
		payloads = append(payloads, copied)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	return eventIDs, payloads, nil
}
