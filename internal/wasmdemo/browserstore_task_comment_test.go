package wasmdemo

import (
	"context"
	"testing"

	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/task"
)

func TestTaskBrowserStoreAddAndListTaskComments(t *testing.T) {
	taskService, _, _, _ := newTaskTestEnv(t)
	ctx := context.Background()
	owner := auth.UserSubject{ID: testUserID(t, "owner")}

	created := taskService.Create(ctx, testCreateCommand(t, owner.ID, task.NoRewardSpec{}, task.ParticipationPolicyOpen)).(task.TaskCreated)

	body := task.NewCommentBody("Any updates on this one?").(task.CommentBodyAccepted).Value
	addResult := taskService.AddTaskComment(ctx, owner, created.Value.ID, body)
	added, matched := addResult.(task.TaskCommentAdded)
	if !matched {
		t.Fatalf("add task comment: want TaskCommentAdded, got %#v", addResult)
	}
	if added.Value.Body.String() != "Any updates on this one?" {
		t.Fatalf("added comment body = %q, want %q", added.Value.Body.String(), "Any updates on this one?")
	}

	listResult := taskService.ListTaskComments(ctx, owner, created.Value.ID)
	listed, matched := listResult.(task.TaskCommentsListed)
	if !matched {
		t.Fatalf("list task comments: want TaskCommentsListed, got %#v", listResult)
	}
	if len(listed.Values) != 1 || listed.Values[0].ID != added.Value.ID {
		t.Fatalf("listed comments = %+v, want just the added comment", listed.Values)
	}
}

func TestTaskBrowserStoreListTaskCommentsRejectsOutsider(t *testing.T) {
	taskService, _, _, _ := newTaskTestEnv(t)
	ctx := context.Background()
	owner := auth.UserSubject{ID: testUserID(t, "owner")}
	outsider := auth.UserSubject{ID: testUserID(t, "outsider")}

	command := testCreateCommand(t, owner.ID, task.NoRewardSpec{}, task.ParticipationPolicyOpen)
	command.Visibility = task.UserVisibility{UserID: owner.ID}
	created := taskService.Create(ctx, command).(task.TaskCreated)

	result := taskService.ListTaskComments(ctx, outsider, created.Value.ID)
	if _, matched := result.(task.TaskCommentsListRejected); !matched {
		t.Fatalf("outsider listing comments on a draft task: want TaskCommentsListRejected, got %#v", result)
	}
}
