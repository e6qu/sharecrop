package ledger

import "github.com/e6qu/sharecrop/internal/core"

type FundResult interface {
	fundResult()
}

type TaskFunded struct {
	Fund TaskFund
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
}

type GrantRejected struct {
	Reason core.DomainError
}

func (CreditsGranted) grantResult() {}

func (GrantRejected) grantResult() {}
