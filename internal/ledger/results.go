package ledger

import (
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/event"
)

// Execution says whether a store command performed its mutation now or
// recognized an idempotent replay of an earlier execution. Replays record no
// new domain events (the original execution already recorded them), so the
// service skips the dispatch step.
type Execution interface {
	execution()
}

type FirstExecution struct{}

type IdempotentReplay struct{}

func (FirstExecution) execution() {}

func (IdempotentReplay) execution() {}

type FundResult interface {
	fundResult()
}

type TaskFunded struct {
	Fund TaskFund
	// Execution distinguishes a fresh fund from an idempotent replay.
	Execution Execution
	// RecordedEvents are the event drafts the store recorded inside the fund
	// transaction (with the store-resolved recipients merged in); the service
	// dispatches them after commit. Empty on a replay.
	RecordedEvents []event.Draft
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
	WorkerUserID   core.UserID
	CollectibleIDs []core.CollectibleID
}

type BundlePayout struct {
	WorkerUserID   core.UserID
	Amount         CreditAmount
	CollectibleIDs []core.CollectibleID
}

func (NoPayout) payoutOutcome() {}

func (CreditPayout) payoutOutcome() {}

func (CollectiblePayout) payoutOutcome() {}

func (BundlePayout) payoutOutcome() {}

type AcceptResult interface {
	acceptResult()
}

type SubmissionAccepted struct {
	TaskID       core.TaskID
	SubmissionID core.SubmissionID
	// WorkerUserID is the reviewed submission's author, present even when the
	// review moved no credits (NoPayout/NoTip), so review events always know
	// who to notify.
	WorkerUserID core.UserID
	Payout       PayoutOutcome
	Tip          TipOutcome
	// Execution distinguishes a fresh accept from an idempotent replay.
	Execution Execution
	// RecordedEvents are the drafts recorded inside the accept transaction:
	// the accepted-review event, the payout/tip events the review produced,
	// and one submission_superseded event per competing submission the close
	// superseded. Empty on a replay.
	RecordedEvents []event.Draft
}

type AcceptRejected struct {
	Reason core.DomainError
}

func (SubmissionAccepted) acceptResult() {}

func (AcceptRejected) acceptResult() {}

type TipOutcome interface {
	tipOutcome()
}

type NoTip struct{}

type CreditTip struct {
	WorkerUserID core.UserID
	Amount       CreditAmount
}

type CollectibleTip struct {
	WorkerUserID  core.UserID
	CollectibleID core.CollectibleID
}

type BundleTip struct {
	WorkerUserID  core.UserID
	Amount        CreditAmount
	CollectibleID core.CollectibleID
}

func (NoTip) tipOutcome() {}

func (CreditTip) tipOutcome() {}

func (CollectibleTip) tipOutcome() {}

func (BundleTip) tipOutcome() {}

type RequestChangesResult interface {
	requestChangesResult()
}

type ChangesRequested struct {
	TaskID       core.TaskID
	SubmissionID core.SubmissionID
	// WorkerUserID is the reviewed submission's author (see SubmissionAccepted).
	WorkerUserID core.UserID
	ReviewNote   string
	// Execution distinguishes a fresh request from an idempotent replay.
	Execution Execution
	// RecordedEvents are the drafts recorded inside the transaction; empty
	// on a replay.
	RecordedEvents []event.Draft
}

type RequestChangesRejected struct {
	Reason core.DomainError
}

func (ChangesRequested) requestChangesResult() {}

func (RequestChangesRejected) requestChangesResult() {}

type RejectResult interface {
	rejectResult()
}

type SubmissionRejected struct {
	TaskID       core.TaskID
	SubmissionID core.SubmissionID
	// WorkerUserID is the reviewed submission's author (see SubmissionAccepted).
	WorkerUserID core.UserID
	Payout       PayoutOutcome
	Tip          TipOutcome
	// Execution distinguishes a fresh reject from an idempotent replay.
	Execution Execution
	// RecordedEvents are the drafts recorded inside the reject transaction
	// (the rejected-review event plus the partial payout/tip events); empty
	// on a replay.
	RecordedEvents []event.Draft
}

type RejectRejected struct {
	Reason core.DomainError
}

func (SubmissionRejected) rejectResult() {}

func (RejectRejected) rejectResult() {}

type RefundResult interface {
	refundResult()
}

type TaskRefunded struct {
	Fund TaskFund
	// Execution distinguishes a fresh refund from an idempotent replay.
	Execution Execution
	// RecordedEvents are the drafts recorded inside the refund transaction
	// (the task_cancelled event, with the released reservation holders the
	// store resolved merged into its recipients); empty on a replay.
	RecordedEvents []event.Draft
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

// TaskAllocatedResult reports the credits currently allocated (locked) to a
// single task via the stateless task_funds store - 0 when the task holds no
// credit funding.
type TaskAllocatedResult interface {
	taskAllocatedResult()
}

type TaskAllocatedFound struct {
	Amount int64
}

type TaskAllocatedRejected struct {
	Reason core.DomainError
}

func (TaskAllocatedFound) taskAllocatedResult()    {}
func (TaskAllocatedRejected) taskAllocatedResult() {}

type ListEntriesResult interface {
	listEntriesResult()
}

type EntriesListed struct {
	Values []LedgerEntry
	// Total counts every row matching the filter, ignoring limit/offset.
	Total int64
}

type ListEntriesRejected struct {
	Reason core.DomainError
}

func (EntriesListed) listEntriesResult() {}

func (ListEntriesRejected) listEntriesResult() {}

// GrantResult is the outcome of a platform-admin manual credit grant. A
// replayed idempotency key returns the same CreditsGranted shape as the
// original grant.
type GrantResult interface {
	grantResult()
}

type CreditsGranted struct {
	EntryID core.LedgerEntryID
	Amount  CreditAmount
	// RecipientUserIDs are the beneficiaries to notify: the granted user, or
	// the grantee organization's owner/admin/billing members. The store
	// resolves them inside the grant transaction so the service does not need
	// a separate membership lookup.
	RecipientUserIDs []core.UserID
	// Execution distinguishes a fresh grant from an idempotent replay.
	Execution Execution
	// RecordedEvents are the drafts recorded inside the grant transaction
	// (the credit_granted event with the resolved beneficiaries merged into
	// its recipients); empty on a replay.
	RecordedEvents []event.Draft
}

type GrantRejected struct {
	Reason core.DomainError
}

func (CreditsGranted) grantResult() {}

func (GrantRejected) grantResult() {}
