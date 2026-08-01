//go:build integration

package integration_test

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/db"
	"github.com/e6qu/sharecrop/internal/event"
	httpserver "github.com/e6qu/sharecrop/internal/http"
	"github.com/e6qu/sharecrop/internal/ledger"
)

func insertActiveReservation(t *testing.T, pool *pgxpool.Pool, taskID core.TaskID, worker core.UserID, expiresPast bool) core.TaskReservationID {
	t.Helper()
	created, matched := core.NewTaskReservationID().(core.TaskReservationIDCreated)
	if !matched {
		t.Fatalf("reservation id rejected")
	}
	expiry := "now() + interval '1 day'"
	if expiresPast {
		expiry = "now() - interval '1 hour'"
	}
	_, err := pool.Exec(context.Background(), `
		insert into task_reservations (id, task_id, assignee_kind, user_id, state, requested_by_user_id, expires_at)
		values ($1, $2, 'user', $3, 'active', $3, `+expiry+`)
	`, created.Value.String(), taskID.String(), worker.String())
	if err != nil {
		t.Fatalf("insert reservation: %v", err)
	}
	return created.Value
}

func TestExpireDueTasksSweepRefundsAndReleases(t *testing.T) {
	pool := newPool(t)
	owner := createUser(t, pool, "expiry-owner")
	worker := createUser(t, pool, "expiry-worker")
	store := db.NewTaskStore(pool)
	ledgerStore := db.NewLedgerStore(pool)

	taskID := insertTaskWithRewardKind(t, pool, owner, "draft", "none")
	fundResult := ledgerStore.FundTask(context.Background(), fundCommand(t, owner, taskID, 25, "fund-expiry-"+taskID.String()))
	if _, matched := fundResult.(ledger.TaskFunded); !matched {
		t.Fatalf("fund task rejected: %T", fundResult)
	}
	setTaskState(t, pool, taskID, "open")
	reservationID := insertActiveReservation(t, pool, taskID, worker, false)
	if _, err := pool.Exec(context.Background(), "update tasks set expires_at = now() - interval '1 hour' where id = $1", taskID.String()); err != nil {
		t.Fatalf("set task expiry: %v", err)
	}

	result := store.ExpireDueTasks(context.Background())
	completed, matched := result.(db.ExpireDueTasksCompleted)
	if !matched {
		t.Fatalf("expire sweep rejected: %#v", result)
	}
	var draft event.Draft
	found := false
	for _, candidate := range completed.RecordedEvents {
		if subject, isTask := candidate.Subject.Task.(event.TaskSubject); isTask && subject.ID == taskID {
			draft = candidate
			found = true
		}
	}
	if !found {
		t.Fatalf("sweep did not record a draft for the due task")
	}
	if draft.Kind != event.KindTaskExpired {
		t.Fatalf("draft kind = %q, want task_expired", draft.Kind.String())
	}
	if !recipientsContain(draft.Recipients, owner) || !recipientsContain(draft.Recipients, worker) {
		t.Fatalf("draft recipients = %#v, want owner and released holder", draft.Recipients.Users)
	}
	// The draft was recorded inside the expire transaction (the outbox), so
	// a crash before the runner's inline dispatch cannot lose the event.
	if got := countEventsByKindForTask(t, pool, taskID, event.KindTaskExpired); got != 1 {
		t.Fatalf("recorded task_expired events = %d, want 1", got)
	}

	var state string
	if err := pool.QueryRow(context.Background(), "select state from tasks where id = $1", taskID.String()).Scan(&state); err != nil {
		t.Fatalf("read task state: %v", err)
	}
	if state != "expired" {
		t.Fatalf("task state = %q, want expired", state)
	}

	var reservationState string
	if err := pool.QueryRow(context.Background(), "select state from task_reservations where id = $1", reservationID.String()).Scan(&reservationState); err != nil {
		t.Fatalf("read reservation state: %v", err)
	}
	if reservationState != "cancelled_by_requester" {
		t.Fatalf("reservation state = %q, want cancelled_by_requester", reservationState)
	}

	var fundCount int
	if err := pool.QueryRow(context.Background(), "select count(*) from task_funds where task_id = $1", taskID.String()).Scan(&fundCount); err != nil {
		t.Fatalf("count task funds: %v", err)
	}
	if fundCount != 0 {
		t.Fatalf("task funds remain after expiry")
	}
	var refundCount int
	if err := pool.QueryRow(context.Background(), "select count(*) from ledger_entries where task_id = $1 and kind = 'task_refund' and idempotency_key = $2", taskID.String(), "expire:"+taskID.String()).Scan(&refundCount); err != nil {
		t.Fatalf("count refund entries: %v", err)
	}
	if refundCount != 1 {
		t.Fatalf("refund entries = %d, want 1", refundCount)
	}
	balance := mustBalance(t, ledgerStore, owner)
	if balance.Spendable() != 100 || balance.Allocated() != 0 {
		t.Fatalf("owner balance after expiry = %d/%d, want 100/0", balance.Spendable(), balance.Allocated())
	}

	// A second run is a no-op: the task is no longer open, so it is skipped
	// and no duplicate refund is written (the expire:<id> key would also
	// refuse one).
	second := store.ExpireDueTasks(context.Background())
	secondCompleted, secondMatched := second.(db.ExpireDueTasksCompleted)
	if !secondMatched {
		t.Fatalf("second sweep rejected: %#v", second)
	}
	for _, candidate := range secondCompleted.RecordedEvents {
		if subject, isTask := candidate.Subject.Task.(event.TaskSubject); isTask && subject.ID == taskID {
			t.Fatalf("second sweep reported the already-expired task")
		}
	}
	if err := pool.QueryRow(context.Background(), "select count(*) from ledger_entries where task_id = $1 and kind = 'task_refund'", taskID.String()).Scan(&refundCount); err != nil {
		t.Fatalf("recount refund entries: %v", err)
	}
	if refundCount != 1 {
		t.Fatalf("refund entries after second run = %d, want 1", refundCount)
	}
}

func TestExpireDueReservationsSweepReturnsReleasedRows(t *testing.T) {
	pool := newPool(t)
	owner := createUser(t, pool, "res-expiry-owner")
	worker := createUser(t, pool, "res-expiry-worker")
	store := db.NewTaskStore(pool)

	taskID := insertTaskWithRewardKind(t, pool, owner, "open", "none")
	reservationID := insertActiveReservation(t, pool, taskID, worker, true)

	result := store.ExpireDueReservations(context.Background())
	completed, matched := result.(db.ExpireDueReservationsCompleted)
	if !matched {
		t.Fatalf("reservation sweep rejected: %#v", result)
	}
	found := false
	for _, draft := range completed.RecordedEvents {
		subject, isReservation := draft.Subject.Reservation.(event.ReservationSubject)
		if !isReservation || subject.ID != reservationID {
			continue
		}
		found = true
		if draft.Kind != event.KindReservationExpired {
			t.Fatalf("draft kind = %q, want reservation_expired", draft.Kind.String())
		}
		taskSubject, isTask := draft.Subject.Task.(event.TaskSubject)
		if !isTask || taskSubject.ID != taskID {
			t.Fatalf("draft task subject = %#v", draft.Subject.Task)
		}
		if !recipientsContain(draft.Recipients, worker) || !recipientsContain(draft.Recipients, owner) {
			t.Fatalf("draft recipients = %#v, want holder and owner", draft.Recipients.Users)
		}
	}
	if !found {
		t.Fatalf("sweep did not record a draft for the due reservation")
	}

	var state string
	if err := pool.QueryRow(context.Background(), "select state from task_reservations where id = $1", reservationID.String()).Scan(&state); err != nil {
		t.Fatalf("read reservation state: %v", err)
	}
	if state != "expired" {
		t.Fatalf("reservation state = %q, want expired", state)
	}

	// Released once: the next sweep does not report it again, and no second
	// reservation_expired event was recorded.
	second := store.ExpireDueReservations(context.Background()).(db.ExpireDueReservationsCompleted)
	for _, draft := range second.RecordedEvents {
		if subject, isReservation := draft.Subject.Reservation.(event.ReservationSubject); isReservation && subject.ID == reservationID {
			t.Fatalf("second sweep reported the already-released reservation")
		}
	}
	if got := countEventsByKindForTask(t, pool, taskID, event.KindReservationExpired); got != 1 {
		t.Fatalf("recorded reservation_expired events = %d, want 1", got)
	}
}

// recipientsContain reports whether the recipient set names the user.
func recipientsContain(recipients event.Recipients, user core.UserID) bool {
	for _, candidate := range recipients.Users {
		if candidate == user {
			return true
		}
	}
	return false
}

func TestRateLimiterEvictExpiredAndFailClosed(t *testing.T) {
	pool := newPool(t)
	limiter := db.NewRateLimiter(pool, "sweeptest", 5, 100)

	key := "evict-" + newUserID(t).String()
	if !limiter.Allow(key) {
		t.Fatalf("fresh key should be allowed")
	}
	if _, err := pool.Exec(context.Background(), "update rate_limit_buckets set updated_at = now() - interval '1 day' where key = $1", "sweeptest:"+key); err != nil {
		t.Fatalf("age bucket: %v", err)
	}
	if err := limiter.EvictExpired(context.Background()); err != nil {
		t.Fatalf("evict expired: %v", err)
	}
	var count int
	if err := pool.QueryRow(context.Background(), "select count(*) from rate_limit_buckets where key = $1", "sweeptest:"+key).Scan(&count); err != nil {
		t.Fatalf("count buckets: %v", err)
	}
	if count != 0 {
		t.Fatalf("stale bucket survived eviction")
	}

	// Storage failure fails closed: a limiter over a closed pool denies
	// (and must not panic).
	closedPool, err := db.Open(context.Background(), requireEnv(t, "DATABASE_URL"))
	if err != nil {
		t.Fatalf("open second pool: %v", err)
	}
	brokenLimiter := db.NewRateLimiter(closedPool, "sweeptest", 5, 100)
	closedPool.Close()
	if brokenLimiter.Allow("broken-" + key) {
		t.Fatalf("limiter over a closed pool must deny (fail closed)")
	}
	// ActiveBuckets over a closed pool reports the unavailable variant
	// instead of panicking or fabricating a count.
	activeResult := brokenLimiter.ActiveBuckets()
	if _, unavailable := activeResult.(httpserver.ActiveBucketsUnavailable); !unavailable {
		t.Fatalf("unexpected active-buckets result: %#v", activeResult)
	}
}

func TestDeleteExpiredMCPSessionsSweep(t *testing.T) {
	pool := newPool(t)
	store := db.NewMCPSessionStore(pool)
	now := time.Now().UTC()

	staleID := "sweep-stale-" + newUserID(t).String()
	if err := store.CreateMCPSession(context.Background(), staleID, "user:sweep", now.Add(-2*time.Hour)); err != nil {
		t.Fatalf("create stale session: %v", err)
	}
	if _, _, err := store.AppendMCPEvent(context.Background(), staleID, []byte(`{"ok":true}`), now.Add(-2*time.Hour)); err != nil {
		t.Fatalf("append stale event: %v", err)
	}
	freshID := "sweep-fresh-" + newUserID(t).String()
	if err := store.CreateMCPSession(context.Background(), freshID, "user:sweep", now); err != nil {
		t.Fatalf("create fresh session: %v", err)
	}

	deleted, err := store.DeleteExpiredMCPSessions(context.Background(), now.Add(-30*time.Minute))
	if err != nil {
		t.Fatalf("delete expired sessions: %v", err)
	}
	if deleted < 1 {
		t.Fatalf("deleted = %d, want >= 1", deleted)
	}

	var staleSessions int
	if err := pool.QueryRow(context.Background(), "select count(*) from mcp_http_sessions where id = $1", staleID).Scan(&staleSessions); err != nil {
		t.Fatalf("count stale sessions: %v", err)
	}
	if staleSessions != 0 {
		t.Fatalf("stale session survived the sweep")
	}
	var staleEvents int
	if err := pool.QueryRow(context.Background(), "select count(*) from mcp_http_events where session_id = $1", staleID).Scan(&staleEvents); err != nil {
		t.Fatalf("count stale events: %v", err)
	}
	if staleEvents != 0 {
		t.Fatalf("stale session events survived the sweep")
	}
	var freshSessions int
	if err := pool.QueryRow(context.Background(), "select count(*) from mcp_http_sessions where id = $1", freshID).Scan(&freshSessions); err != nil {
		t.Fatalf("count fresh sessions: %v", err)
	}
	if freshSessions != 1 {
		t.Fatalf("fresh session was deleted")
	}
}
