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
		return ledger.FundRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "only draft tasks can be funded")}
	}

	var escrowExists bool
	if err := tx.QueryRow(ctx, "select exists(select 1 from task_escrows where task_id = $1)", command.TaskID.String()).Scan(&escrowExists); err != nil {
		return ledger.FundRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "check existing escrow failed")}
	}
	if escrowExists {
		return ledger.FundRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "task is already funded")}
	}

	var balance int64
	if err := tx.QueryRow(ctx, "select coalesce(sum(amount), 0) from ledger_entries where account_id = $1", account.id).Scan(&balance); err != nil {
		return ledger.FundRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "read account balance failed")}
	}
	if balance < command.Amount.Int64() {
		return ledger.FundRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "insufficient credits to fund the task")}
	}

	_, err = tx.Exec(ctx, `
		insert into task_escrows (task_id, funder_account_id, amount, state)
		values ($1, $2, $3, 'held')
	`, command.TaskID.String(), account.id, command.Amount.Int64())
	if err != nil {
		return ledger.FundRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "insert task escrow failed")}
	}

	_, err = tx.Exec(ctx, `
		insert into ledger_entries (id, account_id, kind, amount, task_id, idempotency_key)
		values ($1, $2, 'task_escrow', $3, $4, $5)
	`, command.EntryID.String(), account.id, -command.Amount.Int64(), command.TaskID.String(), command.IdempotencyKey.String())
	if err != nil {
		return ledger.FundRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "insert task escrow ledger entry failed")}
	}

	if err := tx.Commit(ctx); err != nil {
		return ledger.FundRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "commit fund task transaction failed")}
	}

	return ledger.TaskFunded{Escrow: ledger.TaskEscrow{
		TaskID:          command.TaskID,
		FunderAccountID: account.parsedID,
		Amount:          command.Amount,
		State:           ledger.EscrowStateHeld,
	}}
}

func (store LedgerStore) AcceptSubmission(ctx context.Context, command ledger.AcceptStoreCommand) ledger.AcceptResult {
	tx, err := store.pool.Begin(ctx)
	if err != nil {
		return ledger.AcceptRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "begin accept submission transaction failed")}
	}
	defer func() { _ = tx.Rollback(ctx) }()

	taskResult := lockTaskOwnedBy(ctx, tx, command.TaskID, command.RequesterUserID, "accept submissions for")
	taskRow, taskMatched := taskResult.(taskLocked)
	if !taskMatched {
		return ledger.AcceptRejected{Reason: taskResult.(taskLockRejected).reason}
	}

	var submissionState string
	var submitterKind string
	var rawWorkerID string
	scanErr := tx.QueryRow(ctx, `
		select state, submitter_kind, coalesce(user_id::text, '')
		from submissions
		where id = $1 and task_id = $2
	`, command.SubmissionID.String(), command.TaskID.String()).Scan(&submissionState, &submitterKind, &rawWorkerID)
	if errors.Is(scanErr, pgx.ErrNoRows) {
		return ledger.AcceptRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "submission was not found for the task")}
	}
	if scanErr != nil {
		return ledger.AcceptRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "read submission failed")}
	}

	if submissionState == "accepted" {
		return idempotentAccept(ctx, tx, command, submitterKind, rawWorkerID)
	}
	if submissionState != "submitted" {
		return ledger.AcceptRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "only valid submissions can be accepted")}
	}
	if taskRow.state != "open" {
		return ledger.AcceptRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "only open tasks can accept submissions")}
	}

	if _, err := tx.Exec(ctx, "update submissions set state = 'accepted', state_recorded_at = now() where id = $1", command.SubmissionID.String()); err != nil {
		if isUniqueViolation(err) {
			return ledger.AcceptRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "task already has an accepted submission")}
		}
		return ledger.AcceptRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "accept submission failed")}
	}

	if _, err := tx.Exec(ctx, "update tasks set state = 'closed', state_recorded_at = now() where id = $1", command.TaskID.String()); err != nil {
		return ledger.AcceptRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "close task failed")}
	}

	payoutResult := payOutEscrow(ctx, tx, command, submitterKind, rawWorkerID)
	payout, payoutMatched := payoutResult.(payoutResolved)
	if !payoutMatched {
		return ledger.AcceptRejected{Reason: payoutResult.(payoutRejected).reason}
	}

	if err := tx.Commit(ctx); err != nil {
		return ledger.AcceptRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "commit accept submission transaction failed")}
	}

	return ledger.SubmissionAccepted{TaskID: command.TaskID, SubmissionID: command.SubmissionID, Payout: payout.outcome}
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

func (store LedgerStore) ListEntries(ctx context.Context, owner core.UserID) ledger.ListEntriesResult {
	rows, err := store.pool.Query(ctx, `
		select ledger_entries.id::text, ledger_entries.kind, ledger_entries.amount, coalesce(ledger_entries.task_id::text, '')
		from ledger_entries
		join credit_accounts on credit_accounts.id = ledger_entries.account_id
		where credit_accounts.user_id = $1
		order by ledger_entries.created_at, ledger_entries.id
	`, owner.String())
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
