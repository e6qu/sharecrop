package wasmdemo

import (
	"context"
	"testing"

	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/task"
)

func TestTaskBrowserStoreCreateSeriesAndList(t *testing.T) {
	taskService, _, _ := newTaskTestEnv(t)
	ctx := context.Background()
	owner := auth.UserSubject{ID: testUserID(t, "owner")}

	titleResult := task.NewSeriesTitle("Sprint 1")
	title := titleResult.(task.SeriesTitleAccepted).Value
	descriptionResult := task.NewSeriesDescription("A batch of related tasks.")
	description := descriptionResult.(task.SeriesDescriptionAccepted).Value

	createResult := taskService.CreateSeries(ctx, owner, title, description)
	created, matched := createResult.(task.SeriesMutated)
	if !matched {
		t.Fatalf("create series: want SeriesMutated, got %#v", createResult)
	}
	if created.Value.Series.State != task.SeriesStateDraft {
		t.Fatalf("new series state = %v, want draft", created.Value.Series.State)
	}
	if created.Value.Series.Title.String() != "Sprint 1" {
		t.Fatalf("series title = %q, want %q", created.Value.Series.Title.String(), "Sprint 1")
	}

	listResult := taskService.ListSeries(ctx, owner, testPage(t, 10, 0))
	listed, matched := listResult.(task.SeriesListed)
	if !matched {
		t.Fatalf("list series: want SeriesListed, got %#v", listResult)
	}
	if len(listed.Values) != 1 || listed.Values[0].ID != created.Value.Series.ID {
		t.Fatalf("listed series = %+v, want just %v", listed.Values, created.Value.Series.ID)
	}

	getResult := taskService.GetSeries(ctx, owner, created.Value.Series.ID)
	got, matched := getResult.(task.SeriesGot)
	if !matched {
		t.Fatalf("get series: want SeriesGot, got %#v", getResult)
	}
	if got.Value.Series.Title.String() != "Sprint 1" {
		t.Fatalf("got series title = %q, want %q", got.Value.Series.Title.String(), "Sprint 1")
	}
}

func TestTaskBrowserStoreSeriesAddRemoveReorderTasks(t *testing.T) {
	taskService, _, _ := newTaskTestEnv(t)
	ctx := context.Background()
	owner := auth.UserSubject{ID: testUserID(t, "owner")}

	title := task.NewSeriesTitle("Sprint 1").(task.SeriesTitleAccepted).Value
	description := task.NewSeriesDescription("").(task.SeriesDescriptionAccepted).Value
	series := taskService.CreateSeries(ctx, owner, title, description).(task.SeriesMutated)

	firstTask := taskService.Create(ctx, testCreateCommand(t, owner.ID, task.NoRewardSpec{}, task.ParticipationPolicyOpen)).(task.TaskCreated)
	secondTask := taskService.Create(ctx, testCreateCommand(t, owner.ID, task.NoRewardSpec{}, task.ParticipationPolicyOpen)).(task.TaskCreated)

	addFirst := taskService.AddTaskToSeries(ctx, owner, series.Value.Series.ID, firstTask.Value.ID)
	addedFirst, matched := addFirst.(task.SeriesMutated)
	if !matched {
		t.Fatalf("add first task to series: want SeriesMutated, got %#v", addFirst)
	}
	if len(addedFirst.Value.Tasks) != 1 {
		t.Fatalf("series tasks after first add = %+v, want 1", addedFirst.Value.Tasks)
	}

	addSecond := taskService.AddTaskToSeries(ctx, owner, series.Value.Series.ID, secondTask.Value.ID)
	addedSecond, matched := addSecond.(task.SeriesMutated)
	if !matched {
		t.Fatalf("add second task to series: want SeriesMutated, got %#v", addSecond)
	}
	if len(addedSecond.Value.Tasks) != 2 {
		t.Fatalf("series tasks after second add = %+v, want 2", addedSecond.Value.Tasks)
	}
	if addedSecond.Value.Tasks[0].ID != firstTask.Value.ID || addedSecond.Value.Tasks[1].ID != secondTask.Value.ID {
		t.Fatalf("series task order = %+v, want first then second", addedSecond.Value.Tasks)
	}

	reorderResult := taskService.ReorderSeries(ctx, owner, series.Value.Series.ID, []core.TaskID{secondTask.Value.ID, firstTask.Value.ID})
	reordered, matched := reorderResult.(task.SeriesMutated)
	if !matched {
		t.Fatalf("reorder series: want SeriesMutated, got %#v", reorderResult)
	}
	if reordered.Value.Tasks[0].ID != secondTask.Value.ID || reordered.Value.Tasks[1].ID != firstTask.Value.ID {
		t.Fatalf("series task order after reorder = %+v, want second then first", reordered.Value.Tasks)
	}
}

func TestTaskBrowserStoreSeriesUpdateTitleAndState(t *testing.T) {
	taskService, _, _ := newTaskTestEnv(t)
	ctx := context.Background()
	owner := auth.UserSubject{ID: testUserID(t, "owner")}

	title := task.NewSeriesTitle("Sprint 1").(task.SeriesTitleAccepted).Value
	description := task.NewSeriesDescription("v1").(task.SeriesDescriptionAccepted).Value
	series := taskService.CreateSeries(ctx, owner, title, description).(task.SeriesMutated)

	newTitle := task.NewSeriesTitle("Sprint 1 (updated)").(task.SeriesTitleAccepted).Value
	newDescription := task.NewSeriesDescription("v2").(task.SeriesDescriptionAccepted).Value
	updateResult := taskService.UpdateSeries(ctx, owner, series.Value.Series.ID, newTitle, newDescription)
	updated, matched := updateResult.(task.SeriesMutated)
	if !matched {
		t.Fatalf("update series: want SeriesMutated, got %#v", updateResult)
	}
	if updated.Value.Series.Title.String() != "Sprint 1 (updated)" {
		t.Fatalf("updated series title = %q, want %q", updated.Value.Series.Title.String(), "Sprint 1 (updated)")
	}

	publishResult := taskService.ChangeSeriesState(ctx, owner, series.Value.Series.ID, task.PublishSeriesState)
	published, matched := publishResult.(task.SeriesMutated)
	if !matched {
		t.Fatalf("publish series: want SeriesMutated, got %#v", publishResult)
	}
	if published.Value.Series.State != task.SeriesStatePublished {
		t.Fatalf("published series state = %v, want published", published.Value.Series.State)
	}
}

func TestTaskBrowserStoreRemoveTaskFromSeries(t *testing.T) {
	taskService, _, _ := newTaskTestEnv(t)
	ctx := context.Background()
	owner := auth.UserSubject{ID: testUserID(t, "owner")}

	title := task.NewSeriesTitle("Sprint 1").(task.SeriesTitleAccepted).Value
	description := task.NewSeriesDescription("").(task.SeriesDescriptionAccepted).Value
	series := taskService.CreateSeries(ctx, owner, title, description).(task.SeriesMutated)
	created := taskService.Create(ctx, testCreateCommand(t, owner.ID, task.NoRewardSpec{}, task.ParticipationPolicyOpen)).(task.TaskCreated)
	taskService.AddTaskToSeries(ctx, owner, series.Value.Series.ID, created.Value.ID)

	removeResult := taskService.RemoveTaskFromSeries(ctx, owner, series.Value.Series.ID, created.Value.ID)
	removed, matched := removeResult.(task.SeriesMutated)
	if !matched {
		t.Fatalf("remove task from series: want SeriesMutated, got %#v", removeResult)
	}
	if len(removed.Value.Tasks) != 0 {
		t.Fatalf("series tasks after remove = %+v, want none", removed.Value.Tasks)
	}
}

func TestTaskBrowserStoreCreateTaskWithNewSeriesPlacement(t *testing.T) {
	taskService, _, _ := newTaskTestEnv(t)
	ctx := context.Background()
	owner := testUserID(t, "owner")
	ownerSubject := auth.UserSubject{ID: owner}

	titleResult := task.NewSeriesTitle("Batch A")
	seriesTitle := titleResult.(task.SeriesTitleAccepted).Value
	positionResult := task.NewSeriesPosition(1)
	position := positionResult.(task.SeriesPositionAccepted).Value

	command := testCreateCommand(t, owner, task.NoRewardSpec{}, task.ParticipationPolicyOpen)
	command.Placement = task.NewSeriesPlacement{Title: seriesTitle, Position: position}
	created := taskService.Create(ctx, command)
	createdTask, matched := created.(task.TaskCreated)
	if !matched {
		t.Fatalf("create task with new series placement: want TaskCreated, got %#v", created)
	}
	placement, placementMatched := createdTask.Value.Placement.(task.ExistingSeriesPlacement)
	if !placementMatched {
		t.Fatalf("created task placement = %#v, want ExistingSeriesPlacement", createdTask.Value.Placement)
	}

	getResult := taskService.GetSeries(ctx, ownerSubject, placement.SeriesID)
	got, matched := getResult.(task.SeriesGot)
	if !matched {
		t.Fatalf("get series: want SeriesGot, got %#v", getResult)
	}
	if got.Value.Series.Title.String() != "Batch A" {
		t.Fatalf("auto-created series title = %q, want %q", got.Value.Series.Title.String(), "Batch A")
	}
	if len(got.Value.Tasks) != 1 || got.Value.Tasks[0].ID != createdTask.Value.ID {
		t.Fatalf("auto-created series tasks = %+v, want just the created task", got.Value.Tasks)
	}
}
