//go:build integration

package integration_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/db"
	"github.com/e6qu/sharecrop/internal/event"
	"github.com/e6qu/sharecrop/internal/task"
	"github.com/jackc/pgx/v5/pgxpool"
)

// insertRequestedReservation inserts a historical-style requested reservation
// (the pre-migration approval-gate shape; the task_reservations state check
// still allows the stored value).
func insertRequestedReservation(t *testing.T, pool *pgxpool.Pool, taskID core.TaskID, worker core.UserID, createdOffset string) core.TaskReservationID {
	t.Helper()
	created, matched := core.NewTaskReservationID().(core.TaskReservationIDCreated)
	if !matched {
		t.Fatalf("reservation id rejected")
	}
	_, err := pool.Exec(context.Background(), `
		insert into task_reservations (id, task_id, assignee_kind, user_id, state, requested_by_user_id, expires_at, created_at)
		values ($1, $2, 'user', $3, 'requested', $3, now() + interval '1 day', now() + `+createdOffset+`)
	`, created.Value.String(), taskID.String(), worker.String())
	if err != nil {
		t.Fatalf("insert requested reservation: %v", err)
	}
	return created.Value
}

func reservationStateOf(t *testing.T, pool *pgxpool.Pool, id core.TaskReservationID) string {
	t.Helper()
	var state string
	if err := pool.QueryRow(context.Background(), "select state from task_reservations where id = $1", id.String()).Scan(&state); err != nil {
		t.Fatalf("read reservation state: %v", err)
	}
	return state
}

// TestApprovalRemovalMigrationPromotesOldestRequested executes the
// data-migration statements from migrations/000049 against freshly crafted
// pre-migration-shaped rows: per task without an active reservation the
// oldest requested reservation is promoted to active, every other requested
// reservation is declined, and a task that already has an active holder gets
// no second active reservation.
func TestApprovalRemovalMigrationPromotesOldestRequested(t *testing.T) {
	pool := newPool(t)
	owner := createUser(t, pool, "promo-owner")
	first := createUser(t, pool, "promo-first")
	second := createUser(t, pool, "promo-second")
	third := createUser(t, pool, "promo-third")

	// Task A: no active reservation; three requests in age order.
	taskA := insertTaskWithRewardKind(t, pool, owner, "open", "none")
	oldest := insertRequestedReservation(t, pool, taskA, first, "interval '0 hours'")
	middle := insertRequestedReservation(t, pool, taskA, second, "interval '1 hour'")
	newest := insertRequestedReservation(t, pool, taskA, third, "interval '2 hours'")

	// Task B: already has an active holder plus a pending request.
	taskB := insertTaskWithRewardKind(t, pool, owner, "open", "none")
	activeB := insertActiveReservation(t, pool, taskB, first, false)
	pendingB := insertRequestedReservation(t, pool, taskB, second, "interval '0 hours'")

	migrationSQL, err := os.ReadFile(filepath.Join("..", "..", "migrations", "000049_remove_reservation_approval.sql"))
	if err != nil {
		t.Fatalf("read migration file: %v", err)
	}
	// Strip comment lines before splitting so a semicolon inside a comment
	// can never truncate a statement.
	executed := 0
	for _, statement := range strings.Split(stripSQLComments(string(migrationSQL)), ";") {
		trimmed := strings.TrimSpace(statement)
		if !strings.HasPrefix(strings.ToLower(trimmed), "update task_reservations") {
			continue
		}
		if _, err := pool.Exec(context.Background(), trimmed); err != nil {
			t.Fatalf("run migration statement %q: %v", trimmed, err)
		}
		executed++
	}
	if executed != 2 {
		t.Fatalf("executed %d task_reservations migration statements, want the promote and decline pair", executed)
	}

	if got := reservationStateOf(t, pool, oldest); got != "active" {
		t.Fatalf("oldest request state = %q, want active (promoted)", got)
	}
	if got := reservationStateOf(t, pool, middle); got != "declined" {
		t.Fatalf("middle request state = %q, want declined", got)
	}
	if got := reservationStateOf(t, pool, newest); got != "declined" {
		t.Fatalf("newest request state = %q, want declined", got)
	}
	if got := reservationStateOf(t, pool, activeB); got != "active" {
		t.Fatalf("existing active reservation state = %q, want untouched active", got)
	}
	if got := reservationStateOf(t, pool, pendingB); got != "declined" {
		t.Fatalf("pending request on the actively-held task = %q, want declined (no second active holder)", got)
	}
}

// stripSQLComments drops leading "-- ..." comment lines so a statement's
// first executable keyword can be inspected.
func stripSQLComments(statement string) string {
	lines := strings.Split(statement, "\n")
	kept := make([]string, 0, len(lines))
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "--") {
			continue
		}
		kept = append(kept, line)
	}
	return strings.TrimSpace(strings.Join(kept, "\n"))
}

// TestReserveThenSubmitDirectly pins the no-approval-gate flow at the store
// level: creating a reservation on a reservation_required task yields an
// immediately-active reservation, and the submitter is submission-eligible
// with no approval step in between.
func TestReserveThenSubmitDirectly(t *testing.T) {
	pool := newPool(t)
	store := db.NewTaskStore(pool)
	owner := createUser(t, pool, "direct-owner")
	worker := createUser(t, pool, "direct-worker")
	taskID := insertTaskWithRewardKind(t, pool, owner, "open", "none")
	if _, err := pool.Exec(context.Background(), "update tasks set participation_policy = 'reservation_required' where id = $1", taskID.String()); err != nil {
		t.Fatalf("set participation policy: %v", err)
	}

	reservationID, matched := core.NewTaskReservationID().(core.TaskReservationIDCreated)
	if !matched {
		t.Fatalf("reservation id rejected")
	}
	command := task.ReservationCommand{
		TaskID:      taskID,
		Assignee:    task.UserAssignee{UserID: worker},
		RequestedBy: worker,
		Draft:       testEventDraft(t, event.KindReservationRequested, worker),
	}
	result := store.CreateReservation(context.Background(), reservationID.Value, command)
	created, createdMatched := result.(task.CreateReservationStoreAccepted)
	if !createdMatched {
		t.Fatalf("create reservation rejected: %#v", result)
	}
	if created.Value.State != task.ReservationStateActive {
		t.Fatalf("reservation state = %s, want active immediately", created.Value.State.String())
	}

	eligibility := store.CheckSubmissionEligibility(context.Background(), taskID, worker)
	if _, eligible := eligibility.(task.SubmissionEligible); !eligible {
		t.Fatalf("submitter not eligible after reserving: %#v", eligibility)
	}
}
