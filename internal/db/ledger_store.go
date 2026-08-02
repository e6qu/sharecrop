package db

import (
	"context"
	"errors"
	"time"

	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/event"
	"github.com/e6qu/sharecrop/internal/ledger"
	"github.com/e6qu/sharecrop/internal/org"
	"github.com/jackc/pgx/v5/pgxpool"
)

type LedgerStore struct {
	db Beginner
}

func NewLedgerStore(pool *pgxpool.Pool) LedgerStore {
	return NewLedgerStoreFromHandle(NewPGX(pool))
}

func NewLedgerStoreFromHandle(handle Beginner) LedgerStore {
	return LedgerStore{db: handle}
}

func (store LedgerStore) FundTask(ctx context.Context, command ledger.FundStoreCommand) ledger.FundResult {
	tx, err := store.db.Begin(ctx)
	if err != nil {
		return ledger.FundRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "begin fund task transaction failed")}
	}
	defer func() { _ = tx.Rollback(ctx) }()

	accountResult := lockUserAccount(ctx, tx, command.FunderUserID)
	account, matched := accountResult.(accountLocked)
	if !matched {
		return ledger.FundRejected{Reason: accountResult.(accountLockRejected).reason}
	}

	if existing := findFundForKey(ctx, tx, command.IdempotencyKey.String(), command.TaskID, account.id); existing != nil {
		return existing
	}

	taskResult := lockTaskOwnedBy(ctx, tx, command.TaskID, command.FunderUserID, "fund")
	taskRow, taskMatched := taskResult.(taskLocked)
	if !taskMatched {
		return ledger.FundRejected{Reason: taskResult.(taskLockRejected).reason}
	}
	if rejected := requireFundableTask(ctx, tx, command.TaskID, taskRow, command.Amount); rejected != nil {
		return rejected
	}

	if problem := applySpendCharge(ctx, tx, command.Spend); problem != nil {
		recordBudgetRefusal(ctx, store.db, utcDay(time.Now()))
		return ledger.FundRejected{Reason: *problem}
	}

	return completeFunding(ctx, tx, account, command.TaskID, command.Amount, command.EntryID, command.IdempotencyKey, "insufficient credits to fund the task", command.Draft)
}

func (store LedgerStore) FundTaskFromOrganization(ctx context.Context, command ledger.OrganizationFundStoreCommand) ledger.FundResult {
	tx, err := store.db.Begin(ctx)
	if err != nil {
		return ledger.FundRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "begin fund task transaction failed")}
	}
	defer func() { _ = tx.Rollback(ctx) }()

	accountResult := lockOrganizationAccount(ctx, tx, command.OrganizationID)
	account, matched := accountResult.(accountLocked)
	if !matched {
		return ledger.FundRejected{Reason: accountResult.(accountLockRejected).reason}
	}

	if existing := findFundForKey(ctx, tx, command.IdempotencyKey.String(), command.TaskID, account.id); existing != nil {
		return existing
	}

	taskResult := lockTaskOwnedByOrganization(ctx, tx, command.TaskID, command.OrganizationID)
	taskRow, taskMatched := taskResult.(taskLocked)
	if !taskMatched {
		return ledger.FundRejected{Reason: taskResult.(taskLockRejected).reason}
	}
	if rejected := requireFundableTask(ctx, tx, command.TaskID, taskRow, command.Amount); rejected != nil {
		return rejected
	}

	// The task owner is resolved here (only the store sees the task row in
	// this transaction) and merged into the recipients alongside the acting
	// user the service already named.
	ownerID, ownerProblem := parseReviewWorker(taskRow.rawCreatedBy)
	if ownerProblem != nil {
		return ledger.FundRejected{Reason: *ownerProblem}
	}
	draft := command.Draft.WithRecipients(ownerID)

	if problem := applySpendCharge(ctx, tx, command.Spend); problem != nil {
		recordBudgetRefusal(ctx, store.db, utcDay(time.Now()))
		return ledger.FundRejected{Reason: *problem}
	}

	return completeFunding(ctx, tx, account, command.TaskID, command.Amount, command.EntryID, command.IdempotencyKey, "insufficient organization credits to fund the task", draft)
}

func (store LedgerStore) OrganizationBalance(ctx context.Context, organizationID core.OrganizationID) ledger.BalanceResult {
	var spendable, allocated int64
	err := store.db.QueryRow(ctx, `
		select
			coalesce((select sum(ledger_entries.amount) from ledger_entries join credit_accounts on credit_accounts.id = ledger_entries.account_id where credit_accounts.organization_id = $1), 0),
			coalesce((select sum(task_funds.credit_amount) from task_funds join credit_accounts on credit_accounts.id = task_funds.funder_account_id where credit_accounts.organization_id = $1), 0)
	`, organizationID.String()).Scan(&spendable, &allocated)
	if err != nil {
		return ledger.BalanceRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "read organization balance failed")}
	}
	return ledger.BalanceFound{Value: ledger.NewBalance(spendable, allocated)}
}

func (store LedgerStore) AcceptSubmission(ctx context.Context, command ledger.AcceptStoreCommand) ledger.AcceptResult {
	tx, err := store.db.Begin(ctx)
	if err != nil {
		return ledger.AcceptRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "begin accept submission transaction failed")}
	}
	defer func() { _ = tx.Rollback(ctx) }()

	taskResult := lockTaskForReview(ctx, tx, command.TaskID, command.Reviewer, "accept submissions for")
	taskRow, taskMatched := taskResult.(taskLocked)
	if !taskMatched {
		return ledger.AcceptRejected{Reason: taskResult.(taskLockRejected).reason}
	}

	var submissionState string
	var rawWorkerID string
	var acceptedKey string
	scanErr := tx.QueryRow(ctx, `
		select state, user_id::text, coalesce(accepted_idempotency_key, '')
		from submissions
		where id = $1 and task_id = $2
	`, command.SubmissionID.String(), command.TaskID.String()).Scan(&submissionState, &rawWorkerID, &acceptedKey)
	if errors.Is(scanErr, ErrNoRows) {
		return ledger.AcceptRejected{Reason: core.NewDomainError(core.ErrorCodeNotFound, "submission was not found for the task")}
	}
	if scanErr != nil {
		return ledger.AcceptRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "read submission failed")}
	}

	if submissionState == "accepted" {
		if acceptedKey != command.IdempotencyKey.String() {
			return ledger.AcceptRejected{Reason: core.NewDomainError(core.ErrorCodeConflict, "submission was already accepted with a different idempotency key")}
		}
		return idempotentAccept(ctx, tx, command, rawWorkerID)
	}
	if submissionState != "submitted" {
		return ledger.AcceptRejected{Reason: core.NewDomainError(core.ErrorCodeConflict, "only valid submissions can be accepted")}
	}
	if taskRow.state != "open" {
		return ledger.AcceptRejected{Reason: core.NewDomainError(core.ErrorCodeConflict, "only open tasks can accept submissions")}
	}

	if _, err := tx.Exec(ctx, "update submissions set state = 'accepted', accepted_idempotency_key = $2, state_recorded_at = now() where id = $1", command.SubmissionID.String(), command.IdempotencyKey.String()); err != nil {
		if isUniqueViolation(err) {
			return ledger.AcceptRejected{Reason: core.NewDomainError(core.ErrorCodeConflict, "task already has an accepted submission")}
		}
		return ledger.AcceptRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "accept submission failed")}
	}

	if _, err := tx.Exec(ctx, "update tasks set state = 'closed', state_recorded_at = now() where id = $1", command.TaskID.String()); err != nil {
		return ledger.AcceptRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "close task failed")}
	}

	payoutResult := payOutEscrow(ctx, tx, command, rawWorkerID)
	payout, payoutMatched := payoutResult.(payoutResolved)
	if !payoutMatched {
		return ledger.AcceptRejected{Reason: payoutResult.(payoutRejected).reason}
	}

	outcome := payout.outcome
	collectibleResult := payOutCollectible(ctx, tx, command.TaskID, rawWorkerID)
	collectible, collectibleMatched := collectibleResult.(payoutResolved)
	if !collectibleMatched {
		return ledger.AcceptRejected{Reason: collectibleResult.(payoutRejected).reason}
	}
	outcome = combinePayouts(outcome, collectible.outcome)
	if _, noPayout := outcome.(ledger.NoPayout); noPayout {
		if taskRow.rewardKind == "credit" || taskRow.rewardKind == "bundle" {
			return ledger.AcceptRejected{Reason: core.NewDomainError(core.ErrorCodeConflict, "credit reward fund is missing")}
		}
	}

	if problem := applySpendCharge(ctx, tx, command.Spend); problem != nil {
		recordBudgetRefusal(ctx, store.db, utcDay(time.Now()))
		return ledger.AcceptRejected{Reason: *problem}
	}
	tipResult := payCreditTip(ctx, tx, command.TaskID, command.Reviewer, rawWorkerID, command.TipDebitEntryID, command.TipCreditEntryID, command.IdempotencyKey, command.TipSelection)
	tip, tipMatched := tipResult.(tipResolved)
	if !tipMatched {
		return ledger.AcceptRejected{Reason: tipResult.(tipRejected).reason}
	}
	collectibleTipResult := payCollectibleTip(ctx, tx, command.Reviewer, rawWorkerID, command.CollectibleTip)
	collectibleTip, collectibleTipMatched := collectibleTipResult.(tipResolved)
	if !collectibleTipMatched {
		return ledger.AcceptRejected{Reason: collectibleTipResult.(tipRejected).reason}
	}
	tipOutcome := combineTips(tip.outcome, collectibleTip.outcome)

	workerID, workerProblem := parseReviewWorker(rawWorkerID)
	if workerProblem != nil {
		return ledger.AcceptRejected{Reason: *workerProblem}
	}

	// The accept closes the task, so every other submission still in
	// 'submitted' is superseded in this same transaction, each with its own
	// event addressed to its submitter.
	supersededDrafts, supersedeProblem := supersedeCompetingSubmissions(ctx, tx, command.TaskID, command.SubmissionID, command.Draft.Actor)
	if supersedeProblem != nil {
		return ledger.AcceptRejected{Reason: *supersedeProblem}
	}

	acceptedDraft := command.Draft.WithRecipients(workerID)
	payoutDrafts, payoutDraftProblem := ledger.ReviewEventDrafts(acceptedDraft, command.Reviewer, command.TaskID, outcome, tipOutcome)
	if payoutDraftProblem != nil {
		return ledger.AcceptRejected{Reason: *payoutDraftProblem}
	}
	recorded := append(append([]event.Draft{acceptedDraft}, payoutDrafts...), supersededDrafts...)
	for _, draft := range recorded {
		if err := recordEventDraftInTx(ctx, tx, draft); err != nil {
			return ledger.AcceptRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "record accept events failed")}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return ledger.AcceptRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "commit accept submission transaction failed")}
	}

	return ledger.SubmissionAccepted{TaskID: command.TaskID, SubmissionID: command.SubmissionID, WorkerUserID: workerID, Payout: outcome, Tip: tipOutcome, Execution: ledger.FirstExecution{}, RecordedEvents: recorded}
}

func combinePayouts(first ledger.PayoutOutcome, second ledger.PayoutOutcome) ledger.PayoutOutcome {
	credit, hasCredit := first.(ledger.CreditPayout)
	collectible, hasCollectible := second.(ledger.CollectiblePayout)
	if hasCredit && hasCollectible && credit.WorkerUserID == collectible.WorkerUserID {
		return ledger.BundlePayout{WorkerUserID: credit.WorkerUserID, Amount: credit.Amount, CollectibleIDs: collectible.CollectibleIDs}
	}
	if _, firstNone := first.(ledger.NoPayout); firstNone {
		return second
	}
	return first
}

func combineTips(first ledger.TipOutcome, second ledger.TipOutcome) ledger.TipOutcome {
	credit, hasCredit := first.(ledger.CreditTip)
	collectible, hasCollectible := second.(ledger.CollectibleTip)
	if hasCredit && hasCollectible && credit.WorkerUserID == collectible.WorkerUserID {
		return ledger.BundleTip{WorkerUserID: credit.WorkerUserID, Amount: credit.Amount, CollectibleID: collectible.CollectibleID}
	}
	if _, firstNone := first.(ledger.NoTip); firstNone {
		return second
	}
	return first
}

func (store LedgerStore) RequestChanges(ctx context.Context, command ledger.RequestChangesStoreCommand) ledger.RequestChangesResult {
	tx, err := store.db.Begin(ctx)
	if err != nil {
		return ledger.RequestChangesRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "begin request changes transaction failed")}
	}
	defer func() { _ = tx.Rollback(ctx) }()

	taskResult := lockTaskForReview(ctx, tx, command.TaskID, command.Reviewer, "review submissions for")
	taskRow, taskMatched := taskResult.(taskLocked)
	if !taskMatched {
		return ledger.RequestChangesRejected{Reason: taskResult.(taskLockRejected).reason}
	}
	if taskRow.state != "open" {
		return ledger.RequestChangesRejected{Reason: core.NewDomainError(core.ErrorCodeConflict, "only open tasks can request submission changes")}
	}

	var submissionState string
	var rawWorkerID string
	var changesKey string
	var storedNote string
	scanErr := tx.QueryRow(ctx, `
		select state, user_id::text, coalesce(changes_idempotency_key, ''), coalesce(review_note, '')
		from submissions
		where id = $1 and task_id = $2
		for update
	`, command.SubmissionID.String(), command.TaskID.String()).Scan(&submissionState, &rawWorkerID, &changesKey, &storedNote)
	if errors.Is(scanErr, ErrNoRows) {
		return ledger.RequestChangesRejected{Reason: core.NewDomainError(core.ErrorCodeNotFound, "submission was not found for the task")}
	}
	if scanErr != nil {
		return ledger.RequestChangesRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "read submission failed")}
	}
	workerID, workerProblem := parseReviewWorker(rawWorkerID)
	if workerProblem != nil {
		return ledger.RequestChangesRejected{Reason: *workerProblem}
	}
	if submissionState == "changes_requested" {
		if changesKey == command.IdempotencyKey.String() {
			return ledger.ChangesRequested{TaskID: command.TaskID, SubmissionID: command.SubmissionID, WorkerUserID: workerID, ReviewNote: storedNote, Execution: ledger.IdempotentReplay{}, RecordedEvents: []event.Draft{}}
		}
		return ledger.RequestChangesRejected{Reason: core.NewDomainError(core.ErrorCodeConflict, "changes were already requested with a different idempotency key")}
	}
	if submissionState != "submitted" {
		return ledger.RequestChangesRejected{Reason: core.NewDomainError(core.ErrorCodeConflict, "only submitted work can receive requested changes")}
	}

	if _, err := tx.Exec(ctx, `
		update submissions
		set state = 'changes_requested', review_note = $2, reviewed_by_user_id = $3, review_recorded_at = now(), changes_idempotency_key = $4, state_recorded_at = now()
		where id = $1
	`, command.SubmissionID.String(), command.ReviewNote.String(), reviewerUserColumn(command.Reviewer), command.IdempotencyKey.String()); err != nil {
		return ledger.RequestChangesRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "request submission changes failed")}
	}

	if _, err := tx.Exec(ctx, `
		update task_reservations
		set state = 'active', state_recorded_at = now()
		where task_id = $1 and assignee_kind = 'user' and user_id = $2 and state = 'submitted'
	`, command.TaskID.String(), rawWorkerID); err != nil {
		return ledger.RequestChangesRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "reactivate task reservation failed")}
	}

	draft := command.Draft.WithRecipients(workerID)
	if err := recordEventDraftInTx(ctx, tx, draft); err != nil {
		return ledger.RequestChangesRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "record changes requested event failed")}
	}

	if err := tx.Commit(ctx); err != nil {
		return ledger.RequestChangesRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "commit request changes transaction failed")}
	}

	return ledger.ChangesRequested{TaskID: command.TaskID, SubmissionID: command.SubmissionID, WorkerUserID: workerID, ReviewNote: command.ReviewNote.String(), Execution: ledger.FirstExecution{}, RecordedEvents: []event.Draft{draft}}
}

func (store LedgerStore) RejectSubmission(ctx context.Context, command ledger.RejectStoreCommand) ledger.RejectResult {
	tx, err := store.db.Begin(ctx)
	if err != nil {
		return ledger.RejectRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "begin reject submission transaction failed")}
	}
	defer func() { _ = tx.Rollback(ctx) }()

	taskResult := lockTaskForReview(ctx, tx, command.TaskID, command.Reviewer, "review submissions for")
	taskRow, taskMatched := taskResult.(taskLocked)
	if !taskMatched {
		return ledger.RejectRejected{Reason: taskResult.(taskLockRejected).reason}
	}
	if taskRow.state != "open" {
		return ledger.RejectRejected{Reason: core.NewDomainError(core.ErrorCodeConflict, "only open tasks can reject submissions")}
	}

	var submissionState string
	var rawWorkerID string
	var reviewKey string
	scanErr := tx.QueryRow(ctx, `
		select state, user_id::text, coalesce(review_idempotency_key, '')
		from submissions
		where id = $1 and task_id = $2
		for update
	`, command.SubmissionID.String(), command.TaskID.String()).Scan(&submissionState, &rawWorkerID, &reviewKey)
	if errors.Is(scanErr, ErrNoRows) {
		return ledger.RejectRejected{Reason: core.NewDomainError(core.ErrorCodeNotFound, "submission was not found for the task")}
	}
	if scanErr != nil {
		return ledger.RejectRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "read submission failed")}
	}
	workerID, workerProblem := parseReviewWorker(rawWorkerID)
	if workerProblem != nil {
		return ledger.RejectRejected{Reason: *workerProblem}
	}
	if submissionState == "rejected" {
		if reviewKey == command.IdempotencyKey.String() {
			return ledger.SubmissionRejected{TaskID: command.TaskID, SubmissionID: command.SubmissionID, WorkerUserID: workerID, Payout: ledger.NoPayout{}, Tip: ledger.NoTip{}, Execution: ledger.IdempotentReplay{}, RecordedEvents: []event.Draft{}}
		}
		return ledger.RejectRejected{Reason: core.NewDomainError(core.ErrorCodeConflict, "submission was already rejected with a different idempotency key")}
	}
	if submissionState != "submitted" {
		return ledger.RejectRejected{Reason: core.NewDomainError(core.ErrorCodeConflict, "only submitted work can be rejected")}
	}

	payoutResult := payReviewFund(ctx, tx, reviewFundCommand{
		taskID:            command.TaskID,
		rawWorkerID:       rawWorkerID,
		payoutEntryID:     command.PayoutEntryID,
		idempotencyKey:    command.IdempotencyKey,
		selection:         command.CreditSelection,
		closeTask:         false,
		missingIsNoPayout: true,
	})
	payout, payoutMatched := payoutResult.(payoutResolved)
	if !payoutMatched {
		return ledger.RejectRejected{Reason: payoutResult.(payoutRejected).reason}
	}

	if problem := applySpendCharge(ctx, tx, command.Spend); problem != nil {
		recordBudgetRefusal(ctx, store.db, utcDay(time.Now()))
		return ledger.RejectRejected{Reason: *problem}
	}
	tipResult := payCreditTip(ctx, tx, command.TaskID, command.Reviewer, rawWorkerID, command.TipDebitEntryID, command.TipCreditEntryID, command.IdempotencyKey, command.TipSelection)
	tip, tipMatched := tipResult.(tipResolved)
	if !tipMatched {
		return ledger.RejectRejected{Reason: tipResult.(tipRejected).reason}
	}

	if _, err := tx.Exec(ctx, `
		update submissions
		set state = 'rejected', review_note = $2, reviewed_by_user_id = $3, review_recorded_at = now(), review_idempotency_key = $4, state_recorded_at = now()
		where id = $1
	`, command.SubmissionID.String(), command.ReviewNote.String(), reviewerUserColumn(command.Reviewer), command.IdempotencyKey.String()); err != nil {
		return ledger.RejectRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "reject submission failed")}
	}

	if _, err := tx.Exec(ctx, `
		update task_reservations
		set state = 'cancelled_by_requester', state_recorded_at = now()
		where task_id = $1 and assignee_kind = 'user' and user_id = $2 and state in ('active', 'submitted')
	`, command.TaskID.String(), rawWorkerID); err != nil {
		return ledger.RejectRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "release rejected reservation failed")}
	}

	if _, ban := command.BanSelection.(ledger.BanImplementorSelection); ban {
		// Bans record the banning user; the service refuses ban selections
		// from an organization reviewer before the command is built.
		bannedBy := reviewerUserColumn(command.Reviewer)
		if bannedBy == nil {
			return ledger.RejectRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "an organization credential cannot ban an implementor")}
		}
		if _, err := tx.Exec(ctx, `
			insert into task_implementor_bans (task_id, assignee_kind, assignee_key, user_id, banned_by_user_id)
			values ($1, 'user', $2, $3, $4)
			on conflict (task_id, assignee_kind, assignee_key) do nothing
		`, command.TaskID.String(), rawWorkerID, rawWorkerID, *bannedBy); err != nil {
			return ledger.RejectRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "ban task implementor failed")}
		}
	}

	rejectedDraft := command.Draft.WithRecipients(workerID)
	payoutDrafts, payoutDraftProblem := ledger.ReviewEventDrafts(rejectedDraft, command.Reviewer, command.TaskID, payout.outcome, tip.outcome)
	if payoutDraftProblem != nil {
		return ledger.RejectRejected{Reason: *payoutDraftProblem}
	}
	recorded := append([]event.Draft{rejectedDraft}, payoutDrafts...)
	for _, draft := range recorded {
		if err := recordEventDraftInTx(ctx, tx, draft); err != nil {
			return ledger.RejectRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "record reject events failed")}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return ledger.RejectRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "commit reject submission transaction failed")}
	}

	return ledger.SubmissionRejected{TaskID: command.TaskID, SubmissionID: command.SubmissionID, WorkerUserID: workerID, Payout: payout.outcome, Tip: tip.outcome, Execution: ledger.FirstExecution{}, RecordedEvents: recorded}
}

// GrantCredits writes a platform-admin manual_adjustment ledger entry
// crediting the target account's spendable balance, and resolves the
// beneficiary users to notify (the granted user, or the grantee
// organization's owner/admin/billing members) inside the same transaction. A
// replayed idempotency key for the same manual_adjustment returns the same
// CreditsGranted shape without writing a second entry.
func (store LedgerStore) GrantCredits(ctx context.Context, command ledger.GrantStoreCommand) ledger.GrantResult {
	tx, err := store.db.Begin(ctx)
	if err != nil {
		return ledger.GrantRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "begin credit grant transaction failed")}
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var accountResult accountLockResult
	switch typed := command.Target.(type) {
	case ledger.GrantToUser:
		accountResult = lockUserAccount(ctx, tx, typed.ID)
	case ledger.GrantToOrganization:
		accountResult = lockOrganizationAccount(ctx, tx, typed.ID)
	default:
		return ledger.GrantRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "credit grant target is invalid")}
	}
	account, matched := accountResult.(accountLocked)
	if !matched {
		return ledger.GrantRejected{Reason: accountResult.(accountLockRejected).reason}
	}

	var existingID string
	var existingKind string
	var existingAmount int64
	scanErr := tx.QueryRow(ctx, "select id::text, kind, amount from ledger_entries where idempotency_key = $1 and account_id = $2",
		command.IdempotencyKey.String(), account.id).Scan(&existingID, &existingKind, &existingAmount)
	if scanErr != nil && !errors.Is(scanErr, ErrNoRows) {
		return ledger.GrantRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "check credit grant idempotency failed")}
	}
	entryID := command.EntryID
	replayed := scanErr == nil
	if replayed {
		if existingKind != ledger.EntryKindManualAdjustment.String() || existingAmount != command.Amount.Int64() {
			return ledger.GrantRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "idempotency key was used for a different command")}
		}
		parsedResult := core.ParseLedgerEntryID(existingID)
		parsed, parsedMatched := parsedResult.(core.LedgerEntryIDCreated)
		if !parsedMatched {
			return ledger.GrantRejected{Reason: parsedResult.(core.LedgerEntryIDRejected).Reason}
		}
		entryID = parsed.Value
	} else {
		if _, err := tx.Exec(ctx, `
			insert into ledger_entries (id, account_id, kind, amount, idempotency_key, note)
			values ($1, $2, 'manual_adjustment', $3, $4, $5)
		`, command.EntryID.String(), account.id, command.Amount.Int64(), command.IdempotencyKey.String(), command.Note.String()); err != nil {
			return ledger.GrantRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "insert credit grant ledger entry failed")}
		}
	}

	recipients, recipientsReason := grantRecipients(ctx, tx, command.Target)
	if recipientsReason != nil {
		return ledger.GrantRejected{Reason: *recipientsReason}
	}

	execution := ledger.Execution(ledger.FirstExecution{})
	recorded := []event.Draft{}
	if replayed {
		execution = ledger.IdempotentReplay{}
	} else {
		draft := command.Draft.WithRecipients(recipients...)
		if err := recordEventDraftInTx(ctx, tx, draft); err != nil {
			return ledger.GrantRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "record credit grant event failed")}
		}
		recorded = []event.Draft{draft}
	}

	if err := tx.Commit(ctx); err != nil {
		return ledger.GrantRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "commit credit grant transaction failed")}
	}
	return ledger.CreditsGranted{EntryID: entryID, Amount: command.Amount, RecipientUserIDs: recipients, Execution: execution, RecordedEvents: recorded}
}

// grantRecipients resolves who a credit grant should notify: the granted user
// directly, or every active member of the grantee organization holding the
// owner, admin, or billing role.
func grantRecipients(ctx context.Context, tx Tx, target ledger.GrantTarget) ([]core.UserID, *core.DomainError) {
	switch typed := target.(type) {
	case ledger.GrantToUser:
		return []core.UserID{typed.ID}, nil
	case ledger.GrantToOrganization:
		rows, err := tx.Query(ctx, `
			select distinct organization_memberships.user_id::text
			from organization_memberships
			join organization_membership_roles
				on organization_membership_roles.membership_id = organization_memberships.id
			where organization_memberships.organization_id = $1
			and organization_memberships.status = 'active'
			and organization_membership_roles.role in ('owner', 'admin', 'billing')
		`, typed.ID.String())
		if err != nil {
			reason := core.NewDomainError(core.ErrorCodeInvalidState, "read credit grant recipients failed")
			return nil, &reason
		}
		defer rows.Close()
		recipients := make([]core.UserID, 0)
		for rows.Next() {
			var rawID string
			if err := rows.Scan(&rawID); err != nil {
				reason := core.NewDomainError(core.ErrorCodeInvalidState, "scan credit grant recipient failed")
				return nil, &reason
			}
			parsedResult := core.ParseUserID(rawID)
			parsed, matched := parsedResult.(core.UserIDCreated)
			if !matched {
				reason := parsedResult.(core.UserIDRejected).Reason
				return nil, &reason
			}
			recipients = append(recipients, parsed.Value)
		}
		if rows.Err() != nil {
			reason := core.NewDomainError(core.ErrorCodeInvalidState, "read credit grant recipients failed")
			return nil, &reason
		}
		return recipients, nil
	default:
		reason := core.NewDomainError(core.ErrorCodeInvalidArgument, "credit grant target is invalid")
		return nil, &reason
	}
}

func (store LedgerStore) RefundTask(ctx context.Context, command ledger.RefundStoreCommand) ledger.RefundResult {
	tx, err := store.db.Begin(ctx)
	if err != nil {
		return ledger.RefundRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "begin refund task transaction failed")}
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if reason := lockTaskRefundable(ctx, tx, command.TaskID, command.RequesterUserID); reason != nil {
		return ledger.RefundRejected{Reason: *reason}
	}

	// Idempotency keys are unique per account, not globally, so the reuse
	// checks are scoped: first to this task's entries (catches reusing this
	// task's fund/payout key for its refund), then to the funder's account
	// once the fund row names it (catches reusing the funder's key from
	// another task).
	var keyExists bool
	if err := tx.QueryRow(ctx, "select exists(select 1 from ledger_entries where idempotency_key = $1 and task_id = $2)", command.IdempotencyKey.String(), command.TaskID.String()).Scan(&keyExists); err != nil {
		return ledger.RefundRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "check refund idempotency failed")}
	}

	var rawFunderAccountID string
	var amount int64
	scanErr := tx.QueryRow(ctx, "select funder_account_id::text, credit_amount from task_funds where task_id = $1 for update", command.TaskID.String()).Scan(&rawFunderAccountID, &amount)
	if errors.Is(scanErr, ErrNoRows) {
		return replayOrRejectRefund(ctx, tx, command, keyExists)
	}
	if scanErr != nil {
		return ledger.RefundRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "read task fund failed")}
	}
	if !keyExists {
		if err := tx.QueryRow(ctx, "select exists(select 1 from ledger_entries where idempotency_key = $1 and account_id = $2)", command.IdempotencyKey.String(), rawFunderAccountID).Scan(&keyExists); err != nil {
			return ledger.RefundRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "check refund idempotency failed")}
		}
	}
	if keyExists {
		return ledger.RefundRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "idempotency key was used for a different command")}
	}

	fundResult := buildFund(command.TaskID, rawFunderAccountID, amount)
	fundValue, fundMatched := fundResult.(fundBuilt)
	if !fundMatched {
		return ledger.RefundRejected{Reason: fundResult.(fundBuildRejected).reason}
	}

	if _, err := tx.Exec(ctx, `
		insert into ledger_entries (id, account_id, kind, amount, task_id, idempotency_key)
		values ($1, $2, 'task_refund', $3, $4, $5)
	`, command.EntryID.String(), rawFunderAccountID, amount, command.TaskID.String(), command.IdempotencyKey.String()); err != nil {
		return ledger.RefundRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "insert task refund ledger entry failed")}
	}

	if _, err := tx.Exec(ctx, "delete from task_funds where task_id = $1", command.TaskID.String()); err != nil {
		return ledger.RefundRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "clear task fund failed")}
	}

	if reason, rejected := refundHeldCollectibleReward(ctx, tx, command.TaskID); rejected {
		return ledger.RefundRejected{Reason: reason}
	}

	if _, err := tx.Exec(ctx, "update tasks set state = 'cancelled', state_recorded_at = now() where id = $1", command.TaskID.String()); err != nil {
		return ledger.RefundRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "cancel task failed")}
	}

	// A refund cancels the task, so release the worker's reservation too -
	// otherwise it would dangle in an active/submitted state on a cancelled
	// task, exactly like the cancel path (releaseReservationsOnCancel). The
	// holders are read first (inside this transaction) and merged into the
	// event draft's recipients, so the released workers learn the task is
	// gone.
	holders, holdersReason := lockedReservationHolders(ctx, tx, command.TaskID.String())
	if holdersReason != nil {
		return ledger.RefundRejected{Reason: *holdersReason}
	}
	if reason := releaseReservationsOnCancel(ctx, tx, command.TaskID); reason != nil {
		return ledger.RefundRejected{Reason: *reason}
	}

	draft := command.Draft.WithRecipients(holders...)
	if err := recordEventDraftInTx(ctx, tx, draft); err != nil {
		return ledger.RefundRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "record task refund event failed")}
	}

	if err := tx.Commit(ctx); err != nil {
		return ledger.RefundRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "commit refund task transaction failed")}
	}

	return ledger.TaskRefunded{Fund: fundValue.value, Execution: ledger.FirstExecution{}, RecordedEvents: []event.Draft{draft}}
}

// lockTaskRefundable locks the task and authorizes the refund. A refund is
// permitted for the task owner (its creator) or the user currently holding the
// active reservation (the implementor), and only while the task is not yet
// awarded (still draft or open).
func lockTaskRefundable(ctx context.Context, tx Tx, taskID core.TaskID, requester core.UserID) *core.DomainError {
	var state string
	var rawCreatedBy string
	scanErr := tx.QueryRow(ctx, "select state, created_by_user_id::text from tasks where id = $1 for update", taskID.String()).Scan(&state, &rawCreatedBy)
	if errors.Is(scanErr, ErrNoRows) {
		reason := core.NewDomainError(core.ErrorCodeNotFound, "task was not found")
		return &reason
	}
	if scanErr != nil {
		reason := core.NewDomainError(core.ErrorCodeInvalidState, "lock task failed")
		return &reason
	}
	if rawCreatedBy != requester.String() {
		var isActiveImplementor bool
		if err := tx.QueryRow(ctx, `
			select exists(
				select 1 from task_reservations
				where task_id = $1 and assignee_kind = 'user' and user_id = $2 and state = 'active'
			)
		`, taskID.String(), requester.String()).Scan(&isActiveImplementor); err != nil {
			reason := core.NewDomainError(core.ErrorCodeInvalidState, "check active reservation failed")
			return &reason
		}
		if !isActiveImplementor {
			reason := core.NewDomainError(core.ErrorCodePermissionDenied, "only the task owner or the active implementor can refund the task")
			return &reason
		}
	}
	if state != "draft" && state != "open" {
		reason := core.NewDomainError(core.ErrorCodeInvalidState, "only tasks that are not yet awarded can be refunded")
		return &reason
	}
	return nil
}

// replayOrRejectRefund handles a refund of a task with no task_funds row. If the
// idempotency key was already used, it replays the recorded refund from the
// durable task_refund ledger entry; otherwise the task has nothing to refund.
func replayOrRejectRefund(ctx context.Context, tx Tx, command ledger.RefundStoreCommand, keyExists bool) ledger.RefundResult {
	if !keyExists {
		return ledger.RefundRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "task has nothing to refund")}
	}
	var rawFunderAccountID string
	var amount int64
	scanErr := tx.QueryRow(ctx, "select account_id::text, amount from ledger_entries where task_id = $1 and kind = 'task_refund' and idempotency_key = $2", command.TaskID.String(), command.IdempotencyKey.String()).Scan(&rawFunderAccountID, &amount)
	if errors.Is(scanErr, ErrNoRows) {
		return ledger.RefundRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "idempotency key was used for a different command")}
	}
	if scanErr != nil {
		return ledger.RefundRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "read task refund failed")}
	}
	fundResult := buildFund(command.TaskID, rawFunderAccountID, amount)
	fundValue, fundMatched := fundResult.(fundBuilt)
	if !fundMatched {
		return ledger.RefundRejected{Reason: fundResult.(fundBuildRejected).reason}
	}
	return ledger.TaskRefunded{Fund: fundValue.value, Execution: ledger.IdempotentReplay{}, RecordedEvents: []event.Draft{}}
}

func (store LedgerStore) Balance(ctx context.Context, owner core.UserID) ledger.BalanceResult {
	var spendable, allocated int64
	err := store.db.QueryRow(ctx, `
		select
			coalesce((select sum(ledger_entries.amount) from ledger_entries join credit_accounts on credit_accounts.id = ledger_entries.account_id where credit_accounts.user_id = $1), 0),
			coalesce((select sum(task_funds.credit_amount) from task_funds join credit_accounts on credit_accounts.id = task_funds.funder_account_id where credit_accounts.user_id = $1), 0)
	`, owner.String()).Scan(&spendable, &allocated)
	if err != nil {
		return ledger.BalanceRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "read balance failed")}
	}
	return ledger.BalanceFound{Value: ledger.NewBalance(spendable, allocated)}
}

func (store LedgerStore) TaskAllocatedCredits(ctx context.Context, taskID core.TaskID) ledger.TaskAllocatedResult {
	var allocated int64
	err := store.db.QueryRow(ctx,
		"select coalesce((select credit_amount from task_funds where task_id = $1), 0)",
		taskID.String()).Scan(&allocated)
	if err != nil {
		return ledger.TaskAllocatedRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "read task allocated credits failed")}
	}
	return ledger.TaskAllocatedFound{Amount: allocated}
}

func (store LedgerStore) ListEntries(ctx context.Context, owner core.UserID, page core.Page) ledger.ListEntriesResult {
	var total int64
	if err := store.db.QueryRow(ctx, `
		select count(*)
		from ledger_entries
		join credit_accounts on credit_accounts.id = ledger_entries.account_id
		where credit_accounts.user_id = $1
	`, owner.String()).Scan(&total); err != nil {
		return ledger.ListEntriesRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "count ledger entries failed")}
	}
	rows, err := store.db.Query(ctx, `
		select ledger_entries.id::text, ledger_entries.kind, ledger_entries.amount, coalesce(ledger_entries.task_id::text, ''), coalesce(ledger_entries.note, '')
		from ledger_entries
		join credit_accounts on credit_accounts.id = ledger_entries.account_id
		where credit_accounts.user_id = $1
		order by ledger_entries.created_at desc, ledger_entries.id desc
		limit $2 offset $3
	`, owner.String(), page.Limit(), page.Offset())
	if err != nil {
		return ledger.ListEntriesRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "list ledger entries failed")}
	}
	defer rows.Close()

	entries := make([]ledger.LedgerEntry, 0)
	for rows.Next() {
		var rawID string
		var rawKind string
		var amount int64
		var rawTaskID string
		var note string
		if err := rows.Scan(&rawID, &rawKind, &amount, &rawTaskID, &note); err != nil {
			return ledger.ListEntriesRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "scan ledger entry failed")}
		}
		entryResult := parseLedgerEntry(rawID, rawKind, amount, rawTaskID, note)
		entry, matched := entryResult.(ledgerEntryParsed)
		if !matched {
			return ledger.ListEntriesRejected{Reason: entryResult.(ledgerEntryParseRejected).reason}
		}
		entries = append(entries, entry.value)
	}
	if err := rows.Err(); err != nil {
		return ledger.ListEntriesRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "read ledger entries failed")}
	}
	return ledger.EntriesListed{Values: entries, Total: total}
}

func (store LedgerStore) ListOrganizationEntries(ctx context.Context, organizationID core.OrganizationID, page core.Page) ledger.ListEntriesResult {
	var total int64
	if err := store.db.QueryRow(ctx, `
		select count(*)
		from ledger_entries
		join credit_accounts on credit_accounts.id = ledger_entries.account_id
		where credit_accounts.organization_id = $1
	`, organizationID.String()).Scan(&total); err != nil {
		return ledger.ListEntriesRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "count organization ledger entries failed")}
	}
	rows, err := store.db.Query(ctx, `
		select ledger_entries.id::text, ledger_entries.kind, ledger_entries.amount, coalesce(ledger_entries.task_id::text, ''), coalesce(ledger_entries.note, '')
		from ledger_entries
		join credit_accounts on credit_accounts.id = ledger_entries.account_id
		where credit_accounts.organization_id = $1
		order by ledger_entries.created_at desc, ledger_entries.id desc
		limit $2 offset $3
	`, organizationID.String(), page.Limit(), page.Offset())
	if err != nil {
		return ledger.ListEntriesRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "list organization ledger entries failed")}
	}
	defer rows.Close()

	entries := make([]ledger.LedgerEntry, 0)
	for rows.Next() {
		var rawID string
		var rawKind string
		var amount int64
		var rawTaskID string
		var note string
		if err := rows.Scan(&rawID, &rawKind, &amount, &rawTaskID, &note); err != nil {
			return ledger.ListEntriesRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "scan organization ledger entry failed")}
		}
		entryResult := parseLedgerEntry(rawID, rawKind, amount, rawTaskID, note)
		entry, matched := entryResult.(ledgerEntryParsed)
		if !matched {
			return ledger.ListEntriesRejected{Reason: entryResult.(ledgerEntryParseRejected).reason}
		}
		entries = append(entries, entry.value)
	}
	if err := rows.Err(); err != nil {
		return ledger.ListEntriesRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "read organization ledger entries failed")}
	}
	return ledger.EntriesListed{Values: entries, Total: total}
}

// transferAccountRef normalizes a transfer side onto the account owner it
// resolves to, with a stable ordering key so two crossing sends always lock
// the two accounts in the same order (no AB-BA deadlock).
type transferAccountRef struct {
	key          string
	user         *core.UserID
	organization *core.OrganizationID
}

func transferSourceRef(source ledger.TransferSource, actor core.UserID) (transferAccountRef, *core.DomainError) {
	switch typed := source.(type) {
	case ledger.TransferFromSelf:
		user := actor
		return transferAccountRef{key: "user:" + user.String(), user: &user}, nil
	case ledger.TransferFromOrganization:
		organization := typed.ID
		return transferAccountRef{key: "organization:" + organization.String(), organization: &organization}, nil
	default:
		reason := core.NewDomainError(core.ErrorCodeInvalidArgument, "credit send source is invalid")
		return transferAccountRef{}, &reason
	}
}

func transferTargetRef(target ledger.TransferTarget) (transferAccountRef, *core.DomainError) {
	switch typed := target.(type) {
	case ledger.TransferToUser:
		user := typed.ID
		return transferAccountRef{key: "user:" + user.String(), user: &user}, nil
	case ledger.TransferToOrganization:
		organization := typed.ID
		return transferAccountRef{key: "organization:" + organization.String(), organization: &organization}, nil
	default:
		reason := core.NewDomainError(core.ErrorCodeInvalidArgument, "credit send target is invalid")
		return transferAccountRef{}, &reason
	}
}

func lockTransferAccount(ctx context.Context, tx Tx, ref transferAccountRef) accountLockResult {
	if ref.user != nil {
		return lockUserAccount(ctx, tx, *ref.user)
	}
	return lockOrganizationAccount(ctx, tx, *ref.organization)
}

// PeerTransfer moves credits between two accounts as an atomic peer_transfer
// double entry. An organization source is authorized in-transaction: the
// acting user must hold the billing permission in that organization. The
// idempotency key is scoped per side (":send-debit" / ":send-credit" under
// the per-account unique index), and a replayed key returns the original
// outcome without new ledger rows or events.
func (store LedgerStore) PeerTransfer(ctx context.Context, command ledger.PeerTransferStoreCommand) ledger.SendResult {
	sourceRef, sourceProblem := transferSourceRef(command.Source, command.ActingUserID)
	if sourceProblem != nil {
		return ledger.SendRejected{Reason: *sourceProblem}
	}
	targetRef, targetProblem := transferTargetRef(command.Target)
	if targetProblem != nil {
		return ledger.SendRejected{Reason: *targetProblem}
	}
	if sourceRef.key == targetRef.key {
		return ledger.SendRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "credits cannot be sent to the sending account")}
	}

	tx, err := store.db.Begin(ctx)
	if err != nil {
		return ledger.SendRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "begin peer transfer transaction failed")}
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if organizationSource, isOrganization := command.Source.(ledger.TransferFromOrganization); isOrganization {
		check := organizationMemberPermission(ctx, tx, organizationSource.ID.String(), command.ActingUserID, org.PermissionManageBilling)
		if _, denied := check.(org.PermissionDenied); denied {
			return ledger.SendRejected{Reason: core.NewDomainError(core.ErrorCodePermissionDenied, "sending organization credits requires the billing permission")}
		}
	}

	// Lock both accounts in stable key order.
	firstRef, secondRef := sourceRef, targetRef
	if secondRef.key < firstRef.key {
		firstRef, secondRef = secondRef, firstRef
	}
	locked := map[string]accountLocked{}
	for _, ref := range []transferAccountRef{firstRef, secondRef} {
		result := lockTransferAccount(ctx, tx, ref)
		account, matched := result.(accountLocked)
		if !matched {
			return ledger.SendRejected{Reason: result.(accountLockRejected).reason}
		}
		locked[ref.key] = account
	}
	sourceAccount := locked[sourceRef.key]
	targetAccount := locked[targetRef.key]

	debitKey := command.IdempotencyKey.String() + ":send-debit"
	creditKey := command.IdempotencyKey.String() + ":send-credit"

	receivers, receiversReason := transferReceivers(ctx, tx, command.Target)
	if receiversReason != nil {
		return ledger.SendRejected{Reason: *receiversReason}
	}

	var existingID string
	var existingKind string
	var existingAmount int64
	scanErr := tx.QueryRow(ctx, "select id::text, kind, amount from ledger_entries where idempotency_key = $1 and account_id = $2",
		debitKey, sourceAccount.id).Scan(&existingID, &existingKind, &existingAmount)
	if scanErr != nil && !errors.Is(scanErr, ErrNoRows) {
		return ledger.SendRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "check peer transfer idempotency failed")}
	}
	if scanErr == nil {
		if existingKind != ledger.EntryKindPeerTransfer.String() || existingAmount != -command.Amount.Int64() {
			return ledger.SendRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "idempotency key was used for a different command")}
		}
		parsedResult := core.ParseLedgerEntryID(existingID)
		parsed, parsedMatched := parsedResult.(core.LedgerEntryIDCreated)
		if !parsedMatched {
			return ledger.SendRejected{Reason: parsedResult.(core.LedgerEntryIDRejected).Reason}
		}
		if err := tx.Commit(ctx); err != nil {
			return ledger.SendRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "commit peer transfer replay failed")}
		}
		return ledger.CreditsSent{DebitEntryID: parsed.Value, Amount: command.Amount, ReceiverUserIDs: receivers, Execution: ledger.IdempotentReplay{}, RecordedEvents: []event.Draft{}}
	}

	var spendable int64
	if err := tx.QueryRow(ctx, "select coalesce(sum(amount), 0) from ledger_entries where account_id = $1", sourceAccount.id).Scan(&spendable); err != nil {
		return ledger.SendRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "read sender balance failed")}
	}
	if spendable < command.Amount.Int64() {
		return ledger.SendRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "insufficient credits to send")}
	}

	if problem := applyPeerTransferVelocity(ctx, tx, sourceAccount.id, command.Amount.Int64()); problem != nil {
		recordBudgetRefusal(ctx, store.db, utcDay(time.Now()))
		return ledger.SendRejected{Reason: *problem}
	}
	if problem := applySpendCharge(ctx, tx, command.Spend); problem != nil {
		recordBudgetRefusal(ctx, store.db, utcDay(time.Now()))
		return ledger.SendRejected{Reason: *problem}
	}

	noteText := ""
	if provided, hasNote := command.Note.(ledger.TransferNoteProvided); hasNote {
		noteText = provided.Note.String()
	}
	if _, err := tx.Exec(ctx, `
		insert into ledger_entries (id, account_id, kind, amount, idempotency_key, note)
		values ($1, $2, 'peer_transfer', $3, $4, $5)
	`, command.DebitEntryID.String(), sourceAccount.id, -command.Amount.Int64(), debitKey, noteText); err != nil {
		return ledger.SendRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "insert peer transfer debit failed")}
	}
	if _, err := tx.Exec(ctx, `
		insert into ledger_entries (id, account_id, kind, amount, idempotency_key, note)
		values ($1, $2, 'peer_transfer', $3, $4, $5)
	`, command.CreditEntryID.String(), targetAccount.id, command.Amount.Int64(), creditKey, noteText); err != nil {
		return ledger.SendRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "insert peer transfer credit failed")}
	}

	draft := command.Draft.WithRecipients(receivers...)
	if err := recordEventDraftInTx(ctx, tx, draft); err != nil {
		return ledger.SendRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "record credits sent event failed")}
	}

	if err := tx.Commit(ctx); err != nil {
		return ledger.SendRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "commit peer transfer transaction failed")}
	}
	return ledger.CreditsSent{DebitEntryID: command.DebitEntryID, Amount: command.Amount, ReceiverUserIDs: receivers, Execution: ledger.FirstExecution{}, RecordedEvents: []event.Draft{draft}}
}

// transferReceivers resolves who a peer transfer notifies: the receiving
// user, or the receiving organization's owner/admin/billing members (the
// same audience a credit grant to that organization notifies).
func transferReceivers(ctx context.Context, tx Tx, target ledger.TransferTarget) ([]core.UserID, *core.DomainError) {
	switch typed := target.(type) {
	case ledger.TransferToUser:
		return []core.UserID{typed.ID}, nil
	case ledger.TransferToOrganization:
		return grantRecipients(ctx, tx, ledger.GrantToOrganization{ID: typed.ID})
	default:
		reason := core.NewDomainError(core.ErrorCodeInvalidArgument, "credit send target is invalid")
		return nil, &reason
	}
}
