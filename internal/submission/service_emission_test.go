package submission

import (
	"context"
	"testing"

	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/event"
	"github.com/e6qu/sharecrop/internal/event/eventtest"
	"github.com/e6qu/sharecrop/internal/task"
)

func submissionRecipientsContain(recipients event.Recipients, user core.UserID) bool {
	for _, value := range recipients.Users {
		if value == user {
			return true
		}
	}
	return false
}

func TestSubmitEmitsSubmissionCreatedToOwnerAndSubmitter(t *testing.T) {
	store := newSubmissionMemoryStore()
	taskStore := newSubmissionTaskStore(t, task.PublicVisibility{}, `{"kind":"freeform"}`)
	events := eventtest.NewCapturingStore()
	service := NewService(store, taskStore, submissionPermissionStore{}, eventtest.RecorderOver(events))
	command := testSubmitCommand(t, taskStore.value.ID, `{"answer":"done"}`)

	created, matched := service.Submit(context.Background(), command).(SubmissionCreated)
	if !matched {
		t.Fatalf("submit rejected")
	}

	appended := events.Appended()
	if len(appended) != 1 {
		t.Fatalf("emitted %d events, want 1", len(appended))
	}
	if appended[0].Kind != event.KindSubmissionCreated {
		t.Fatalf("emitted kind = %s, want submission_created", appended[0].Kind.String())
	}
	subjectSubmission, matched := appended[0].Subject.Submission.(event.SubmissionSubject)
	if !matched || subjectSubmission.ID != created.Value.ID {
		t.Fatalf("submission subject did not carry the created submission")
	}
	if _, matched := appended[0].Subject.Task.(event.TaskSubject); !matched {
		t.Fatalf("submission_created event has no task subject")
	}
	recipients := events.RecipientsAt(0)
	if !submissionRecipientsContain(recipients, taskStore.value.CreatedBy) || !submissionRecipientsContain(recipients, command.SubmitterID) {
		t.Fatalf("submission_created recipients = %v, want owner and submitter", recipients.Users)
	}
}

func TestAddSubmissionCommentEmitsSubmissionCommented(t *testing.T) {
	store := newSubmissionMemoryStore()
	taskStore := newSubmissionTaskStore(t, task.PublicVisibility{}, `{"kind":"freeform"}`)
	events := eventtest.NewCapturingStore()
	service := NewService(store, taskStore, submissionPermissionStore{}, eventtest.RecorderOver(events))
	command := testSubmitCommand(t, taskStore.value.ID, `{"answer":"done"}`)
	created, matched := service.Submit(context.Background(), command).(SubmissionCreated)
	if !matched {
		t.Fatalf("submit rejected")
	}

	body, matched := task.NewCommentBody("How fresh should the data be?").(task.CommentBodyAccepted)
	if !matched {
		t.Fatalf("comment body rejected")
	}
	commenter := auth.UserSubject{ID: command.SubmitterID}
	if _, matched := service.AddSubmissionComment(context.Background(), commenter, created.Value.ID, body.Value).(SubmissionCommentAdded); !matched {
		t.Fatalf("comment rejected")
	}

	appended := events.Appended()
	last := appended[len(appended)-1]
	if last.Kind != event.KindSubmissionCommented {
		t.Fatalf("emitted kind = %s, want submission_commented", last.Kind.String())
	}
	recipients := events.RecipientsAt(len(appended) - 1)
	if !submissionRecipientsContain(recipients, taskStore.value.CreatedBy) || !submissionRecipientsContain(recipients, command.SubmitterID) {
		t.Fatalf("submission_commented recipients = %v, want owner and submitter", recipients.Users)
	}
}
