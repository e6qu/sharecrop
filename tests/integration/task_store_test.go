//go:build integration

package integration_test

import (
	"context"
	"testing"

	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/db"
	"github.com/e6qu/sharecrop/internal/task"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TestTaskStoreCancelReleasesSubmittedReservation covers the data-hygiene
// invariant that cancelling a task releases every non-terminal reservation still
// held on it. A worker's reservation left in the submitted state must move to a
// terminal state when the task is cancelled, rather than dangling forever
// (releaseExpiredReservations never touches submitted reservations).
func TestTaskStoreCancelReleasesSubmittedReservation(t *testing.T) {
	pool := newPool(t)
	owner := createUser(t, pool, "cancel-release-owner")
	worker := createUser(t, pool, "cancel-release-worker")
	taskID := insertTask(t, pool, owner, "open", 30)
	insertTaskUserVisibility(t, pool, taskID, owner)
	insertSubmittedReservation(t, pool, taskID, worker)

	store := db.NewTaskStore(pool)
	result := store.ChangeTaskState(context.Background(), taskID, task.StateCancelled)
	changed, matched := result.(task.ChangeTaskStateStoreAccepted)
	if !matched {
		t.Fatalf("cancel task with submitted reservation: want ChangeTaskStateStoreAccepted, got %#v", result)
	}
	if changed.Value.State != task.StateCancelled {
		t.Fatalf("cancelled task state = %v, want cancelled", changed.Value.State)
	}

	var reservationState string
	if err := pool.QueryRow(context.Background(),
		"select state from task_reservations where task_id = $1", taskID.String()).Scan(&reservationState); err != nil {
		t.Fatalf("read reservation state: %v", err)
	}
	if reservationState != task.ReservationStateCancelledByRequester.String() {
		t.Fatalf("reservation state after cancel = %q, want %q (no longer submitted)", reservationState, task.ReservationStateCancelledByRequester.String())
	}
}

func insertTaskUserVisibility(t *testing.T, pool *pgxpool.Pool, taskID core.TaskID, owner core.UserID) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `
		insert into task_visibility_scopes (task_id, visibility_kind, scope_key, user_id)
		values ($1, 'user', $2, $3)
	`, taskID.String(), owner.String(), owner.String())
	if err != nil {
		t.Fatalf("insert task visibility: %v", err)
	}
}
