package ledger

import (
	"context"

	"github.com/e6qu/sharecrop/internal/core"
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
	PayoutEntryID   core.LedgerEntryID
	RequesterUserID core.UserID
	TaskID          core.TaskID
	SubmissionID    core.SubmissionID
	IdempotencyKey  IdempotencyKey
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
	entryResult := core.NewLedgerEntryID()
	entryCreated, matched := entryResult.(core.LedgerEntryIDCreated)
	if !matched {
		rejected := entryResult.(core.LedgerEntryIDRejected)
		return AcceptRejected{Reason: rejected.Reason}
	}

	return service.store.AcceptSubmission(ctx, AcceptStoreCommand{
		PayoutEntryID:   entryCreated.Value,
		RequesterUserID: requester,
		TaskID:          taskID,
		SubmissionID:    submissionID,
		IdempotencyKey:  key,
	})
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
