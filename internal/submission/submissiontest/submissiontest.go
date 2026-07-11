// Package submissiontest holds test-support helpers for submission.Submission,
// shared by the submission bridge's codec tests and the integration dual-run
// test so the two do not carry duplicate comparisons.
package submissiontest

import (
	"fmt"

	"github.com/e6qu/sharecrop/internal/attachment"
	"github.com/e6qu/sharecrop/internal/submission"
)

// SubmissionDiff returns a description of the first field in which got and want
// differ, or "" if they are equal. It compares every field the store persists,
// so a bridge that drops or garbles one is caught.
func SubmissionDiff(got, want submission.Submission) string {
	switch {
	case got.ID != want.ID:
		return fmt.Sprintf("id: %s != %s", got.ID, want.ID)
	case got.TaskID != want.TaskID:
		return fmt.Sprintf("task_id: %s != %s", got.TaskID, want.TaskID)
	case got.SubmitterID != want.SubmitterID:
		return fmt.Sprintf("submitter_id: %s != %s", got.SubmitterID, want.SubmitterID)
	case got.State.String() != want.State.String():
		return fmt.Sprintf("state: %s != %s", got.State, want.State)
	case got.ResponseSource.String() != want.ResponseSource.String():
		return fmt.Sprintf("response_source: %s != %s", got.ResponseSource, want.ResponseSource)
	case got.ReviewNote.String() != want.ReviewNote.String():
		return fmt.Sprintf("review_note: %s != %s", got.ReviewNote, want.ReviewNote)
	}
	if diff := attachmentsDiff(got.Attachments, want.Attachments); diff != "" {
		return diff
	}
	if diff := validationDiff(got.Validation, want.Validation); diff != "" {
		return diff
	}
	return sensitiveFieldsDiff(got.SensitiveFields, want.SensitiveFields)
}

// CommentDiff returns a description of the first field in which two submission
// comments differ, or "" if they are equal.
func CommentDiff(got, want submission.SubmissionComment) string {
	switch {
	case got.ID != want.ID:
		return fmt.Sprintf("id: %s != %s", got.ID, want.ID)
	case got.SubmissionID != want.SubmissionID:
		return fmt.Sprintf("submission_id: %s != %s", got.SubmissionID, want.SubmissionID)
	case got.AuthorID != want.AuthorID:
		return fmt.Sprintf("author_id: %s != %s", got.AuthorID, want.AuthorID)
	case got.Body.String() != want.Body.String():
		return fmt.Sprintf("body: %s != %s", got.Body, want.Body)
	case !got.CreatedAt.Equal(want.CreatedAt):
		return fmt.Sprintf("created_at: %s != %s", got.CreatedAt, want.CreatedAt)
	default:
		return ""
	}
}

func attachmentsDiff(got, want []attachment.Attachment) string {
	if len(got) != len(want) {
		return fmt.Sprintf("attachments: length %d != %d", len(got), len(want))
	}
	for index := range want {
		if got[index].Name.String() != want[index].Name.String() {
			return fmt.Sprintf("attachment %d name: %s != %s", index, got[index].Name, want[index].Name)
		}
		if got[index].ContentType.String() != want[index].ContentType.String() {
			return fmt.Sprintf("attachment %d content_type: %s != %s", index, got[index].ContentType, want[index].ContentType)
		}
		if string(got[index].Content.Bytes()) != string(want[index].Content.Bytes()) {
			return fmt.Sprintf("attachment %d content differs", index)
		}
	}
	return ""
}

func validationDiff(got, want submission.ValidationOutcome) string {
	_, gotPassed := got.(submission.ValidationPassed)
	_, wantPassed := want.(submission.ValidationPassed)
	if gotPassed != wantPassed {
		return fmt.Sprintf("validation: passed %t != %t", gotPassed, wantPassed)
	}
	gotFailed, gotHasErrors := got.(submission.ValidationFailed)
	wantFailed, wantHasErrors := want.(submission.ValidationFailed)
	if gotHasErrors != wantHasErrors {
		return fmt.Sprintf("validation: failed %t != %t", gotHasErrors, wantHasErrors)
	}
	if !gotHasErrors {
		return ""
	}
	if len(gotFailed.Errors) != len(wantFailed.Errors) {
		return fmt.Sprintf("validation errors: length %d != %d", len(gotFailed.Errors), len(wantFailed.Errors))
	}
	for index := range wantFailed.Errors {
		if gotFailed.Errors[index] != wantFailed.Errors[index] {
			return fmt.Sprintf("validation error %d: %+v != %+v", index, gotFailed.Errors[index], wantFailed.Errors[index])
		}
	}
	return ""
}

func sensitiveFieldsDiff(got, want []submission.SensitiveField) string {
	if len(got) != len(want) {
		return fmt.Sprintf("sensitive_fields: length %d != %d", len(got), len(want))
	}
	for index := range want {
		if got[index] != want[index] {
			return fmt.Sprintf("sensitive_field %d: %+v != %+v", index, got[index], want[index])
		}
	}
	return ""
}
