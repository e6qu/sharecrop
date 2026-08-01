//go:build integration

package integration_test

import (
	"context"
	"testing"
	"time"

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

// TestTaskStoreListCreatedAfterFilter proves the created-after filter excludes
// tasks created at or before the instant and composes with the list scope.
func TestTaskStoreListCreatedAfterFilter(t *testing.T) {
	pool := newPool(t)
	owner := createUser(t, pool, "created-after-owner")
	older := insertTask(t, pool, owner, "open", 30)
	insertTaskUserVisibility(t, pool, older, owner)
	newer := insertTask(t, pool, owner, "open", 30)
	insertTaskUserVisibility(t, pool, newer, owner)
	if _, err := pool.Exec(context.Background(), "update tasks set created_at = now() - interval '2 hours' where id = $1", older.String()); err != nil {
		t.Fatalf("age older task: %v", err)
	}

	var cutoff time.Time
	if err := pool.QueryRow(context.Background(), "select now() - interval '1 hour'").Scan(&cutoff); err != nil {
		t.Fatalf("read cutoff: %v", err)
	}

	store := db.NewTaskStore(pool)
	filters := task.NoListFilters()
	filters.Created = task.CreatedAfter{Instant: cutoff}
	result := store.ListTasks(context.Background(), task.UserListScope{UserID: owner}, filters, core.DefaultPage())
	listed, matched := result.(task.ListTasksStoreAccepted)
	if !matched {
		t.Fatalf("list with created-after filter rejected: %#v", result)
	}
	if len(listed.Values) != 1 {
		t.Fatalf("filtered task count = %d, want 1", len(listed.Values))
	}
	if listed.Values[0].Task.ID != newer {
		t.Fatalf("filtered task = %s, want the newer task %s", listed.Values[0].Task.ID.String(), newer.String())
	}

	unfiltered := store.ListTasks(context.Background(), task.UserListScope{UserID: owner}, task.NoListFilters(), core.DefaultPage())
	unfilteredListed, unfilteredMatched := unfiltered.(task.ListTasksStoreAccepted)
	if !unfilteredMatched {
		t.Fatalf("unfiltered list rejected: %#v", unfiltered)
	}
	if len(unfilteredListed.Values) != 2 {
		t.Fatalf("unfiltered task count = %d, want 2", len(unfilteredListed.Values))
	}
}

// TestTaskStoreListPendingReviewCount proves owned-task list rows carry the
// count of submissions still in state submitted (and only those).
func TestTaskStoreListPendingReviewCount(t *testing.T) {
	pool := newPool(t)
	owner := createUser(t, pool, "pending-count-owner")
	workerA := createUser(t, pool, "pending-count-worker-a")
	workerB := createUser(t, pool, "pending-count-worker-b")
	reviewed := createUser(t, pool, "pending-count-worker-c")

	busyTask := insertTask(t, pool, owner, "open", 30)
	insertTaskUserVisibility(t, pool, busyTask, owner)
	quietTask := insertTask(t, pool, owner, "open", 30)
	insertTaskUserVisibility(t, pool, quietTask, owner)

	insertSubmission(t, pool, busyTask, workerA)
	insertSubmission(t, pool, busyTask, workerB)
	reviewedSubmission := insertSubmission(t, pool, busyTask, reviewed)
	if _, err := pool.Exec(context.Background(), "update submissions set state = 'rejected' where id = $1", reviewedSubmission.String()); err != nil {
		t.Fatalf("mark submission reviewed: %v", err)
	}

	store := db.NewTaskStore(pool)
	result := store.ListTasks(context.Background(), task.UserListScope{UserID: owner}, task.NoListFilters(), core.DefaultPage())
	listed, matched := result.(task.ListTasksStoreAccepted)
	if !matched {
		t.Fatalf("list owned tasks rejected: %#v", result)
	}
	counts := map[string]int64{}
	for _, item := range listed.Values {
		counts[item.Task.ID.String()] = item.PendingReviewCount
	}
	if counts[busyTask.String()] != 2 {
		t.Fatalf("busy task pending review count = %d, want 2 (rejected submission must not count)", counts[busyTask.String()])
	}
	if counts[quietTask.String()] != 0 {
		t.Fatalf("quiet task pending review count = %d, want 0", counts[quietTask.String()])
	}
}
