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

// Balance is the derived total of an account's ledger entries.
type Balance struct {
	value int64
}

func NewBalance(value int64) Balance {
	return Balance{value: value}
}

func (balance Balance) Int64() int64 {
	return balance.value
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
	EntryKindTaskEscrow       = EntryKind{value: "task_escrow"}
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
	case EntryKindTaskEscrow.value:
		return EntryKindAccepted{Value: EntryKindTaskEscrow}
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

// EscrowState is the lifecycle of a task escrow.
type EscrowState struct {
	value string
}

var (
	EscrowStateHeld     = EscrowState{value: "held"}
	EscrowStateReleased = EscrowState{value: "released"}
	EscrowStateRefunded = EscrowState{value: "refunded"}
)

type EscrowStateResult interface {
	escrowStateResult()
}

type EscrowStateAccepted struct {
	Value EscrowState
}

type EscrowStateRejected struct {
	Reason core.DomainError
}

func (EscrowStateAccepted) escrowStateResult() {}

func (EscrowStateRejected) escrowStateResult() {}

func ParseEscrowState(raw string) EscrowStateResult {
	switch raw {
	case EscrowStateHeld.value:
		return EscrowStateAccepted{Value: EscrowStateHeld}
	case EscrowStateReleased.value:
		return EscrowStateAccepted{Value: EscrowStateReleased}
	case EscrowStateRefunded.value:
		return EscrowStateAccepted{Value: EscrowStateRefunded}
	default:
		return EscrowStateRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidEnum, "task escrow state is invalid")}
	}
}

func (state EscrowState) String() string {
	return state.value
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

type BanSelection interface {
	banSelection()
}

type NoBanSelection struct{}

type BanImplementorSelection struct{}

func (NoBanSelection) banSelection() {}

func (BanImplementorSelection) banSelection() {}
