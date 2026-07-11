package db

import (
	"context"
	"errors"

	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/ledger"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type LedgerStore struct {
	pool *pgxpool.Pool
}

func NewLedgerStore(pool *pgxpool.Pool) LedgerStore {
	return LedgerStore{pool: pool}
}

func (store LedgerStore) FundTask(ctx context.Context, command ledger.FundStoreCommand) ledger.FundResult {
	tx, err := store.pool.Begin(ctx)
	if err != nil {
		return ledger.FundRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "begin fund task transaction failed")}
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if existing := findFundForKey(ctx, tx, command.IdempotencyKey.String(), command.TaskID); existing != nil {
		return existing
	}

	accountResult := lockUserAccount(ctx, tx, command.FunderUserID)
	account, matched := accountResult.(accountLocked)
	if !matched {
		return ledger.FundRejected{Reason: accountResult.(accountLockRejected).reason}
	}

	taskResult := lockTaskOwnedBy(ctx, tx, command.TaskID, command.FunderUserID, "fund")
	taskRow, taskMatched := taskResult.(taskLocked)
	if !taskMatched {
		return ledger.FundRejected{Reason: taskResult.(taskLockRejected).reason}
	}
	if rejected := requireFundableTask(ctx, tx, command.TaskID, taskRow, command.Amount); rejected != nil {
		return rejected
	}

	return completeFunding(ctx, tx, account, command.TaskID, command.Amount, command.EntryID, command.IdempotencyKey, "insufficient credits to fund the task")
}

func (store LedgerStore) FundTaskFromOrganization(ctx context.Context, command ledger.OrganizationFundStoreCommand) ledger.FundResult {
	tx, err := store.pool.Begin(ctx)
	if err != nil {
		return ledger.FundRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "begin fund task transaction failed")}
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if existing := findFundForKey(ctx, tx, command.IdempotencyKey.String(), command.TaskID); existing != nil {
		return existing
	}

	accountResult := lockOrganizationAccount(ctx, tx, command.OrganizationID)
	account, matched := accountResult.(accountLocked)
	if !matched {
		return ledger.FundRejected{Reason: accountResult.(accountLockRejected).reason}
	}

	taskResult := lockTaskOwnedByOrganization(ctx, tx, command.TaskID, command.OrganizationID)
	taskRow, taskMatched := taskResult.(taskLocked)
	if !taskMatched {
		return ledger.FundRejected{Reason: taskResult.(taskLockRejected).reason}
	}
	if rejected := requireFundableTask(ctx, tx, command.TaskID, taskRow, command.Amount); rejected != nil {
		return rejected
	}

	return completeFunding(ctx, tx, account, command.TaskID, command.Amount, command.EntryID, command.IdempotencyKey, "insufficient organization credits to fund the task")
}

func (store LedgerStore) OrganizationBalance(ctx context.Context, organizationID core.OrganizationID) ledger.BalanceResult {
	var spendable, allocated int64
	err := store.pool.QueryRow(ctx, `
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
	tx, err := store.pool.Begin(ctx)
	if err != nil {
		return ledger.AcceptRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "begin accept submission transaction failed")}
	}
	defer func() { _ = tx.Rollback(ctx) }()

	taskResult := lockTaskForReview(ctx, tx, command.TaskID, command.RequesterUserID, "accept submissions for")
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
	if errors.Is(scanErr, pgx.ErrNoRows) {
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

	tipResult := payCreditTip(ctx, tx, command.TaskID, command.RequesterUserID, rawWorkerID, command.TipDebitEntryID, command.TipCreditEntryID, command.IdempotencyKey, command.TipSelection)
	tip, tipMatched := tipResult.(tipResolved)
	if !tipMatched {
		return ledger.AcceptRejected{Reason: tipResult.(tipRejected).reason}
	}
	collectibleTipResult := payCollectibleTip(ctx, tx, command.RequesterUserID, rawWorkerID, command.CollectibleTip)
	collectibleTip, collectibleTipMatched := collectibleTipResult.(tipResolved)
	if !collectibleTipMatched {
		return ledger.AcceptRejected{Reason: collectibleTipResult.(tipRejected).reason}
	}
	tipOutcome := combineTips(tip.outcome, collectibleTip.outcome)

	if err := tx.Commit(ctx); err != nil {
		return ledger.AcceptRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "commit accept submission transaction failed")}
	}

	return ledger.SubmissionAccepted{TaskID: command.TaskID, SubmissionID: command.SubmissionID, Payout: outcome, Tip: tipOutcome}
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
	tx, err := store.pool.Begin(ctx)
	if err != nil {
		return ledger.RequestChangesRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "begin request changes transaction failed")}
	}
	defer func() { _ = tx.Rollback(ctx) }()

	taskResult := lockTaskForReview(ctx, tx, command.TaskID, command.RequesterUserID, "review submissions for")
	taskRow, taskMatched := taskResult.(taskLocked)
	if !taskMatched {
		return ledger.RequestChangesRejected{Reason: taskResult.(taskLockRejected).reason}
	}
	if taskRow.state != "open" {
		return ledger.RequestChangesRejected{Reason: core.NewDomainError(core.ErrorCodeConflict, "only open tasks can request submission changes")}
	}

	var submissionState string
	var rawWorkerID string
	scanErr := tx.QueryRow(ctx, `
		select state, user_id::text
		from submissions
		where id = $1 and task_id = $2
		for update
	`, command.SubmissionID.String(), command.TaskID.String()).Scan(&submissionState, &rawWorkerID)
	if errors.Is(scanErr, pgx.ErrNoRows) {
		return ledger.RequestChangesRejected{Reason: core.NewDomainError(core.ErrorCodeNotFound, "submission was not found for the task")}
	}
	if scanErr != nil {
		return ledger.RequestChangesRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "read submission failed")}
	}
	if submissionState != "submitted" {
		return ledger.RequestChangesRejected{Reason: core.NewDomainError(core.ErrorCodeConflict, "only submitted work can receive requested changes")}
	}

	if _, err := tx.Exec(ctx, `
		update submissions
		set state = 'changes_requested', review_note = $2, reviewed_by_user_id = $3, review_recorded_at = now(), state_recorded_at = now()
		where id = $1
	`, command.SubmissionID.String(), command.ReviewNote.String(), command.RequesterUserID.String()); err != nil {
		return ledger.RequestChangesRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "request submission changes failed")}
	}

	if _, err := tx.Exec(ctx, `
		update task_reservations
		set state = 'active', state_recorded_at = now()
		where task_id = $1 and assignee_kind = 'user' and user_id = $2 and state = 'submitted'
	`, command.TaskID.String(), rawWorkerID); err != nil {
		return ledger.RequestChangesRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "reactivate task reservation failed")}
	}

	if err := tx.Commit(ctx); err != nil {
		return ledger.RequestChangesRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "commit request changes transaction failed")}
	}

	return ledger.ChangesRequested{TaskID: command.TaskID, SubmissionID: command.SubmissionID, ReviewNote: command.ReviewNote.String()}
}

func (store LedgerStore) RejectSubmission(ctx context.Context, command ledger.RejectStoreCommand) ledger.RejectResult {
	tx, err := store.pool.Begin(ctx)
	if err != nil {
		return ledger.RejectRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "begin reject submission transaction failed")}
	}
	defer func() { _ = tx.Rollback(ctx) }()

	taskResult := lockTaskForReview(ctx, tx, command.TaskID, command.RequesterUserID, "review submissions for")
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
	if errors.Is(scanErr, pgx.ErrNoRows) {
		return ledger.RejectRejected{Reason: core.NewDomainError(core.ErrorCodeNotFound, "submission was not found for the task")}
	}
	if scanErr != nil {
		return ledger.RejectRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "read submission failed")}
	}
	if submissionState == "rejected" {
		if reviewKey == command.IdempotencyKey.String() {
			return ledger.SubmissionRejected{TaskID: command.TaskID, SubmissionID: command.SubmissionID, Payout: ledger.NoPayout{}, Tip: ledger.NoTip{}}
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

	tipResult := payCreditTip(ctx, tx, command.TaskID, command.RequesterUserID, rawWorkerID, command.TipDebitEntryID, command.TipCreditEntryID, command.IdempotencyKey, command.TipSelection)
	tip, tipMatched := tipResult.(tipResolved)
	if !tipMatched {
		return ledger.RejectRejected{Reason: tipResult.(tipRejected).reason}
	}

	if _, err := tx.Exec(ctx, `
		update submissions
		set state = 'rejected', review_note = $2, reviewed_by_user_id = $3, review_recorded_at = now(), review_idempotency_key = $4, state_recorded_at = now()
		where id = $1
	`, command.SubmissionID.String(), command.ReviewNote.String(), command.RequesterUserID.String(), command.IdempotencyKey.String()); err != nil {
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
		if _, err := tx.Exec(ctx, `
			insert into task_implementor_bans (task_id, assignee_kind, assignee_key, user_id, banned_by_user_id)
			values ($1, 'user', $2, $3, $4)
			on conflict (task_id, assignee_kind, assignee_key) do nothing
		`, command.TaskID.String(), rawWorkerID, rawWorkerID, command.RequesterUserID.String()); err != nil {
			return ledger.RejectRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "ban task implementor failed")}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return ledger.RejectRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "commit reject submission transaction failed")}
	}

	return ledger.SubmissionRejected{TaskID: command.TaskID, SubmissionID: command.SubmissionID, Payout: payout.outcome, Tip: tip.outcome}
}

func (store LedgerStore) RefundTask(ctx context.Context, command ledger.RefundStoreCommand) ledger.RefundResult {
	tx, err := store.pool.Begin(ctx)
	if err != nil {
		return ledger.RefundRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "begin refund task transaction failed")}
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if reason := lockTaskRefundable(ctx, tx, command.TaskID, command.RequesterUserID); reason != nil {
		return ledger.RefundRejected{Reason: *reason}
	}

	var keyExists bool
	if err := tx.QueryRow(ctx, "select exists(select 1 from ledger_entries where idempotency_key = $1)", command.IdempotencyKey.String()).Scan(&keyExists); err != nil {
		return ledger.RefundRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "check refund idempotency failed")}
	}

	var rawFunderAccountID string
	var amount int64
	scanErr := tx.QueryRow(ctx, "select funder_account_id::text, credit_amount from task_funds where task_id = $1 for update", command.TaskID.String()).Scan(&rawFunderAccountID, &amount)
	if errors.Is(scanErr, pgx.ErrNoRows) {
		return replayOrRejectRefund(ctx, tx, command, keyExists)
	}
	if scanErr != nil {
		return ledger.RefundRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "read task fund failed")}
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

	if err := tx.Commit(ctx); err != nil {
		return ledger.RefundRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "commit refund task transaction failed")}
	}

	return ledger.TaskRefunded{Fund: fundValue.value}
}

// lockTaskRefundable locks the task and authorizes the refund. A refund is
// permitted for the task owner (its creator) or the user currently holding the
// active reservation (the implementor), and only while the task is not yet
// awarded (still draft or open).
func lockTaskRefundable(ctx context.Context, tx pgx.Tx, taskID core.TaskID, requester core.UserID) *core.DomainError {
	var state string
	var rawCreatedBy string
	scanErr := tx.QueryRow(ctx, "select state, created_by_user_id::text from tasks where id = $1 for update", taskID.String()).Scan(&state, &rawCreatedBy)
	if errors.Is(scanErr, pgx.ErrNoRows) {
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
func replayOrRejectRefund(ctx context.Context, tx pgx.Tx, command ledger.RefundStoreCommand, keyExists bool) ledger.RefundResult {
	if !keyExists {
		return ledger.RefundRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "task has nothing to refund")}
	}
	var rawFunderAccountID string
	var amount int64
	scanErr := tx.QueryRow(ctx, "select account_id::text, amount from ledger_entries where task_id = $1 and kind = 'task_refund' and idempotency_key = $2", command.TaskID.String(), command.IdempotencyKey.String()).Scan(&rawFunderAccountID, &amount)
	if errors.Is(scanErr, pgx.ErrNoRows) {
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
	return ledger.TaskRefunded{Fund: fundValue.value}
}

func (store LedgerStore) Balance(ctx context.Context, owner core.UserID) ledger.BalanceResult {
	var spendable, allocated int64
	err := store.pool.QueryRow(ctx, `
		select
			coalesce((select sum(ledger_entries.amount) from ledger_entries join credit_accounts on credit_accounts.id = ledger_entries.account_id where credit_accounts.user_id = $1), 0),
			coalesce((select sum(task_funds.credit_amount) from task_funds join credit_accounts on credit_accounts.id = task_funds.funder_account_id where credit_accounts.user_id = $1), 0)
	`, owner.String()).Scan(&spendable, &allocated)
	if err != nil {
		return ledger.BalanceRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "read balance failed")}
	}
	return ledger.BalanceFound{Value: ledger.NewBalance(spendable, allocated)}
}

func (store LedgerStore) ListEntries(ctx context.Context, owner core.UserID, page core.Page) ledger.ListEntriesResult {
	rows, err := store.pool.Query(ctx, `
		select ledger_entries.id::text, ledger_entries.kind, ledger_entries.amount, coalesce(ledger_entries.task_id::text, '')
		from ledger_entries
		join credit_accounts on credit_accounts.id = ledger_entries.account_id
		where credit_accounts.user_id = $1
		order by ledger_entries.created_at, ledger_entries.id
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
		if err := rows.Scan(&rawID, &rawKind, &amount, &rawTaskID); err != nil {
			return ledger.ListEntriesRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "scan ledger entry failed")}
		}
		entryResult := parseLedgerEntry(rawID, rawKind, amount, rawTaskID)
		entry, matched := entryResult.(ledgerEntryParsed)
		if !matched {
			return ledger.ListEntriesRejected{Reason: entryResult.(ledgerEntryParseRejected).reason}
		}
		entries = append(entries, entry.value)
	}
	if err := rows.Err(); err != nil {
		return ledger.ListEntriesRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "read ledger entries failed")}
	}
	return ledger.EntriesListed{Values: entries}
}

func (store LedgerStore) ListOrganizationEntries(ctx context.Context, organizationID core.OrganizationID, page core.Page) ledger.ListEntriesResult {
	rows, err := store.pool.Query(ctx, `
		select ledger_entries.id::text, ledger_entries.kind, ledger_entries.amount, coalesce(ledger_entries.task_id::text, '')
		from ledger_entries
		join credit_accounts on credit_accounts.id = ledger_entries.account_id
		where credit_accounts.organization_id = $1
		order by ledger_entries.created_at, ledger_entries.id
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
		if err := rows.Scan(&rawID, &rawKind, &amount, &rawTaskID); err != nil {
			return ledger.ListEntriesRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "scan organization ledger entry failed")}
		}
		entryResult := parseLedgerEntry(rawID, rawKind, amount, rawTaskID)
		entry, matched := entryResult.(ledgerEntryParsed)
		if !matched {
			return ledger.ListEntriesRejected{Reason: entryResult.(ledgerEntryParseRejected).reason}
		}
		entries = append(entries, entry.value)
	}
	if err := rows.Err(); err != nil {
		return ledger.ListEntriesRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "read organization ledger entries failed")}
	}
	return ledger.EntriesListed{Values: entries}
}
