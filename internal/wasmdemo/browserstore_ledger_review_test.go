package wasmdemo

import (
	"context"
	"testing"

	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/ledger"
	"github.com/e6qu/sharecrop/internal/submission"
	"github.com/e6qu/sharecrop/internal/task"
)

func newReviewTestEnv(t *testing.T) (task.Service, ledger.Service, submission.Service, BrowserStorage) {
	t.Helper()
	storage := newTestBrowserStorage()
	ids := &counterLedgerIDs{}
	taskStore := NewTaskBrowserStore(storage, ids)
	taskService := task.NewService(taskStore, noopOrganizationPermissions{}, nil)
	ledgerService := ledger.NewService(NewLedgerBrowserStore(storage, ids))
	submissionService := submission.NewService(NewSubmissionBrowserStore(storage, ids), taskStore, noopOrganizationPermissions{})
	return taskService, ledgerService, submissionService, storage
}

func TestLedgerBrowserStoreAcceptSubmissionPaysCreditAndTip(t *testing.T) {
	taskService, ledgerService, submissionService, _ := newReviewTestEnv(t)
	ctx := context.Background()
	owner := testUserID(t, "owner")
	ownerSubject := auth.UserSubject{ID: owner}
	worker := testUserID(t, "worker")

	rewardResult := task.NewCreditRewardAmount(30)
	reward := rewardResult.(task.CreditRewardAmountAccepted).Value
	created := taskService.Create(ctx, testCreateCommand(t, owner, task.CreditRewardSpec{Amount: reward}, task.ParticipationPolicyOpen)).(task.TaskCreated)
	ledgerService.FundTask(ctx, owner, created.Value.ID, testCreditAmount(t, 30), testIdempotencyKey(t, "fund-1"))
	taskService.Open(ctx, ownerSubject, created.Value.ID)

	submitted := submissionService.Submit(ctx, submission.SubmitCommand{TaskID: created.Value.ID, SubmitterID: worker, ResponseSource: testResponseSource(t, `{"note":"done"}`)}).(submission.SubmissionCreated)

	acceptResult := ledgerService.ReviewAcceptSubmission(ctx, owner, created.Value.ID, submitted.Value.ID, testIdempotencyKey(t, "accept-1"),
		ledger.FullCreditReviewSelection{}, ledger.CreditTipSelection{Amount: testCreditAmount(t, 5)}, ledger.NoCollectibleTipSelection{})
	accepted, matched := acceptResult.(ledger.SubmissionAccepted)
	if !matched {
		t.Fatalf("accept submission: want SubmissionAccepted, got %#v", acceptResult)
	}
	payout, payoutMatched := accepted.Payout.(ledger.CreditPayout)
	if !payoutMatched || payout.Amount.Int64() != 30 || payout.WorkerUserID != worker {
		t.Fatalf("accept payout = %+v, want credit payout of 30 to worker", accepted.Payout)
	}
	tip, tipMatched := accepted.Tip.(ledger.CreditTip)
	if !tipMatched || tip.Amount.Int64() != 5 {
		t.Fatalf("accept tip = %+v, want credit tip of 5", accepted.Tip)
	}

	workerBalance := ledgerService.Balance(ctx, worker).(ledger.BalanceFound)
	// +100 baseline quirk (see browserstore_ledger_test.go) + 30 payout + 5 tip.
	if workerBalance.Value.Int64() != 135 {
		t.Fatalf("worker balance after accept = %d, want 135", workerBalance.Value.Int64())
	}

	taskAfter := taskService.Get(ctx, ownerSubject, created.Value.ID).(task.TaskGot)
	if taskAfter.Value.State != task.StateClosed {
		t.Fatalf("task state after accept = %v, want closed", taskAfter.Value.State)
	}
}

func TestLedgerBrowserStoreAcceptSubmissionIdempotentReplay(t *testing.T) {
	taskService, ledgerService, submissionService, _ := newReviewTestEnv(t)
	ctx := context.Background()
	owner := testUserID(t, "owner")
	ownerSubject := auth.UserSubject{ID: owner}
	worker := testUserID(t, "worker")

	created := taskService.Create(ctx, testCreateCommand(t, owner, task.NoRewardSpec{}, task.ParticipationPolicyOpen)).(task.TaskCreated)
	taskService.Open(ctx, ownerSubject, created.Value.ID)
	submitted := submissionService.Submit(ctx, submission.SubmitCommand{TaskID: created.Value.ID, SubmitterID: worker, ResponseSource: testResponseSource(t, `{"note":"done"}`)}).(submission.SubmissionCreated)

	key := testIdempotencyKey(t, "accept-1")
	first := ledgerService.AcceptSubmission(ctx, owner, created.Value.ID, submitted.Value.ID, key)
	if _, matched := first.(ledger.SubmissionAccepted); !matched {
		t.Fatalf("first accept: want SubmissionAccepted, got %#v", first)
	}
	retry := ledgerService.AcceptSubmission(ctx, owner, created.Value.ID, submitted.Value.ID, key)
	if _, matched := retry.(ledger.SubmissionAccepted); !matched {
		t.Fatalf("retried accept with same idempotency key: want SubmissionAccepted (replayed), got %#v", retry)
	}

	differentKey := testIdempotencyKey(t, "accept-2")
	conflict := ledgerService.AcceptSubmission(ctx, owner, created.Value.ID, submitted.Value.ID, differentKey)
	if _, matched := conflict.(ledger.AcceptRejected); !matched {
		t.Fatalf("second genuine accept with different key: want AcceptRejected, got %#v", conflict)
	}
}

func TestLedgerBrowserStoreRequestChangesReactivatesReservation(t *testing.T) {
	taskService, ledgerService, submissionService, _ := newReviewTestEnv(t)
	ctx := context.Background()
	owner := testUserID(t, "owner")
	ownerSubject := auth.UserSubject{ID: owner}
	worker := auth.UserSubject{ID: testUserID(t, "worker")}

	created := taskService.Create(ctx, testCreateCommand(t, owner, task.NoRewardSpec{}, task.ParticipationPolicyReservationRequired)).(task.TaskCreated)
	taskService.Open(ctx, ownerSubject, created.Value.ID)
	taskService.Reserve(ctx, worker, created.Value.ID)
	submitted := submissionService.Submit(ctx, submission.SubmitCommand{TaskID: created.Value.ID, SubmitterID: worker.ID, ResponseSource: testResponseSource(t, `{"note":"done"}`)}).(submission.SubmissionCreated)

	note := submission.NewRequiredReviewNote("Please add more detail.").(submission.ReviewNoteAccepted).Value
	changesResult := ledgerService.RequestChanges(ctx, owner, created.Value.ID, submitted.Value.ID, note)
	changed, matched := changesResult.(ledger.ChangesRequested)
	if !matched {
		t.Fatalf("request changes: want ChangesRequested, got %#v", changesResult)
	}
	if changed.ReviewNote != "Please add more detail." {
		t.Fatalf("review note = %q, want %q", changed.ReviewNote, "Please add more detail.")
	}

	reservations := taskService.ListReservations(ctx, ownerSubject, created.Value.ID).(task.ReservationsListed)
	if len(reservations.Values) != 1 || reservations.Values[0].State != task.ReservationStateActive {
		t.Fatalf("reservation after request-changes = %+v, want state=active", reservations.Values)
	}
}

func TestLedgerBrowserStoreRejectSubmissionCancelsReservation(t *testing.T) {
	taskService, ledgerService, submissionService, _ := newReviewTestEnv(t)
	ctx := context.Background()
	owner := testUserID(t, "owner")
	ownerSubject := auth.UserSubject{ID: owner}
	worker := auth.UserSubject{ID: testUserID(t, "worker")}

	rewardResult := task.NewCreditRewardAmount(20)
	reward := rewardResult.(task.CreditRewardAmountAccepted).Value
	created := taskService.Create(ctx, testCreateCommand(t, owner, task.CreditRewardSpec{Amount: reward}, task.ParticipationPolicyReservationRequired)).(task.TaskCreated)
	ledgerService.FundTask(ctx, owner, created.Value.ID, testCreditAmount(t, 20), testIdempotencyKey(t, "fund-1"))
	taskService.Open(ctx, ownerSubject, created.Value.ID)
	taskService.Reserve(ctx, worker, created.Value.ID)
	submitted := submissionService.Submit(ctx, submission.SubmitCommand{TaskID: created.Value.ID, SubmitterID: worker.ID, ResponseSource: testResponseSource(t, `{"note":"done"}`)}).(submission.SubmissionCreated)

	note := submission.NewRequiredReviewNote("Not a good fit.").(submission.ReviewNoteAccepted).Value
	rejectResult := ledgerService.RejectSubmission(ctx, owner, created.Value.ID, submitted.Value.ID, testIdempotencyKey(t, "reject-1"), note,
		ledger.NoCreditReviewSelection{}, ledger.NoTipSelection{}, ledger.NoBanSelection{})
	rejected, matched := rejectResult.(ledger.SubmissionRejected)
	if !matched {
		t.Fatalf("reject submission: want SubmissionRejected, got %#v", rejectResult)
	}
	if _, matched := rejected.Payout.(ledger.NoPayout); !matched {
		t.Fatalf("reject payout with NoCreditReviewSelection = %#v, want NoPayout", rejected.Payout)
	}

	// Escrow was never released, so it can still be refunded to the owner.
	refundResult := ledgerService.RefundTask(ctx, owner, created.Value.ID, testIdempotencyKey(t, "refund-1"))
	if _, matched := refundResult.(ledger.TaskRefunded); !matched {
		t.Fatalf("refund after reject-with-no-payout: want TaskRefunded, got %#v", refundResult)
	}

	reservations := taskService.ListReservations(ctx, ownerSubject, created.Value.ID).(task.ReservationsListed)
	if len(reservations.Values) != 1 || reservations.Values[0].State != task.ReservationStateCancelledByRequester {
		t.Fatalf("reservation after reject = %+v, want state=cancelled_by_requester", reservations.Values)
	}
}
