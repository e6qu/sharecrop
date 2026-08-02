//go:build integration

package integration_test

import (
	"context"
	"testing"

	"github.com/e6qu/sharecrop/internal/attachment"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/db"
	"github.com/e6qu/sharecrop/internal/event"
	"github.com/e6qu/sharecrop/internal/event/eventtest"
	"github.com/e6qu/sharecrop/internal/submission"
	"github.com/e6qu/sharecrop/internal/task"
	"github.com/jackc/pgx/v5/pgxpool"
)

// insertReservationRequiredTask creates an open, publicly visible task that
// requires a reservation and validates submissions against an object schema
// with one required string field ("answer").
func insertReservationRequiredTask(t *testing.T, pool *pgxpool.Pool, owner core.UserID) core.TaskID {
	t.Helper()
	taskID := newTaskID(t)
	schemaJSON := `{"kind":"object","fields":[{"name":"answer","presence":"required","schema":{"kind":"string"}}]}`
	_, err := pool.Exec(context.Background(), `
		insert into tasks (id, owner_kind, user_id, title, description, reward_kind, state, participation_policy, response_schema_json, data_payload_kind, created_by_user_id)
		values ($1, 'user', $2, 'Reservation flow task', 'Submit an answer string.', 'none', 'open', 'reservation_required', $3::jsonb, 'none', $2)
	`, taskID.String(), owner.String(), schemaJSON)
	if err != nil {
		t.Fatalf("insert reservation-required task: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `
		insert into task_visibility_scopes (task_id, visibility_kind, scope_key)
		values ($1, 'public', 'public')
	`, taskID.String()); err != nil {
		t.Fatalf("insert public visibility: %v", err)
	}
	return taskID
}

func reservationStateInDB(t *testing.T, pool *pgxpool.Pool, reservationID core.TaskReservationID) string {
	t.Helper()
	var state string
	if err := pool.QueryRow(context.Background(), "select state from task_reservations where id = $1", reservationID.String()).Scan(&state); err != nil {
		t.Fatalf("read reservation state: %v", err)
	}
	return state
}

func submitResponse(t *testing.T, service submission.Service, taskID core.TaskID, worker core.UserID, responseJSON string) submission.SubmitResult {
	t.Helper()
	source, matched := submission.NewResponseSource(responseJSON).(submission.ResponseSourceAccepted)
	if !matched {
		t.Fatalf("response source rejected")
	}
	return service.Submit(context.Background(), task.WorkerIsUser{}, submission.SubmitCommand{
		TaskID:         taskID,
		SubmitterID:    worker,
		ResponseSource: source.Value,
		Attachments:    []attachment.Attachment{},
	})
}

// TestInvalidSubmissionKeepsReservationActive proves a schema-invalid
// submission is recorded with its validation errors but does NOT consume the
// worker's active reservation: the corrected resubmission goes through
// immediately, and only the valid submission moves the reservation to
// submitted.
func TestInvalidSubmissionKeepsReservationActive(t *testing.T) {
	pool := newPool(t)
	owner := createUser(t, pool, "resubmit-owner")
	worker := createUser(t, pool, "resubmit-worker")
	taskID := insertReservationRequiredTask(t, pool, owner)

	taskStore := db.NewTaskStore(pool)
	reservationID, matched := core.NewTaskReservationID().(core.TaskReservationIDCreated)
	if !matched {
		t.Fatalf("reservation id rejected")
	}
	reserved, reservedMatched := taskStore.CreateReservation(context.Background(), reservationID.Value, task.ReservationCommand{
		Origin:      task.ReservedByUserSession{},
		TaskID:      taskID,
		Assignee:    task.UserAssignee{UserID: worker},
		RequestedBy: worker,
		Draft:       testEventDraft(t, event.KindReservationRequested, worker),
	}).(task.CreateReservationStoreAccepted)
	if !reservedMatched {
		t.Fatalf("create reservation rejected")
	}
	if reserved.Value.State != task.ReservationStateActive {
		t.Fatalf("reservation state = %q, want active", reserved.Value.State.String())
	}

	service := submission.NewService(db.NewSubmissionStore(pool), taskStore, nil, eventtest.NewRecorder())

	invalidResult := submitResponse(t, service, taskID, worker, `{"answer":12}`)
	invalidCreated, invalidMatched := invalidResult.(submission.SubmissionCreated)
	if !invalidMatched {
		t.Fatalf("invalid submission result = %#v, want SubmissionCreated", invalidResult)
	}
	if invalidCreated.Value.State != submission.StateInvalid {
		t.Fatalf("submission state = %q, want invalid", invalidCreated.Value.State.String())
	}
	failed, failedMatched := invalidCreated.Value.Validation.(submission.ValidationFailed)
	if !failedMatched || len(failed.Errors) == 0 {
		t.Fatalf("invalid submission must carry its validation errors, got %#v", invalidCreated.Value.Validation)
	}
	if state := reservationStateInDB(t, pool, reservationID.Value); state != task.ReservationStateActive.String() {
		t.Fatalf("reservation state after invalid submission = %q, want active (the worker must be able to resubmit)", state)
	}

	validResult := submitResponse(t, service, taskID, worker, `{"answer":"a corrected response"}`)
	validCreated, validMatched := validResult.(submission.SubmissionCreated)
	if !validMatched {
		t.Fatalf("valid resubmission result = %#v, want SubmissionCreated", validResult)
	}
	if validCreated.Value.State != submission.StateSubmitted {
		t.Fatalf("resubmission state = %q, want submitted", validCreated.Value.State.String())
	}
	if state := reservationStateInDB(t, pool, reservationID.Value); state != task.ReservationStateSubmitted.String() {
		t.Fatalf("reservation state after valid submission = %q, want submitted", state)
	}
}
