package ledger

import (
	"context"

	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/event"
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
	CollectibleTip   CollectibleTipSelection
}

type RequestChangesStoreCommand struct {
	RequesterUserID core.UserID
	TaskID          core.TaskID
	SubmissionID    core.SubmissionID
	IdempotencyKey  IdempotencyKey
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
	TaskAllocatedCredits(context.Context, core.TaskID) TaskAllocatedResult
	Balance(context.Context, core.UserID) BalanceResult
	OrganizationBalance(context.Context, core.OrganizationID) BalanceResult
	ListEntries(context.Context, core.UserID, core.Page) ListEntriesResult
	ListOrganizationEntries(context.Context, core.OrganizationID, core.Page) ListEntriesResult
}

type Service struct {
	store    Store
	recorder event.Recorder
}

func NewService(store Store, recorder event.Recorder) Service {
	return Service{store: store, recorder: recorder}
}

// emitLedgerEvent emits one event after a committed ledger mutation. Emission
// is post-commit best-effort: a rejected emission never fails the operation
// that already committed.
func (service Service) emitLedgerEvent(ctx context.Context, kind event.Kind, actor event.Actor, subject event.Subject, metadata event.Metadata, recipients event.Recipients) {
	_ = service.recorder.Emit(ctx, event.EmitCommand{Kind: kind, Actor: actor, Subject: subject, Metadata: metadata, Recipients: recipients})
}

func reviewSubject(taskID core.TaskID, submissionID core.SubmissionID) event.Subject {
	subject := event.NoSubjectRefs()
	subject.Task = event.TaskSubject{ID: taskID}
	subject.Submission = event.SubmissionSubject{ID: submissionID}
	return subject
}

func taskSubject(taskID core.TaskID) event.Subject {
	subject := event.NoSubjectRefs()
	subject.Task = event.TaskSubject{ID: taskID}
	return subject
}

// emitPayoutEvents emits the payout_received / tip_received events a review
// produced. The worker is both variants' only meaningful audience; the
// reviewing requester is the actor, so their inbox stays clean (the
// notification fan-out skips the actor) while their feed still shows the
// movement.
func (service Service) emitPayoutEvents(ctx context.Context, requester core.UserID, subject event.Subject, taskID core.TaskID, payout PayoutOutcome, tip TipOutcome) {
	actor := event.ActorUser{ID: requester}
	switch typed := payout.(type) {
	case CreditPayout:
		service.emitLedgerEvent(ctx, event.KindPayoutReceived, actor, subject, event.TaskAmountMetadata(taskID, typed.Amount.Int64()), event.NewRecipients(typed.WorkerUserID, requester))
	case CollectiblePayout:
		service.emitLedgerEvent(ctx, event.KindPayoutReceived, actor, subject, event.TaskMetadata(taskID), event.NewRecipients(typed.WorkerUserID, requester))
	case BundlePayout:
		service.emitLedgerEvent(ctx, event.KindPayoutReceived, actor, subject, event.TaskAmountMetadata(taskID, typed.Amount.Int64()), event.NewRecipients(typed.WorkerUserID, requester))
	}
	switch typed := tip.(type) {
	case CreditTip:
		service.emitLedgerEvent(ctx, event.KindTipReceived, actor, subject, event.TaskAmountMetadata(taskID, typed.Amount.Int64()), event.NewRecipients(typed.WorkerUserID, requester))
	case CollectibleTip:
		service.emitLedgerEvent(ctx, event.KindTipReceived, actor, subject, event.TaskCollectibleMetadata(taskID, typed.CollectibleID), event.NewRecipients(typed.WorkerUserID, requester))
	case BundleTip:
		service.emitLedgerEvent(ctx, event.KindTipReceived, actor, subject, event.TaskAmountMetadata(taskID, typed.Amount.Int64()), event.NewRecipients(typed.WorkerUserID, requester))
	}
}

func (service Service) FundTask(ctx context.Context, funder core.UserID, taskID core.TaskID, amount CreditAmount, key IdempotencyKey) FundResult {
	entryResult := core.NewLedgerEntryID()
	entryCreated, matched := entryResult.(core.LedgerEntryIDCreated)
	if !matched {
		rejected := entryResult.(core.LedgerEntryIDRejected)
		return FundRejected{Reason: rejected.Reason}
	}

	result := service.store.FundTask(ctx, FundStoreCommand{
		EntryID:        entryCreated.Value,
		FunderUserID:   funder,
		TaskID:         taskID,
		Amount:         amount,
		IdempotencyKey: key,
	})
	if _, funded := result.(TaskFunded); funded {
		// The fund result cannot distinguish a fresh fund from an idempotent
		// replay, so a replayed request re-emits task_funded; the recipient
		// set makes the duplicate harmless (see below). The store result does
		// not carry the task owner either, so the funder (the actor, and in
		// the common self-funding flow also the owner) is the only recipient;
		// a cross-service owner lookup is deliberately avoided here.
		service.emitLedgerEvent(ctx, event.KindTaskFunded, event.ActorUser{ID: funder}, taskSubject(taskID),
			event.TaskAmountMetadata(taskID, amount.Int64()), event.NewRecipients(funder))
	}
	return result
}

func (service Service) AcceptSubmission(ctx context.Context, requester core.UserID, taskID core.TaskID, submissionID core.SubmissionID, key IdempotencyKey) AcceptResult {
	return service.ReviewAcceptSubmission(ctx, requester, taskID, submissionID, key, FullCreditReviewSelection{}, NoTipSelection{}, NoCollectibleTipSelection{})
}

func (service Service) ReviewAcceptSubmission(ctx context.Context, requester core.UserID, taskID core.TaskID, submissionID core.SubmissionID, key IdempotencyKey, creditSelection CreditReviewSelection, tipSelection TipSelection, collectibleTip CollectibleTipSelection) AcceptResult {
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

	result := service.store.AcceptSubmission(ctx, AcceptStoreCommand{
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
		CollectibleTip:   collectibleTip,
	})
	if accepted, matched := result.(SubmissionAccepted); matched {
		// The accept result cannot distinguish a fresh accept from an
		// idempotent replay, so a replay re-emits; correct retries reuse the
		// same idempotency key and simply refresh the same inbox rows.
		subject := reviewSubject(taskID, submissionID)
		service.emitLedgerEvent(ctx, event.KindSubmissionAccepted, event.ActorUser{ID: requester}, subject,
			event.TaskMetadata(taskID), event.NewRecipients(accepted.WorkerUserID, requester))
		service.emitPayoutEvents(ctx, requester, subject, taskID, accepted.Payout, accepted.Tip)
	}
	return result
}

func (service Service) RequestChanges(ctx context.Context, requester core.UserID, taskID core.TaskID, submissionID core.SubmissionID, key IdempotencyKey, note submission.ReviewNote) RequestChangesResult {
	result := service.store.RequestChanges(ctx, RequestChangesStoreCommand{
		RequesterUserID: requester,
		TaskID:          taskID,
		SubmissionID:    submissionID,
		IdempotencyKey:  key,
		ReviewNote:      note,
	})
	if changed, matched := result.(ChangesRequested); matched {
		// Fresh request and idempotent replay share the same result shape, so
		// a replay re-emits (same trade-off as ReviewAcceptSubmission).
		service.emitLedgerEvent(ctx, event.KindSubmissionChangesRequested, event.ActorUser{ID: requester},
			reviewSubject(taskID, submissionID), event.TaskMetadata(taskID), event.NewRecipients(changed.WorkerUserID, requester))
	}
	return result
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
	result := service.store.RejectSubmission(ctx, RejectStoreCommand{
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
	if rejected, matched := result.(SubmissionRejected); matched {
		// Fresh reject and idempotent replay share the same result shape, so
		// a replay re-emits (same trade-off as ReviewAcceptSubmission).
		subject := reviewSubject(taskID, submissionID)
		service.emitLedgerEvent(ctx, event.KindSubmissionRejected, event.ActorUser{ID: requester}, subject,
			event.TaskMetadata(taskID), event.NewRecipients(rejected.WorkerUserID, requester))
		service.emitPayoutEvents(ctx, requester, subject, taskID, rejected.Payout, rejected.Tip)
	}
	return result
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

	result := service.store.RefundTask(ctx, RefundStoreCommand{
		EntryID:         entryCreated.Value,
		RequesterUserID: requester,
		TaskID:          taskID,
		IdempotencyKey:  key,
	})
	if _, refunded := result.(TaskRefunded); refunded {
		// A refund cancels the task. The refund result does not expose the
		// released reservation holder, so the requesting user (owner or
		// active implementor - the store only permits those two) is the only
		// recipient; the metadata marks the cancellation's refund cause.
		service.emitLedgerEvent(ctx, event.KindTaskCancelled, event.ActorUser{ID: requester}, taskSubject(taskID),
			event.TaskRefundMetadata(taskID), event.NewRecipients(requester))
	}
	return result
}

func (service Service) FundTaskFromOrganization(ctx context.Context, organizationID core.OrganizationID, taskID core.TaskID, amount CreditAmount, key IdempotencyKey) FundResult {
	entryResult := core.NewLedgerEntryID()
	entryCreated, matched := entryResult.(core.LedgerEntryIDCreated)
	if !matched {
		return FundRejected{Reason: entryResult.(core.LedgerEntryIDRejected).Reason}
	}

	result := service.store.FundTaskFromOrganization(ctx, OrganizationFundStoreCommand{
		EntryID:        entryCreated.Value,
		OrganizationID: organizationID,
		TaskID:         taskID,
		Amount:         amount,
		IdempotencyKey: key,
	})
	if _, funded := result.(TaskFunded); funded {
		// Organization funding carries no acting user identity into this
		// service and the fund result names no task owner, so the event is
		// recorded with the system actor and an empty recipient set: it lands
		// in the durable stream (for feeds/webhooks reading by subject) but
		// fans out no inbox rows.
		service.emitLedgerEvent(ctx, event.KindTaskFunded, event.ActorSystem{}, taskSubject(taskID),
			event.TaskAmountMetadata(taskID, amount.Int64()), event.NewRecipients())
	}
	return result
}

func (service Service) Balance(ctx context.Context, owner core.UserID) BalanceResult {
	return service.store.Balance(ctx, owner)
}

func (service Service) OrganizationBalance(ctx context.Context, organizationID core.OrganizationID) BalanceResult {
	return service.store.OrganizationBalance(ctx, organizationID)
}

// TaskAllocatedCredits reports the credits currently allocated to a task
// (0 when it holds no credit funding). Used to show a task's live funding on
// its detail page and to gate the refund control.
func (service Service) TaskAllocatedCredits(ctx context.Context, taskID core.TaskID) TaskAllocatedResult {
	return service.store.TaskAllocatedCredits(ctx, taskID)
}

func (service Service) ListEntries(ctx context.Context, owner core.UserID, page core.Page) ListEntriesResult {
	return service.store.ListEntries(ctx, owner, page)
}

func (service Service) ListOrganizationEntries(ctx context.Context, organizationID core.OrganizationID, page core.Page) ListEntriesResult {
	return service.store.ListOrganizationEntries(ctx, organizationID, page)
}
