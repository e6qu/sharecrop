package ledger

import (
	"context"
	"testing"

	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/submission"
)

func TestNewCreditAmountRejectsNonPositive(t *testing.T) {
	for _, value := range []int64{0, -1, -100} {
		if _, matched := NewCreditAmount(value).(CreditAmountRejected); !matched {
			t.Fatalf("NewCreditAmount(%d) was accepted, want rejected", value)
		}
	}
}

func TestNewCreditAmountAcceptsPositive(t *testing.T) {
	accepted, matched := NewCreditAmount(250).(CreditAmountAccepted)
	if !matched {
		t.Fatalf("NewCreditAmount(250) was rejected")
	}
	if accepted.Value.Int64() != 250 {
		t.Fatalf("amount = %d, want 250", accepted.Value.Int64())
	}
}

func TestSignupGrantAmountIsHundred(t *testing.T) {
	if SignupGrantAmount().Int64() != 100 {
		t.Fatalf("signup grant = %d, want 100", SignupGrantAmount().Int64())
	}
}

func TestParseSignedAmountRejectsZero(t *testing.T) {
	if _, matched := ParseSignedAmount(0).(SignedAmountRejected); !matched {
		t.Fatalf("ParseSignedAmount(0) was accepted, want rejected")
	}
}

func TestNewIdempotencyKeyRejectsEmpty(t *testing.T) {
	if _, matched := NewIdempotencyKey("").(IdempotencyKeyRejected); !matched {
		t.Fatalf("empty idempotency key was accepted")
	}
}

func TestParseEntryKindRoundTrips(t *testing.T) {
	kinds := []EntryKind{
		EntryKindSignupGrant,
		EntryKindTaskFund,
		EntryKindTaskRefund,
		EntryKindTaskPayout,
		EntryKindManualAdjustment,
	}
	for _, kind := range kinds {
		accepted, matched := ParseEntryKind(kind.String()).(EntryKindAccepted)
		if !matched {
			t.Fatalf("ParseEntryKind(%q) was rejected", kind.String())
		}
		if accepted.Value != kind {
			t.Fatalf("parsed = %q, want %q", accepted.Value.String(), kind.String())
		}
	}
}

func TestParseEntryKindRejectsUnknown(t *testing.T) {
	if _, matched := ParseEntryKind("bonus").(EntryKindRejected); !matched {
		t.Fatalf("unknown entry kind was accepted")
	}
}

func TestDeriveSpendableSumsSignedAmounts(t *testing.T) {
	entries := []LedgerEntry{
		newTestEntry(t, 100),
		newTestEntry(t, -40),
		newTestEntry(t, 15),
	}
	if spendable := DeriveSpendable(entries); spendable != 75 {
		t.Fatalf("spendable = %d, want 75", spendable)
	}
}

func TestDeriveSpendableOfNoEntriesIsZero(t *testing.T) {
	if DeriveSpendable(nil) != 0 {
		t.Fatalf("empty spendable was not zero")
	}
}

func TestBalanceSectionsAndTotal(t *testing.T) {
	balance := NewBalance(70, 30)
	if balance.Spendable() != 70 || balance.Allocated() != 30 || balance.Total() != 100 {
		t.Fatalf("balance = (%d, %d, total %d), want (70, 30, total 100)", balance.Spendable(), balance.Allocated(), balance.Total())
	}
}

func TestServiceFundTaskGeneratesEntryAndDelegates(t *testing.T) {
	store := &memoryStore{}
	service := NewService(store)
	amount := newTestAmount(t, 50)

	result := service.FundTask(context.Background(), newTestUserID(t), newTestTaskID(t), amount, newTestKey(t, "fund-1"))
	if _, matched := result.(TaskFunded); !matched {
		t.Fatalf("result = %T, want TaskFunded", result)
	}
	if store.fundCommand.EntryID.String() == "" {
		t.Fatalf("service did not generate a ledger entry id")
	}
	if store.fundCommand.Amount.Int64() != 50 {
		t.Fatalf("amount = %d, want 50", store.fundCommand.Amount.Int64())
	}
}

func TestServiceAcceptSubmissionDelegates(t *testing.T) {
	store := &memoryStore{}
	service := NewService(store)

	result := service.AcceptSubmission(context.Background(), newTestUserID(t), newTestTaskID(t), newTestSubmissionID(t), newTestKey(t, "accept-1"))
	if _, matched := result.(SubmissionAccepted); !matched {
		t.Fatalf("result = %T, want SubmissionAccepted", result)
	}
	if store.acceptCommand.PayoutEntryID.String() == "" {
		t.Fatalf("service did not generate a payout entry id")
	}
}

func TestServiceRejectSubmissionDelegates(t *testing.T) {
	store := &memoryStore{}
	service := NewService(store)
	note := submissionNote(t, "needs current data")

	result := service.RejectSubmission(context.Background(), newTestUserID(t), newTestTaskID(t), newTestSubmissionID(t), newTestKey(t, "reject-1"), note, NoCreditReviewSelection{}, NoTipSelection{}, BanImplementorSelection{})
	if _, matched := result.(SubmissionRejected); !matched {
		t.Fatalf("result = %T, want SubmissionRejected", result)
	}
	if store.rejectCommand.PayoutEntryID.String() == "" {
		t.Fatalf("service did not generate a payout entry id")
	}
	if _, matched := store.rejectCommand.BanSelection.(BanImplementorSelection); !matched {
		t.Fatalf("ban selection = %T, want BanImplementorSelection", store.rejectCommand.BanSelection)
	}
}

type memoryStore struct {
	fundCommand   FundStoreCommand
	acceptCommand AcceptStoreCommand
	refundCommand RefundStoreCommand
	rejectCommand RejectStoreCommand
}

func (store *memoryStore) FundTask(_ context.Context, command FundStoreCommand) FundResult {
	store.fundCommand = command
	return TaskFunded{Fund: TaskFund{TaskID: command.TaskID, CreditAmount: command.Amount}}
}

func (store *memoryStore) AcceptSubmission(_ context.Context, command AcceptStoreCommand) AcceptResult {
	store.acceptCommand = command
	return SubmissionAccepted{TaskID: command.TaskID, SubmissionID: command.SubmissionID, Payout: NoPayout{}, Tip: NoTip{}}
}

func (store *memoryStore) RequestChanges(_ context.Context, command RequestChangesStoreCommand) RequestChangesResult {
	return ChangesRequested{TaskID: command.TaskID, SubmissionID: command.SubmissionID, ReviewNote: command.ReviewNote.String()}
}

func (store *memoryStore) RejectSubmission(_ context.Context, command RejectStoreCommand) RejectResult {
	store.rejectCommand = command
	return SubmissionRejected{TaskID: command.TaskID, SubmissionID: command.SubmissionID, Payout: NoPayout{}, Tip: NoTip{}}
}

func (store *memoryStore) RefundTask(_ context.Context, command RefundStoreCommand) RefundResult {
	store.refundCommand = command
	return TaskRefunded{Fund: TaskFund{TaskID: command.TaskID}}
}

func (store *memoryStore) FundTaskFromOrganization(_ context.Context, command OrganizationFundStoreCommand) FundResult {
	return TaskFunded{Fund: TaskFund{TaskID: command.TaskID, CreditAmount: command.Amount}}
}

func (store *memoryStore) TaskAllocatedCredits(_ context.Context, _ core.TaskID) TaskAllocatedResult {
	return TaskAllocatedFound{Amount: 0}
}

func (store *memoryStore) Balance(_ context.Context, _ core.UserID) BalanceResult {
	return BalanceFound{Value: NewBalance(0, 0)}
}

func (store *memoryStore) OrganizationBalance(_ context.Context, _ core.OrganizationID) BalanceResult {
	return BalanceFound{Value: NewBalance(0, 0)}
}

func (store *memoryStore) ListEntries(_ context.Context, _ core.UserID, _ core.Page) ListEntriesResult {
	return EntriesListed{Values: nil}
}

func (store *memoryStore) ListOrganizationEntries(_ context.Context, _ core.OrganizationID, _ core.Page) ListEntriesResult {
	return EntriesListed{Values: nil}
}

func newTestEntry(t *testing.T, amount int64) LedgerEntry {
	t.Helper()
	signed, matched := ParseSignedAmount(amount).(SignedAmountAccepted)
	if !matched {
		t.Fatalf("ParseSignedAmount(%d) rejected", amount)
	}
	return LedgerEntry{ID: newTestEntryID(t), Kind: EntryKindManualAdjustment, Amount: signed.Value, TaskRef: NoTaskReference{}}
}

func submissionNote(t *testing.T, raw string) submission.ReviewNote {
	t.Helper()
	accepted, matched := submission.NewRequiredReviewNote(raw).(submission.ReviewNoteAccepted)
	if !matched {
		t.Fatalf("review note rejected")
	}
	return accepted.Value
}

func newTestAmount(t *testing.T, value int64) CreditAmount {
	t.Helper()
	accepted, matched := NewCreditAmount(value).(CreditAmountAccepted)
	if !matched {
		t.Fatalf("NewCreditAmount(%d) rejected", value)
	}
	return accepted.Value
}

func newTestKey(t *testing.T, raw string) IdempotencyKey {
	t.Helper()
	accepted, matched := NewIdempotencyKey(raw).(IdempotencyKeyAccepted)
	if !matched {
		t.Fatalf("NewIdempotencyKey(%q) rejected", raw)
	}
	return accepted.Value
}

func newTestUserID(t *testing.T) core.UserID {
	t.Helper()
	created, matched := core.NewUserID().(core.UserIDCreated)
	if !matched {
		t.Fatalf("new user id rejected")
	}
	return created.Value
}

func newTestTaskID(t *testing.T) core.TaskID {
	t.Helper()
	created, matched := core.NewTaskID().(core.TaskIDCreated)
	if !matched {
		t.Fatalf("new task id rejected")
	}
	return created.Value
}

func newTestSubmissionID(t *testing.T) core.SubmissionID {
	t.Helper()
	created, matched := core.NewSubmissionID().(core.SubmissionIDCreated)
	if !matched {
		t.Fatalf("new submission id rejected")
	}
	return created.Value
}

func newTestEntryID(t *testing.T) core.LedgerEntryID {
	t.Helper()
	created, matched := core.NewLedgerEntryID().(core.LedgerEntryIDCreated)
	if !matched {
		t.Fatalf("new ledger entry id rejected")
	}
	return created.Value
}
