//go:build integration

package integration_test

import (
	"context"
	"testing"

	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/db"
	"github.com/e6qu/sharecrop/internal/task"
	"github.com/e6qu/sharecrop/internal/task/tasktest"
	"github.com/e6qu/sharecrop/internal/wasibridge/rpc"
	"github.com/e6qu/sharecrop/internal/wasibridge/taskbridge"
)

// TestTaskBridgeDualRun drives the task store - the widest store - through the
// compiled wasip1 guest + host bridge against real Postgres: create a task,
// find it, change its state, list it, reserve it, comment on it, create a series
// and attach the task to it, and comment on the series. Every read path is
// checked against a direct db call so the whole Task/Series/Reservation/comment
// serialization is verified end to end.
func TestTaskBridgeDualRun(t *testing.T) {
	ctx := context.Background()
	pool := newPool(t)
	dbStore := db.NewTaskStore(pool)

	guestWASM, err := compileWASIGuest(t, "github.com/e6qu/sharecrop/cmd/sharecrop-wasi-store-guest")
	if err != nil {
		t.Fatalf("compile store guest: %v", err)
	}
	host, err := rpc.NewHost(ctx, guestWASM, func(ctx context.Context, method string, args []byte) ([]byte, error) {
		return taskbridge.Dispatch(ctx, dbStore, method, args)
	})
	if err != nil {
		t.Fatalf("new host: %v", err)
	}
	t.Cleanup(func() { _ = host.Close(ctx) })
	bridgeStore := taskbridge.NewGuestStore(func(method string, args []byte) ([]byte, error) {
		return host.Call(ctx, method, args)
	})

	owner := createUser(t, pool, "task-owner")
	worker := createUser(t, pool, "task-worker")
	page := requirePage(t, 50, 0)

	seriesID := tasktest.NewSeriesID(t)
	taskID := tasktest.NewTaskID(t)
	command := buildCreateCommand(t, owner)

	t.Run("create task then find matches a direct call", func(t *testing.T) {
		result := bridgeStore.CreateTask(ctx, seriesID, taskID, command)
		if rejected, matched := result.(task.CreateTaskStoreRejected); matched {
			t.Fatalf("bridge CreateTask rejected: %s", rejected.Reason.Description())
		}
		if _, matched := result.(task.CreateTaskStoreAccepted); !matched {
			t.Fatalf("bridge CreateTask = %T, want accepted", result)
		}
		viaBridge := requireTaskFound(t, bridgeStore.FindTask(ctx, taskID))
		direct := requireTaskFound(t, dbStore.FindTask(ctx, taskID))
		if diff := tasktest.TaskDiff(viaBridge, direct); diff != "" {
			t.Errorf("found task mismatch: %s", diff)
		}
	})

	t.Run("change task state to open through the bridge", func(t *testing.T) {
		result := bridgeStore.ChangeTaskState(ctx, taskID, task.StateOpen)
		accepted, matched := result.(task.ChangeTaskStateStoreAccepted)
		if !matched {
			t.Fatalf("bridge ChangeTaskState = %T, want accepted", result)
		}
		if accepted.Value.State.String() != "open" {
			t.Errorf("task state = %s, want open", accepted.Value.State)
		}
	})

	t.Run("list tasks by creator matches a direct call", func(t *testing.T) {
		scope := task.CreatorListScope{CreatorID: owner}
		viaBridge := requireTasksListed(t, bridgeStore.ListTasks(ctx, scope, task.NoListFilters(), page))
		direct := requireTasksListed(t, dbStore.ListTasks(ctx, scope, task.NoListFilters(), page))
		if len(viaBridge) != len(direct) || len(viaBridge) != 1 {
			t.Fatalf("task counts: bridge %d, direct %d, want 1", len(viaBridge), len(direct))
		}
		if diff := tasktest.ListItemDiff(viaBridge[0], direct[0]); diff != "" {
			t.Errorf("list item mismatch: %s", diff)
		}
	})

	t.Run("reserve the task through the bridge", func(t *testing.T) {
		reservationID := tasktest.NewReservationID(t)
		result := bridgeStore.CreateReservation(ctx, reservationID, task.ReservationCommand{
			TaskID:      taskID,
			Assignee:    task.UserAssignee{UserID: worker},
			RequestedBy: worker,
		})
		if _, matched := result.(task.CreateReservationStoreAccepted); !matched {
			t.Fatalf("bridge CreateReservation = %T, want accepted", result)
		}
		viaBridge := requireReservationsListed(t, bridgeStore.ListReservations(ctx, taskID))
		direct := requireReservationsListed(t, dbStore.ListReservations(ctx, taskID))
		if len(viaBridge) != len(direct) || len(viaBridge) != 1 {
			t.Fatalf("reservation counts: bridge %d, direct %d, want 1", len(viaBridge), len(direct))
		}
		if diff := tasktest.ReservationDiff(viaBridge[0], direct[0]); diff != "" {
			t.Errorf("reservation mismatch: %s", diff)
		}
	})

	t.Run("comment on the task through the bridge", func(t *testing.T) {
		comment := task.TaskComment{ID: newTaskCommentID(t), TaskID: taskID, AuthorID: owner, Body: tasktest.CommentBody(t, "progress update?")}
		if _, matched := bridgeStore.CreateTaskComment(ctx, comment).(task.CreateTaskCommentStoreAccepted); !matched {
			t.Fatalf("bridge CreateTaskComment did not accept")
		}
		viaBridge := requireTaskComments(t, bridgeStore.ListTaskComments(ctx, taskID))
		direct := requireTaskComments(t, dbStore.ListTaskComments(ctx, taskID))
		if len(viaBridge) != len(direct) || len(viaBridge) != 1 {
			t.Fatalf("task comment counts: bridge %d, direct %d, want 1", len(viaBridge), len(direct))
		}
		if viaBridge[0].Body.String() != direct[0].Body.String() {
			t.Errorf("task comment body mismatch: %q != %q", viaBridge[0].Body, direct[0].Body)
		}
	})

	t.Run("submission eligibility matches a direct call", func(t *testing.T) {
		_, bridgeRejected := bridgeStore.CheckSubmissionEligibility(ctx, taskID, worker).(task.SubmissionEligibilityRejected)
		_, directRejected := dbStore.CheckSubmissionEligibility(ctx, taskID, worker).(task.SubmissionEligibilityRejected)
		if bridgeRejected != directRejected {
			t.Errorf("submission eligibility disagrees: bridge rejected=%t, direct rejected=%t", bridgeRejected, directRejected)
		}
	})

	seriesID2 := tasktest.NewSeriesID(t)

	t.Run("create a series, attach the task, and find it", func(t *testing.T) {
		series := task.Series{
			ID:          seriesID2,
			Owner:       task.UserOwner{UserID: owner},
			Title:       seriesTitle(t, "Bridge series"),
			Description: seriesDescription(t, "A series created through the bridge."),
			State:       task.SeriesStateDraft,
			CreatedBy:   owner,
		}
		if _, matched := bridgeStore.CreateSeries(ctx, series).(task.SeriesMutationStoreAccepted); !matched {
			t.Fatalf("bridge CreateSeries did not accept")
		}
		if _, matched := bridgeStore.AddTaskToSeries(ctx, seriesID2, taskID).(task.SeriesMutationStoreAccepted); !matched {
			t.Fatalf("bridge AddTaskToSeries did not accept")
		}
		viaBridge := requireSeriesDetail(t, bridgeStore.FindSeries(ctx, seriesID2))
		direct := requireSeriesDetail(t, dbStore.FindSeries(ctx, seriesID2))
		if diff := tasktest.SeriesDiff(viaBridge.Series, direct.Series); diff != "" {
			t.Errorf("series mismatch: %s", diff)
		}
		if len(viaBridge.Tasks) != len(direct.Tasks) || len(viaBridge.Tasks) != 1 {
			t.Fatalf("series task counts: bridge %d, direct %d, want 1", len(viaBridge.Tasks), len(direct.Tasks))
		}
		if diff := tasktest.TaskDiff(viaBridge.Tasks[0], direct.Tasks[0]); diff != "" {
			t.Errorf("series task mismatch: %s", diff)
		}
	})

	t.Run("comment on the series through the bridge", func(t *testing.T) {
		comment := task.SeriesComment{ID: newSeriesCommentID(t), SeriesID: seriesID2, AuthorID: owner, Body: tasktest.CommentBody(t, "kickoff")}
		if _, matched := bridgeStore.CreateSeriesComment(ctx, comment).(task.CreateSeriesCommentStoreAccepted); !matched {
			t.Fatalf("bridge CreateSeriesComment did not accept")
		}
		viaBridge := requireSeriesComments(t, bridgeStore.ListSeriesComments(ctx, seriesID2))
		direct := requireSeriesComments(t, dbStore.ListSeriesComments(ctx, seriesID2))
		if len(viaBridge) != len(direct) || len(viaBridge) != 1 {
			t.Fatalf("series comment counts: bridge %d, direct %d, want 1", len(viaBridge), len(direct))
		}
	})
}

func buildCreateCommand(t *testing.T, owner core.UserID) task.CreateCommand {
	t.Helper()
	reference, matched := task.NewReferenceURL("").(task.ReferenceURLAccepted)
	if !matched {
		t.Fatalf("reference rejected")
	}
	return task.CreateCommand{
		Actor:          auth.UserSubject{ID: owner},
		Owner:          task.UserOwner{UserID: owner},
		Title:          tasktest.Title(t, "Review the bridge"),
		Description:    tasktest.Description(t, "Check the WASI bridge end to end."),
		Type:           task.TaskTypeGeneral,
		Reference:      reference.Value,
		Reward:         task.NoRewardSpec{},
		Participation:  task.ParticipationPolicyReservationRequired,
		AssigneeScope:  task.AssigneeScopeUser,
		ReservationTTL: task.DefaultReservationTTL(),
		Visibility:     task.PublicVisibility{},
		Placement:      task.StandalonePlacement{},
		ResponseSchema: tasktest.SchemaSource(t, `{"kind":"freeform"}`),
		Payload:        task.NoDataPayload{},
	}
}

func requireTaskFound(t *testing.T, result task.FindTaskStoreResult) task.Task {
	t.Helper()
	found, matched := result.(task.FindTaskStoreAccepted)
	if !matched {
		t.Fatalf("find task result = %T, want accepted", result)
	}
	return found.Value
}

func requireTasksListed(t *testing.T, result task.ListTasksStoreResult) []task.ListItem {
	t.Helper()
	listed, matched := result.(task.ListTasksStoreAccepted)
	if !matched {
		t.Fatalf("list tasks result = %T, want accepted", result)
	}
	return listed.Values
}

func requireReservationsListed(t *testing.T, result task.ListReservationsStoreResult) []task.Reservation {
	t.Helper()
	listed, matched := result.(task.ListReservationsStoreAccepted)
	if !matched {
		t.Fatalf("list reservations result = %T, want accepted", result)
	}
	return listed.Values
}

func requireTaskComments(t *testing.T, result task.ListTaskCommentsStoreResult) []task.TaskComment {
	t.Helper()
	listed, matched := result.(task.ListTaskCommentsStoreAccepted)
	if !matched {
		t.Fatalf("list task comments result = %T, want accepted", result)
	}
	return listed.Values
}

func requireSeriesDetail(t *testing.T, result task.FindSeriesStoreResult) task.SeriesDetail {
	t.Helper()
	found, matched := result.(task.FindSeriesStoreAccepted)
	if !matched {
		t.Fatalf("find series result = %T, want accepted", result)
	}
	return found.Value
}

func requireSeriesComments(t *testing.T, result task.ListSeriesCommentsStoreResult) []task.SeriesComment {
	t.Helper()
	listed, matched := result.(task.ListSeriesCommentsStoreAccepted)
	if !matched {
		t.Fatalf("list series comments result = %T, want accepted", result)
	}
	return listed.Values
}

func seriesTitle(t *testing.T, raw string) task.SeriesTitle {
	t.Helper()
	accepted, matched := task.NewSeriesTitle(raw).(task.SeriesTitleAccepted)
	if !matched {
		t.Fatalf("series title rejected")
	}
	return accepted.Value
}

func seriesDescription(t *testing.T, raw string) task.SeriesDescription {
	t.Helper()
	accepted, matched := task.NewSeriesDescription(raw).(task.SeriesDescriptionAccepted)
	if !matched {
		t.Fatalf("series description rejected")
	}
	return accepted.Value
}

func newTaskCommentID(t *testing.T) core.TaskCommentID {
	t.Helper()
	created, matched := core.NewTaskCommentID().(core.TaskCommentIDCreated)
	if !matched {
		t.Fatalf("task comment id rejected")
	}
	return created.Value
}

func newSeriesCommentID(t *testing.T) core.SeriesCommentID {
	t.Helper()
	created, matched := core.NewSeriesCommentID().(core.SeriesCommentIDCreated)
	if !matched {
		t.Fatalf("series comment id rejected")
	}
	return created.Value
}
