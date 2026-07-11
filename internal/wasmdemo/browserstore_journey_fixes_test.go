package wasmdemo

// Tests for the browser-store fixes from the journey correctness review:
// db-parity for collectible reward counts, reservation expiry and the
// one-active-reservation rule, implementor bans, ledger idempotency
// semantics, pending-review refund/cancel guards, deactivation guards, and
// series bookkeeping. Each test pins a behavior the Postgres store already
// had and the browser store was missing or diverging on.

import (
	"context"
	"testing"
	"time"

	"github.com/e6qu/sharecrop/internal/assets"
	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/ledger"
	"github.com/e6qu/sharecrop/internal/org"
	"github.com/e6qu/sharecrop/internal/submission"
	"github.com/e6qu/sharecrop/internal/task"
)

// newFundedSubmittedCreditTask builds the shared preface of the pending-
// review guard and replay tests: a credit-reward task funded with amount,
// opened, and holding one submitted (pending-review) submission from worker.
func newFundedSubmittedCreditTask(t *testing.T, taskService task.Service, ledgerService ledger.Service, submissionService submission.Service, owner core.UserID, worker core.UserID, amount int64) (core.TaskID, core.SubmissionID) {
	t.Helper()
	ctx := context.Background()
	reward := task.NewCreditRewardAmount(amount).(task.CreditRewardAmountAccepted).Value
	created := taskService.Create(ctx, testCreateCommand(t, owner, task.CreditRewardSpec{Amount: reward}, task.ParticipationPolicyOpen)).(task.TaskCreated)
	if _, matched := ledgerService.FundTask(ctx, owner, created.Value.ID, testCreditAmount(t, amount), testIdempotencyKey(t, "fund-1")).(ledger.TaskFunded); !matched {
		t.Fatalf("fund task failed")
	}
	if _, matched := taskService.Open(ctx, auth.UserSubject{ID: owner}, created.Value.ID).(task.TaskStateChanged); !matched {
		t.Fatalf("open task failed")
	}
	submitted := submissionService.Submit(ctx, submission.SubmitCommand{TaskID: created.Value.ID, SubmitterID: worker, ResponseSource: testResponseSource(t, `{"note":"done"}`)}).(submission.SubmissionCreated)
	return created.Value.ID, submitted.Value.ID
}

func testCollectibleRewardSpec(t *testing.T, count int) task.CollectibleRewardSpec {
	t.Helper()
	result := task.NewCollectibleRewardCount(count)
	accepted, matched := result.(task.CollectibleRewardCountAccepted)
	if !matched {
		t.Fatalf("new collectible reward count %d failed", count)
	}
	return task.CollectibleRewardSpec{Count: accepted.Value}
}

func TestTaskBrowserStoreFindTaskUnknownIsNotFound(t *testing.T) {
	storage := newTestBrowserStorage()
	store := NewTaskBrowserStore(storage, &counterLedgerIDs{}, systemTestClock{})

	unknown := core.NewTaskID().(core.TaskIDCreated).Value
	result := store.FindTask(context.Background(), unknown)
	rejected, matched := result.(task.FindTaskStoreRejected)
	if !matched {
		t.Fatalf("find unknown task: want FindTaskStoreRejected, got %#v", result)
	}
	if rejected.Reason.Code() != core.ErrorCodeNotFound {
		t.Fatalf("find unknown task code = %v, want not_found", rejected.Reason.Code())
	}
}

func TestTaskBrowserStoreOpenCollectibleRewardRequiresFunding(t *testing.T) {
	storage := newTestBrowserStorage()
	ids := &counterLedgerIDs{}
	taskService := task.NewService(NewTaskBrowserStore(storage, ids, systemTestClock{}), noopOrganizationPermissions{}, nil)
	assetService := assets.NewService(NewAssetBrowserStore(storage, ids))
	ctx := context.Background()
	owner := testUserID(t, "owner")
	ownerSubject := auth.UserSubject{ID: owner}

	created := taskService.Create(ctx, testCreateCommand(t, owner, testCollectibleRewardSpec(t, 1), task.ParticipationPolicyOpen)).(task.TaskCreated)

	openBefore := taskService.Open(ctx, ownerSubject, created.Value.ID)
	if _, matched := openBefore.(task.ChangeStateRejected); !matched {
		t.Fatalf("open unfunded collectible-reward task: want ChangeStateRejected, got %#v", openBefore)
	}

	minted := assetService.Mint(ctx, assets.CollectibleOwnerKindUser, owner.String(), "", testCollectibleName(t, "Golden Badge"), assets.CollectibleKindBadge, assets.TransferPolicyTransferableBetweenUsers, "art-1").(assets.CollectibleMinted)
	fundResult := assetService.FundReward(ctx, owner, created.Value.ID, minted.Value.ID)
	if _, matched := fundResult.(assets.RewardFunded); !matched {
		t.Fatalf("fund collectible reward: want RewardFunded, got %#v", fundResult)
	}

	openAfter := taskService.Open(ctx, ownerSubject, created.Value.ID)
	if _, matched := openAfter.(task.TaskStateChanged); !matched {
		t.Fatalf("open funded collectible-reward task: want TaskStateChanged, got %#v", openAfter)
	}
}

func TestTaskBrowserStoreCancelSettlesHeldCollectible(t *testing.T) {
	storage := newTestBrowserStorage()
	ids := &counterLedgerIDs{}
	taskService := task.NewService(NewTaskBrowserStore(storage, ids, systemTestClock{}), noopOrganizationPermissions{}, nil)
	assetService := assets.NewService(NewAssetBrowserStore(storage, ids))
	ctx := context.Background()
	owner := testUserID(t, "owner")
	ownerSubject := auth.UserSubject{ID: owner}

	created := taskService.Create(ctx, testCreateCommand(t, owner, testCollectibleRewardSpec(t, 1), task.ParticipationPolicyOpen)).(task.TaskCreated)
	minted := assetService.Mint(ctx, assets.CollectibleOwnerKindUser, owner.String(), "", testCollectibleName(t, "Golden Badge"), assets.CollectibleKindBadge, assets.TransferPolicyTransferableBetweenUsers, "art-1").(assets.CollectibleMinted)
	assetService.FundReward(ctx, owner, created.Value.ID, minted.Value.ID)
	taskService.Open(ctx, ownerSubject, created.Value.ID)

	// Cancelling a funded task settles it: the held collectible returns to its
	// funder (minted) before the task is cancelled.
	cancelResult := taskService.Cancel(ctx, ownerSubject, created.Value.ID)
	if _, matched := cancelResult.(task.TaskStateChanged); !matched {
		t.Fatalf("cancel with held collectible reward: want TaskStateChanged, got %#v", cancelResult)
	}
	collectible, found, _ := (AssetBrowserStore{storage: storage, ids: ids}).loadCollectible(minted.Value.ID.String())
	if !found || collectible.State != assets.CollectibleStateMinted.String() || collectible.OwnerID != owner.String() {
		t.Fatalf("collectible after cancel = %+v, want minted owned by funder", collectible)
	}
	held, _ := countHeldCollectibleRewards(storage, created.Value.ID.String())
	if held != 0 {
		t.Fatalf("held collectible rewards after cancel = %d, want 0", held)
	}
}

func TestTaskBrowserStoreReservationsExpire(t *testing.T) {
	storage := newTestBrowserStorage()
	ids := &counterLedgerIDs{}
	clock := &adjustableTestClock{now: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	taskService := task.NewService(NewTaskBrowserStore(storage, ids, clock), noopOrganizationPermissions{}, nil)
	ctx := context.Background()
	owner := auth.UserSubject{ID: testUserID(t, "owner")}
	worker := auth.UserSubject{ID: testUserID(t, "worker")}

	created := taskService.Create(ctx, testCreateCommand(t, owner.ID, task.NoRewardSpec{}, task.ParticipationPolicyReservationRequired)).(task.TaskCreated)
	taskService.Open(ctx, owner, created.Value.ID)
	reserved := taskService.Reserve(ctx, worker, created.Value.ID).(task.ReservationCreated)
	if reserved.Value.State != task.ReservationStateActive {
		t.Fatalf("reservation state = %v, want active", reserved.Value.State)
	}

	// The default reservation TTL is 48 hours; past it, the reservation
	// expires on the next reservation-observing call, like the real store's
	// expireReservationsSQL sweep.
	clock.now = clock.now.Add(49 * time.Hour)

	reservations := taskService.ListReservations(ctx, owner, created.Value.ID).(task.ReservationsListed)
	if len(reservations.Values) != 1 || reservations.Values[0].State != task.ReservationStateExpired {
		t.Fatalf("reservations after TTL = %+v, want one expired", reservations.Values)
	}

	eligibility := taskService.CheckSubmissionEligibility(ctx, created.Value.ID, worker.ID)
	if _, matched := eligibility.(task.SubmissionEligibilityRejected); !matched {
		t.Fatalf("submission eligibility after reservation expiry: want SubmissionEligibilityRejected, got %#v", eligibility)
	}
}

func TestTaskBrowserStoreSecondActiveReservationRejected(t *testing.T) {
	taskService, _, _, _ := newTaskTestEnv(t)
	ctx := context.Background()
	owner := auth.UserSubject{ID: testUserID(t, "owner")}
	firstWorker := auth.UserSubject{ID: testUserID(t, "first-worker")}
	secondWorker := auth.UserSubject{ID: testUserID(t, "second-worker")}

	created := taskService.Create(ctx, testCreateCommand(t, owner.ID, task.NoRewardSpec{}, task.ParticipationPolicyReservationRequired)).(task.TaskCreated)
	taskService.Open(ctx, owner, created.Value.ID)
	if _, matched := taskService.Reserve(ctx, firstWorker, created.Value.ID).(task.ReservationCreated); !matched {
		t.Fatalf("first reservation failed")
	}

	secondResult := taskService.Reserve(ctx, secondWorker, created.Value.ID)
	rejected, matched := secondResult.(task.ReservationRejected)
	if !matched {
		t.Fatalf("second active reservation: want ReservationRejected, got %#v", secondResult)
	}
	if rejected.Reason.Description() != "task already has an active reservation" {
		t.Fatalf("second reservation rejection = %q, want %q", rejected.Reason.Description(), "task already has an active reservation")
	}
}

func TestTaskBrowserStoreApproveSecondReservationRejectedWhileActive(t *testing.T) {
	taskService, _, _, _ := newTaskTestEnv(t)
	ctx := context.Background()
	owner := auth.UserSubject{ID: testUserID(t, "owner")}
	firstWorker := auth.UserSubject{ID: testUserID(t, "first-worker")}
	secondWorker := auth.UserSubject{ID: testUserID(t, "second-worker")}

	created := taskService.Create(ctx, testCreateCommand(t, owner.ID, task.NoRewardSpec{}, task.ParticipationPolicyApprovalRequired)).(task.TaskCreated)
	taskService.Open(ctx, owner, created.Value.ID)
	firstReserved := taskService.Reserve(ctx, firstWorker, created.Value.ID).(task.ReservationCreated)
	secondReserved := taskService.Reserve(ctx, secondWorker, created.Value.ID).(task.ReservationCreated)

	if _, matched := taskService.ApproveReservation(ctx, owner, created.Value.ID, firstReserved.Value.ID).(task.ReservationStateChanged); !matched {
		t.Fatalf("approve first reservation failed")
	}
	secondApproval := taskService.ApproveReservation(ctx, owner, created.Value.ID, secondReserved.Value.ID)
	if _, matched := secondApproval.(task.ReservationStateChangeRejected); !matched {
		t.Fatalf("approve second reservation while first is active: want ReservationStateChangeRejected, got %#v", secondApproval)
	}
}

func TestTaskBrowserStoreTeamListIncludesActivelyReservedTask(t *testing.T) {
	storage := newTestBrowserStorage()
	ids := &counterLedgerIDs{}
	taskService := task.NewService(NewTaskBrowserStore(storage, ids, systemTestClock{}), allowAllOrganizationPermissions{}, nil)
	ctx := context.Background()
	owner := auth.UserSubject{ID: testUserID(t, "owner")}
	reservingMember := auth.UserSubject{ID: testUserID(t, "reserving-member")}
	viewer := auth.UserSubject{ID: testUserID(t, "viewer")}
	teamID := core.NewTeamID().(core.TeamIDCreated).Value

	command := testCreateCommand(t, owner.ID, task.NoRewardSpec{}, task.ParticipationPolicyReservationRequired)
	command.AssigneeScope = task.AssigneeScopeTeam
	created := taskService.Create(ctx, command).(task.TaskCreated)
	taskService.Open(ctx, owner, created.Value.ID)

	// The task is public, not team-visible: before the team reserves it, the
	// team's queue does not include it.
	before := taskService.List(ctx, viewer, task.TeamListScope{TeamID: teamID}, task.NoListFilters(), testPage(t, 10, 0)).(task.TasksListed)
	if len(before.Values) != 0 {
		t.Fatalf("team list before reservation = %+v, want none", before.Values)
	}

	if _, matched := taskService.ReserveForTeam(ctx, reservingMember, created.Value.ID, teamID).(task.ReservationCreated); !matched {
		t.Fatalf("reserve for team failed")
	}

	after := taskService.List(ctx, viewer, task.TeamListScope{TeamID: teamID}, task.NoListFilters(), testPage(t, 10, 0)).(task.TasksListed)
	if len(after.Values) != 1 || after.Values[0].Task.ID != created.Value.ID {
		t.Fatalf("team list after reservation = %+v, want just the reserved task", after.Values)
	}
}

func TestTaskBrowserStoreCancelWithPendingReviewSucceeds(t *testing.T) {
	taskService, _, submissionService, _ := newReviewTestEnv(t)
	ctx := context.Background()
	owner := testUserID(t, "owner")
	ownerSubject := auth.UserSubject{ID: owner}
	worker := testUserID(t, "worker")

	created := taskService.Create(ctx, testCreateCommand(t, owner, task.NoRewardSpec{}, task.ParticipationPolicyOpen)).(task.TaskCreated)
	taskService.Open(ctx, ownerSubject, created.Value.ID)
	submissionService.Submit(ctx, submission.SubmitCommand{TaskID: created.Value.ID, SubmitterID: worker, ResponseSource: testResponseSource(t, `{"note":"done"}`)})

	// Cancelling is allowed even with a submission pending review; the task is
	// simply cancelled (unfunded here, so nothing to settle).
	cancelResult := taskService.Cancel(ctx, ownerSubject, created.Value.ID)
	if _, matched := cancelResult.(task.TaskStateChanged); !matched {
		t.Fatalf("cancel with pending review: want TaskStateChanged, got %#v", cancelResult)
	}
}

func TestLedgerBrowserStoreRefundWithPendingReviewSucceeds(t *testing.T) {
	taskService, ledgerService, submissionService, storage := newReviewTestEnv(t)
	ctx := context.Background()
	owner := testUserID(t, "owner")
	worker := testUserID(t, "worker")
	grantFunderBalance(t, storage, owner)

	taskID, _ := newFundedSubmittedCreditTask(t, taskService, ledgerService, submissionService, owner, worker, 30)

	// Refunds are auto-granted: the owner can refund even with a submission
	// pending review. The allocated credits return to the owner and the worker
	// is not paid.
	refundResult := ledgerService.RefundTask(ctx, owner, taskID, testIdempotencyKey(t, "refund-1"))
	if _, matched := refundResult.(ledger.TaskRefunded); !matched {
		t.Fatalf("refund with pending review: want TaskRefunded, got %#v", refundResult)
	}
	ownerBalance := ledgerService.Balance(ctx, owner).(ledger.BalanceFound)
	if ownerBalance.Value.Spendable() != 100 || ownerBalance.Value.Allocated() != 0 {
		t.Fatalf("owner balance after refund = (spendable %d, allocated %d), want (100, 0)", ownerBalance.Value.Spendable(), ownerBalance.Value.Allocated())
	}
	workerBalance := ledgerService.Balance(ctx, worker).(ledger.BalanceFound)
	if workerBalance.Value.Spendable() != 0 {
		t.Fatalf("worker balance after refund = %d, want 0 (not paid)", workerBalance.Value.Spendable())
	}
}

func TestLedgerBrowserStoreFundKeyReusedForDifferentTaskRejected(t *testing.T) {
	store, storage, funder, taskID, ctx := newFundableTaskTestEnv(t, "funder", true)

	key := testIdempotencyKey(t, "fund-1")
	first := store.FundTask(ctx, ledger.FundStoreCommand{
		EntryID: core.NewLedgerEntryID().(core.LedgerEntryIDCreated).Value, FunderUserID: funder, TaskID: taskID,
		Amount: testCreditAmount(t, 30), IdempotencyKey: key,
	})
	if _, matched := first.(ledger.TaskFunded); !matched {
		t.Fatalf("first fund: want TaskFunded, got %#v", first)
	}

	otherTaskID := core.NewTaskID().(core.TaskIDCreated).Value
	seedDraftTask(t, storage, otherTaskID.String(), funder.String(), "none", 0)
	reuse := store.FundTask(ctx, ledger.FundStoreCommand{
		EntryID: core.NewLedgerEntryID().(core.LedgerEntryIDCreated).Value, FunderUserID: funder, TaskID: otherTaskID,
		Amount: testCreditAmount(t, 30), IdempotencyKey: key,
	})
	rejected, matched := reuse.(ledger.FundRejected)
	if !matched {
		t.Fatalf("fund a different task with a spent key: want FundRejected, got %#v", reuse)
	}
	if rejected.Reason.Description() != "idempotency key was used for a different command" {
		t.Fatalf("reused-key rejection = %q, want %q", rejected.Reason.Description(), "idempotency key was used for a different command")
	}
	if _, fundFound, _ := (LedgerBrowserStore{storage: storage}).loadFund(otherTaskID.String()); fundFound {
		t.Fatalf("reused-key fund must not allocate credits to the other task")
	}
}

func TestLedgerBrowserStoreRefundReplayRequiresSameKey(t *testing.T) {
	store, storage, funder, taskID, ctx := newFundableTaskTestEnv(t, "funder", true)

	store.FundTask(ctx, ledger.FundStoreCommand{
		EntryID: core.NewLedgerEntryID().(core.LedgerEntryIDCreated).Value, FunderUserID: funder, TaskID: taskID,
		Amount: testCreditAmount(t, 30), IdempotencyKey: testIdempotencyKey(t, "fund-1"),
	})
	firstKey := testIdempotencyKey(t, "refund-1")
	first := store.RefundTask(ctx, ledger.RefundStoreCommand{
		EntryID: core.NewLedgerEntryID().(core.LedgerEntryIDCreated).Value, RequesterUserID: funder, TaskID: taskID,
		IdempotencyKey: firstKey,
	})
	if _, matched := first.(ledger.TaskRefunded); !matched {
		t.Fatalf("first refund: want TaskRefunded, got %#v", first)
	}

	// Reopen the task record so the refund path reaches the fund-lookup
	// checks (a completed refund also cancels the task, which rejects
	// earlier on both backends).
	record, _, _ := loadStoredTaskRecord(storage, taskID.String())
	record.State = "open"
	if !saveStoredTaskRecord(storage, record) {
		t.Fatalf("reopen task failed")
	}

	differentKey := store.RefundTask(ctx, ledger.RefundStoreCommand{
		EntryID: core.NewLedgerEntryID().(core.LedgerEntryIDCreated).Value, RequesterUserID: funder, TaskID: taskID,
		IdempotencyKey: testIdempotencyKey(t, "refund-2"),
	})
	rejected, matched := differentKey.(ledger.RefundRejected)
	if !matched {
		t.Fatalf("refund an already-refunded task with a new key: want RefundRejected, got %#v", differentKey)
	}
	if rejected.Reason.Description() != "task has nothing to refund" {
		t.Fatalf("refund rejection = %q, want %q", rejected.Reason.Description(), "task has nothing to refund")
	}

	replay := store.RefundTask(ctx, ledger.RefundStoreCommand{
		EntryID: core.NewLedgerEntryID().(core.LedgerEntryIDCreated).Value, RequesterUserID: funder, TaskID: taskID,
		IdempotencyKey: firstKey,
	})
	if _, matched := replay.(ledger.TaskRefunded); !matched {
		t.Fatalf("retried refund with the original key: want TaskRefunded (replayed), got %#v", replay)
	}
	balance := store.Balance(ctx, funder).(ledger.BalanceFound)
	if balance.Value.Spendable() != 100 {
		t.Fatalf("funder balance after replay = %d, want 100 (not double-refunded)", balance.Value.Spendable())
	}
}

func TestLedgerBrowserStoreLedgerPaginationHasNoMarkerGaps(t *testing.T) {
	store, storage, funder, taskID, ctx := newFundableTaskTestEnv(t, "funder", true)
	otherTaskID := core.NewTaskID().(core.TaskIDCreated).Value
	seedDraftTask(t, storage, otherTaskID.String(), funder.String(), "none", 0)

	store.FundTask(ctx, ledger.FundStoreCommand{
		EntryID: core.NewLedgerEntryID().(core.LedgerEntryIDCreated).Value, FunderUserID: funder, TaskID: taskID,
		Amount: testCreditAmount(t, 10), IdempotencyKey: testIdempotencyKey(t, "fund-1"),
	})
	store.FundTask(ctx, ledger.FundStoreCommand{
		EntryID: core.NewLedgerEntryID().(core.LedgerEntryIDCreated).Value, FunderUserID: funder, TaskID: otherTaskID,
		Amount: testCreditAmount(t, 10), IdempotencyKey: testIdempotencyKey(t, "fund-2"),
	})

	// 3 real entries exist (signup grant + 2 escrows). A page past the first
	// entry must return 2 full rows; hidden idempotency markers used to
	// occupy page slots and leave gaps.
	page := store.ListEntries(ctx, funder, testPage(t, 2, 1)).(ledger.EntriesListed)
	if len(page.Values) != 2 {
		t.Fatalf("ledger page limit=2 offset=1 = %d entries, want 2", len(page.Values))
	}
}

func TestLedgerBrowserStoreAcceptReplayReconstructsPayout(t *testing.T) {
	taskService, ledgerService, submissionService, storage := newReviewTestEnv(t)
	ctx := context.Background()
	owner := testUserID(t, "owner")
	worker := testUserID(t, "worker")
	grantFunderBalance(t, storage, owner)

	taskID, submissionID := newFundedSubmittedCreditTask(t, taskService, ledgerService, submissionService, owner, worker, 30)

	key := testIdempotencyKey(t, "accept-1")
	first := ledgerService.ReviewAcceptSubmission(ctx, owner, taskID, submissionID, key,
		ledger.FullCreditReviewSelection{}, ledger.NoTipSelection{}, ledger.NoCollectibleTipSelection{}).(ledger.SubmissionAccepted)
	if payout, matched := first.Payout.(ledger.CreditPayout); !matched || payout.Amount.Int64() != 30 {
		t.Fatalf("first accept payout = %#v, want credit payout of 30", first.Payout)
	}

	replayResult := ledgerService.ReviewAcceptSubmission(ctx, owner, taskID, submissionID, key,
		ledger.FullCreditReviewSelection{}, ledger.NoTipSelection{}, ledger.NoCollectibleTipSelection{})
	replay, matched := replayResult.(ledger.SubmissionAccepted)
	if !matched {
		t.Fatalf("replayed accept: want SubmissionAccepted, got %#v", replayResult)
	}
	payout, payoutMatched := replay.Payout.(ledger.CreditPayout)
	if !payoutMatched || payout.Amount.Int64() != 30 || payout.WorkerUserID != worker {
		t.Fatalf("replayed accept payout = %#v, want the original credit payout of 30 to worker", replay.Payout)
	}
	workerBalance := ledgerService.Balance(ctx, worker).(ledger.BalanceFound)
	if workerBalance.Value.Spendable() != 30 {
		t.Fatalf("worker balance after replay = %d, want 30 (not double-paid)", workerBalance.Value.Spendable())
	}
}

func TestLedgerBrowserStoreRejectWithBanBlocksReservationAndSubmission(t *testing.T) {
	taskService, ledgerService, submissionService, _ := newReviewTestEnv(t)
	ctx := context.Background()
	owner := testUserID(t, "owner")
	worker := auth.UserSubject{ID: testUserID(t, "worker")}

	taskID, submissionID := newReservedSubmittedTask(t, taskService, submissionService, owner, worker)

	note := submission.NewRequiredReviewNote("Not a good fit.").(submission.ReviewNoteAccepted).Value
	rejectResult := ledgerService.RejectSubmission(ctx, owner, taskID, submissionID, testIdempotencyKey(t, "reject-1"), note,
		ledger.NoCreditReviewSelection{}, ledger.NoTipSelection{}, ledger.BanImplementorSelection{})
	if _, matched := rejectResult.(ledger.SubmissionRejected); !matched {
		t.Fatalf("reject with ban: want SubmissionRejected, got %#v", rejectResult)
	}

	reserveAgain := taskService.Reserve(ctx, worker, taskID)
	rejected, matched := reserveAgain.(task.ReservationRejected)
	if !matched {
		t.Fatalf("banned worker re-reserving: want ReservationRejected, got %#v", reserveAgain)
	}
	if rejected.Reason.Description() != "implementor is banned from the task" {
		t.Fatalf("banned reservation rejection = %q, want ban message", rejected.Reason.Description())
	}

	eligibility := taskService.CheckSubmissionEligibility(ctx, taskID, worker.ID)
	if _, matched := eligibility.(task.SubmissionEligibilityRejected); !matched {
		t.Fatalf("banned worker submission eligibility: want SubmissionEligibilityRejected, got %#v", eligibility)
	}
}

func TestAssetBrowserStoreFundedCollectibleTaskParsesRewardCount(t *testing.T) {
	storage := newTestBrowserStorage()
	ids := &counterLedgerIDs{}
	taskService := task.NewService(NewTaskBrowserStore(storage, ids, systemTestClock{}), noopOrganizationPermissions{}, nil)
	assetService := assets.NewService(NewAssetBrowserStore(storage, ids))
	ctx := context.Background()
	owner := testUserID(t, "owner")
	ownerSubject := auth.UserSubject{ID: owner}

	// A no-reward task funded with a collectible flips to reward_kind
	// collectible; its detail and every task list must keep parsing (the
	// stored record has no counter of its own - the count derives from the
	// reward records, like the real store's subquery).
	created := taskService.Create(ctx, testCreateCommand(t, owner, task.NoRewardSpec{}, task.ParticipationPolicyOpen)).(task.TaskCreated)
	minted := assetService.Mint(ctx, assets.CollectibleOwnerKindUser, owner.String(), "", testCollectibleName(t, "Golden Badge"), assets.CollectibleKindBadge, assets.TransferPolicyTransferableBetweenUsers, "art-1").(assets.CollectibleMinted)
	if _, matched := assetService.FundReward(ctx, owner, created.Value.ID, minted.Value.ID).(assets.RewardFunded); !matched {
		t.Fatalf("fund collectible reward failed")
	}

	got := taskService.Get(ctx, ownerSubject, created.Value.ID)
	detail, matched := got.(task.TaskGot)
	if !matched {
		t.Fatalf("task detail after collectible funding: want TaskGot, got %#v", got)
	}
	reward, rewardMatched := detail.Value.Reward.(task.CollectibleRewardSpec)
	if !rewardMatched || reward.Count.Int() != 1 {
		t.Fatalf("task reward after collectible funding = %#v, want collectible count 1", detail.Value.Reward)
	}

	if _, matched := taskService.Open(ctx, ownerSubject, created.Value.ID).(task.TaskStateChanged); !matched {
		t.Fatalf("open funded collectible task failed")
	}
	listed := taskService.List(ctx, ownerSubject, task.PublicListScope{ViewerID: owner}, task.NoListFilters(), testPage(t, 10, 0)).(task.TasksListed)
	if len(listed.Values) != 1 {
		t.Fatalf("public list after collectible funding = %+v, want one task", listed.Values)
	}
}

func TestAssetBrowserStoreFundRewardAfterRefundReEscrows(t *testing.T) {
	service, storage, _ := newAssetsTestEnv(t)
	ctx := context.Background()
	owner := testUserID(t, "owner")

	minted := service.Mint(ctx, assets.CollectibleOwnerKindUser, owner.String(), "", testCollectibleName(t, "Golden Badge"), assets.CollectibleKindBadge, assets.TransferPolicyTransferableBetweenUsers, "art-1").(assets.CollectibleMinted)
	taskID := core.NewTaskID().(core.TaskIDCreated).Value
	seedDraftTask(t, storage, taskID.String(), owner.String(), "none", 0)

	if _, matched := service.FundReward(ctx, owner, taskID, minted.Value.ID).(assets.RewardFunded); !matched {
		t.Fatalf("first fund failed")
	}
	if _, matched := service.RefundReward(ctx, owner, taskID).(assets.RewardRefunded); !matched {
		t.Fatalf("refund failed")
	}
	// The refund cleared the stateless held record and returned the collectible
	// to minted. Reset the cancelled task to draft: the same collectible can now
	// be escrowed on it again.
	record, _, _ := loadStoredTaskRecord(storage, taskID.String())
	record.State = "draft"
	if !saveStoredTaskRecord(storage, record) {
		t.Fatalf("reset task state failed")
	}

	again := service.FundReward(ctx, owner, taskID, minted.Value.ID)
	if _, matched := again.(assets.RewardFunded); !matched {
		t.Fatalf("re-escrowing the collectible after refund: want RewardFunded, got %#v", again)
	}
}

func TestAssetBrowserStoreRefundRewardReturnsRemainingHeld(t *testing.T) {
	service, storage, _ := newAssetsTestEnv(t)
	ctx := context.Background()
	owner := testUserID(t, "owner")

	first := service.Mint(ctx, assets.CollectibleOwnerKindUser, owner.String(), "", testCollectibleName(t, "Badge A"), assets.CollectibleKindBadge, assets.TransferPolicyTransferableBetweenUsers, "art-1").(assets.CollectibleMinted)
	second := service.Mint(ctx, assets.CollectibleOwnerKindUser, owner.String(), "", testCollectibleName(t, "Badge B"), assets.CollectibleKindBadge, assets.TransferPolicyTransferableBetweenUsers, "art-2").(assets.CollectibleMinted)
	taskID := core.NewTaskID().(core.TaskIDCreated).Value
	seedDraftTask(t, storage, taskID.String(), owner.String(), "none", 0)
	service.FundReward(ctx, owner, taskID, first.Value.ID)
	service.FundReward(ctx, owner, taskID, second.Value.ID)

	// Simulate one collectible having already been awarded: its stateless
	// held record is gone. Refund returns only the still-held collectible.
	if !deleteTaskFundCollectible(storage, taskID.String(), first.Value.ID.String()) {
		t.Fatalf("remove awarded reward record failed")
	}

	refundResult := service.RefundReward(ctx, owner, taskID)
	refunded, matched := refundResult.(assets.RewardRefunded)
	if !matched {
		t.Fatalf("refund remaining held collectible: want RewardRefunded, got %#v", refundResult)
	}
	if len(refunded.Values) != 1 || refunded.Values[0].ID != second.Value.ID {
		t.Fatalf("refunded = %#v, want just the remaining held collectible", refunded.Values)
	}
}

func TestAuthBrowserStoreDeactivateRejectsWhileHoldingFundedRewards(t *testing.T) {
	storage := newTestBrowserStorage()
	ids := &counterLedgerIDs{}
	authStore := NewAuthBrowserStore(storage, ids)
	ledgerStore := NewLedgerBrowserStore(storage, ids)
	ctx := context.Background()

	email := testEmail(t, "funder@example.com")
	serviceResult := auth.NewService(authStore, testAccessTokenSecret(t), fixedTestClock{now: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)})
	service := serviceResult.(auth.ServiceCreated).Value
	if _, matched := service.Register(ctx, email, testPassword(t, "correct horse battery staple")).(auth.RegisterAccepted); !matched {
		t.Fatalf("register failed")
	}
	credential := authStore.FindCredentialByEmail(ctx, email).(auth.CredentialFound)
	funder := credential.Record.UserID

	taskID := core.NewTaskID().(core.TaskIDCreated).Value
	seedDraftTask(t, storage, taskID.String(), funder.String(), "none", 0)
	if _, matched := appendStringIndex(storage, taskUserIndexKey(funder.String()), taskID.String(), "task").(stringIndexStored); !matched {
		t.Fatalf("index task for funder failed")
	}
	fundResult := ledgerStore.FundTask(ctx, ledger.FundStoreCommand{
		EntryID: core.NewLedgerEntryID().(core.LedgerEntryIDCreated).Value, FunderUserID: funder, TaskID: taskID,
		Amount: testCreditAmount(t, 30), IdempotencyKey: testIdempotencyKey(t, "fund-1"),
	})
	if _, matched := fundResult.(ledger.TaskFunded); !matched {
		t.Fatalf("fund failed: %#v", fundResult)
	}

	deactivate := authStore.DeactivateUser(ctx, funder)
	rejected, matched := deactivate.(auth.AccountMutationRejected)
	if !matched {
		t.Fatalf("deactivate with held escrow: want AccountMutationRejected, got %#v", deactivate)
	}
	if rejected.Reason.Code() != core.ErrorCodeConflict {
		t.Fatalf("deactivate rejection code = %v, want conflict", rejected.Reason.Code())
	}

	refundResult := ledgerStore.RefundTask(ctx, ledger.RefundStoreCommand{
		EntryID: core.NewLedgerEntryID().(core.LedgerEntryIDCreated).Value, RequesterUserID: funder, TaskID: taskID,
		IdempotencyKey: testIdempotencyKey(t, "refund-1"),
	})
	if _, matched := refundResult.(ledger.TaskRefunded); !matched {
		t.Fatalf("refund failed: %#v", refundResult)
	}
	if _, matched := authStore.DeactivateUser(ctx, funder).(auth.AccountMutationAccepted); !matched {
		t.Fatalf("deactivate after refund: want AccountMutationAccepted")
	}
}

func TestAuthBrowserStoreMutationsOnDeactivatedAccountAreNotFound(t *testing.T) {
	storage := newTestBrowserStorage()
	ids := &counterLedgerIDs{}
	authStore := NewAuthBrowserStore(storage, ids)
	ctx := context.Background()

	email := testEmail(t, "person@example.com")
	serviceResult := auth.NewService(authStore, testAccessTokenSecret(t), fixedTestClock{now: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)})
	service := serviceResult.(auth.ServiceCreated).Value
	if _, matched := service.Register(ctx, email, testPassword(t, "correct horse battery staple")).(auth.RegisterAccepted); !matched {
		t.Fatalf("register failed")
	}
	credential := authStore.FindCredentialByEmail(ctx, email).(auth.CredentialFound)
	userID := credential.Record.UserID
	if _, matched := authStore.DeactivateUser(ctx, userID).(auth.AccountMutationAccepted); !matched {
		t.Fatalf("deactivate failed")
	}

	hash := credential.Record.PasswordHash
	checks := map[string]auth.AccountMutationResult{
		"update email":     authStore.UpdateUserEmail(ctx, userID, testEmail(t, "new@example.com")),
		"update password":  authStore.UpdatePassword(ctx, userID, hash),
		"deactivate again": authStore.DeactivateUser(ctx, userID),
	}
	for label, result := range checks {
		rejected, matched := result.(auth.AccountMutationRejected)
		if !matched {
			t.Fatalf("%s on deactivated account: want AccountMutationRejected, got %#v", label, result)
		}
		if rejected.Reason.Code() != core.ErrorCodeNotFound {
			t.Fatalf("%s rejection code = %v, want not_found", label, rejected.Reason.Code())
		}
	}
}

func TestOrgBrowserStoreFindMemberRolesZeroRolesIsMissing(t *testing.T) {
	storage := newTestBrowserStorage()
	store := NewOrgBrowserStore(storage, &counterLedgerIDs{})
	organizationID := core.NewOrganizationID().(core.OrganizationIDCreated).Value
	userID := testUserID(t, "member")

	membershipID := "membership-zero-roles"
	membership := storedMembership{ID: membershipID, OrganizationID: organizationID.String(), UserID: userID.String(), Status: org.MembershipStatusActive.String(), Roles: []string{}}
	if !putStoredMembershipJSON(storage, orgMembershipKey(membershipID), membership) {
		t.Fatalf("write membership failed")
	}
	if !putStorageString(storage, orgActiveMembershipKey(organizationID.String(), userID.String()), membershipID) {
		t.Fatalf("write membership pointer failed")
	}

	result := store.FindMemberRoles(context.Background(), organizationID, userID)
	if _, matched := result.(org.MemberRolesMissing); !matched {
		t.Fatalf("member with zero roles: want MemberRolesMissing (matching internal/db), got %#v", result)
	}
}

func TestTaskBrowserStoreSeriesMoveTaskBetweenSeries(t *testing.T) {
	taskService, _, _, _ := newTaskTestEnv(t)
	ctx := context.Background()
	owner := auth.UserSubject{ID: testUserID(t, "owner")}

	title := task.NewSeriesTitle("Series A").(task.SeriesTitleAccepted).Value
	description := task.NewSeriesDescription("").(task.SeriesDescriptionAccepted).Value
	seriesA := taskService.CreateSeries(ctx, owner, title, description).(task.SeriesMutated)
	titleB := task.NewSeriesTitle("Series B").(task.SeriesTitleAccepted).Value
	seriesB := taskService.CreateSeries(ctx, owner, titleB, description).(task.SeriesMutated)
	created := taskService.Create(ctx, testCreateCommand(t, owner.ID, task.NoRewardSpec{}, task.ParticipationPolicyOpen)).(task.TaskCreated)

	taskService.AddTaskToSeries(ctx, owner, seriesA.Value.Series.ID, created.Value.ID)
	moved := taskService.AddTaskToSeries(ctx, owner, seriesB.Value.Series.ID, created.Value.ID).(task.SeriesMutated)
	if len(moved.Value.Tasks) != 1 || moved.Value.Tasks[0].ID != created.Value.ID {
		t.Fatalf("series B after move = %+v, want just the moved task", moved.Value.Tasks)
	}

	seriesAAfter := taskService.GetSeries(ctx, owner, seriesA.Value.Series.ID).(task.SeriesGot)
	if len(seriesAAfter.Value.Tasks) != 0 {
		t.Fatalf("series A after move = %+v, want none (task moved to B)", seriesAAfter.Value.Tasks)
	}
}

func TestTaskBrowserStoreSeriesNextPositionAfterRemoval(t *testing.T) {
	taskService, _, _, _ := newTaskTestEnv(t)
	ctx := context.Background()
	owner := auth.UserSubject{ID: testUserID(t, "owner")}

	title := task.NewSeriesTitle("Sprint 1").(task.SeriesTitleAccepted).Value
	description := task.NewSeriesDescription("").(task.SeriesDescriptionAccepted).Value
	series := taskService.CreateSeries(ctx, owner, title, description).(task.SeriesMutated)
	firstTask := taskService.Create(ctx, testCreateCommand(t, owner.ID, task.NoRewardSpec{}, task.ParticipationPolicyOpen)).(task.TaskCreated)
	secondTask := taskService.Create(ctx, testCreateCommand(t, owner.ID, task.NoRewardSpec{}, task.ParticipationPolicyOpen)).(task.TaskCreated)
	thirdTask := taskService.Create(ctx, testCreateCommand(t, owner.ID, task.NoRewardSpec{}, task.ParticipationPolicyOpen)).(task.TaskCreated)

	taskService.AddTaskToSeries(ctx, owner, series.Value.Series.ID, firstTask.Value.ID)
	taskService.AddTaskToSeries(ctx, owner, series.Value.Series.ID, secondTask.Value.ID)
	taskService.RemoveTaskFromSeries(ctx, owner, series.Value.Series.ID, firstTask.Value.ID)

	// The remaining task holds position 2; the next position must be max+1
	// (3), not index-length+1 (2), which would collide.
	added := taskService.AddTaskToSeries(ctx, owner, series.Value.Series.ID, thirdTask.Value.ID).(task.SeriesMutated)
	var thirdPlacement task.ExistingSeriesPlacement
	for _, member := range added.Value.Tasks {
		if member.ID == thirdTask.Value.ID {
			thirdPlacement = member.Placement.(task.ExistingSeriesPlacement)
		}
	}
	if thirdPlacement.Position.Int() != 3 {
		t.Fatalf("third task position after removal = %d, want 3 (max+1)", thirdPlacement.Position.Int())
	}
}
