package db

import (
	"context"
	"errors"

	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/ledger"
	"github.com/jackc/pgx/v5"
)

type accountLockResult interface {
	accountLockResult()
}

type accountLocked struct {
	id       string
	parsedID core.CreditAccountID
}

type accountLockRejected struct {
	reason core.DomainError
}

func (accountLocked) accountLockResult() {}

func (accountLockRejected) accountLockResult() {}

func lockUserAccount(ctx context.Context, tx pgx.Tx, userID core.UserID) accountLockResult {
	var rawID string
	scanErr := tx.QueryRow(ctx, "select id::text from credit_accounts where user_id = $1 for update", userID.String()).Scan(&rawID)
	if errors.Is(scanErr, pgx.ErrNoRows) {
		return accountLockRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "user has no credit account")}
	}
	if scanErr != nil {
		return accountLockRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "lock credit account failed")}
	}
	parsedResult := core.ParseCreditAccountID(rawID)
	parsed, matched := parsedResult.(core.CreditAccountIDCreated)
	if !matched {
		return accountLockRejected{reason: parsedResult.(core.CreditAccountIDRejected).Reason}
	}
	return accountLocked{id: rawID, parsedID: parsed.Value}
}

type taskLockResult interface {
	taskLockResult()
}

type taskLocked struct {
	state string
}

type taskLockRejected struct {
	reason core.DomainError
}

func (taskLocked) taskLockResult() {}

func (taskLockRejected) taskLockResult() {}

func lockTaskOwnedBy(ctx context.Context, tx pgx.Tx, taskID core.TaskID, requester core.UserID, action string) taskLockResult {
	var state string
	var rawCreatedBy string
	scanErr := tx.QueryRow(ctx, "select state, created_by_user_id::text from tasks where id = $1 for update", taskID.String()).Scan(&state, &rawCreatedBy)
	if errors.Is(scanErr, pgx.ErrNoRows) {
		return taskLockRejected{reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "task was not found")}
	}
	if scanErr != nil {
		return taskLockRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "lock task failed")}
	}
	if rawCreatedBy != requester.String() {
		return taskLockRejected{reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "only the task owner can "+action+" the task")}
	}
	return taskLocked{state: state}
}

// findEscrowForKey returns a non-nil FundResult when a fund command with the
// given idempotency key has already been recorded for the task.
func findEscrowForKey(ctx context.Context, tx pgx.Tx, key string, taskID core.TaskID) ledger.FundResult {
	var exists bool
	if err := tx.QueryRow(ctx, "select exists(select 1 from ledger_entries where idempotency_key = $1)", key).Scan(&exists); err != nil {
		return ledger.FundRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "check fund idempotency failed")}
	}
	if !exists {
		return nil
	}

	var rawFunderAccountID string
	var amount int64
	var escrowState string
	scanErr := tx.QueryRow(ctx, "select funder_account_id::text, amount, state from task_escrows where task_id = $1", taskID.String()).Scan(&rawFunderAccountID, &amount, &escrowState)
	if errors.Is(scanErr, pgx.ErrNoRows) {
		return ledger.FundRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "idempotency key was used for a different command")}
	}
	if scanErr != nil {
		return ledger.FundRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "read existing escrow failed")}
	}

	stateResult := ledger.ParseEscrowState(escrowState)
	stateAccepted, matched := stateResult.(ledger.EscrowStateAccepted)
	if !matched {
		return ledger.FundRejected{Reason: stateResult.(ledger.EscrowStateRejected).Reason}
	}
	escrowResult := buildEscrow(taskID, rawFunderAccountID, amount, stateAccepted.Value)
	built, builtMatched := escrowResult.(escrowBuilt)
	if !builtMatched {
		return ledger.FundRejected{Reason: escrowResult.(escrowBuildRejected).reason}
	}
	return ledger.TaskFunded{Escrow: built.value}
}

type payoutResult interface {
	payoutResult()
}

type payoutResolved struct {
	outcome ledger.PayoutOutcome
}

type payoutRejected struct {
	reason core.DomainError
}

func (payoutResolved) payoutResult() {}

func (payoutRejected) payoutResult() {}

func payOutEscrow(ctx context.Context, tx pgx.Tx, command ledger.AcceptStoreCommand, submitterKind string, rawWorkerID string) payoutResult {
	var rawFunderAccountID string
	var amount int64
	var escrowState string
	scanErr := tx.QueryRow(ctx, "select funder_account_id::text, amount, state from task_escrows where task_id = $1 for update", command.TaskID.String()).Scan(&rawFunderAccountID, &amount, &escrowState)
	if errors.Is(scanErr, pgx.ErrNoRows) {
		return payoutResolved{outcome: ledger.NoPayout{}}
	}
	if scanErr != nil {
		return payoutRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "read task escrow failed")}
	}
	if escrowState != "held" {
		return payoutResolved{outcome: ledger.NoPayout{}}
	}

	if submitterKind != "authenticated" {
		return payoutRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "anonymous submissions cannot receive a credit payout")}
	}

	workerResult := core.ParseUserID(rawWorkerID)
	worker, workerMatched := workerResult.(core.UserIDCreated)
	if !workerMatched {
		return payoutRejected{reason: workerResult.(core.UserIDRejected).Reason}
	}

	var workerAccountID string
	accountErr := tx.QueryRow(ctx, "select id::text from credit_accounts where user_id = $1 for update", rawWorkerID).Scan(&workerAccountID)
	if errors.Is(accountErr, pgx.ErrNoRows) {
		return payoutRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "submission author has no credit account")}
	}
	if accountErr != nil {
		return payoutRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "lock worker credit account failed")}
	}

	if _, err := tx.Exec(ctx, `
		insert into ledger_entries (id, account_id, kind, amount, task_id, idempotency_key)
		values ($1, $2, 'task_payout', $3, $4, $5)
	`, command.PayoutEntryID.String(), workerAccountID, amount, command.TaskID.String(), command.IdempotencyKey.String()); err != nil {
		return payoutRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "insert task payout ledger entry failed")}
	}

	if _, err := tx.Exec(ctx, "update task_escrows set state = 'released', state_recorded_at = now() where task_id = $1", command.TaskID.String()); err != nil {
		return payoutRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "release task escrow failed")}
	}

	amountResult := ledger.NewCreditAmount(amount)
	amountAccepted, amountMatched := amountResult.(ledger.CreditAmountAccepted)
	if !amountMatched {
		return payoutRejected{reason: amountResult.(ledger.CreditAmountRejected).Reason}
	}
	return payoutResolved{outcome: ledger.CreditPayout{WorkerUserID: worker.Value, Amount: amountAccepted.Value}}
}

// idempotentAccept returns the prior acceptance outcome when a submission is
// already accepted, so accept commands can be retried safely.
func idempotentAccept(ctx context.Context, tx pgx.Tx, command ledger.AcceptStoreCommand, submitterKind string, rawWorkerID string) ledger.AcceptResult {
	var escrowState string
	var amount int64
	scanErr := tx.QueryRow(ctx, "select state, amount from task_escrows where task_id = $1", command.TaskID.String()).Scan(&escrowState, &amount)
	if errors.Is(scanErr, pgx.ErrNoRows) {
		return ledger.SubmissionAccepted{TaskID: command.TaskID, SubmissionID: command.SubmissionID, Payout: ledger.NoPayout{}}
	}
	if scanErr != nil {
		return ledger.AcceptRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "read task escrow failed")}
	}
	if escrowState != "released" || submitterKind != "authenticated" {
		return ledger.SubmissionAccepted{TaskID: command.TaskID, SubmissionID: command.SubmissionID, Payout: ledger.NoPayout{}}
	}

	worker, workerMatched := core.ParseUserID(rawWorkerID).(core.UserIDCreated)
	amountAccepted, amountMatched := ledger.NewCreditAmount(amount).(ledger.CreditAmountAccepted)
	if !workerMatched || !amountMatched {
		return ledger.SubmissionAccepted{TaskID: command.TaskID, SubmissionID: command.SubmissionID, Payout: ledger.NoPayout{}}
	}
	return ledger.SubmissionAccepted{TaskID: command.TaskID, SubmissionID: command.SubmissionID, Payout: ledger.CreditPayout{WorkerUserID: worker.Value, Amount: amountAccepted.Value}}
}

type escrowBuildResult interface {
	escrowBuildResult()
}

type escrowBuilt struct {
	value ledger.TaskEscrow
}

type escrowBuildRejected struct {
	reason core.DomainError
}

func (escrowBuilt) escrowBuildResult() {}

func (escrowBuildRejected) escrowBuildResult() {}

func buildEscrow(taskID core.TaskID, rawFunderAccountID string, amount int64, state ledger.EscrowState) escrowBuildResult {
	accountResult := core.ParseCreditAccountID(rawFunderAccountID)
	account, matched := accountResult.(core.CreditAccountIDCreated)
	if !matched {
		return escrowBuildRejected{reason: accountResult.(core.CreditAccountIDRejected).Reason}
	}
	amountResult := ledger.NewCreditAmount(amount)
	amountAccepted, amountMatched := amountResult.(ledger.CreditAmountAccepted)
	if !amountMatched {
		return escrowBuildRejected{reason: amountResult.(ledger.CreditAmountRejected).Reason}
	}
	return escrowBuilt{value: ledger.TaskEscrow{
		TaskID:          taskID,
		FunderAccountID: account.Value,
		Amount:          amountAccepted.Value,
		State:           state,
	}}
}

type ledgerEntryParseResult interface {
	ledgerEntryParseResult()
}

type ledgerEntryParsed struct {
	value ledger.LedgerEntry
}

type ledgerEntryParseRejected struct {
	reason core.DomainError
}

func (ledgerEntryParsed) ledgerEntryParseResult() {}

func (ledgerEntryParseRejected) ledgerEntryParseResult() {}

func parseLedgerEntry(rawID string, rawKind string, amount int64, rawTaskID string) ledgerEntryParseResult {
	idResult := core.ParseLedgerEntryID(rawID)
	entryID, idMatched := idResult.(core.LedgerEntryIDCreated)
	if !idMatched {
		return ledgerEntryParseRejected{reason: idResult.(core.LedgerEntryIDRejected).Reason}
	}
	kindResult := ledger.ParseEntryKind(rawKind)
	kind, kindMatched := kindResult.(ledger.EntryKindAccepted)
	if !kindMatched {
		return ledgerEntryParseRejected{reason: kindResult.(ledger.EntryKindRejected).Reason}
	}
	amountResult := ledger.ParseSignedAmount(amount)
	signed, amountMatched := amountResult.(ledger.SignedAmountAccepted)
	if !amountMatched {
		return ledgerEntryParseRejected{reason: amountResult.(ledger.SignedAmountRejected).Reason}
	}
	taskRefResult := parseTaskReference(rawTaskID)
	taskRef, taskRefMatched := taskRefResult.(taskReferenceParsed)
	if !taskRefMatched {
		return ledgerEntryParseRejected{reason: taskRefResult.(taskReferenceParseRejected).reason}
	}
	return ledgerEntryParsed{value: ledger.LedgerEntry{
		ID:      entryID.Value,
		Kind:    kind.Value,
		Amount:  signed.Value,
		TaskRef: taskRef.value,
	}}
}

type taskReferenceResult interface {
	taskReferenceResult()
}

type taskReferenceParsed struct {
	value ledger.TaskReference
}

type taskReferenceParseRejected struct {
	reason core.DomainError
}

func (taskReferenceParsed) taskReferenceResult() {}

func (taskReferenceParseRejected) taskReferenceResult() {}

func parseTaskReference(rawTaskID string) taskReferenceResult {
	if rawTaskID == "" {
		return taskReferenceParsed{value: ledger.NoTaskReference{}}
	}
	result := core.ParseTaskID(rawTaskID)
	taskID, matched := result.(core.TaskIDCreated)
	if !matched {
		return taskReferenceParseRejected{reason: result.(core.TaskIDRejected).Reason}
	}
	return taskReferenceParsed{value: ledger.TaskReferenced{TaskID: taskID.Value}}
}
