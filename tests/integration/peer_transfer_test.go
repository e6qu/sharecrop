//go:build integration

package integration_test

import (
	"context"
	"testing"

	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/db"
	"github.com/e6qu/sharecrop/internal/event"
	"github.com/e6qu/sharecrop/internal/ledger"
	"github.com/jackc/pgx/v5/pgxpool"
)

func mustTransferAmount(t *testing.T, value int64) ledger.CreditAmount {
	t.Helper()
	accepted, matched := ledger.NewCreditAmount(value).(ledger.CreditAmountAccepted)
	if !matched {
		t.Fatalf("credit amount rejected")
	}
	return accepted.Value
}

func mustTransferKey(t *testing.T, raw string) ledger.IdempotencyKey {
	t.Helper()
	accepted, matched := ledger.NewIdempotencyKey(raw).(ledger.IdempotencyKeyAccepted)
	if !matched {
		t.Fatalf("idempotency key rejected")
	}
	return accepted.Value
}

func mustTransferNote(t *testing.T, raw string) ledger.TransferNote {
	t.Helper()
	accepted, matched := ledger.NewGrantNote(raw).(ledger.GrantNoteAccepted)
	if !matched {
		t.Fatalf("note rejected")
	}
	return ledger.TransferNoteProvided{Note: accepted.Value}
}

func countPeerTransferRows(t *testing.T, pool *pgxpool.Pool, key string) int {
	t.Helper()
	var count int
	if err := pool.QueryRow(context.Background(), `
		select count(*) from ledger_entries where kind = 'peer_transfer' and idempotency_key like $1
	`, key+"%").Scan(&count); err != nil {
		t.Fatalf("count peer transfer rows: %v", err)
	}
	return count
}

// TestPeerTransferUserToUser sends credits between two users: the balances
// move atomically, both peer_transfer rows carry the note, the credits_sent
// event notifies the receiver exactly once, and an idempotent replay changes
// nothing.
func TestPeerTransferUserToUser(t *testing.T) {
	pool := newPool(t)
	service := newDBLedgerService(pool)
	sender := createUser(t, pool, "p2p-sender")
	receiver := createUser(t, pool, "p2p-receiver")
	key := "p2p-" + sender.String()

	result := service.SendCredits(context.Background(), sender, ledger.TransferFromSelf{}, ledger.TransferToUser{ID: receiver},
		mustTransferAmount(t, 30), mustTransferNote(t, "thanks for the review"), mustTransferKey(t, key), ledger.SpendByUser{})
	sent, matched := result.(ledger.CreditsSent)
	if !matched {
		t.Fatalf("send rejected: %#v", result)
	}
	if _, first := sent.Execution.(ledger.FirstExecution); !first {
		t.Fatalf("execution = %T, want FirstExecution", sent.Execution)
	}
	if len(sent.ReceiverUserIDs) != 1 || sent.ReceiverUserIDs[0] != receiver {
		t.Fatalf("receivers = %#v, want the receiving user", sent.ReceiverUserIDs)
	}

	store := db.NewLedgerStore(pool)
	senderBalance := mustBalance(t, store, sender)
	receiverBalance := mustBalance(t, store, receiver)
	if senderBalance.Spendable() != 70 {
		t.Fatalf("sender spendable = %d, want 70", senderBalance.Spendable())
	}
	if receiverBalance.Spendable() != 130 {
		t.Fatalf("receiver spendable = %d, want 130", receiverBalance.Spendable())
	}
	if got := countPeerTransferRows(t, pool, key); got != 2 {
		t.Fatalf("peer transfer rows = %d, want the debit/credit pair", got)
	}
	var note string
	if err := pool.QueryRow(context.Background(), "select note from ledger_entries where id = $1", sent.DebitEntryID.String()).Scan(&note); err != nil {
		t.Fatalf("read debit note: %v", err)
	}
	if note != "thanks for the review" {
		t.Fatalf("debit note = %q", note)
	}

	// The credits_sent event was recorded in the transfer transaction and
	// dispatched inline; the receiver has exactly one notification.
	if len(sent.RecordedEvents) != 1 {
		t.Fatalf("recorded events = %d, want 1", len(sent.RecordedEvents))
	}
	draft := sent.RecordedEvents[0]
	if draft.Kind != event.KindCreditsSent {
		t.Fatalf("event kind = %q, want credits_sent", draft.Kind.String())
	}
	if got := countNotificationsForEvent(t, pool, draft.ID, receiver); got != 1 {
		t.Fatalf("receiver notifications = %d, want 1", got)
	}
	if row := readEventOutboxRow(t, pool, draft.ID); row.dispatchState != "dispatched" {
		t.Fatalf("event state = %q, want dispatched", row.dispatchState)
	}

	// Idempotent replay: same key, no new rows, no new events, same outcome.
	replay := service.SendCredits(context.Background(), sender, ledger.TransferFromSelf{}, ledger.TransferToUser{ID: receiver},
		mustTransferAmount(t, 30), mustTransferNote(t, "thanks for the review"), mustTransferKey(t, key), ledger.SpendByUser{})
	replayed, replayMatched := replay.(ledger.CreditsSent)
	if !replayMatched {
		t.Fatalf("replay rejected: %#v", replay)
	}
	if _, isReplay := replayed.Execution.(ledger.IdempotentReplay); !isReplay {
		t.Fatalf("replay execution = %T, want IdempotentReplay", replayed.Execution)
	}
	if len(replayed.RecordedEvents) != 0 {
		t.Fatalf("replay recorded %d events, want 0", len(replayed.RecordedEvents))
	}
	if replayed.DebitEntryID != sent.DebitEntryID {
		t.Fatalf("replay debit entry = %s, want the original %s", replayed.DebitEntryID.String(), sent.DebitEntryID.String())
	}
	if got := countPeerTransferRows(t, pool, key); got != 2 {
		t.Fatalf("peer transfer rows after replay = %d, want 2", got)
	}
	if mustBalance(t, store, sender).Spendable() != 70 {
		t.Fatalf("sender balance moved on replay")
	}
	if got := countNotificationsForEvent(t, pool, draft.ID, receiver); got != 1 {
		t.Fatalf("receiver notifications after replay = %d, want 1", got)
	}
}

// TestPeerTransferRejectsInsufficientFundsAndSelfSend covers the two core
// refusals: overdrawing the spendable balance, and sending to yourself.
func TestPeerTransferRejectsInsufficientFundsAndSelfSend(t *testing.T) {
	pool := newPool(t)
	service := newDBLedgerService(pool)
	sender := createUser(t, pool, "p2p-reject-sender")
	receiver := createUser(t, pool, "p2p-reject-receiver")

	overdraw := service.SendCredits(context.Background(), sender, ledger.TransferFromSelf{}, ledger.TransferToUser{ID: receiver},
		mustTransferAmount(t, 1000), ledger.NoTransferNote{}, mustTransferKey(t, "p2p-overdraw-"+sender.String()), ledger.SpendByUser{})
	if _, rejected := overdraw.(ledger.SendRejected); !rejected {
		t.Fatalf("overdraw = %#v, want rejected", overdraw)
	}
	if mustBalance(t, db.NewLedgerStore(pool), sender).Spendable() != 100 {
		t.Fatalf("sender balance moved on a rejected overdraw")
	}

	selfSend := service.SendCredits(context.Background(), sender, ledger.TransferFromSelf{}, ledger.TransferToUser{ID: sender},
		mustTransferAmount(t, 5), ledger.NoTransferNote{}, mustTransferKey(t, "p2p-self-"+sender.String()), ledger.SpendByUser{})
	if _, rejected := selfSend.(ledger.SendRejected); !rejected {
		t.Fatalf("self send = %#v, want rejected", selfSend)
	}
}

// TestPeerTransferOrganizationPaths covers both organization directions:
// user-to-organization deposits into the organization wallet (notifying its
// owner/admin/billing members), and organization-to-user requires the
// acting member to hold the billing permission.
func TestPeerTransferOrganizationPaths(t *testing.T) {
	pool := newPool(t)
	service := newDBLedgerService(pool)
	store := db.NewLedgerStore(pool)
	organizationID := createOrganization(t, pool, "p2p-org")
	billingUser := createUser(t, pool, "p2p-org-billing")
	plainMember := createUser(t, pool, "p2p-org-member")
	outsider := createUser(t, pool, "p2p-org-outsider")
	addOrganizationMember(t, pool, organizationID, billingUser, "billing")
	addOrganizationMember(t, pool, organizationID, plainMember, "member")

	orgBalanceBefore := mustOrgSpendable(t, store, organizationID)

	// user -> organization: the depositor needs no membership at all.
	deposit := service.SendCredits(context.Background(), outsider, ledger.TransferFromSelf{}, ledger.TransferToOrganization{ID: organizationID},
		mustTransferAmount(t, 40), ledger.NoTransferNote{}, mustTransferKey(t, "p2p-deposit-"+outsider.String()), ledger.SpendByUser{})
	deposited, depositMatched := deposit.(ledger.CreditsSent)
	if !depositMatched {
		t.Fatalf("deposit rejected: %#v", deposit)
	}
	if mustOrgSpendable(t, store, organizationID) != orgBalanceBefore+40 {
		t.Fatalf("organization balance did not rise by the deposit")
	}
	if mustBalance(t, store, outsider).Spendable() != 60 {
		t.Fatalf("depositor balance = %d, want 60", mustBalance(t, store, outsider).Spendable())
	}
	// The billing member is among the notified receivers.
	foundBilling := false
	for _, receiver := range deposited.ReceiverUserIDs {
		if receiver == billingUser {
			foundBilling = true
		}
		if receiver == plainMember {
			t.Fatalf("plain member must not be notified of an organization deposit")
		}
	}
	if !foundBilling {
		t.Fatalf("billing member missing from deposit receivers: %#v", deposited.ReceiverUserIDs)
	}

	// organization -> user: a plain member is refused, the billing member
	// succeeds.
	denied := service.SendCredits(context.Background(), plainMember, ledger.TransferFromOrganization{ID: organizationID}, ledger.TransferToUser{ID: outsider},
		mustTransferAmount(t, 10), ledger.NoTransferNote{}, mustTransferKey(t, "p2p-org-denied-"+plainMember.String()), ledger.SpendByUser{})
	deniedRejected, deniedMatched := denied.(ledger.SendRejected)
	if !deniedMatched {
		t.Fatalf("plain-member org send = %#v, want rejected", denied)
	}
	if deniedRejected.Reason.Code() != core.ErrorCodePermissionDenied {
		t.Fatalf("plain-member org send code = %v, want permission denied", deniedRejected.Reason.Code())
	}

	orgSend := service.SendCredits(context.Background(), billingUser, ledger.TransferFromOrganization{ID: organizationID}, ledger.TransferToUser{ID: outsider},
		mustTransferAmount(t, 15), ledger.NoTransferNote{}, mustTransferKey(t, "p2p-org-send-"+billingUser.String()), ledger.SpendByUser{})
	orgSent, orgSentMatched := orgSend.(ledger.CreditsSent)
	if !orgSentMatched {
		t.Fatalf("billing-member org send rejected: %#v", orgSend)
	}
	if mustOrgSpendable(t, store, organizationID) != orgBalanceBefore+40-15 {
		t.Fatalf("organization balance did not drop by the org send")
	}
	if mustBalance(t, store, outsider).Spendable() != 75 {
		t.Fatalf("outsider balance = %d, want 75", mustBalance(t, store, outsider).Spendable())
	}
	if len(orgSent.ReceiverUserIDs) != 1 || orgSent.ReceiverUserIDs[0] != outsider {
		t.Fatalf("org send receivers = %#v, want the receiving user", orgSent.ReceiverUserIDs)
	}

	// organization -> organization is rejected outright.
	otherOrganization := createOrganization(t, pool, "p2p-org-other")
	orgToOrg := service.SendCredits(context.Background(), billingUser, ledger.TransferFromOrganization{ID: organizationID}, ledger.TransferToOrganization{ID: otherOrganization},
		mustTransferAmount(t, 5), ledger.NoTransferNote{}, mustTransferKey(t, "p2p-orgorg-"+billingUser.String()), ledger.SpendByUser{})
	if _, rejected := orgToOrg.(ledger.SendRejected); !rejected {
		t.Fatalf("org-to-org send = %#v, want rejected", orgToOrg)
	}
}

func mustOrgSpendable(t *testing.T, store db.LedgerStore, organizationID core.OrganizationID) int64 {
	t.Helper()
	found, matched := store.OrganizationBalance(context.Background(), organizationID).(ledger.BalanceFound)
	if !matched {
		t.Fatalf("organization balance rejected")
	}
	return found.Value.Spendable()
}
