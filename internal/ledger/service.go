package ledger

import (
	"context"

	"github.com/e6qu/sharecrop/internal/audit"
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
	// Draft is the task_funded event recorded inside the fund transaction.
	Draft event.Draft
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
	// Draft is the submission_accepted event recorded inside the accept
	// transaction; the store merges the worker into its recipients and
	// derives the payout/tip and submission_superseded events from it.
	Draft event.Draft
}

type RequestChangesStoreCommand struct {
	RequesterUserID core.UserID
	TaskID          core.TaskID
	SubmissionID    core.SubmissionID
	IdempotencyKey  IdempotencyKey
	ReviewNote      submission.ReviewNote
	// Draft is the submission_changes_requested event recorded inside the
	// transaction; the store merges the worker into its recipients.
	Draft event.Draft
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
	// Draft is the submission_rejected event recorded inside the reject
	// transaction; the store merges the worker into its recipients and
	// derives the partial payout/tip events from it.
	Draft event.Draft
}

// RefundStoreCommand carries a validated task-refund request to the store.
type RefundStoreCommand struct {
	EntryID         core.LedgerEntryID
	RequesterUserID core.UserID
	TaskID          core.TaskID
	IdempotencyKey  IdempotencyKey
	// Draft is the task_cancelled (refund cause) event recorded inside the
	// refund transaction; the store merges the released reservation holders
	// into its recipients so workers learn the task is gone.
	Draft event.Draft
}

// OrganizationFundStoreCommand carries a validated organization task-funding
// request to the store.
type OrganizationFundStoreCommand struct {
	EntryID        core.LedgerEntryID
	OrganizationID core.OrganizationID
	TaskID         core.TaskID
	Amount         CreditAmount
	IdempotencyKey IdempotencyKey
	// ActingUserID is the organization member who initiated the funding, so
	// the recorded event and audit trail carry the real actor.
	ActingUserID core.UserID
	// Draft is the task_funded event recorded inside the fund transaction;
	// the store merges the task owner into its recipients.
	Draft event.Draft
}

// GrantStoreCommand carries a validated platform-admin credit grant to the
// store: a manual_adjustment ledger entry crediting the target account's
// spendable balance.
type GrantStoreCommand struct {
	EntryID        core.LedgerEntryID
	Target         GrantTarget
	Amount         CreditAmount
	Note           GrantNote
	IdempotencyKey IdempotencyKey
	// Draft is the credit_granted event recorded inside the grant
	// transaction; the store merges the resolved beneficiaries into its
	// recipients.
	Draft event.Draft
}

type Store interface {
	FundTask(context.Context, FundStoreCommand) FundResult
	FundTaskFromOrganization(context.Context, OrganizationFundStoreCommand) FundResult
	AcceptSubmission(context.Context, AcceptStoreCommand) AcceptResult
	RequestChanges(context.Context, RequestChangesStoreCommand) RequestChangesResult
	RejectSubmission(context.Context, RejectStoreCommand) RejectResult
	RefundTask(context.Context, RefundStoreCommand) RefundResult
	GrantCredits(context.Context, GrantStoreCommand) GrantResult
	TaskAllocatedCredits(context.Context, core.TaskID) TaskAllocatedResult
	Balance(context.Context, core.UserID) BalanceResult
	OrganizationBalance(context.Context, core.OrganizationID) BalanceResult
	ListEntries(context.Context, core.UserID, core.Page) ListEntriesResult
	ListOrganizationEntries(context.Context, core.OrganizationID, core.Page) ListEntriesResult
}

// AuditRecorder records an audit event for an admin ledger action. audit.Service
// satisfies it; the indirection keeps ledger's store surface free of the audit
// store.
type AuditRecorder interface {
	Record(context.Context, core.UserID, audit.Action, audit.Subject, audit.Metadata) audit.RecordResult
}

type Service struct {
	store    Store
	recorder event.Recorder
	audit    AuditRecorder
}

func NewService(store Store, recorder event.Recorder, auditRecorder AuditRecorder) Service {
	return Service{store: store, recorder: recorder, audit: auditRecorder}
}

// ledgerDraft builds the event draft a ledger mutation records inside its
// store transaction. The store merges recipients only it can resolve (the
// worker, the task owner, released reservation holders) before recording.
func ledgerDraft(kind event.Kind, actor event.Actor, subject event.Subject, metadata event.Metadata, recipients event.Recipients) (event.Draft, *core.DomainError) {
	draftResult := event.NewDraft(kind, actor, subject, metadata, recipients)
	created, matched := draftResult.(event.DraftCreated)
	if !matched {
		reason := draftResult.(event.DraftRejected).Reason
		return event.Draft{}, &reason
	}
	return created.Value, nil
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

func (service Service) FundTask(ctx context.Context, funder core.UserID, taskID core.TaskID, amount CreditAmount, key IdempotencyKey) FundResult {
	entryResult := core.NewLedgerEntryID()
	entryCreated, matched := entryResult.(core.LedgerEntryIDCreated)
	if !matched {
		rejected := entryResult.(core.LedgerEntryIDRejected)
		return FundRejected{Reason: rejected.Reason}
	}

	// A user fund requires the funder to own the task, so the funder is the
	// full recipient set. The draft is recorded inside the fund transaction;
	// an idempotent replay records nothing and dispatches nothing.
	draft, draftProblem := ledgerDraft(event.KindTaskFunded, event.ActorUser{ID: funder}, taskSubject(taskID),
		event.TaskAmountMetadata(taskID, amount.Int64()), event.NewRecipients(funder))
	if draftProblem != nil {
		return FundRejected{Reason: *draftProblem}
	}

	result := service.store.FundTask(ctx, FundStoreCommand{
		EntryID:        entryCreated.Value,
		FunderUserID:   funder,
		TaskID:         taskID,
		Amount:         amount,
		IdempotencyKey: key,
		Draft:          draft,
	})
	if funded, matched := result.(TaskFunded); matched {
		service.recorder.Dispatch(ctx, funded.RecordedEvents...)
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

	// The worker is resolved inside the store transaction and merged into
	// the draft's recipients there; the payout/tip and submission_superseded
	// events are derived from this draft in the same transaction.
	draft, draftProblem := ledgerDraft(event.KindSubmissionAccepted, event.ActorUser{ID: requester},
		reviewSubject(taskID, submissionID), event.TaskMetadata(taskID), event.NewRecipients(requester))
	if draftProblem != nil {
		return AcceptRejected{Reason: *draftProblem}
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
		Draft:            draft,
	})
	if accepted, matched := result.(SubmissionAccepted); matched {
		service.recorder.Dispatch(ctx, accepted.RecordedEvents...)
	}
	return result
}

func (service Service) RequestChanges(ctx context.Context, requester core.UserID, taskID core.TaskID, submissionID core.SubmissionID, key IdempotencyKey, note submission.ReviewNote) RequestChangesResult {
	draft, draftProblem := ledgerDraft(event.KindSubmissionChangesRequested, event.ActorUser{ID: requester},
		reviewSubject(taskID, submissionID), event.TaskMetadata(taskID), event.NewRecipients(requester))
	if draftProblem != nil {
		return RequestChangesRejected{Reason: *draftProblem}
	}

	result := service.store.RequestChanges(ctx, RequestChangesStoreCommand{
		RequesterUserID: requester,
		TaskID:          taskID,
		SubmissionID:    submissionID,
		IdempotencyKey:  key,
		ReviewNote:      note,
		Draft:           draft,
	})
	if changed, matched := result.(ChangesRequested); matched {
		service.recorder.Dispatch(ctx, changed.RecordedEvents...)
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
	draft, draftProblem := ledgerDraft(event.KindSubmissionRejected, event.ActorUser{ID: requester},
		reviewSubject(taskID, submissionID), event.TaskMetadata(taskID), event.NewRecipients(requester))
	if draftProblem != nil {
		return RejectRejected{Reason: *draftProblem}
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
		Draft:            draft,
	})
	if rejected, matched := result.(SubmissionRejected); matched {
		service.recorder.Dispatch(ctx, rejected.RecordedEvents...)
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

	// A refund cancels the task; the metadata marks the refund cause. The
	// store merges the released active-reservation holders into the draft's
	// recipients inside the refund transaction, so workers learn the task is
	// gone.
	draft, draftProblem := ledgerDraft(event.KindTaskCancelled, event.ActorUser{ID: requester}, taskSubject(taskID),
		event.TaskRefundMetadata(taskID), event.NewRecipients(requester))
	if draftProblem != nil {
		return RefundRejected{Reason: *draftProblem}
	}

	result := service.store.RefundTask(ctx, RefundStoreCommand{
		EntryID:         entryCreated.Value,
		RequesterUserID: requester,
		TaskID:          taskID,
		IdempotencyKey:  key,
		Draft:           draft,
	})
	if refunded, matched := result.(TaskRefunded); matched {
		service.recorder.Dispatch(ctx, refunded.RecordedEvents...)
	}
	return result
}

// FundTaskFromOrganization escrows organization credits onto a task on
// behalf of actingUser (the organization member who initiated the funding,
// verified by the API boundary). The recorded event names that user as its
// actor, and the store merges the task owner into the recipients inside the
// fund transaction.
func (service Service) FundTaskFromOrganization(ctx context.Context, actingUser core.UserID, organizationID core.OrganizationID, taskID core.TaskID, amount CreditAmount, key IdempotencyKey) FundResult {
	entryResult := core.NewLedgerEntryID()
	entryCreated, matched := entryResult.(core.LedgerEntryIDCreated)
	if !matched {
		return FundRejected{Reason: entryResult.(core.LedgerEntryIDRejected).Reason}
	}

	subject := taskSubject(taskID)
	subject.Organization = event.OrganizationSubject{ID: organizationID}
	draft, draftProblem := ledgerDraft(event.KindTaskFunded, event.ActorUser{ID: actingUser}, subject,
		event.TaskAmountMetadata(taskID, amount.Int64()), event.NewRecipients(actingUser))
	if draftProblem != nil {
		return FundRejected{Reason: *draftProblem}
	}

	result := service.store.FundTaskFromOrganization(ctx, OrganizationFundStoreCommand{
		EntryID:        entryCreated.Value,
		OrganizationID: organizationID,
		TaskID:         taskID,
		Amount:         amount,
		IdempotencyKey: key,
		ActingUserID:   actingUser,
		Draft:          draft,
	})
	if funded, fundedMatched := result.(TaskFunded); fundedMatched {
		service.recorder.Dispatch(ctx, funded.RecordedEvents...)
	}
	return result
}

// GrantCredits writes a platform-admin manual_adjustment ledger entry that
// credits the target account's spendable balance. The caller (REST / MCP
// handler) is responsible for verifying the actor is a platform admin before
// calling. The credit_granted event (to the beneficiaries — the granted
// user, or the grantee organization's owner/admin/billing members — plus the
// acting admin's own feed) is recorded inside the grant transaction and
// dispatched after commit; the admin audit entry stays post-commit
// best-effort.
func (service Service) GrantCredits(ctx context.Context, actingAdmin core.UserID, target GrantTarget, amount CreditAmount, note GrantNote, key IdempotencyKey) GrantResult {
	subject := event.NoSubjectRefs()
	auditSubject := audit.Subject{}
	switch typed := target.(type) {
	case GrantToUser:
		auditSubject = audit.Subject{Kind: "user", ID: typed.ID.String()}
	case GrantToOrganization:
		subject.Organization = event.OrganizationSubject{ID: typed.ID}
		auditSubject = audit.Subject{Kind: "organization", ID: typed.ID.String()}
	default:
		return GrantRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "credit grant target is invalid")}
	}

	entryResult := core.NewLedgerEntryID()
	entryCreated, matched := entryResult.(core.LedgerEntryIDCreated)
	if !matched {
		return GrantRejected{Reason: entryResult.(core.LedgerEntryIDRejected).Reason}
	}

	// The store resolves the beneficiaries (the granted user or the grantee
	// organization's owner/admin/billing members) inside the grant
	// transaction and merges them into this draft's recipients; a replayed
	// idempotency key records and dispatches nothing.
	draft, draftProblem := ledgerDraft(event.KindCreditGranted, event.ActorUser{ID: actingAdmin}, subject,
		event.AmountMetadata(amount.Int64()), event.NewRecipients(actingAdmin))
	if draftProblem != nil {
		return GrantRejected{Reason: *draftProblem}
	}

	result := service.store.GrantCredits(ctx, GrantStoreCommand{
		EntryID:        entryCreated.Value,
		Target:         target,
		Amount:         amount,
		Note:           note,
		IdempotencyKey: key,
		Draft:          draft,
	})
	granted, grantedMatched := result.(CreditsGranted)
	if !grantedMatched {
		return result
	}

	service.recorder.Dispatch(ctx, granted.RecordedEvents...)
	_ = service.audit.Record(ctx, actingAdmin, audit.ActionAdminCreditGranted, auditSubject, audit.Metadata{JSON: event.AmountMetadata(amount.Int64()).JSON})
	return granted
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
