package db

import (
	"context"
	"errors"

	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/ledger"
	"github.com/e6qu/sharecrop/internal/org"
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

// completeFunding holds an escrow against a locked, draft, owner-verified task,
// debiting the locked funder account. It is shared by user and organization
// funding so the escrow mechanics live in one place.
func completeFunding(ctx context.Context, tx pgx.Tx, account accountLocked, taskID core.TaskID, amount ledger.CreditAmount, entryID core.LedgerEntryID, key ledger.IdempotencyKey, insufficientMessage string) ledger.FundResult {
	var escrowExists bool
	if err := tx.QueryRow(ctx, "select exists(select 1 from task_escrows where task_id = $1)", taskID.String()).Scan(&escrowExists); err != nil {
		return ledger.FundRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "check existing escrow failed")}
	}
	if escrowExists {
		return ledger.FundRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "task is already funded")}
	}

	var balance int64
	if err := tx.QueryRow(ctx, "select coalesce(sum(amount), 0) from ledger_entries where account_id = $1", account.id).Scan(&balance); err != nil {
		return ledger.FundRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "read account balance failed")}
	}
	if balance < amount.Int64() {
		return ledger.FundRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, insufficientMessage)}
	}

	if _, err := tx.Exec(ctx, "insert into task_escrows (task_id, funder_account_id, amount, state) values ($1, $2, $3, 'held')", taskID.String(), account.id, amount.Int64()); err != nil {
		return ledger.FundRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "insert task escrow failed")}
	}
	if _, err := tx.Exec(ctx, "insert into ledger_entries (id, account_id, kind, amount, task_id, idempotency_key) values ($1, $2, 'task_escrow', $3, $4, $5)", entryID.String(), account.id, -amount.Int64(), taskID.String(), key.String()); err != nil {
		return ledger.FundRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "insert task escrow ledger entry failed")}
	}

	if err := tx.Commit(ctx); err != nil {
		return ledger.FundRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "commit fund task transaction failed")}
	}

	return ledger.TaskFunded{Escrow: ledger.TaskEscrow{
		TaskID:          taskID,
		FunderAccountID: account.parsedID,
		Amount:          amount,
		State:           ledger.EscrowStateHeld,
	}}
}

func lockOrganizationAccount(ctx context.Context, tx pgx.Tx, organizationID core.OrganizationID) accountLockResult {
	var rawID string
	scanErr := tx.QueryRow(ctx, "select id::text from credit_accounts where organization_id = $1 for update", organizationID.String()).Scan(&rawID)
	if errors.Is(scanErr, pgx.ErrNoRows) {
		return accountLockRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "organization has no credit account")}
	}
	if scanErr != nil {
		return accountLockRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "lock organization credit account failed")}
	}
	parsedResult := core.ParseCreditAccountID(rawID)
	parsed, matched := parsedResult.(core.CreditAccountIDCreated)
	if !matched {
		return accountLockRejected{reason: parsedResult.(core.CreditAccountIDRejected).Reason}
	}
	return accountLocked{id: rawID, parsedID: parsed.Value}
}

func lockTaskOwnedByOrganization(ctx context.Context, tx pgx.Tx, taskID core.TaskID, organizationID core.OrganizationID) taskLockResult {
	var state string
	var rawOrganizationID string
	var rewardKind string
	var rewardCreditAmount int64
	scanErr := tx.QueryRow(ctx, "select state, coalesce(organization_id::text, ''), reward_kind, coalesce(reward_credit_amount, 0) from tasks where id = $1 for update", taskID.String()).Scan(&state, &rawOrganizationID, &rewardKind, &rewardCreditAmount)
	if errors.Is(scanErr, pgx.ErrNoRows) {
		return taskLockRejected{reason: core.NewDomainError(core.ErrorCodeNotFound, "task was not found")}
	}
	if scanErr != nil {
		return taskLockRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "lock task failed")}
	}
	if rawOrganizationID != organizationID.String() {
		return taskLockRejected{reason: core.NewDomainError(core.ErrorCodePermissionDenied, "task is not owned by the organization")}
	}
	return taskLocked{state: state, rewardKind: rewardKind, rewardCreditAmount: rewardCreditAmount}
}

type taskLockResult interface {
	taskLockResult()
}

type taskLocked struct {
	state              string
	rewardKind         string
	rewardCreditAmount int64
}

type taskLockRejected struct {
	reason core.DomainError
}

func (taskLocked) taskLockResult() {}

func (taskLockRejected) taskLockResult() {}

func lockTaskOwnedBy(ctx context.Context, tx pgx.Tx, taskID core.TaskID, requester core.UserID, action string) taskLockResult {
	var state string
	var rawCreatedBy string
	var rewardKind string
	var rewardCreditAmount int64
	scanErr := tx.QueryRow(ctx, "select state, created_by_user_id::text, reward_kind, coalesce(reward_credit_amount, 0) from tasks where id = $1 for update", taskID.String()).Scan(&state, &rawCreatedBy, &rewardKind, &rewardCreditAmount)
	if errors.Is(scanErr, pgx.ErrNoRows) {
		return taskLockRejected{reason: core.NewDomainError(core.ErrorCodeNotFound, "task was not found")}
	}
	if scanErr != nil {
		return taskLockRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "lock task failed")}
	}
	if rawCreatedBy != requester.String() {
		return taskLockRejected{reason: core.NewDomainError(core.ErrorCodePermissionDenied, "only the task owner can "+action+" the task")}
	}
	return taskLocked{state: state, rewardKind: rewardKind, rewardCreditAmount: rewardCreditAmount}
}

// lockTaskForReview locks a task for a review action. The direct task creator is
// always authorized. For organization-owned tasks, a member with the
// review-submissions permission is also authorized. This mirrors the
// submission service's review-permission check, but resolves authorization in
// the same transaction as the review write so authorization cannot drift
// between the check and the mutation.
func lockTaskForReview(ctx context.Context, tx pgx.Tx, taskID core.TaskID, requester core.UserID, action string) taskLockResult {
	var state string
	var rawCreatedBy string
	var rawOrganizationID string
	var rewardKind string
	var rewardCreditAmount int64
	scanErr := tx.QueryRow(ctx, "select state, created_by_user_id::text, coalesce(organization_id::text, ''), reward_kind, coalesce(reward_credit_amount, 0) from tasks where id = $1 for update", taskID.String()).Scan(&state, &rawCreatedBy, &rawOrganizationID, &rewardKind, &rewardCreditAmount)
	if errors.Is(scanErr, pgx.ErrNoRows) {
		return taskLockRejected{reason: core.NewDomainError(core.ErrorCodeNotFound, "task was not found")}
	}
	if scanErr != nil {
		return taskLockRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "lock task failed")}
	}
	if rawCreatedBy == requester.String() {
		return taskLocked{state: state, rewardKind: rewardKind, rewardCreditAmount: rewardCreditAmount}
	}
	if rawOrganizationID != "" {
		check := reviewerOrganizationPermission(ctx, tx, rawOrganizationID, requester)
		if _, granted := check.(org.PermissionGranted); granted {
			return taskLocked{state: state, rewardKind: rewardKind, rewardCreditAmount: rewardCreditAmount}
		}
	}
	return taskLockRejected{reason: core.NewDomainError(core.ErrorCodePermissionDenied, "only the task owner or an organization reviewer can "+action+" the task")}
}

// reviewerOrganizationPermission resolves whether the requester holds the
// review-submissions permission in the given organization, evaluated in-tx.
func reviewerOrganizationPermission(ctx context.Context, tx pgx.Tx, rawOrganizationID string, requester core.UserID) org.PermissionCheck {
	rows, err := tx.Query(ctx, `
		select organization_membership_roles.role
		from organization_memberships
		join organization_membership_roles on organization_membership_roles.membership_id = organization_memberships.id
		where organization_memberships.organization_id = $1
			and organization_memberships.user_id = $2
			and organization_memberships.status = $3
	`, rawOrganizationID, requester.String(), org.MembershipStatusActive.String())
	if err != nil {
		return org.PermissionDenied{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "read reviewer roles failed")}
	}
	defer rows.Close()

	roles := make([]org.Role, 0)
	for rows.Next() {
		var rawRole string
		if err := rows.Scan(&rawRole); err != nil {
			return org.PermissionDenied{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "scan reviewer role failed")}
		}
		roleResult := org.ParseRole(rawRole)
		roleAccepted, matched := roleResult.(org.RoleAccepted)
		if !matched {
			return org.PermissionDenied{Reason: roleResult.(org.RoleRejected).Reason}
		}
		roles = append(roles, roleAccepted.Value)
	}
	if err := rows.Err(); err != nil {
		return org.PermissionDenied{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "read reviewer roles failed")}
	}
	return org.CheckPermission(roles, org.PermissionReviewSubmissions)
}

type fundingRewardResult interface {
	fundingRewardResult()
}

type fundingRewardAccepted struct{}

type fundingRewardRejected struct {
	reason core.DomainError
}

func (fundingRewardAccepted) fundingRewardResult() {}

func (fundingRewardRejected) fundingRewardResult() {}

func requireCreditRewardFunding(taskRow taskLocked, amount ledger.CreditAmount) fundingRewardResult {
	if taskRow.rewardKind != "credit" && taskRow.rewardKind != "bundle" {
		return fundingRewardRejected{reason: core.NewDomainError(core.ErrorCodeConflict, "task does not declare a credit reward")}
	}
	if taskRow.rewardCreditAmount != amount.Int64() {
		return fundingRewardRejected{reason: core.NewDomainError(core.ErrorCodeConflict, "funding amount must match the declared credit reward")}
	}
	return fundingRewardAccepted{}
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

func payOutEscrow(ctx context.Context, tx pgx.Tx, command ledger.AcceptStoreCommand, rawWorkerID string) payoutResult {
	return payReviewEscrow(ctx, tx, reviewEscrowCommand{
		taskID:            command.TaskID,
		rawWorkerID:       rawWorkerID,
		payoutEntryID:     command.PayoutEntryID,
		refundEntryID:     command.RefundEntryID,
		idempotencyKey:    command.IdempotencyKey,
		selection:         command.CreditSelection,
		closeTask:         true,
		missingIsNoPayout: true,
	})
}

type reviewEscrowCommand struct {
	taskID            core.TaskID
	rawWorkerID       string
	payoutEntryID     core.LedgerEntryID
	refundEntryID     core.LedgerEntryID
	idempotencyKey    ledger.IdempotencyKey
	selection         ledger.CreditReviewSelection
	closeTask         bool
	missingIsNoPayout bool
}

func payReviewEscrow(ctx context.Context, tx pgx.Tx, command reviewEscrowCommand) payoutResult {
	if _, noPayout := command.selection.(ledger.NoCreditReviewSelection); noPayout {
		return payoutResolved{outcome: ledger.NoPayout{}}
	}

	var rawFunderAccountID string
	var amount int64
	var escrowState string
	scanErr := tx.QueryRow(ctx, "select funder_account_id::text, amount, state from task_escrows where task_id = $1 for update", command.taskID.String()).Scan(&rawFunderAccountID, &amount, &escrowState)
	if errors.Is(scanErr, pgx.ErrNoRows) {
		if _, partial := command.selection.(ledger.PartialCreditReviewSelection); partial {
			return payoutRejected{reason: core.NewDomainError(core.ErrorCodeConflict, "credit reward escrow is missing")}
		}
		if command.missingIsNoPayout {
			return payoutResolved{outcome: ledger.NoPayout{}}
		}
		return payoutRejected{reason: core.NewDomainError(core.ErrorCodeConflict, "credit reward escrow is missing")}
	}
	if scanErr != nil {
		return payoutRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "read task escrow failed")}
	}
	if escrowState != "held" {
		if _, partial := command.selection.(ledger.PartialCreditReviewSelection); partial {
			return payoutRejected{reason: core.NewDomainError(core.ErrorCodeConflict, "credit reward escrow is not held")}
		}
		return payoutResolved{outcome: ledger.NoPayout{}}
	}

	payoutAmount := amount
	if partial, matched := command.selection.(ledger.PartialCreditReviewSelection); matched {
		payoutAmount = partial.Amount.Int64()
	}
	if payoutAmount > amount {
		return payoutRejected{reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "credit payout cannot exceed held escrow")}
	}

	workerResult := core.ParseUserID(command.rawWorkerID)
	worker, workerMatched := workerResult.(core.UserIDCreated)
	if !workerMatched {
		return payoutRejected{reason: workerResult.(core.UserIDRejected).Reason}
	}

	var workerAccountID string
	accountErr := tx.QueryRow(ctx, "select id::text from credit_accounts where user_id = $1 for update", command.rawWorkerID).Scan(&workerAccountID)
	if errors.Is(accountErr, pgx.ErrNoRows) {
		return payoutRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "submission author has no credit account")}
	}
	if accountErr != nil {
		return payoutRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "lock worker credit account failed")}
	}

	if _, err := tx.Exec(ctx, `
		insert into ledger_entries (id, account_id, kind, amount, task_id, idempotency_key)
		values ($1, $2, 'task_payout', $3, $4, $5)
	`, command.payoutEntryID.String(), workerAccountID, payoutAmount, command.taskID.String(), command.idempotencyKey.String()); err != nil {
		return payoutRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "insert task payout ledger entry failed")}
	}

	remaining := amount - payoutAmount
	if command.closeTask {
		if remaining > 0 {
			if _, err := tx.Exec(ctx, `
				insert into ledger_entries (id, account_id, kind, amount, task_id)
				values ($1, $2, 'task_refund', $3, $4)
			`, command.refundEntryID.String(), rawFunderAccountID, remaining, command.taskID.String()); err != nil {
				return payoutRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "insert partial accept refund ledger entry failed")}
			}
		}
		if _, err := tx.Exec(ctx, "update task_escrows set amount = $2, state = 'released', state_recorded_at = now() where task_id = $1", command.taskID.String(), payoutAmount); err != nil {
			return payoutRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "release task escrow failed")}
		}
	} else {
		nextState := "held"
		if remaining == 0 {
			nextState = "released"
		}
		nextAmount := remaining
		if nextAmount == 0 {
			nextAmount = payoutAmount
		}
		if _, err := tx.Exec(ctx, "update task_escrows set amount = $2, state = $3, state_recorded_at = now() where task_id = $1", command.taskID.String(), nextAmount, nextState); err != nil {
			return payoutRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "update task escrow after partial payout failed")}
		}
	}

	amountResult := ledger.NewCreditAmount(payoutAmount)
	amountAccepted, amountMatched := amountResult.(ledger.CreditAmountAccepted)
	if !amountMatched {
		return payoutRejected{reason: amountResult.(ledger.CreditAmountRejected).Reason}
	}
	return payoutResolved{outcome: ledger.CreditPayout{WorkerUserID: worker.Value, Amount: amountAccepted.Value}}
}

func payCreditTip(ctx context.Context, tx pgx.Tx, taskID core.TaskID, requester core.UserID, rawWorkerID string, debitEntryID core.LedgerEntryID, creditEntryID core.LedgerEntryID, selection ledger.TipSelection) tipResult {
	tip, matched := selection.(ledger.CreditTipSelection)
	if !matched {
		return tipResolved{outcome: ledger.NoTip{}}
	}

	requesterAccount := lockUserAccount(ctx, tx, requester)
	requesterLocked, requesterMatched := requesterAccount.(accountLocked)
	if !requesterMatched {
		return tipRejected{reason: requesterAccount.(accountLockRejected).reason}
	}

	var balance int64
	if err := tx.QueryRow(ctx, "select coalesce(sum(amount), 0) from ledger_entries where account_id = $1", requesterLocked.id).Scan(&balance); err != nil {
		return tipRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "read requester balance failed")}
	}
	if balance < tip.Amount.Int64() {
		return tipRejected{reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "insufficient credits to tip the implementor")}
	}

	workerResult := core.ParseUserID(rawWorkerID)
	worker, workerMatched := workerResult.(core.UserIDCreated)
	if !workerMatched {
		return tipRejected{reason: workerResult.(core.UserIDRejected).Reason}
	}

	var workerAccountID string
	accountErr := tx.QueryRow(ctx, "select id::text from credit_accounts where user_id = $1 for update", rawWorkerID).Scan(&workerAccountID)
	if errors.Is(accountErr, pgx.ErrNoRows) {
		return tipRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "submission author has no credit account")}
	}
	if accountErr != nil {
		return tipRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "lock worker credit account failed")}
	}

	if _, err := tx.Exec(ctx, "insert into ledger_entries (id, account_id, kind, amount, task_id) values ($1, $2, 'task_tip', $3, $4)", debitEntryID.String(), requesterLocked.id, -tip.Amount.Int64(), taskID.String()); err != nil {
		return tipRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "insert task tip debit failed")}
	}
	if _, err := tx.Exec(ctx, "insert into ledger_entries (id, account_id, kind, amount, task_id) values ($1, $2, 'task_tip', $3, $4)", creditEntryID.String(), workerAccountID, tip.Amount.Int64(), taskID.String()); err != nil {
		return tipRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "insert task tip credit failed")}
	}

	return tipResolved{outcome: ledger.CreditTip{WorkerUserID: worker.Value, Amount: tip.Amount}}
}

type tipResult interface {
	tipResult()
}

type tipResolved struct {
	outcome ledger.TipOutcome
}

type tipRejected struct {
	reason core.DomainError
}

func (tipResolved) tipResult() {}

func (tipRejected) tipResult() {}

func payOutCollectible(ctx context.Context, tx pgx.Tx, taskID core.TaskID, rawWorkerID string) payoutResult {
	var rawCollectibleID string
	var rewardState string
	scanErr := tx.QueryRow(ctx, "select collectible_id::text, state from task_collectible_rewards where task_id = $1 for update", taskID.String()).Scan(&rawCollectibleID, &rewardState)
	if errors.Is(scanErr, pgx.ErrNoRows) {
		return payoutResolved{outcome: ledger.NoPayout{}}
	}
	if scanErr != nil {
		return payoutRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "read collectible reward failed")}
	}
	if rewardState != "held" {
		return payoutResolved{outcome: ledger.NoPayout{}}
	}

	workerResult := core.ParseUserID(rawWorkerID)
	worker, workerMatched := workerResult.(core.UserIDCreated)
	if !workerMatched {
		return payoutRejected{reason: workerResult.(core.UserIDRejected).Reason}
	}
	collectibleResult := core.ParseCollectibleID(rawCollectibleID)
	collectibleID, collectibleMatched := collectibleResult.(core.CollectibleIDCreated)
	if !collectibleMatched {
		return payoutRejected{reason: collectibleResult.(core.CollectibleIDRejected).Reason}
	}

	if _, err := tx.Exec(ctx, "update collectibles set state = 'awarded', owner_user_id = $2, state_recorded_at = now() where id = $1", rawCollectibleID, rawWorkerID); err != nil {
		return payoutRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "award collectible failed")}
	}
	if _, err := tx.Exec(ctx, "update task_collectible_rewards set state = 'released', state_recorded_at = now() where task_id = $1", taskID.String()); err != nil {
		return payoutRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "release collectible reward failed")}
	}

	return payoutResolved{outcome: ledger.CollectiblePayout{WorkerUserID: worker.Value, CollectibleID: collectibleID.Value}}
}

func releasedCollectiblePayout(ctx context.Context, tx pgx.Tx, taskID core.TaskID, rawWorkerID string) payoutResult {
	var rawCollectibleID string
	var rewardState string
	scanErr := tx.QueryRow(ctx, "select collectible_id::text, state from task_collectible_rewards where task_id = $1", taskID.String()).Scan(&rawCollectibleID, &rewardState)
	if errors.Is(scanErr, pgx.ErrNoRows) {
		return payoutResolved{outcome: ledger.NoPayout{}}
	}
	if scanErr != nil {
		return payoutRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "read collectible reward failed")}
	}
	if rewardState != "released" {
		return payoutResolved{outcome: ledger.NoPayout{}}
	}

	workerResult := core.ParseUserID(rawWorkerID)
	worker, workerMatched := workerResult.(core.UserIDCreated)
	if !workerMatched {
		return payoutRejected{reason: workerResult.(core.UserIDRejected).Reason}
	}
	collectibleResult := core.ParseCollectibleID(rawCollectibleID)
	collectibleID, collectibleMatched := collectibleResult.(core.CollectibleIDCreated)
	if !collectibleMatched {
		return payoutRejected{reason: collectibleResult.(core.CollectibleIDRejected).Reason}
	}
	return payoutResolved{outcome: ledger.CollectiblePayout{WorkerUserID: worker.Value, CollectibleID: collectibleID.Value}}
}

func refundHeldCollectibleReward(ctx context.Context, tx pgx.Tx, taskID core.TaskID) (core.DomainError, bool) {
	var rawCollectibleID string
	var rewardState string
	scanErr := tx.QueryRow(ctx, "select collectible_id::text, state from task_collectible_rewards where task_id = $1 for update", taskID.String()).Scan(&rawCollectibleID, &rewardState)
	if errors.Is(scanErr, pgx.ErrNoRows) {
		return core.DomainError{}, false
	}
	if scanErr != nil {
		return core.NewDomainError(core.ErrorCodeInvalidState, "read collectible reward failed"), true
	}
	if rewardState == "refunded" {
		return core.DomainError{}, false
	}
	if rewardState != "held" {
		return core.NewDomainError(core.ErrorCodeInvalidState, "collectible reward is not held"), true
	}
	if _, err := tx.Exec(ctx, "update collectibles set state = 'minted', state_recorded_at = now() where id = $1", rawCollectibleID); err != nil {
		return core.NewDomainError(core.ErrorCodeInvalidState, "return collectible failed"), true
	}
	if _, err := tx.Exec(ctx, "update task_collectible_rewards set state = 'refunded', state_recorded_at = now() where task_id = $1", taskID.String()); err != nil {
		return core.NewDomainError(core.ErrorCodeInvalidState, "update collectible reward failed"), true
	}
	return core.DomainError{}, false
}

// idempotentAccept returns the prior acceptance outcome when a submission is
// already accepted, so accept commands can be retried safely.
func idempotentAccept(ctx context.Context, tx pgx.Tx, command ledger.AcceptStoreCommand, rawWorkerID string) ledger.AcceptResult {
	var escrowState string
	var amount int64
	scanErr := tx.QueryRow(ctx, "select state, amount from task_escrows where task_id = $1", command.TaskID.String()).Scan(&escrowState, &amount)
	if errors.Is(scanErr, pgx.ErrNoRows) {
		collectibleResult := releasedCollectiblePayout(ctx, tx, command.TaskID, rawWorkerID)
		collectible, matched := collectibleResult.(payoutResolved)
		if !matched {
			return ledger.AcceptRejected{Reason: collectibleResult.(payoutRejected).reason}
		}
		return ledger.SubmissionAccepted{TaskID: command.TaskID, SubmissionID: command.SubmissionID, Payout: collectible.outcome, Tip: ledger.NoTip{}}
	}
	if scanErr != nil {
		return ledger.AcceptRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "read task escrow failed")}
	}

	outcome := ledger.PayoutOutcome(ledger.NoPayout{})
	worker, workerMatched := core.ParseUserID(rawWorkerID).(core.UserIDCreated)
	amountAccepted, amountMatched := ledger.NewCreditAmount(amount).(ledger.CreditAmountAccepted)
	if escrowState == "released" && workerMatched && amountMatched {
		outcome = ledger.CreditPayout{WorkerUserID: worker.Value, Amount: amountAccepted.Value}
	}

	collectibleResult := releasedCollectiblePayout(ctx, tx, command.TaskID, rawWorkerID)
	collectible, collectibleMatched := collectibleResult.(payoutResolved)
	if !collectibleMatched {
		return ledger.AcceptRejected{Reason: collectibleResult.(payoutRejected).reason}
	}
	outcome = combinePayouts(outcome, collectible.outcome)
	return ledger.SubmissionAccepted{TaskID: command.TaskID, SubmissionID: command.SubmissionID, Payout: outcome, Tip: ledger.NoTip{}}
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
