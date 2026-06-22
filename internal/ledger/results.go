package ledger

import "github.com/e6qu/sharecrop/internal/core"

type FundResult interface {
	fundResult()
}

type TaskFunded struct {
	Escrow TaskEscrow
}

type FundRejected struct {
	Reason core.DomainError
}

func (TaskFunded) fundResult() {}

func (FundRejected) fundResult() {}

// PayoutOutcome records whether accepting a submission paid a credit reward.
type PayoutOutcome interface {
	payoutOutcome()
}

type NoPayout struct{}

type CreditPayout struct {
	WorkerUserID core.UserID
	Amount       CreditAmount
}

type CollectiblePayout struct {
	WorkerUserID  core.UserID
	CollectibleID core.CollectibleID
}

func (NoPayout) payoutOutcome() {}

func (CreditPayout) payoutOutcome() {}

func (CollectiblePayout) payoutOutcome() {}

type AcceptResult interface {
	acceptResult()
}

type SubmissionAccepted struct {
	TaskID       core.TaskID
	SubmissionID core.SubmissionID
	Payout       PayoutOutcome
}

type AcceptRejected struct {
	Reason core.DomainError
}

func (SubmissionAccepted) acceptResult() {}

func (AcceptRejected) acceptResult() {}

type RefundResult interface {
	refundResult()
}

type TaskRefunded struct {
	Escrow TaskEscrow
}

type RefundRejected struct {
	Reason core.DomainError
}

func (TaskRefunded) refundResult() {}

func (RefundRejected) refundResult() {}

type BalanceResult interface {
	balanceResult()
}

type BalanceFound struct {
	Value Balance
}

type BalanceRejected struct {
	Reason core.DomainError
}

func (BalanceFound) balanceResult() {}

func (BalanceRejected) balanceResult() {}

type ListEntriesResult interface {
	listEntriesResult()
}

type EntriesListed struct {
	Values []LedgerEntry
}

type ListEntriesRejected struct {
	Reason core.DomainError
}

func (EntriesListed) listEntriesResult() {}

func (ListEntriesRejected) listEntriesResult() {}
