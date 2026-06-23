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

	if existing := findEscrowForKey(ctx, tx, command.IdempotencyKey.String(), command.TaskID); existing != nil {
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
	if taskRow.state != "draft" {
		return ledger.FundRejected{Reason: core.NewDomainError(core.ErrorCodeConflict, "only draft tasks can be funded")}
	}
	if rejected, matched := requireCreditRewardFunding(taskRow, command.Amount).(fundingRewardRejected); matched {
		return ledger.FundRejected{Reason: rejected.reason}
	}

	return completeFunding(ctx, tx, account, command.TaskID, command.Amount, command.EntryID, command.IdempotencyKey, "insufficient credits to fund the task")
}

func (store LedgerStore) FundTaskFromOrganization(ctx context.Context, command ledger.OrganizationFundStoreCommand) ledger.FundResult {
	tx, err := store.pool.Begin(ctx)
	if err != nil {
		return ledger.FundRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "begin fund task transaction failed")}
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if existing := findEscrowForKey(ctx, tx, command.IdempotencyKey.String(), command.TaskID); existing != nil {
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
	if taskRow.state != "draft" {
		return ledger.FundRejected{Reason: core.NewDomainError(core.ErrorCodeConflict, "only draft tasks can be funded")}
	}
	if rejected, matched := requireCreditRewardFunding(taskRow, command.Amount).(fundingRewardRejected); matched {
		return ledger.FundRejected{Reason: rejected.reason}
	}

	return completeFunding(ctx, tx, account, command.TaskID, command.Amount, command.EntryID, command.IdempotencyKey, "insufficient organization credits to fund the task")
}

func (store LedgerStore) OrganizationBalance(ctx context.Context, organizationID core.OrganizationID) ledger.BalanceResult {
	var balance int64
	err := store.pool.QueryRow(ctx, `
		select coalesce(sum(ledger_entries.amount), 0)
		from ledger_entries
		join credit_accounts on credit_accounts.id = ledger_entries.account_id
		where credit_accounts.organization_id = $1
	`, organizationID.String()).Scan(&balance)
	if err != nil {
		return ledger.BalanceRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "read organization balance failed")}
	}
	return ledger.BalanceFound{Value: ledger.NewBalance(balance)}
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
			return ledger.AcceptRejected{Reason: core.NewDomainError(core.ErrorCodeConflict, "credit reward escrow is missing")}
		}
	}

	tipResult := payCreditTip(ctx, tx, command.TaskID, command.RequesterUserID, rawWorkerID, command.TipDebitEntryID, command.TipCreditEntryID, command.TipSelection)
	tip, tipMatched := tipResult.(tipResolved)
	if !tipMatched {
		return ledger.AcceptRejected{Reason: tipResult.(tipRejected).reason}
	}

	if err := tx.Commit(ctx); err != nil {
		return ledger.AcceptRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "commit accept submission transaction failed")}
	}

	return ledger.SubmissionAccepted{TaskID: command.TaskID, SubmissionID: command.SubmissionID, Payout: outcome, Tip: tip.outcome}
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

	payoutResult := payReviewEscrow(ctx, tx, reviewEscrowCommand{
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

	tipResult := payCreditTip(ctx, tx, command.TaskID, command.RequesterUserID, rawWorkerID, command.TipDebitEntryID, command.TipCreditEntryID, command.TipSelection)
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

	taskResult := lockTaskOwnedBy(ctx, tx, command.TaskID, command.RequesterUserID, "refund")
	taskRow, taskMatched := taskResult.(taskLocked)
	if !taskMatched {
		return ledger.RefundRejected{Reason: taskResult.(taskLockRejected).reason}
	}
	if taskRow.state != "draft" && taskRow.state != "open" {
		return ledger.RefundRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "only draft or open tasks can be refunded")}
	}

	var keyExists bool
	if err := tx.QueryRow(ctx, "select exists(select 1 from ledger_entries where idempotency_key = $1)", command.IdempotencyKey.String()).Scan(&keyExists); err != nil {
		return ledger.RefundRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "check refund idempotency failed")}
	}

	var rawFunderAccountID string
	var amount int64
	var escrowState string
	scanErr := tx.QueryRow(ctx, `
		select funder_account_id::text, amount, state
		from task_escrows
		where task_id = $1
		for update
	`, command.TaskID.String()).Scan(&rawFunderAccountID, &amount, &escrowState)
	if errors.Is(scanErr, pgx.ErrNoRows) {
		return ledger.RefundRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "task has no escrow to refund")}
	}
	if scanErr != nil {
		return ledger.RefundRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "read task escrow failed")}
	}

	escrowResult := buildEscrow(command.TaskID, rawFunderAccountID, amount, ledger.EscrowStateRefunded)
	escrowValue, escrowMatched := escrowResult.(escrowBuilt)
	if !escrowMatched {
		return ledger.RefundRejected{Reason: escrowResult.(escrowBuildRejected).reason}
	}

	if keyExists && escrowState == "refunded" {
		return ledger.TaskRefunded{Escrow: escrowValue.value}
	}
	if escrowState != "held" {
		return ledger.RefundRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "task escrow is not held")}
	}

	_, err = tx.Exec(ctx, `
		insert into ledger_entries (id, account_id, kind, amount, task_id, idempotency_key)
		values ($1, $2, 'task_refund', $3, $4, $5)
	`, command.EntryID.String(), rawFunderAccountID, amount, command.TaskID.String(), command.IdempotencyKey.String())
	if err != nil {
		return ledger.RefundRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "insert task refund ledger entry failed")}
	}

	if _, err := tx.Exec(ctx, "update task_escrows set state = 'refunded', state_recorded_at = now() where task_id = $1", command.TaskID.String()); err != nil {
		return ledger.RefundRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "update task escrow failed")}
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

	return ledger.TaskRefunded{Escrow: escrowValue.value}
}

func (store LedgerStore) Balance(ctx context.Context, owner core.UserID) ledger.BalanceResult {
	var balance int64
	err := store.pool.QueryRow(ctx, `
		select coalesce(sum(ledger_entries.amount), 0)
		from ledger_entries
		join credit_accounts on credit_accounts.id = ledger_entries.account_id
		where credit_accounts.user_id = $1
	`, owner.String()).Scan(&balance)
	if err != nil {
		return ledger.BalanceRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "read balance failed")}
	}
	return ledger.BalanceFound{Value: ledger.NewBalance(balance)}
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
