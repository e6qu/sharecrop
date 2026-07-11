package ledger

import "github.com/e6qu/sharecrop/internal/core"

// CreditAmount is a positive magnitude of Sharecrop credits in integer base units.
type CreditAmount struct {
	value int64
}

type CreditAmountResult interface {
	creditAmountResult()
}

type CreditAmountAccepted struct {
	Value CreditAmount
}

type CreditAmountRejected struct {
	Reason core.DomainError
}

func (CreditAmountAccepted) creditAmountResult() {}

func (CreditAmountRejected) creditAmountResult() {}

func NewCreditAmount(value int64) CreditAmountResult {
	if value <= 0 {
		return CreditAmountRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "credit amount must be a positive number of base units")}
	}
	return CreditAmountAccepted{Value: CreditAmount{value: value}}
}

func (amount CreditAmount) Int64() int64 {
	return amount.value
}

// SignedAmount is the effect of a single ledger entry on an account balance.
type SignedAmount struct {
	value int64
}

type SignedAmountResult interface {
	signedAmountResult()
}

type SignedAmountAccepted struct {
	Value SignedAmount
}

type SignedAmountRejected struct {
	Reason core.DomainError
}

func (SignedAmountAccepted) signedAmountResult() {}

func (SignedAmountRejected) signedAmountResult() {}

func ParseSignedAmount(value int64) SignedAmountResult {
	if value == 0 {
		return SignedAmountRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "ledger entry amount must be non-zero")}
	}
	return SignedAmountAccepted{Value: SignedAmount{value: value}}
}

func (amount SignedAmount) Int64() int64 {
	return amount.value
}

// Balance is the two-section wallet for a credit account. Spendable is the sum
// of the account's ledger entries (credits the owner may still spend).
// Allocated is the sum of credits currently locked to tasks the account funded
// (they cannot be spent until the task finishes). Total is their sum.
type Balance struct {
	spendable int64
	allocated int64
}

func NewBalance(spendable int64, allocated int64) Balance {
	return Balance{spendable: spendable, allocated: allocated}
}

func (balance Balance) Spendable() int64 {
	return balance.spendable
}

func (balance Balance) Allocated() int64 {
	return balance.allocated
}

func (balance Balance) Total() int64 {
	return balance.spendable + balance.allocated
}

// IdempotencyKey deduplicates fund, accept, and refund commands.
type IdempotencyKey struct {
	value string
}

type IdempotencyKeyResult interface {
	idempotencyKeyResult()
}

type IdempotencyKeyAccepted struct {
	Value IdempotencyKey
}

type IdempotencyKeyRejected struct {
	Reason core.DomainError
}

func (IdempotencyKeyAccepted) idempotencyKeyResult() {}

func (IdempotencyKeyRejected) idempotencyKeyResult() {}

func NewIdempotencyKey(raw string) IdempotencyKeyResult {
	if raw == "" {
		return IdempotencyKeyRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "idempotency key is required")}
	}
	if len(raw) > 200 {
		return IdempotencyKeyRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "idempotency key is too long")}
	}
	return IdempotencyKeyAccepted{Value: IdempotencyKey{value: raw}}
}

func (key IdempotencyKey) String() string {
	return key.value
}

// EntryKind is the typed reason for a ledger entry.
type EntryKind struct {
	value string
}

var (
	EntryKindSignupGrant      = EntryKind{value: "signup_grant"}
	EntryKindTaskFund         = EntryKind{value: "task_fund"}
	EntryKindTaskRefund       = EntryKind{value: "task_refund"}
	EntryKindTaskPayout       = EntryKind{value: "task_payout"}
	EntryKindTaskTip          = EntryKind{value: "task_tip"}
	EntryKindManualAdjustment = EntryKind{value: "manual_adjustment"}
)

type EntryKindResult interface {
	entryKindResult()
}

type EntryKindAccepted struct {
	Value EntryKind
}

type EntryKindRejected struct {
	Reason core.DomainError
}

func (EntryKindAccepted) entryKindResult() {}

func (EntryKindRejected) entryKindResult() {}

func ParseEntryKind(raw string) EntryKindResult {
	switch raw {
	case EntryKindSignupGrant.value:
		return EntryKindAccepted{Value: EntryKindSignupGrant}
	case EntryKindTaskFund.value:
		return EntryKindAccepted{Value: EntryKindTaskFund}
	case EntryKindTaskRefund.value:
		return EntryKindAccepted{Value: EntryKindTaskRefund}
	case EntryKindTaskPayout.value:
		return EntryKindAccepted{Value: EntryKindTaskPayout}
	case EntryKindTaskTip.value:
		return EntryKindAccepted{Value: EntryKindTaskTip}
	case EntryKindManualAdjustment.value:
		return EntryKindAccepted{Value: EntryKindManualAdjustment}
	default:
		return EntryKindRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidEnum, "ledger entry kind is invalid")}
	}
}

func (kind EntryKind) String() string {
	return kind.value
}

// SignupGrantAmount is the credit grant given to each new registered user.
func SignupGrantAmount() CreditAmount {
	return CreditAmount{value: 100}
}

type CreditReviewSelection interface {
	creditReviewSelection()
}

type FullCreditReviewSelection struct{}

type PartialCreditReviewSelection struct {
	Amount CreditAmount
}

type NoCreditReviewSelection struct{}

func (FullCreditReviewSelection) creditReviewSelection() {}

func (PartialCreditReviewSelection) creditReviewSelection() {}

func (NoCreditReviewSelection) creditReviewSelection() {}

type TipSelection interface {
	tipSelection()
}

type NoTipSelection struct{}

type CreditTipSelection struct {
	Amount CreditAmount
}

func (NoTipSelection) tipSelection() {}

func (CreditTipSelection) tipSelection() {}

type CollectibleTipSelection interface {
	collectibleTipSelection()
}

type NoCollectibleTipSelection struct{}

type CollectibleTipSelected struct {
	ID core.CollectibleID
}

func (NoCollectibleTipSelection) collectibleTipSelection() {}

func (CollectibleTipSelected) collectibleTipSelection() {}

type BanSelection interface {
	banSelection()
}

type NoBanSelection struct{}

type BanImplementorSelection struct{}

func (NoBanSelection) banSelection() {}

func (BanImplementorSelection) banSelection() {}
