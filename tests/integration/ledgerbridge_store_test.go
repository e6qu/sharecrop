//go:build integration

package integration_test

import (
	"context"
	"testing"

	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/db"
	"github.com/e6qu/sharecrop/internal/ledger"
	"github.com/e6qu/sharecrop/internal/wasibridge/ledgerbridge"
	"github.com/e6qu/sharecrop/internal/wasibridge/rpc"
)

// TestLedgerBridgeDualRun drives a full fund -> accept -> refund flow through
// the compiled wasip1 guest + host bridge against real Postgres, and checks the
// read paths (balance, task-allocated credits, ledger entries) return byte-for-
// byte the same values as a direct db call. Accept exercises the hardest result:
// SubmissionAccepted carrying a nested PayoutOutcome.
func TestLedgerBridgeDualRun(t *testing.T) {
	ctx := context.Background()
	pool := newPool(t)
	dbStore := db.NewLedgerStore(pool)

	guestWASM, err := compileWASIGuest(t, "github.com/e6qu/sharecrop/cmd/sharecrop-wasi-store-guest")
	if err != nil {
		t.Fatalf("compile store guest: %v", err)
	}
	host, err := rpc.NewHost(ctx, guestWASM, func(ctx context.Context, method string, args []byte) ([]byte, error) {
		return ledgerbridge.Dispatch(ctx, dbStore, method, args)
	})
	if err != nil {
		t.Fatalf("new host: %v", err)
	}
	t.Cleanup(func() { _ = host.Close(ctx) })
	bridgeStore := ledgerbridge.NewGuestStore(func(method string, args []byte) ([]byte, error) {
		return host.Call(ctx, method, args)
	})

	owner := createUser(t, pool, "ledger-owner")
	worker := createUser(t, pool, "ledger-worker")
	taskID := insertTask(t, pool, owner, "draft", 40)
	page := requirePage(t, 50, 0)

	t.Run("fund through the bridge moves credits and matches direct reads", func(t *testing.T) {
		result := bridgeStore.FundTask(ctx, fundCommand(t, owner, taskID, 40, "bridge-fund-"+taskID.String()))
		funded, matched := result.(ledger.TaskFunded)
		if !matched {
			t.Fatalf("bridge FundTask = %T, want TaskFunded", result)
		}
		if funded.Fund.CreditAmount.Int64() != 40 || funded.Fund.TaskID != taskID {
			t.Errorf("funded task fund = %+v", funded.Fund)
		}
		assertBalancesEqual(t, bridgeStore, dbStore, owner)
		if balance := mustBalanceVia(t, bridgeStore, owner); balance.Spendable() != 60 || balance.Allocated() != 40 {
			t.Errorf("owner balance after funding = (%d, %d), want (60, 40)", balance.Spendable(), balance.Allocated())
		}
	})

	t.Run("task-allocated credits match a direct call", func(t *testing.T) {
		viaBridge := mustAllocated(t, bridgeStore.TaskAllocatedCredits(ctx, taskID))
		direct := mustAllocated(t, dbStore.TaskAllocatedCredits(ctx, taskID))
		if viaBridge != direct || viaBridge != 40 {
			t.Errorf("task allocated: bridge %d, direct %d, want 40", viaBridge, direct)
		}
	})

	t.Run("ledger entries match a direct call", func(t *testing.T) {
		viaBridge := mustEntries(t, bridgeStore.ListEntries(ctx, owner, page))
		direct := mustEntries(t, dbStore.ListEntries(ctx, owner, page))
		assertEntriesEqual(t, viaBridge, direct)
	})

	t.Run("accept through the bridge pays the worker with a credit payout", func(t *testing.T) {
		setTaskState(t, pool, taskID, "open")
		submissionID := insertSubmission(t, pool, taskID, worker)

		result := bridgeStore.AcceptSubmission(ctx, acceptCommand(t, owner, taskID, submissionID, "bridge-accept-"+submissionID.String()))
		accepted, matched := result.(ledger.SubmissionAccepted)
		if !matched {
			t.Fatalf("bridge AcceptSubmission = %T, want SubmissionAccepted", result)
		}
		payout, matched := accepted.Payout.(ledger.CreditPayout)
		if !matched {
			t.Fatalf("payout = %T, want CreditPayout", accepted.Payout)
		}
		if payout.WorkerUserID != worker || payout.Amount.Int64() != 40 {
			t.Errorf("credit payout = %+v, want worker %s / 40", payout, worker)
		}
		assertBalancesEqual(t, bridgeStore, dbStore, worker)
		if balance := mustBalanceVia(t, bridgeStore, worker); balance.Spendable() != 140 {
			t.Errorf("worker balance after payout = %d, want 140", balance.Spendable())
		}
	})

	t.Run("refund through the bridge returns credits to the owner", func(t *testing.T) {
		refundTaskID := insertTask(t, pool, owner, "draft", 20)
		if _, matched := bridgeStore.FundTask(ctx, fundCommand(t, owner, refundTaskID, 20, "bridge-fund-"+refundTaskID.String())).(ledger.TaskFunded); !matched {
			t.Fatalf("bridge fund for refund did not succeed")
		}
		result := bridgeStore.RefundTask(ctx, refundCommand(t, owner, refundTaskID, "bridge-refund-"+refundTaskID.String()))
		refunded, matched := result.(ledger.TaskRefunded)
		if !matched {
			t.Fatalf("bridge RefundTask = %T, want TaskRefunded", result)
		}
		if refunded.Fund.TaskID != refundTaskID {
			t.Errorf("refunded fund task = %s, want %s", refunded.Fund.TaskID, refundTaskID)
		}
		assertBalancesEqual(t, bridgeStore, dbStore, owner)
	})
}

func mustBalanceVia(t *testing.T, store ledger.Store, account core.UserID) ledger.Balance {
	t.Helper()
	found, matched := store.Balance(context.Background(), account).(ledger.BalanceFound)
	if !matched {
		t.Fatalf("balance was rejected")
	}
	return found.Value
}

func assertBalancesEqual(t *testing.T, bridge, direct ledger.Store, account core.UserID) {
	t.Helper()
	viaBridge := mustBalanceVia(t, bridge, account)
	viaDirect := mustBalanceVia(t, direct, account)
	if viaBridge.Spendable() != viaDirect.Spendable() || viaBridge.Allocated() != viaDirect.Allocated() {
		t.Errorf("balance mismatch: bridge (%d, %d), direct (%d, %d)",
			viaBridge.Spendable(), viaBridge.Allocated(), viaDirect.Spendable(), viaDirect.Allocated())
	}
}

func mustAllocated(t *testing.T, result ledger.TaskAllocatedResult) int64 {
	t.Helper()
	found, matched := result.(ledger.TaskAllocatedFound)
	if !matched {
		t.Fatalf("task allocated result = %T, want found", result)
	}
	return found.Amount
}

func mustEntries(t *testing.T, result ledger.ListEntriesResult) []ledger.LedgerEntry {
	t.Helper()
	listed, matched := result.(ledger.EntriesListed)
	if !matched {
		t.Fatalf("list entries result = %T, want listed", result)
	}
	return listed.Values
}

func assertEntriesEqual(t *testing.T, got, want []ledger.LedgerEntry) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("ledger entry counts: bridge %d, direct %d", len(got), len(want))
	}
	for index := range want {
		if got[index].ID != want[index].ID {
			t.Errorf("entry %d id: %s != %s", index, got[index].ID, want[index].ID)
		}
		if got[index].Kind.String() != want[index].Kind.String() {
			t.Errorf("entry %d kind: %s != %s", index, got[index].Kind, want[index].Kind)
		}
		if got[index].Amount.Int64() != want[index].Amount.Int64() {
			t.Errorf("entry %d amount: %d != %d", index, got[index].Amount.Int64(), want[index].Amount.Int64())
		}
	}
}
