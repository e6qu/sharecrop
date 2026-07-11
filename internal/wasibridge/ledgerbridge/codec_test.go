package ledgerbridge

import (
	"strconv"
	"testing"

	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/ledger"
	"github.com/e6qu/sharecrop/internal/submission"
)

func amount(t *testing.T, value int64) ledger.CreditAmount {
	t.Helper()
	accepted, matched := ledger.NewCreditAmount(value).(ledger.CreditAmountAccepted)
	if !matched {
		t.Fatalf("credit amount %d rejected", value)
	}
	return accepted.Value
}

func userID(t *testing.T) core.UserID {
	t.Helper()
	created, matched := core.NewUserID().(core.UserIDCreated)
	if !matched {
		t.Fatalf("user id rejected")
	}
	return created.Value
}

func collectibleID(t *testing.T) core.CollectibleID {
	t.Helper()
	created, matched := core.NewCollectibleID().(core.CollectibleIDCreated)
	if !matched {
		t.Fatalf("collectible id rejected")
	}
	return created.Value
}

// TestPayoutOutcomeRoundTrips exercises every payout variant, since the accept
// and reject results embed this union.
func TestPayoutOutcomeRoundTrips(t *testing.T) {
	worker := userID(t)
	cases := []ledger.PayoutOutcome{
		ledger.NoPayout{},
		ledger.CreditPayout{WorkerUserID: worker, Amount: amount(t, 40)},
		ledger.CollectiblePayout{WorkerUserID: worker, CollectibleIDs: []core.CollectibleID{collectibleID(t), collectibleID(t)}},
		ledger.BundlePayout{WorkerUserID: worker, Amount: amount(t, 15), CollectibleIDs: []core.CollectibleID{collectibleID(t)}},
	}
	for _, original := range cases {
		restored, err := decodePayoutOutcome(encodePayoutOutcome(original))
		if err != nil {
			t.Fatalf("decode %T: %v", original, err)
		}
		if got, want := describePayout(restored), describePayout(original); got != want {
			t.Errorf("payout round-trip: got %s, want %s", got, want)
		}
	}
}

// TestTipOutcomeRoundTrips exercises every tip variant.
func TestTipOutcomeRoundTrips(t *testing.T) {
	worker := userID(t)
	cases := []ledger.TipOutcome{
		ledger.NoTip{},
		ledger.CreditTip{WorkerUserID: worker, Amount: amount(t, 5)},
		ledger.CollectibleTip{WorkerUserID: worker, CollectibleID: collectibleID(t)},
		ledger.BundleTip{WorkerUserID: worker, Amount: amount(t, 3), CollectibleID: collectibleID(t)},
	}
	for _, original := range cases {
		restored, err := decodeTipOutcome(encodeTipOutcome(original))
		if err != nil {
			t.Fatalf("decode %T: %v", original, err)
		}
		if got, want := describeTip(restored), describeTip(original); got != want {
			t.Errorf("tip round-trip: got %s, want %s", got, want)
		}
	}
}

func TestSelectionRoundTrips(t *testing.T) {
	credit, err := decodeCreditReviewSelection(encodeCreditReviewSelection(ledger.PartialCreditReviewSelection{Amount: amount(t, 12)}))
	if err != nil {
		t.Fatalf("decode credit selection: %v", err)
	}
	partial, matched := credit.(ledger.PartialCreditReviewSelection)
	if !matched || partial.Amount.Int64() != 12 {
		t.Errorf("partial credit selection did not round-trip: %T", credit)
	}
	if _, matched := mustCreditSelection(t, ledger.FullCreditReviewSelection{}).(ledger.FullCreditReviewSelection); !matched {
		t.Errorf("full credit selection did not round-trip")
	}

	tip, err := decodeTipSelection(encodeTipSelection(ledger.CreditTipSelection{Amount: amount(t, 7)}))
	if err != nil {
		t.Fatalf("decode tip selection: %v", err)
	}
	if creditTip, matched := tip.(ledger.CreditTipSelection); !matched || creditTip.Amount.Int64() != 7 {
		t.Errorf("credit tip selection did not round-trip: %T", tip)
	}

	collectibleTip, err := decodeCollectibleTipSelection(encodeCollectibleTipSelection(ledger.CollectibleTipSelected{ID: collectibleID(t)}))
	if err != nil {
		t.Fatalf("decode collectible tip selection: %v", err)
	}
	if _, matched := collectibleTip.(ledger.CollectibleTipSelected); !matched {
		t.Errorf("collectible tip selection did not round-trip: %T", collectibleTip)
	}

	ban, err := decodeBanSelection(encodeBanSelection(ledger.BanImplementorSelection{}))
	if err != nil {
		t.Fatalf("decode ban selection: %v", err)
	}
	if _, matched := ban.(ledger.BanImplementorSelection); !matched {
		t.Errorf("ban selection did not round-trip: %T", ban)
	}
}

func TestLedgerEntryRoundTrip(t *testing.T) {
	entryID, matched := core.NewLedgerEntryID().(core.LedgerEntryIDCreated)
	if !matched {
		t.Fatalf("entry id rejected")
	}
	signed, matched := ledger.ParseSignedAmount(-25).(ledger.SignedAmountAccepted)
	if !matched {
		t.Fatalf("signed amount rejected")
	}
	taskID, matched := core.NewTaskID().(core.TaskIDCreated)
	if !matched {
		t.Fatalf("task id rejected")
	}
	original := ledger.LedgerEntry{
		ID:      entryID.Value,
		Kind:    ledger.EntryKindTaskFund,
		Amount:  signed.Value,
		TaskRef: ledger.TaskReferenced{TaskID: taskID.Value},
	}
	restored, err := decodeLedgerEntry(encodeLedgerEntry(original))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if restored.ID != original.ID || restored.Kind.String() != "task_fund" || restored.Amount.Int64() != -25 {
		t.Errorf("ledger entry scalar fields did not round-trip: %+v", restored)
	}
	if _, matched := restored.TaskRef.(ledger.TaskReferenced); !matched {
		t.Errorf("task reference did not round-trip: %T", restored.TaskRef)
	}
}

func TestAcceptResultRoundTrip(t *testing.T) {
	worker := userID(t)
	taskID, matched := core.NewTaskID().(core.TaskIDCreated)
	if !matched {
		t.Fatalf("task id rejected")
	}
	submissionID, matched := core.NewSubmissionID().(core.SubmissionIDCreated)
	if !matched {
		t.Fatalf("submission id rejected")
	}
	original := ledger.SubmissionAccepted{
		TaskID:       taskID.Value,
		SubmissionID: submissionID.Value,
		Payout:       ledger.BundlePayout{WorkerUserID: worker, Amount: amount(t, 40), CollectibleIDs: []core.CollectibleID{collectibleID(t)}},
		Tip:          ledger.CreditTip{WorkerUserID: worker, Amount: amount(t, 5)},
	}
	restored, err := decodeAcceptResult(encodeAcceptResult(original))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	accepted, matched := restored.(ledger.SubmissionAccepted)
	if !matched {
		t.Fatalf("accept result = %T, want SubmissionAccepted", restored)
	}
	if accepted.TaskID != original.TaskID || accepted.SubmissionID != original.SubmissionID {
		t.Errorf("accept ids did not round-trip")
	}
	if describePayout(accepted.Payout) != describePayout(original.Payout) {
		t.Errorf("accept payout did not round-trip")
	}
	if describeTip(accepted.Tip) != describeTip(original.Tip) {
		t.Errorf("accept tip did not round-trip")
	}
}

func TestBalanceAndRefundResultRoundTrips(t *testing.T) {
	balance, err := decodeBalanceResult(encodeBalanceResult(ledger.BalanceFound{Value: ledger.NewBalance(60, 40)}))
	if err != nil {
		t.Fatalf("decode balance: %v", err)
	}
	found, matched := balance.(ledger.BalanceFound)
	if !matched || found.Value.Spendable() != 60 || found.Value.Allocated() != 40 {
		t.Errorf("balance did not round-trip: %+v", balance)
	}

	taskID, matched := core.NewTaskID().(core.TaskIDCreated)
	if !matched {
		t.Fatalf("task id rejected")
	}
	accountID, matched := core.NewCreditAccountID().(core.CreditAccountIDCreated)
	if !matched {
		t.Fatalf("account id rejected")
	}
	fund := ledger.TaskFund{TaskID: taskID.Value, FunderAccountID: accountID.Value, CreditAmount: amount(t, 40)}
	refund, err := decodeRefundResult(encodeRefundResult(ledger.TaskRefunded{Fund: fund}))
	if err != nil {
		t.Fatalf("decode refund: %v", err)
	}
	refunded, matched := refund.(ledger.TaskRefunded)
	if !matched || refunded.Fund.CreditAmount.Int64() != 40 || refunded.Fund.TaskID != taskID.Value {
		t.Errorf("refund did not round-trip: %+v", refund)
	}
}

func TestRejectCommandRoundTrip(t *testing.T) {
	note, matched := submission.NewStoredReviewNote("needs work").(submission.ReviewNoteAccepted)
	if !matched {
		t.Fatalf("review note rejected")
	}
	original := ledger.RejectStoreCommand{
		PayoutEntryID:    entryID(t),
		TipDebitEntryID:  entryID(t),
		TipCreditEntryID: entryID(t),
		RequesterUserID:  userID(t),
		TaskID:           taskID(t),
		SubmissionID:     submissionID(t),
		IdempotencyKey:   key(t, "reject-1"),
		ReviewNote:       note.Value,
		CreditSelection:  ledger.NoCreditReviewSelection{},
		TipSelection:     ledger.CreditTipSelection{Amount: amount(t, 3)},
		BanSelection:     ledger.BanImplementorSelection{},
	}
	restored, err := decodeRejectCommand(encodeRejectCommand(original))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if restored.ReviewNote.String() != "needs work" || restored.RequesterUserID != original.RequesterUserID {
		t.Errorf("reject command did not round-trip: %+v", restored)
	}
	if _, matched := restored.BanSelection.(ledger.BanImplementorSelection); !matched {
		t.Errorf("ban selection did not round-trip: %T", restored.BanSelection)
	}
	if creditTip, matched := restored.TipSelection.(ledger.CreditTipSelection); !matched || creditTip.Amount.Int64() != 3 {
		t.Errorf("tip selection did not round-trip: %T", restored.TipSelection)
	}
}

// ---- small builders + describers ----

func entryID(t *testing.T) core.LedgerEntryID {
	t.Helper()
	created, matched := core.NewLedgerEntryID().(core.LedgerEntryIDCreated)
	if !matched {
		t.Fatalf("entry id rejected")
	}
	return created.Value
}

func taskID(t *testing.T) core.TaskID {
	t.Helper()
	created, matched := core.NewTaskID().(core.TaskIDCreated)
	if !matched {
		t.Fatalf("task id rejected")
	}
	return created.Value
}

func submissionID(t *testing.T) core.SubmissionID {
	t.Helper()
	created, matched := core.NewSubmissionID().(core.SubmissionIDCreated)
	if !matched {
		t.Fatalf("submission id rejected")
	}
	return created.Value
}

func key(t *testing.T, raw string) ledger.IdempotencyKey {
	t.Helper()
	accepted, matched := ledger.NewIdempotencyKey(raw).(ledger.IdempotencyKeyAccepted)
	if !matched {
		t.Fatalf("idempotency key rejected")
	}
	return accepted.Value
}

func mustCreditSelection(t *testing.T, selection ledger.CreditReviewSelection) ledger.CreditReviewSelection {
	t.Helper()
	restored, err := decodeCreditReviewSelection(encodeCreditReviewSelection(selection))
	if err != nil {
		t.Fatalf("decode credit selection: %v", err)
	}
	return restored
}

func describePayout(outcome ledger.PayoutOutcome) string {
	switch typed := outcome.(type) {
	case ledger.CreditPayout:
		return "credit:" + typed.WorkerUserID.String() + ":" + itoa(typed.Amount.Int64())
	case ledger.CollectiblePayout:
		return "collectible:" + typed.WorkerUserID.String() + ":" + itoa(int64(len(typed.CollectibleIDs)))
	case ledger.BundlePayout:
		return "bundle:" + typed.WorkerUserID.String() + ":" + itoa(typed.Amount.Int64()) + ":" + itoa(int64(len(typed.CollectibleIDs)))
	default:
		return "none"
	}
}

func describeTip(outcome ledger.TipOutcome) string {
	switch typed := outcome.(type) {
	case ledger.CreditTip:
		return "credit:" + typed.WorkerUserID.String() + ":" + itoa(typed.Amount.Int64())
	case ledger.CollectibleTip:
		return "collectible:" + typed.WorkerUserID.String() + ":" + typed.CollectibleID.String()
	case ledger.BundleTip:
		return "bundle:" + typed.WorkerUserID.String() + ":" + itoa(typed.Amount.Int64()) + ":" + typed.CollectibleID.String()
	default:
		return "none"
	}
}

func itoa(value int64) string {
	return strconv.FormatInt(value, 10)
}
