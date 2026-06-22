package ledger

import (
	"context"

	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/submission"
)

// FundStoreCommand carries a validated task-funding request to the store.
type FundStoreCommand struct {
	EntryID        core.LedgerEntryID
	FunderUserID   core.UserID
	TaskID         core.TaskID
	Amount         CreditAmount
	IdempotencyKey IdempotencyKey
}

// AcceptStoreCommand carries a validated submission-acceptance request to the store.
type AcceptStoreCommand struct {
	PayoutEntryID    core.LedgerEntryID
	RefundEntryID    core.LedgerEntryID
	TipDebitEntryID  core.LedgerEntryID
	TipCreditEntryID core.LedgerEntryID
	RequesterUserID  core.UserID
	TaskID           core.TaskID
	SubmissionID     core.SubmissionID
	IdempotencyKey   IdempotencyKey
	CreditSelection  CreditReviewSelection
	TipSelection     TipSelection
}

type RequestChangesStoreCommand struct {
	RequesterUserID core.UserID
	TaskID          core.TaskID
	SubmissionID    core.SubmissionID
	ReviewNote      submission.ReviewNote
}

type RejectStoreCommand struct {
	PayoutEntryID    core.LedgerEntryID
	TipDebitEntryID  core.LedgerEntryID
	TipCreditEntryID core.LedgerEntryID
	RequesterUserID  core.UserID
	TaskID           core.TaskID
	SubmissionID     core.SubmissionID
	IdempotencyKey   IdempotencyKey
	ReviewNote       submission.ReviewNote
	CreditSelection  CreditReviewSelection
	TipSelection     TipSelection
	BanSelection     BanSelection
}

// RefundStoreCommand carries a validated task-refund request to the store.
type RefundStoreCommand struct {
	EntryID         core.LedgerEntryID
	RequesterUserID core.UserID
	TaskID          core.TaskID
	IdempotencyKey  IdempotencyKey
}

// OrganizationFundStoreCommand carries a validated organization task-funding
// request to the store.
type OrganizationFundStoreCommand struct {
	EntryID        core.LedgerEntryID
	OrganizationID core.OrganizationID
	TaskID         core.TaskID
	Amount         CreditAmount
	IdempotencyKey IdempotencyKey
}

type Store interface {
	FundTask(context.Context, FundStoreCommand) FundResult
	FundTaskFromOrganization(context.Context, OrganizationFundStoreCommand) FundResult
	AcceptSubmission(context.Context, AcceptStoreCommand) AcceptResult
	RequestChanges(context.Context, RequestChangesStoreCommand) RequestChangesResult
	RejectSubmission(context.Context, RejectStoreCommand) RejectResult
	RefundTask(context.Context, RefundStoreCommand) RefundResult
	Balance(context.Context, core.UserID) BalanceResult
	OrganizationBalance(context.Context, core.OrganizationID) BalanceResult
	ListEntries(context.Context, core.UserID) ListEntriesResult
}

type Service struct {
	store Store
}

func NewService(store Store) Service {
	return Service{store: store}
}

func (service Service) FundTask(ctx context.Context, funder core.UserID, taskID core.TaskID, amount CreditAmount, key IdempotencyKey) FundResult {
	entryResult := core.NewLedgerEntryID()
	entryCreated, matched := entryResult.(core.LedgerEntryIDCreated)
	if !matched {
		rejected := entryResult.(core.LedgerEntryIDRejected)
		return FundRejected{Reason: rejected.Reason}
	}

	return service.store.FundTask(ctx, FundStoreCommand{
		EntryID:        entryCreated.Value,
		FunderUserID:   funder,
		TaskID:         taskID,
		Amount:         amount,
		IdempotencyKey: key,
	})
}

func (service Service) AcceptSubmission(ctx context.Context, requester core.UserID, taskID core.TaskID, submissionID core.SubmissionID, key IdempotencyKey) AcceptResult {
	return service.ReviewAcceptSubmission(ctx, requester, taskID, submissionID, key, FullCreditReviewSelection{}, NoTipSelection{})
}

func (service Service) ReviewAcceptSubmission(ctx context.Context, requester core.UserID, taskID core.TaskID, submissionID core.SubmissionID, key IdempotencyKey, creditSelection CreditReviewSelection, tipSelection TipSelection) AcceptResult {
	payoutEntryID, idResult := newLedgerEntryID()
	if rejected, matched := idResult.(ledgerEntryIDRejected); matched {
		return AcceptRejected{Reason: rejected.reason}
	}
	refundEntryID, idResult := newLedgerEntryID()
	if rejected, matched := idResult.(ledgerEntryIDRejected); matched {
		return AcceptRejected{Reason: rejected.reason}
	}
	tipDebitEntryID, idResult := newLedgerEntryID()
	if rejected, matched := idResult.(ledgerEntryIDRejected); matched {
		return AcceptRejected{Reason: rejected.reason}
	}
	tipCreditEntryID, idResult := newLedgerEntryID()
	if rejected, matched := idResult.(ledgerEntryIDRejected); matched {
		return AcceptRejected{Reason: rejected.reason}
	}

	return service.store.AcceptSubmission(ctx, AcceptStoreCommand{
		PayoutEntryID:    payoutEntryID,
		RefundEntryID:    refundEntryID,
		TipDebitEntryID:  tipDebitEntryID,
		TipCreditEntryID: tipCreditEntryID,
		RequesterUserID:  requester,
		TaskID:           taskID,
		SubmissionID:     submissionID,
		IdempotencyKey:   key,
		CreditSelection:  creditSelection,
		TipSelection:     tipSelection,
	})
}

func (service Service) RequestChanges(ctx context.Context, requester core.UserID, taskID core.TaskID, submissionID core.SubmissionID, note submission.ReviewNote) RequestChangesResult {
	return service.store.RequestChanges(ctx, RequestChangesStoreCommand{
		RequesterUserID: requester,
		TaskID:          taskID,
		SubmissionID:    submissionID,
		ReviewNote:      note,
	})
}

func (service Service) RejectSubmission(ctx context.Context, requester core.UserID, taskID core.TaskID, submissionID core.SubmissionID, key IdempotencyKey, note submission.ReviewNote, creditSelection CreditReviewSelection, tipSelection TipSelection, banSelection BanSelection) RejectResult {
	payoutEntryID, idResult := newLedgerEntryID()
	if rejected, matched := idResult.(ledgerEntryIDRejected); matched {
		return RejectRejected{Reason: rejected.reason}
	}
	tipDebitEntryID, idResult := newLedgerEntryID()
	if rejected, matched := idResult.(ledgerEntryIDRejected); matched {
		return RejectRejected{Reason: rejected.reason}
	}
	tipCreditEntryID, idResult := newLedgerEntryID()
	if rejected, matched := idResult.(ledgerEntryIDRejected); matched {
		return RejectRejected{Reason: rejected.reason}
	}
	return service.store.RejectSubmission(ctx, RejectStoreCommand{
		PayoutEntryID:    payoutEntryID,
		TipDebitEntryID:  tipDebitEntryID,
		TipCreditEntryID: tipCreditEntryID,
		RequesterUserID:  requester,
		TaskID:           taskID,
		SubmissionID:     submissionID,
		IdempotencyKey:   key,
		ReviewNote:       note,
		CreditSelection:  creditSelection,
		TipSelection:     tipSelection,
		BanSelection:     banSelection,
	})
}

type ledgerEntryIDResult interface {
	ledgerEntryIDResult()
}

type ledgerEntryIDAccepted struct {
	value core.LedgerEntryID
}

type ledgerEntryIDRejected struct {
	reason core.DomainError
}

func (ledgerEntryIDAccepted) ledgerEntryIDResult() {}

func (ledgerEntryIDRejected) ledgerEntryIDResult() {}

func newLedgerEntryID() (core.LedgerEntryID, ledgerEntryIDResult) {
	entryResult := core.NewLedgerEntryID()
	entryCreated, matched := entryResult.(core.LedgerEntryIDCreated)
	if !matched {
		rejected := entryResult.(core.LedgerEntryIDRejected)
		return core.LedgerEntryID{}, ledgerEntryIDRejected{reason: rejected.Reason}
	}
	return entryCreated.Value, ledgerEntryIDAccepted{value: entryCreated.Value}
}

func (service Service) RefundTask(ctx context.Context, requester core.UserID, taskID core.TaskID, key IdempotencyKey) RefundResult {
	entryResult := core.NewLedgerEntryID()
	entryCreated, matched := entryResult.(core.LedgerEntryIDCreated)
	if !matched {
		rejected := entryResult.(core.LedgerEntryIDRejected)
		return RefundRejected{Reason: rejected.Reason}
	}

	return service.store.RefundTask(ctx, RefundStoreCommand{
		EntryID:         entryCreated.Value,
		RequesterUserID: requester,
		TaskID:          taskID,
		IdempotencyKey:  key,
	})
}

func (service Service) FundTaskFromOrganization(ctx context.Context, organizationID core.OrganizationID, taskID core.TaskID, amount CreditAmount, key IdempotencyKey) FundResult {
	entryResult := core.NewLedgerEntryID()
	entryCreated, matched := entryResult.(core.LedgerEntryIDCreated)
	if !matched {
		return FundRejected{Reason: entryResult.(core.LedgerEntryIDRejected).Reason}
	}

	return service.store.FundTaskFromOrganization(ctx, OrganizationFundStoreCommand{
		EntryID:        entryCreated.Value,
		OrganizationID: organizationID,
		TaskID:         taskID,
		Amount:         amount,
		IdempotencyKey: key,
	})
}

func (service Service) Balance(ctx context.Context, owner core.UserID) BalanceResult {
	return service.store.Balance(ctx, owner)
}

func (service Service) OrganizationBalance(ctx context.Context, organizationID core.OrganizationID) BalanceResult {
	return service.store.OrganizationBalance(ctx, organizationID)
}

func (service Service) ListEntries(ctx context.Context, owner core.UserID) ListEntriesResult {
	return service.store.ListEntries(ctx, owner)
}
