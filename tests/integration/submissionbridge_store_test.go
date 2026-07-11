//go:build integration

package integration_test

import (
	"context"
	"testing"

	"github.com/e6qu/sharecrop/internal/attachment"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/db"
	"github.com/e6qu/sharecrop/internal/submission"
	"github.com/e6qu/sharecrop/internal/submission/submissiontest"
	"github.com/e6qu/sharecrop/internal/task"
	"github.com/e6qu/sharecrop/internal/wasibridge/rpc"
	"github.com/e6qu/sharecrop/internal/wasibridge/submissionbridge"
)

// TestSubmissionBridgeDualRun exercises the submission store through both the
// direct-db path and the compiled wasip1 guest + host bridge: create (with an
// attachment and a []SensitiveField argument), find, find-by-receipt, list for
// task, list for submitter, and the submission comment thread.
func TestSubmissionBridgeDualRun(t *testing.T) {
	ctx := context.Background()
	pool := newPool(t)
	dbStore := db.NewSubmissionStore(pool)

	guestWASM, err := compileWASIGuest(t, "github.com/e6qu/sharecrop/cmd/sharecrop-wasi-store-guest")
	if err != nil {
		t.Fatalf("compile store guest: %v", err)
	}
	host, err := rpc.NewHost(ctx, guestWASM, func(ctx context.Context, method string, args []byte) ([]byte, error) {
		return submissionbridge.Dispatch(ctx, dbStore, method, args)
	})
	if err != nil {
		t.Fatalf("new host: %v", err)
	}
	t.Cleanup(func() { _ = host.Close(ctx) })
	bridgeStore := submissionbridge.NewGuestStore(func(method string, args []byte) ([]byte, error) {
		return host.Call(ctx, method, args)
	})

	owner := createUser(t, pool, "submission-owner")
	submitter := createUser(t, pool, "submission-submitter")
	taskID := insertTask(t, pool, owner, "open", 30)
	page := requirePage(t, 50, 0)

	submissionID := newSubmissionID(t)
	receiptID := newReceiptTokenID(t)
	receiptHash := newReceiptHash(t)
	command := submission.SubmitCommand{
		TaskID:         taskID,
		SubmitterID:    submitter,
		ResponseSource: newResponseSource(t, `{"answer":"42"}`),
		Attachments:    []attachment.Attachment{newAttachment(t)},
	}
	sensitiveFields := []submission.SensitiveField{{
		Path:      "/answer",
		Category:  "pii",
		Retention: "standard",
		Redaction: "replace",
		State:     "active",
	}}

	t.Run("create through the bridge, then find matches a direct call", func(t *testing.T) {
		result := bridgeStore.CreateSubmission(ctx, submissionID, receiptID, receiptHash, command, submission.StateSubmitted, submission.ValidationPassed{}, sensitiveFields)
		if rejected, matched := result.(submission.CreateSubmissionStoreRejected); matched {
			t.Fatalf("bridge CreateSubmission rejected: %s / %s", rejected.Reason.Code(), rejected.Reason.Description())
		}
		if _, matched := result.(submission.CreateSubmissionStoreAccepted); !matched {
			t.Fatalf("bridge CreateSubmission = %T, want accepted", result)
		}
		viaBridge := requireFound(t, bridgeStore.FindSubmission(ctx, submissionID))
		direct := requireFound(t, dbStore.FindSubmission(ctx, submissionID))
		if diff := submissiontest.SubmissionDiff(viaBridge, direct); diff != "" {
			t.Errorf("found submission mismatch: %s", diff)
		}
		if len(viaBridge.Attachments) != 1 {
			t.Errorf("bridge found %d attachments, want 1", len(viaBridge.Attachments))
		}
		if len(viaBridge.SensitiveFields) != 1 {
			t.Errorf("bridge found %d sensitive fields, want 1", len(viaBridge.SensitiveFields))
		}
	})

	t.Run("find by receipt token matches a direct call", func(t *testing.T) {
		viaBridge, matched := bridgeStore.FindByReceiptToken(ctx, receiptHash).(submission.ReceiptFound)
		if !matched {
			t.Fatalf("bridge FindByReceiptToken did not find the submission")
		}
		direct, matched := dbStore.FindByReceiptToken(ctx, receiptHash).(submission.ReceiptFound)
		if !matched {
			t.Fatalf("direct FindByReceiptToken did not find the submission")
		}
		if diff := submissiontest.SubmissionDiff(viaBridge.Value, direct.Value); diff != "" {
			t.Errorf("receipt submission mismatch: %s", diff)
		}
	})

	t.Run("list for task matches a direct call", func(t *testing.T) {
		viaBridge := requireListed(t, bridgeStore.ListForTask(ctx, taskID, page))
		direct := requireListed(t, dbStore.ListForTask(ctx, taskID, page))
		assertSubmissionSetsEqual(t, viaBridge, direct)
	})

	t.Run("list for submitter matches a direct call", func(t *testing.T) {
		viaBridge := requireListed(t, bridgeStore.ListForSubmitter(ctx, submitter, page))
		direct := requireListed(t, dbStore.ListForSubmitter(ctx, submitter, page))
		assertSubmissionSetsEqual(t, viaBridge, direct)
	})

	t.Run("comment through the bridge, then list matches a direct call", func(t *testing.T) {
		comment := submission.SubmissionComment{
			ID:           newSubmissionCommentID(t),
			SubmissionID: submissionID,
			AuthorID:     submitter,
			Body:         newCommentBody(t, "please clarify step 2"),
		}
		if _, matched := bridgeStore.CreateSubmissionComment(ctx, comment).(submission.CreateSubmissionCommentStoreAccepted); !matched {
			t.Fatalf("bridge CreateSubmissionComment did not accept")
		}
		viaBridge := requireCommentsListed(t, bridgeStore.ListSubmissionComments(ctx, submissionID))
		direct := requireCommentsListed(t, dbStore.ListSubmissionComments(ctx, submissionID))
		if len(viaBridge) != len(direct) || len(viaBridge) != 1 {
			t.Fatalf("comment counts: bridge %d, direct %d, want 1", len(viaBridge), len(direct))
		}
		if diff := submissiontest.CommentDiff(viaBridge[0], direct[0]); diff != "" {
			t.Errorf("comment mismatch: %s", diff)
		}
	})
}

func requireFound(t *testing.T, result submission.FindSubmissionStoreResult) submission.Submission {
	t.Helper()
	found, matched := result.(submission.FindSubmissionStoreAccepted)
	if !matched {
		t.Fatalf("find result = %T, want accepted", result)
	}
	return found.Value
}

func requireListed(t *testing.T, result submission.ListSubmissionsStoreResult) []submission.Submission {
	t.Helper()
	listed, matched := result.(submission.ListSubmissionsStoreAccepted)
	if !matched {
		t.Fatalf("list result = %T, want accepted", result)
	}
	return listed.Values
}

func requireCommentsListed(t *testing.T, result submission.ListSubmissionCommentsStoreResult) []submission.SubmissionComment {
	t.Helper()
	listed, matched := result.(submission.ListSubmissionCommentsStoreAccepted)
	if !matched {
		t.Fatalf("comments result = %T, want accepted", result)
	}
	return listed.Values
}

func assertSubmissionSetsEqual(t *testing.T, got, want []submission.Submission) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("submission counts: bridge %d, direct %d", len(got), len(want))
	}
	for index := range want {
		if diff := submissiontest.SubmissionDiff(got[index], want[index]); diff != "" {
			t.Errorf("submission %d mismatch: %s", index, diff)
		}
	}
}

func newReceiptTokenID(t *testing.T) core.SubmissionReceiptTokenID {
	t.Helper()
	created, matched := core.NewSubmissionReceiptTokenID().(core.SubmissionReceiptTokenIDCreated)
	if !matched {
		t.Fatalf("receipt token id rejected")
	}
	return created.Value
}

func newSubmissionCommentID(t *testing.T) core.SubmissionCommentID {
	t.Helper()
	created, matched := core.NewSubmissionCommentID().(core.SubmissionCommentIDCreated)
	if !matched {
		t.Fatalf("submission comment id rejected")
	}
	return created.Value
}

func newReceiptHash(t *testing.T) submission.ReceiptTokenHash {
	t.Helper()
	plain, matched := submission.NewReceiptTokenPlain().(submission.ReceiptTokenPlainAccepted)
	if !matched {
		t.Fatalf("receipt token plain rejected")
	}
	return plain.Value.Hash()
}

func newResponseSource(t *testing.T, raw string) submission.ResponseSource {
	t.Helper()
	accepted, matched := submission.NewResponseSource(raw).(submission.ResponseSourceAccepted)
	if !matched {
		t.Fatalf("response source rejected")
	}
	return accepted.Value
}

func newCommentBody(t *testing.T, raw string) task.CommentBody {
	t.Helper()
	accepted, matched := task.NewCommentBody(raw).(task.CommentBodyAccepted)
	if !matched {
		t.Fatalf("comment body rejected")
	}
	return accepted.Value
}

func newAttachment(t *testing.T) attachment.Attachment {
	t.Helper()
	accepted, matched := attachment.NewStoredAttachment("proof.txt", "text/plain", []byte("hello world")).(attachment.AttachmentAccepted)
	if !matched {
		t.Fatalf("attachment rejected")
	}
	return accepted.Value
}
