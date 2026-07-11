package submissionbridge

import (
	"testing"

	"github.com/e6qu/sharecrop/internal/attachment"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/submission"
	"github.com/e6qu/sharecrop/internal/submission/submissiontest"
	"github.com/e6qu/sharecrop/internal/task"
)

func newUserID(t *testing.T) core.UserID {
	t.Helper()
	created, matched := core.NewUserID().(core.UserIDCreated)
	if !matched {
		t.Fatalf("user id rejected")
	}
	return created.Value
}

func newTaskID(t *testing.T) core.TaskID {
	t.Helper()
	created, matched := core.NewTaskID().(core.TaskIDCreated)
	if !matched {
		t.Fatalf("task id rejected")
	}
	return created.Value
}

func sampleSubmission(t *testing.T) submission.Submission {
	t.Helper()
	submissionID, matched := core.NewSubmissionID().(core.SubmissionIDCreated)
	if !matched {
		t.Fatalf("submission id rejected")
	}
	source, matched := submission.NewResponseSource(`{"answer":"yes"}`).(submission.ResponseSourceAccepted)
	if !matched {
		t.Fatalf("response source rejected")
	}
	note, matched := submission.NewRequiredReviewNote("looks good").(submission.ReviewNoteAccepted)
	if !matched {
		t.Fatalf("review note rejected")
	}
	att, matched := attachment.NewStoredAttachment("proof.txt", "text/plain", []byte("hello world")).(attachment.AttachmentAccepted)
	if !matched {
		t.Fatalf("attachment rejected")
	}
	return submission.Submission{
		ID:             submissionID.Value,
		TaskID:         newTaskID(t),
		SubmitterID:    newUserID(t),
		State:          submission.StateChangesRequested,
		ResponseSource: source.Value,
		Attachments:    []attachment.Attachment{att.Value},
		Validation: submission.ValidationFailed{Errors: []submission.ValidationError{
			{Path: "/answer", Message: "must be a number"},
		}},
		SensitiveFields: []submission.SensitiveField{{
			Path:       "/ssn",
			Category:   "pii",
			Retention:  "30d",
			Redaction:  "mask",
			State:      "pending",
			RedactedAt: "",
		}},
		ReviewNote: note.Value,
	}
}

func TestSubmissionRoundTrip(t *testing.T) {
	original := sampleSubmission(t)
	restored, err := decodeSubmission(encodeSubmission(original))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if diff := submissiontest.SubmissionDiff(restored, original); diff != "" {
		t.Errorf("submission mismatch: %s", diff)
	}
}

func TestValidationPassedRoundTrip(t *testing.T) {
	restored, err := decodeValidationOutcome(encodeValidationOutcome(submission.ValidationPassed{}))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if _, matched := restored.(submission.ValidationPassed); !matched {
		t.Errorf("validation passed did not round-trip, got %T", restored)
	}
}

func TestSubmissionCommentRoundTrip(t *testing.T) {
	commentID, matched := core.NewSubmissionCommentID().(core.SubmissionCommentIDCreated)
	if !matched {
		t.Fatalf("comment id rejected")
	}
	submissionID, matched := core.NewSubmissionID().(core.SubmissionIDCreated)
	if !matched {
		t.Fatalf("submission id rejected")
	}
	body, matched := task.NewCommentBody("please clarify step 2").(task.CommentBodyAccepted)
	if !matched {
		t.Fatalf("comment body rejected")
	}
	original := submission.SubmissionComment{
		ID:           commentID.Value,
		SubmissionID: submissionID.Value,
		AuthorID:     newUserID(t),
		Body:         body.Value,
	}
	restored, err := decodeSubmissionComment(encodeSubmissionComment(original))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if diff := submissiontest.CommentDiff(restored, original); diff != "" {
		t.Errorf("comment mismatch: %s", diff)
	}
}

func TestSubmissionResultRoundTrips(t *testing.T) {
	value := sampleSubmission(t)

	created, err := decodeCreateSubmissionResult(encodeCreateSubmissionResult(submission.CreateSubmissionStoreAccepted{Value: value}))
	if err != nil {
		t.Fatalf("decode create: %v", err)
	}
	accepted, matched := created.(submission.CreateSubmissionStoreAccepted)
	if !matched {
		t.Fatalf("create result = %T, want accepted", created)
	}
	if diff := submissiontest.SubmissionDiff(accepted.Value, value); diff != "" {
		t.Errorf("created submission mismatch: %s", diff)
	}

	listed, err := decodeListSubmissionsResult(encodeListSubmissionsResult(submission.ListSubmissionsStoreAccepted{Values: []submission.Submission{value}}))
	if err != nil {
		t.Fatalf("decode list: %v", err)
	}
	values, matched := listed.(submission.ListSubmissionsStoreAccepted)
	if !matched {
		t.Fatalf("list result = %T, want accepted", listed)
	}
	if len(values.Values) != 1 {
		t.Fatalf("listed %d submissions, want 1", len(values.Values))
	}
	if diff := submissiontest.SubmissionDiff(values.Values[0], value); diff != "" {
		t.Errorf("listed submission mismatch: %s", diff)
	}
}
