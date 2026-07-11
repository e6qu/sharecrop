package wasmdemo

import (
	"context"
	"testing"

	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/ledger"
)

func testCreditAmount(t *testing.T, value int64) ledger.CreditAmount {
	t.Helper()
	result := ledger.NewCreditAmount(value)
	accepted, matched := result.(ledger.CreditAmountAccepted)
	if !matched {
		t.Fatalf("new credit amount %d failed", value)
	}
	return accepted.Value
}

func testIdempotencyKey(t *testing.T, raw string) ledger.IdempotencyKey {
	t.Helper()
	result := ledger.NewIdempotencyKey(raw)
	accepted, matched := result.(ledger.IdempotencyKeyAccepted)
	if !matched {
		t.Fatalf("new idempotency key %q failed", raw)
	}
	return accepted.Value
}

// seedDraftTask writes a minimal storedTaskRecord directly (bypassing the
// not-yet-built task browser store, tested separately) so ledger's fund/
// refund logic can be exercised against a real, shared task record.
func seedDraftTask(t *testing.T, storage BrowserStorage, taskID string, createdBy string, rewardKind string, rewardCreditAmount int64) {
	t.Helper()
	record := storedTaskRecord{
		ID: taskID, OwnerKind: "user", OwnerUserID: createdBy, Title: "Test task", Description: "desc",
		TaskType: "general", RewardKind: rewardKind, RewardCreditAmount: rewardCreditAmount,
		Participation: "open", AssigneeScope: "user", ReservationTTLHours: 48, State: "draft",
		VisibilityKind: "public", ResponseSchemaJSON: `{"kind":"freeform"}`, PayloadKind: "none", CreatedBy: createdBy,
	}
	if !saveStoredTaskRecord(storage, record) {
		t.Fatalf("seed draft task failed")
	}
}

// newFundableTaskTestEnv wires a LedgerBrowserStore against a freshly
// seeded draft task with no reward yet. If grantSignupBonus is true, the
// funder is granted a signup-bonus balance the same way auth does - the
// common setup shared by the fund/refund tests below.
func newFundableTaskTestEnv(t *testing.T, funderLabel string, grantSignupBonus bool) (LedgerBrowserStore, BrowserStorage, core.UserID, core.TaskID, context.Context) {
	t.Helper()
	storage := newTestBrowserStorage()
	ids := &counterLedgerIDs{}
	store := NewLedgerBrowserStore(storage, ids)
	funder := testUserID(t, funderLabel)
	if grantSignupBonus {
		NewAuthBrowserStore(storage, ids).insertSignupGrant("user", funder.String())
	}
	taskID := core.NewTaskID().(core.TaskIDCreated).Value
	seedDraftTask(t, storage, taskID.String(), funder.String(), "none", 0)
	return store, storage, funder, taskID, context.Background()
}

func TestLedgerBrowserStoreFundTaskFirstTime(t *testing.T) {
	store, storage, funder, taskID, ctx := newFundableTaskTestEnv(t, "funder", true)

	result := store.FundTask(ctx, ledger.FundStoreCommand{
		EntryID: core.NewLedgerEntryID().(core.LedgerEntryIDCreated).Value, FunderUserID: funder, TaskID: taskID,
		Amount: testCreditAmount(t, 30), IdempotencyKey: testIdempotencyKey(t, "fund-1"),
	})
	funded, matched := result.(ledger.TaskFunded)
	if !matched {
		t.Fatalf("fund task: want TaskFunded, got %#v", result)
	}
	if funded.Fund.CreditAmount.Int64() != 30 {
		t.Fatalf("fund = %+v, want credit_amount=30", funded.Fund)
	}

	record, found, _ := loadStoredTaskRecord(storage, taskID.String())
	if !found || record.RewardKind != "credit" || record.RewardCreditAmount != 30 {
		t.Fatalf("task after funding = %+v, want reward_kind=credit amount=30", record)
	}

	balanceResult := store.Balance(ctx, funder)
	balance, matched := balanceResult.(ledger.BalanceFound)
	if !matched {
		t.Fatalf("balance: want BalanceFound, got %#v", balanceResult)
	}
	// Funding moves 30 from spendable into allocated; the 100-credit total is conserved.
	if balance.Value.Spendable() != 70 || balance.Value.Allocated() != 30 || balance.Value.Total() != 100 {
		t.Fatalf("funder balance = (spendable %d, allocated %d, total %d), want (70, 30, 100)", balance.Value.Spendable(), balance.Value.Allocated(), balance.Value.Total())
	}
}

func TestLedgerBrowserStoreFundTaskRejectsInsufficientBalance(t *testing.T) {
	store, _, funder, taskID, ctx := newFundableTaskTestEnv(t, "funder", true)

	result := store.FundTask(ctx, ledger.FundStoreCommand{
		EntryID: core.NewLedgerEntryID().(core.LedgerEntryIDCreated).Value, FunderUserID: funder, TaskID: taskID,
		Amount: testCreditAmount(t, 1000), IdempotencyKey: testIdempotencyKey(t, "fund-1"),
	})
	if _, matched := result.(ledger.FundRejected); !matched {
		t.Fatalf("fund with insufficient balance: want FundRejected, got %#v", result)
	}
}

func TestLedgerBrowserStoreFundTaskRejectsWrongOwner(t *testing.T) {
	store, storage, _, taskID, ctx := newFundableTaskTestEnv(t, "owner", false)
	other := testUserID(t, "other")
	NewAuthBrowserStore(storage, &counterLedgerIDs{}).insertSignupGrant("user", other.String())

	result := store.FundTask(ctx, ledger.FundStoreCommand{
		EntryID: core.NewLedgerEntryID().(core.LedgerEntryIDCreated).Value, FunderUserID: other, TaskID: taskID,
		Amount: testCreditAmount(t, 10), IdempotencyKey: testIdempotencyKey(t, "fund-1"),
	})
	if _, matched := result.(ledger.FundRejected); !matched {
		t.Fatalf("fund by non-owner: want FundRejected, got %#v", result)
	}
}

func TestLedgerBrowserStoreFundTaskIdempotentRetry(t *testing.T) {
	store, _, funder, taskID, ctx := newFundableTaskTestEnv(t, "funder", true)

	command := ledger.FundStoreCommand{
		EntryID: core.NewLedgerEntryID().(core.LedgerEntryIDCreated).Value, FunderUserID: funder, TaskID: taskID,
		Amount: testCreditAmount(t, 30), IdempotencyKey: testIdempotencyKey(t, "fund-1"),
	}
	first := store.FundTask(ctx, command)
	if _, matched := first.(ledger.TaskFunded); !matched {
		t.Fatalf("first fund: want TaskFunded, got %#v", first)
	}

	// A retried request with the same idempotency key replays the same
	// result instead of erroring "already funded".
	retry := store.FundTask(ctx, command)
	if _, matched := retry.(ledger.TaskFunded); !matched {
		t.Fatalf("retried fund with same idempotency key: want TaskFunded (replayed), got %#v", retry)
	}

	balanceResult := store.Balance(ctx, funder).(ledger.BalanceFound)
	if balanceResult.Value.Spendable() != 70 {
		t.Fatalf("funder balance after retry = %d, want 70 (not double-charged)", balanceResult.Value.Spendable())
	}
}

func TestLedgerBrowserStoreFundTaskRejectsAlreadyFundedWithDifferentKey(t *testing.T) {
	store, _, funder, taskID, ctx := newFundableTaskTestEnv(t, "funder", true)

	store.FundTask(ctx, ledger.FundStoreCommand{
		EntryID: core.NewLedgerEntryID().(core.LedgerEntryIDCreated).Value, FunderUserID: funder, TaskID: taskID,
		Amount: testCreditAmount(t, 30), IdempotencyKey: testIdempotencyKey(t, "fund-1"),
	})

	secondResult := store.FundTask(ctx, ledger.FundStoreCommand{
		EntryID: core.NewLedgerEntryID().(core.LedgerEntryIDCreated).Value, FunderUserID: funder, TaskID: taskID,
		Amount: testCreditAmount(t, 30), IdempotencyKey: testIdempotencyKey(t, "fund-2"),
	})
	if _, matched := secondResult.(ledger.FundRejected); !matched {
		t.Fatalf("second genuine fund attempt: want FundRejected (already funded), got %#v", secondResult)
	}
}

func TestLedgerBrowserStoreRefundTaskReturnsCreditsAndCancelsTask(t *testing.T) {
	store, storage, funder, taskID, ctx := newFundableTaskTestEnv(t, "funder", true)

	store.FundTask(ctx, ledger.FundStoreCommand{
		EntryID: core.NewLedgerEntryID().(core.LedgerEntryIDCreated).Value, FunderUserID: funder, TaskID: taskID,
		Amount: testCreditAmount(t, 30), IdempotencyKey: testIdempotencyKey(t, "fund-1"),
	})

	refundResult := store.RefundTask(ctx, ledger.RefundStoreCommand{
		EntryID: core.NewLedgerEntryID().(core.LedgerEntryIDCreated).Value, RequesterUserID: funder, TaskID: taskID,
		IdempotencyKey: testIdempotencyKey(t, "refund-1"),
	})
	refunded, matched := refundResult.(ledger.TaskRefunded)
	if !matched {
		t.Fatalf("refund: want TaskRefunded, got %#v", refundResult)
	}
	if refunded.Fund.CreditAmount.Int64() != 30 {
		t.Fatalf("refunded fund credit amount = %d, want 30", refunded.Fund.CreditAmount.Int64())
	}

	balance := store.Balance(ctx, funder).(ledger.BalanceFound)
	// The allocated credits return to spendable; total stays 100, allocated 0.
	if balance.Value.Spendable() != 100 || balance.Value.Allocated() != 0 {
		t.Fatalf("funder balance after refund = (spendable %d, allocated %d), want (100, 0)", balance.Value.Spendable(), balance.Value.Allocated())
	}

	record, found, _ := loadStoredTaskRecord(storage, taskID.String())
	if !found || record.State != "cancelled" {
		t.Fatalf("task after refund state = %+v, want cancelled", record)
	}
}

func TestLedgerBrowserStoreRefundTaskRejectsWithoutEscrow(t *testing.T) {
	store, _, funder, taskID, ctx := newFundableTaskTestEnv(t, "funder", false)

	result := store.RefundTask(ctx, ledger.RefundStoreCommand{
		EntryID: core.NewLedgerEntryID().(core.LedgerEntryIDCreated).Value, RequesterUserID: funder, TaskID: taskID,
		IdempotencyKey: testIdempotencyKey(t, "refund-1"),
	})
	if _, matched := result.(ledger.RefundRejected); !matched {
		t.Fatalf("refund without escrow: want RefundRejected, got %#v", result)
	}
}

func TestLedgerBrowserStoreListEntriesExcludesIdempotencyMarkers(t *testing.T) {
	store, _, funder, taskID, ctx := newFundableTaskTestEnv(t, "funder", true)
	store.FundTask(ctx, ledger.FundStoreCommand{
		EntryID: core.NewLedgerEntryID().(core.LedgerEntryIDCreated).Value, FunderUserID: funder, TaskID: taskID,
		Amount: testCreditAmount(t, 30), IdempotencyKey: testIdempotencyKey(t, "fund-1"),
	})

	listResult := store.ListEntries(ctx, funder, testPage(t, 10, 0))
	listed, matched := listResult.(ledger.EntriesListed)
	if !matched {
		t.Fatalf("list entries: want EntriesListed, got %#v", listResult)
	}
	// Signup grant + escrow debit = 2 real entries; the idempotency marker
	// must not appear as a third, spurious entry.
	if len(listed.Values) != 2 {
		t.Fatalf("listed entries count = %d, want 2 (signup grant + escrow)", len(listed.Values))
	}
}
