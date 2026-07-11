package wasmdemo

import (
	"context"
	"testing"

	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/submission"
	"github.com/e6qu/sharecrop/internal/task"
)

func newSubmissionTestEnv(t *testing.T) (submission.Service, task.Service, *counterLedgerIDs) {
	t.Helper()
	storage := newTestBrowserStorage()
	ids := &counterLedgerIDs{}
	taskStore := NewTaskBrowserStore(storage, ids, systemTestClock{})
	taskService := task.NewService(taskStore, noopOrganizationPermissions{}, nil)
	submissionService := submission.NewService(NewSubmissionBrowserStore(storage, ids), taskStore, noopOrganizationPermissions{})
	return submissionService, taskService, ids
}

func testResponseSource(t *testing.T, raw string) submission.ResponseSource {
	t.Helper()
	result := submission.NewResponseSource(raw)
	accepted, matched := result.(submission.ResponseSourceAccepted)
	if !matched {
		t.Fatalf("new response source %q failed", raw)
	}
	return accepted.Value
}

func TestSubmissionBrowserStoreSubmitAndFindByReceipt(t *testing.T) {
	submissionService, taskService, _ := newSubmissionTestEnv(t)
	ctx := context.Background()
	owner := auth.UserSubject{ID: testUserID(t, "owner")}
	worker := testUserID(t, "worker")

	created := taskService.Create(ctx, testCreateCommand(t, owner.ID, task.NoRewardSpec{}, task.ParticipationPolicyOpen)).(task.TaskCreated)
	taskService.Open(ctx, owner, created.Value.ID)

	submitResult := submissionService.Submit(ctx, submission.SubmitCommand{
		TaskID: created.Value.ID, SubmitterID: worker, ResponseSource: testResponseSource(t, `{"note":"done"}`),
	})
	submitted, matched := submitResult.(submission.SubmissionCreated)
	if !matched {
		t.Fatalf("submit: want SubmissionCreated, got %#v", submitResult)
	}
	if submitted.Value.State != submission.StateSubmitted {
		t.Fatalf("submission state = %v, want submitted", submitted.Value.State)
	}

	receiptResult := submissionService.FindByReceipt(ctx, submitted.ReceiptToken)
	found, matched := receiptResult.(submission.ReceiptStatusFound)
	if !matched {
		t.Fatalf("find by receipt: want ReceiptStatusFound, got %#v", receiptResult)
	}
	if found.Value.ID != submitted.Value.ID {
		t.Fatalf("found submission ID = %v, want %v", found.Value.ID, submitted.Value.ID)
	}
}

func TestSubmissionBrowserStoreSubmitMarksReservationSubmitted(t *testing.T) {
	submissionService, taskService, _ := newSubmissionTestEnv(t)
	ctx := context.Background()
	owner := auth.UserSubject{ID: testUserID(t, "owner")}
	worker := auth.UserSubject{ID: testUserID(t, "worker")}

	created := taskService.Create(ctx, testCreateCommand(t, owner.ID, task.NoRewardSpec{}, task.ParticipationPolicyReservationRequired)).(task.TaskCreated)
	taskService.Open(ctx, owner, created.Value.ID)
	reserved := taskService.Reserve(ctx, worker, created.Value.ID).(task.ReservationCreated)
	if reserved.Value.State != task.ReservationStateActive {
		t.Fatalf("reservation state = %v, want active", reserved.Value.State)
	}

	submissionService.Submit(ctx, submission.SubmitCommand{
		TaskID: created.Value.ID, SubmitterID: worker.ID, ResponseSource: testResponseSource(t, `{"note":"done"}`),
	})

	listResult := taskService.ListReservations(ctx, owner, created.Value.ID)
	listed, matched := listResult.(task.ReservationsListed)
	if !matched {
		t.Fatalf("list reservations: want ReservationsListed, got %#v", listResult)
	}
	if len(listed.Values) != 1 || listed.Values[0].State != task.ReservationStateSubmitted {
		t.Fatalf("reservation after submit = %+v, want state=submitted", listed.Values)
	}
}

func TestSubmissionBrowserStoreListForTaskAndSubmitter(t *testing.T) {
	submissionService, taskService, _ := newSubmissionTestEnv(t)
	ctx := context.Background()
	owner := auth.UserSubject{ID: testUserID(t, "owner")}
	worker := testUserID(t, "worker")

	created := taskService.Create(ctx, testCreateCommand(t, owner.ID, task.NoRewardSpec{}, task.ParticipationPolicyOpen)).(task.TaskCreated)
	taskService.Open(ctx, owner, created.Value.ID)
	submissionService.Submit(ctx, submission.SubmitCommand{TaskID: created.Value.ID, SubmitterID: worker, ResponseSource: testResponseSource(t, `{"note":"done"}`)})

	taskListResult := submissionService.ListForTask(ctx, owner, created.Value.ID, testPage(t, 10, 0))
	taskListed, matched := taskListResult.(submission.SubmissionsListed)
	if !matched {
		t.Fatalf("list for task: want SubmissionsListed, got %#v", taskListResult)
	}
	if len(taskListed.Values) != 1 {
		t.Fatalf("submissions for task = %+v, want 1", taskListed.Values)
	}

	submitterListResult := submissionService.ListForSubmitter(ctx, auth.UserSubject{ID: worker}, worker, testPage(t, 10, 0))
	submitterListed, matched := submitterListResult.(submission.SubmissionsListed)
	if !matched {
		t.Fatalf("list for submitter: want SubmissionsListed, got %#v", submitterListResult)
	}
	if len(submitterListed.Values) != 1 {
		t.Fatalf("submissions for submitter = %+v, want 1", submitterListed.Values)
	}
}

func TestSubmissionBrowserStoreComments(t *testing.T) {
	submissionService, taskService, _ := newSubmissionTestEnv(t)
	ctx := context.Background()
	owner := auth.UserSubject{ID: testUserID(t, "owner")}
	worker := auth.UserSubject{ID: testUserID(t, "worker")}

	created := taskService.Create(ctx, testCreateCommand(t, owner.ID, task.NoRewardSpec{}, task.ParticipationPolicyOpen)).(task.TaskCreated)
	taskService.Open(ctx, owner, created.Value.ID)
	submitted := submissionService.Submit(ctx, submission.SubmitCommand{TaskID: created.Value.ID, SubmitterID: worker.ID, ResponseSource: testResponseSource(t, `{"note":"done"}`)}).(submission.SubmissionCreated)

	bodyResult := task.NewCommentBody("Looks good, one question.")
	body := bodyResult.(task.CommentBodyAccepted).Value

	addResult := submissionService.AddSubmissionComment(ctx, owner, submitted.Value.ID, body)
	added, matched := addResult.(submission.SubmissionCommentAdded)
	if !matched {
		t.Fatalf("add comment: want SubmissionCommentAdded, got %#v", addResult)
	}
	if added.TaskID != created.Value.ID || added.SubmitterID != worker.ID {
		t.Fatalf("comment context = %+v, want task=%v submitter=%v", added, created.Value.ID, worker.ID)
	}

	listResult := submissionService.ListSubmissionComments(ctx, worker, submitted.Value.ID)
	listed, matched := listResult.(submission.SubmissionCommentsListed)
	if !matched {
		t.Fatalf("list comments: want SubmissionCommentsListed, got %#v", listResult)
	}
	if len(listed.Values) != 1 || listed.Values[0].Body.String() != "Looks good, one question." {
		t.Fatalf("listed comments = %+v, want the added comment", listed.Values)
	}
}

func TestSubmissionBrowserStoreListCommentsRejectsOutsider(t *testing.T) {
	submissionService, taskService, _ := newSubmissionTestEnv(t)
	ctx := context.Background()
	owner := auth.UserSubject{ID: testUserID(t, "owner")}
	worker := testUserID(t, "worker")
	outsider := auth.UserSubject{ID: testUserID(t, "outsider")}

	created := taskService.Create(ctx, testCreateCommand(t, owner.ID, task.NoRewardSpec{}, task.ParticipationPolicyOpen)).(task.TaskCreated)
	taskService.Open(ctx, owner, created.Value.ID)
	submitted := submissionService.Submit(ctx, submission.SubmitCommand{TaskID: created.Value.ID, SubmitterID: worker, ResponseSource: testResponseSource(t, `{"note":"done"}`)}).(submission.SubmissionCreated)

	result := submissionService.ListSubmissionComments(ctx, outsider, submitted.Value.ID)
	if _, matched := result.(submission.SubmissionCommentsListRejected); !matched {
		t.Fatalf("outsider listing comments: want SubmissionCommentsListRejected, got %#v", result)
	}
}
