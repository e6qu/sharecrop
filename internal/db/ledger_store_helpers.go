package db

import (
	"context"
	"errors"

	"github.com/e6qu/sharecrop/internal/assets"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/ledger"
	"github.com/e6qu/sharecrop/internal/org"
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

func lockUserAccount(ctx context.Context, tx Tx, userID core.UserID) accountLockResult {
	var rawID string
	scanErr := tx.QueryRow(ctx, "select id::text from credit_accounts where user_id = $1 for update", userID.String()).Scan(&rawID)
	if errors.Is(scanErr, ErrNoRows) {
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

// completeFunding allocates credits against a locked, draft, owner-verified
// task: it validates the funder's spendable balance, writes a task_fund ledger
// entry (dropping spendable) and inserts the stateless task_funds row (raising
// allocated). It is shared by user and organization funding so the mechanics
// live in one place.
func completeFunding(ctx context.Context, tx Tx, account accountLocked, taskID core.TaskID, amount ledger.CreditAmount, entryID core.LedgerEntryID, key ledger.IdempotencyKey, insufficientMessage string) ledger.FundResult {
	var fundExists bool
	if err := tx.QueryRow(ctx, "select exists(select 1 from task_funds where task_id = $1)", taskID.String()).Scan(&fundExists); err != nil {
		return ledger.FundRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "check existing task fund failed")}
	}
	if fundExists {
		return ledger.FundRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "task is already funded")}
	}

	var spendable int64
	if err := tx.QueryRow(ctx, "select coalesce(sum(amount), 0) from ledger_entries where account_id = $1", account.id).Scan(&spendable); err != nil {
		return ledger.FundRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "read account balance failed")}
	}
	if spendable < amount.Int64() {
		return ledger.FundRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, insufficientMessage)}
	}

	if _, err := tx.Exec(ctx, "insert into task_funds (task_id, funder_account_id, credit_amount) values ($1, $2, $3)", taskID.String(), account.id, amount.Int64()); err != nil {
		return ledger.FundRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "insert task fund failed")}
	}
	if _, err := tx.Exec(ctx, "insert into ledger_entries (id, account_id, kind, amount, task_id, idempotency_key) values ($1, $2, 'task_fund', $3, $4, $5)", entryID.String(), account.id, -amount.Int64(), taskID.String(), key.String()); err != nil {
		return ledger.FundRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "insert task fund ledger entry failed")}
	}

	if err := tx.Commit(ctx); err != nil {
		return ledger.FundRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "commit fund task transaction failed")}
	}

	return ledger.TaskFunded{Fund: ledger.TaskFund{
		TaskID:          taskID,
		FunderAccountID: account.parsedID,
		CreditAmount:    amount,
	}}
}

func lockOrganizationAccount(ctx context.Context, tx Tx, organizationID core.OrganizationID) accountLockResult {
	var rawID string
	scanErr := tx.QueryRow(ctx, "select id::text from credit_accounts where organization_id = $1 for update", organizationID.String()).Scan(&rawID)
	if errors.Is(scanErr, ErrNoRows) {
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

func lockTaskOwnedByOrganization(ctx context.Context, tx Tx, taskID core.TaskID, organizationID core.OrganizationID) taskLockResult {
	var state string
	var rawOrganizationID string
	var rewardKind string
	var rewardCreditAmount int64
	scanErr := tx.QueryRow(ctx, "select state, coalesce(organization_id::text, ''), reward_kind, coalesce(reward_credit_amount, 0) from tasks where id = $1 for update", taskID.String()).Scan(&state, &rawOrganizationID, &rewardKind, &rewardCreditAmount)
	if errors.Is(scanErr, ErrNoRows) {
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

func lockTaskOwnedBy(ctx context.Context, tx Tx, taskID core.TaskID, requester core.UserID, action string) taskLockResult {
	var state string
	var rawCreatedBy string
	var rewardKind string
	var rewardCreditAmount int64
	scanErr := tx.QueryRow(ctx, "select state, created_by_user_id::text, reward_kind, coalesce(reward_credit_amount, 0) from tasks where id = $1 for update", taskID.String()).Scan(&state, &rawCreatedBy, &rewardKind, &rewardCreditAmount)
	if errors.Is(scanErr, ErrNoRows) {
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
func lockTaskForReview(ctx context.Context, tx Tx, taskID core.TaskID, requester core.UserID, action string) taskLockResult {
	var state string
	var rawCreatedBy string
	var rawOrganizationID string
	var rewardKind string
	var rewardCreditAmount int64
	scanErr := tx.QueryRow(ctx, "select state, created_by_user_id::text, coalesce(organization_id::text, ''), reward_kind, coalesce(reward_credit_amount, 0) from tasks where id = $1 for update", taskID.String()).Scan(&state, &rawCreatedBy, &rawOrganizationID, &rewardKind, &rewardCreditAmount)
	if errors.Is(scanErr, ErrNoRows) {
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
func reviewerOrganizationPermission(ctx context.Context, tx Tx, rawOrganizationID string, requester core.UserID) org.PermissionCheck {
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

// fundingRewardAccepted signals funding may proceed. RewardKindUpdate is
// non-empty when the task's reward_kind must transition as part of this
// funding (none -> credit, collectible -> bundle) - a task is always
// fundable by whoever is authorized to fund it, regardless of the reward
// kind it was created with.
type fundingRewardAccepted struct {
	RewardKindUpdate string
}

type fundingRewardRejected struct {
	reason core.DomainError
}

func (fundingRewardAccepted) fundingRewardResult() {}

func (fundingRewardRejected) fundingRewardResult() {}

func requireCreditRewardFunding(taskRow taskLocked, amount ledger.CreditAmount) fundingRewardResult {
	switch taskRow.rewardKind {
	case "credit", "bundle":
		if taskRow.rewardCreditAmount != amount.Int64() {
			return fundingRewardRejected{reason: core.NewDomainError(core.ErrorCodeConflict, "funding amount must match the declared credit reward")}
		}
		return fundingRewardAccepted{}
	case "collectible":
		return fundingRewardAccepted{RewardKindUpdate: "bundle"}
	default:
		return fundingRewardAccepted{RewardKindUpdate: "credit"}
	}
}

// requireFundableTask checks a locked task is still draft and, if needed,
// persists the reward_kind transition a first-time credit funding requires
// (none -> credit, collectible -> bundle) - shared by personal and
// organization funding so this policy lives in exactly one place.
func requireFundableTask(ctx context.Context, tx Tx, taskID core.TaskID, taskRow taskLocked, amount ledger.CreditAmount) ledger.FundResult {
	if taskRow.state != "draft" {
		return ledger.FundRejected{Reason: core.NewDomainError(core.ErrorCodeConflict, "only draft tasks can be funded")}
	}
	rewardResult := requireCreditRewardFunding(taskRow, amount)
	if rejected, matched := rewardResult.(fundingRewardRejected); matched {
		return ledger.FundRejected{Reason: rejected.reason}
	}
	rewardKindUpdate := rewardResult.(fundingRewardAccepted).RewardKindUpdate
	if rewardKindUpdate == "" {
		return nil
	}
	if _, err := tx.Exec(ctx, "update tasks set reward_kind = $1, reward_credit_amount = $2 where id = $3", rewardKindUpdate, amount.Int64(), taskID.String()); err != nil {
		return ledger.FundRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "update task reward kind failed")}
	}
	return nil
}

// findFundForKey returns a non-nil FundResult when a fund command with the
// given idempotency key has already been recorded for the task. The reply is
// reconstructed from the durable task_fund ledger entry, so it replays even
// after the task_funds row has been consumed by an award or refund.
func findFundForKey(ctx context.Context, tx Tx, key string, taskID core.TaskID) ledger.FundResult {
	var kind string
	var rawFunderAccountID string
	var amount int64
	var rawTaskID string
	scanErr := tx.QueryRow(ctx, "select kind, account_id::text, amount, coalesce(task_id::text, '') from ledger_entries where idempotency_key = $1", key).Scan(&kind, &rawFunderAccountID, &amount, &rawTaskID)
	if errors.Is(scanErr, ErrNoRows) {
		return nil
	}
	if scanErr != nil {
		return ledger.FundRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "check fund idempotency failed")}
	}
	if kind != ledger.EntryKindTaskFund.String() || rawTaskID != taskID.String() {
		return ledger.FundRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "idempotency key was used for a different command")}
	}
	fundResult := buildFund(taskID, rawFunderAccountID, -amount)
	built, builtMatched := fundResult.(fundBuilt)
	if !builtMatched {
		return ledger.FundRejected{Reason: fundResult.(fundBuildRejected).reason}
	}
	return ledger.TaskFunded{Fund: built.value}
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

func payOutEscrow(ctx context.Context, tx Tx, command ledger.AcceptStoreCommand, rawWorkerID string) payoutResult {
	return payReviewFund(ctx, tx, reviewFundCommand{
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

type reviewFundCommand struct {
	taskID            core.TaskID
	rawWorkerID       string
	payoutEntryID     core.LedgerEntryID
	refundEntryID     core.LedgerEntryID
	idempotencyKey    ledger.IdempotencyKey
	selection         ledger.CreditReviewSelection
	closeTask         bool
	missingIsNoPayout bool
}

// payReviewFund pays a task's allocated credits (from task_funds) to the worker
// when a submission is accepted (closeTask) or rejected. On accept it pays the
// selected amount, refunds the remainder to the funder, and deletes the
// task_funds row (credits have changed hands). On reject it pays the selected
// amount and keeps the remainder allocated (updating task_funds) so a later
// submission can still be accepted against it; a fully-paid reject deletes the
// row.
func payReviewFund(ctx context.Context, tx Tx, command reviewFundCommand) payoutResult {
	if _, noPayout := command.selection.(ledger.NoCreditReviewSelection); noPayout {
		return payoutResolved{outcome: ledger.NoPayout{}}
	}

	var rawFunderAccountID string
	var amount int64
	scanErr := tx.QueryRow(ctx, "select funder_account_id::text, credit_amount from task_funds where task_id = $1 for update", command.taskID.String()).Scan(&rawFunderAccountID, &amount)
	if errors.Is(scanErr, ErrNoRows) {
		if _, partial := command.selection.(ledger.PartialCreditReviewSelection); partial {
			return payoutRejected{reason: core.NewDomainError(core.ErrorCodeConflict, "credit reward fund is missing")}
		}
		if command.missingIsNoPayout {
			return payoutResolved{outcome: ledger.NoPayout{}}
		}
		return payoutRejected{reason: core.NewDomainError(core.ErrorCodeConflict, "credit reward fund is missing")}
	}
	if scanErr != nil {
		return payoutRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "read task fund failed")}
	}

	payoutAmount := amount
	if partial, matched := command.selection.(ledger.PartialCreditReviewSelection); matched {
		payoutAmount = partial.Amount.Int64()
	}
	if payoutAmount > amount {
		return payoutRejected{reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "credit payout cannot exceed allocated credits")}
	}

	workerResult := core.ParseUserID(command.rawWorkerID)
	worker, workerMatched := workerResult.(core.UserIDCreated)
	if !workerMatched {
		return payoutRejected{reason: workerResult.(core.UserIDRejected).Reason}
	}

	var workerAccountID string
	accountErr := tx.QueryRow(ctx, "select id::text from credit_accounts where user_id = $1 for update", command.rawWorkerID).Scan(&workerAccountID)
	if errors.Is(accountErr, ErrNoRows) {
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
		if _, err := tx.Exec(ctx, "delete from task_funds where task_id = $1", command.taskID.String()); err != nil {
			return payoutRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "clear task fund failed")}
		}
	} else if remaining == 0 {
		if _, err := tx.Exec(ctx, "delete from task_funds where task_id = $1", command.taskID.String()); err != nil {
			return payoutRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "clear task fund failed")}
		}
	} else {
		if _, err := tx.Exec(ctx, "update task_funds set credit_amount = $2 where task_id = $1", command.taskID.String(), remaining); err != nil {
			return payoutRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "update task fund after partial payout failed")}
		}
	}

	amountResult := ledger.NewCreditAmount(payoutAmount)
	amountAccepted, amountMatched := amountResult.(ledger.CreditAmountAccepted)
	if !amountMatched {
		return payoutRejected{reason: amountResult.(ledger.CreditAmountRejected).Reason}
	}
	return payoutResolved{outcome: ledger.CreditPayout{WorkerUserID: worker.Value, Amount: amountAccepted.Value}}
}

func payCreditTip(ctx context.Context, tx Tx, taskID core.TaskID, requester core.UserID, rawWorkerID string, debitEntryID core.LedgerEntryID, creditEntryID core.LedgerEntryID, idempotencyKey ledger.IdempotencyKey, selection ledger.TipSelection) tipResult {
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
	if errors.Is(accountErr, ErrNoRows) {
		return tipRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "submission author has no credit account")}
	}
	if accountErr != nil {
		return tipRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "lock worker credit account failed")}
	}

	// Derive per-side idempotency keys (distinct from the bare payout key) so the
	// unique constraint would catch a repeated double-tip if the task-lock ordering
	// ever changed; today the FOR UPDATE task lock already serializes this.
	debitKey := idempotencyKey.String() + ":tip-debit"
	creditKey := idempotencyKey.String() + ":tip-credit"
	if _, err := tx.Exec(ctx, "insert into ledger_entries (id, account_id, kind, amount, task_id, idempotency_key) values ($1, $2, 'task_tip', $3, $4, $5)", debitEntryID.String(), requesterLocked.id, -tip.Amount.Int64(), taskID.String(), debitKey); err != nil {
		return tipRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "insert task tip debit failed")}
	}
	if _, err := tx.Exec(ctx, "insert into ledger_entries (id, account_id, kind, amount, task_id, idempotency_key) values ($1, $2, 'task_tip', $3, $4, $5)", creditEntryID.String(), workerAccountID, tip.Amount.Int64(), taskID.String(), creditKey); err != nil {
		return tipRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "insert task tip credit failed")}
	}

	return tipResolved{outcome: ledger.CreditTip{WorkerUserID: worker.Value, Amount: tip.Amount}}
}

func payCollectibleTip(ctx context.Context, tx Tx, requester core.UserID, rawWorkerID string, selection ledger.CollectibleTipSelection) tipResult {
	selected, matched := selection.(ledger.CollectibleTipSelected)
	if !matched {
		return tipResolved{outcome: ledger.NoTip{}}
	}

	workerResult := core.ParseUserID(rawWorkerID)
	worker, workerMatched := workerResult.(core.UserIDCreated)
	if !workerMatched {
		return tipRejected{reason: workerResult.(core.UserIDRejected).Reason}
	}

	collectibleResult := lockCollectible(ctx, tx, selected.ID)
	collectible, collectibleMatched := collectibleResult.(collectibleParsed)
	if !collectibleMatched {
		return tipRejected{reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "collectible is not available to tip")}
	}
	if collectible.value.OwnerKind == assets.CollectibleOwnerKindUser && collectible.value.OwnerID == worker.Value.String() {
		return tipResolved{outcome: ledger.CollectibleTip{WorkerUserID: worker.Value, CollectibleID: selected.ID}}
	}
	if collectible.value.OwnerKind != assets.CollectibleOwnerKindUser || collectible.value.OwnerID != requester.String() {
		return tipRejected{reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "collectible is not available to tip")}
	}
	if collectible.value.State != assets.CollectibleStateMinted {
		return tipRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "collectible is not available to tip")}
	}
	if denied, matched := assets.AllowsTip(collectible.value.Policy).(assets.RewardDenied); matched {
		return tipRejected{reason: denied.Reason}
	}
	if collectible.value.Policy == assets.TransferPolicyTransferableWithinOrg {
		if reason, rejected := requireSharedCollectibleOrganization(ctx, tx, collectible.value, requester, worker.Value); rejected {
			return tipRejected{reason: reason}
		}
	}

	if _, err := tx.Exec(ctx, "update collectibles set owner_user_id = $2, owner_kind = 'user', state_recorded_at = now() where id = $1", selected.ID.String(), worker.Value.String()); err != nil {
		return tipRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "transfer collectible tip failed")}
	}
	return tipResolved{outcome: ledger.CollectibleTip{WorkerUserID: worker.Value, CollectibleID: selected.ID}}
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

// payOutCollectible transfers every currently-held collectible reward for the
// task to the accepted worker (escrowed -> minted, owner = worker) and deletes
// the task_fund_collectibles rows. A task may bundle more than one collectible,
// so all held rewards are awarded together.
func payOutCollectible(ctx context.Context, tx Tx, taskID core.TaskID, rawWorkerID string) payoutResult {
	rawCollectibleIDs, scanRejected := heldFundCollectibleIDs(ctx, tx, taskID, true)
	if scanRejected != nil {
		return payoutRejected{reason: *scanRejected}
	}
	resolved, resolveRejected := resolveCollectiblePayout(rawWorkerID, rawCollectibleIDs)
	if resolveRejected != nil {
		return payoutRejected{reason: *resolveRejected}
	}
	if _, empty := resolved.outcome.(ledger.NoPayout); empty {
		return payoutResolved{outcome: ledger.NoPayout{}}
	}

	for _, rawCollectibleID := range rawCollectibleIDs {
		if _, err := tx.Exec(ctx, "update collectibles set state = 'minted', owner_user_id = $2, owner_kind = 'user', state_recorded_at = now() where id = $1", rawCollectibleID, rawWorkerID); err != nil {
			return payoutRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "award collectible failed")}
		}
	}
	if _, err := tx.Exec(ctx, "delete from task_fund_collectibles where task_id = $1", taskID.String()); err != nil {
		return payoutRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "clear collectible reward failed")}
	}

	return resolved
}

// resolveCollectiblePayout parses the worker and the held collectible IDs into a
// collectible payout outcome. It returns a NoPayout outcome when there are no
// collectible rewards, so both the award and refund paths share one parse.
func resolveCollectiblePayout(rawWorkerID string, rawCollectibleIDs []string) (payoutResolved, *core.DomainError) {
	if len(rawCollectibleIDs) == 0 {
		return payoutResolved{outcome: ledger.NoPayout{}}, nil
	}

	workerResult := core.ParseUserID(rawWorkerID)
	worker, workerMatched := workerResult.(core.UserIDCreated)
	if !workerMatched {
		reason := workerResult.(core.UserIDRejected).Reason
		return payoutResolved{}, &reason
	}

	collectibleIDs := make([]core.CollectibleID, 0, len(rawCollectibleIDs))
	for _, rawCollectibleID := range rawCollectibleIDs {
		collectibleResult := core.ParseCollectibleID(rawCollectibleID)
		collectibleID, collectibleMatched := collectibleResult.(core.CollectibleIDCreated)
		if !collectibleMatched {
			reason := collectibleResult.(core.CollectibleIDRejected).Reason
			return payoutResolved{}, &reason
		}
		collectibleIDs = append(collectibleIDs, collectibleID.Value)
	}
	return payoutResolved{outcome: ledger.CollectiblePayout{WorkerUserID: worker.Value, CollectibleIDs: collectibleIDs}}, nil
}

// heldFundCollectibleIDs returns the raw collectible IDs currently held for the
// task's reward (task_fund_collectibles), optionally taking a row lock.
func heldFundCollectibleIDs(ctx context.Context, tx Tx, taskID core.TaskID, lock bool) ([]string, *core.DomainError) {
	query := "select collectible_id::text from task_fund_collectibles where task_id = $1 order by collectible_id"
	if lock {
		query += " for update"
	}
	rows, err := tx.Query(ctx, query, taskID.String())
	if err != nil {
		reason := core.NewDomainError(core.ErrorCodeInvalidState, "read collectible reward failed")
		return nil, &reason
	}
	defer rows.Close()
	ids := make([]string, 0)
	for rows.Next() {
		var rawCollectibleID string
		if err := rows.Scan(&rawCollectibleID); err != nil {
			reason := core.NewDomainError(core.ErrorCodeInvalidState, "scan collectible reward failed")
			return nil, &reason
		}
		ids = append(ids, rawCollectibleID)
	}
	if err := rows.Err(); err != nil {
		reason := core.NewDomainError(core.ErrorCodeInvalidState, "read collectible reward failed")
		return nil, &reason
	}
	return ids, nil
}

// refundHeldCollectibleReward returns every held collectible reward on the task
// to its funder (escrowed -> minted; the escrowed collectible still records its
// funder as owner) and deletes the task_fund_collectibles rows. A task may
// bundle more than one collectible, so all held rewards are returned together.
func refundHeldCollectibleReward(ctx context.Context, tx Tx, taskID core.TaskID) (core.DomainError, bool) {
	rawCollectibleIDs, scanRejected := heldFundCollectibleIDs(ctx, tx, taskID, true)
	if scanRejected != nil {
		return *scanRejected, true
	}
	if len(rawCollectibleIDs) == 0 {
		return core.DomainError{}, false
	}
	for _, rawCollectibleID := range rawCollectibleIDs {
		if _, err := tx.Exec(ctx, "update collectibles set state = 'minted', state_recorded_at = now() where id = $1", rawCollectibleID); err != nil {
			return core.NewDomainError(core.ErrorCodeInvalidState, "return collectible failed"), true
		}
	}
	if _, err := tx.Exec(ctx, "delete from task_fund_collectibles where task_id = $1", taskID.String()); err != nil {
		return core.NewDomainError(core.ErrorCodeInvalidState, "clear collectible reward failed"), true
	}
	return core.DomainError{}, false
}

// idempotentAccept returns the prior acceptance outcome when a submission is
// already accepted, so accept commands can be retried safely. The stateless
// task_funds/task_fund_collectibles rows were consumed by the first accept, so
// the credit payout is reconstructed from the durable task_payout ledger entry
// keyed on the accept's idempotency key.
func idempotentAccept(ctx context.Context, tx Tx, command ledger.AcceptStoreCommand, rawWorkerID string) ledger.AcceptResult {
	outcome := ledger.PayoutOutcome(ledger.NoPayout{})
	var amount int64
	scanErr := tx.QueryRow(ctx, "select amount from ledger_entries where task_id = $1 and kind = 'task_payout' and idempotency_key = $2", command.TaskID.String(), command.IdempotencyKey.String()).Scan(&amount)
	if scanErr != nil && !errors.Is(scanErr, ErrNoRows) {
		return ledger.AcceptRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "read task payout failed")}
	}
	if scanErr == nil {
		worker, workerMatched := core.ParseUserID(rawWorkerID).(core.UserIDCreated)
		amountAccepted, amountMatched := ledger.NewCreditAmount(amount).(ledger.CreditAmountAccepted)
		if workerMatched && amountMatched {
			outcome = ledger.CreditPayout{WorkerUserID: worker.Value, Amount: amountAccepted.Value}
		}
	}
	return ledger.SubmissionAccepted{TaskID: command.TaskID, SubmissionID: command.SubmissionID, Payout: outcome, Tip: ledger.NoTip{}}
}

type fundBuildResult interface {
	fundBuildResult()
}

type fundBuilt struct {
	value ledger.TaskFund
}

type fundBuildRejected struct {
	reason core.DomainError
}

func (fundBuilt) fundBuildResult() {}

func (fundBuildRejected) fundBuildResult() {}

func buildFund(taskID core.TaskID, rawFunderAccountID string, amount int64) fundBuildResult {
	accountResult := core.ParseCreditAccountID(rawFunderAccountID)
	account, matched := accountResult.(core.CreditAccountIDCreated)
	if !matched {
		return fundBuildRejected{reason: accountResult.(core.CreditAccountIDRejected).Reason}
	}
	amountResult := ledger.NewCreditAmount(amount)
	amountAccepted, amountMatched := amountResult.(ledger.CreditAmountAccepted)
	if !amountMatched {
		return fundBuildRejected{reason: amountResult.(ledger.CreditAmountRejected).Reason}
	}
	return fundBuilt{value: ledger.TaskFund{
		TaskID:          taskID,
		FunderAccountID: account.Value,
		CreditAmount:    amountAccepted.Value,
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
