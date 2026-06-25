package db

import (
	"context"

	"github.com/e6qu/sharecrop/internal/assets"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CollectibleStore struct {
	pool *pgxpool.Pool
}

func NewCollectibleStore(pool *pgxpool.Pool) CollectibleStore {
	return CollectibleStore{pool: pool}
}

func (store CollectibleStore) CreateCollectible(ctx context.Context, collectible assets.Collectible) assets.CreateStoreResult {
	_, err := store.pool.Exec(ctx, `
		insert into collectibles (id, name, kind, state, transfer_policy, owner_user_id, owner_kind, art)
		values ($1, $2, $3, $4, $5, $6, $7, $8)
	`, collectible.ID.String(), collectible.Name.String(), collectible.Kind.String(), collectible.State.String(), collectible.Policy.String(), collectible.OwnerID, collectible.OwnerKind, collectible.Art)
	if err != nil {
		return assets.CreateStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "insert collectible failed")}
	}
	return assets.CreateStoreAccepted{}
}

func (store CollectibleStore) ListCollectibles(ctx context.Context, owner core.UserID, page core.Page) assets.ListStoreResult {
	rows, err := store.pool.Query(ctx, `
		select id::text, name, kind, state, transfer_policy, owner_user_id::text, owner_kind, art
		from collectibles
		where owner_user_id = $1 and owner_kind = 'user'
		order by created_at desc, id
		limit $2 offset $3
	`, owner.String(), page.Limit(), page.Offset())
	if err != nil {
		return assets.ListStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "list collectibles failed")}
	}
	defer rows.Close()
	return collectListedCollectibles(rows)
}

func (store CollectibleStore) ListCollectiblesByOwner(ctx context.Context, ownerKind string, ownerID string, page core.Page) assets.ListStoreResult {
	rows, err := store.pool.Query(ctx, `
		select id::text, name, kind, state, transfer_policy, owner_user_id::text, owner_kind, art
		from collectibles
		where owner_user_id = $1 and owner_kind = $2
		order by created_at desc, id
		limit $3 offset $4
	`, ownerID, ownerKind, page.Limit(), page.Offset())
	if err != nil {
		return assets.ListStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "list collectibles failed")}
	}
	defer rows.Close()
	return collectListedCollectibles(rows)
}

// collectListedCollectibles scans every row of a collectibles query into the
// listed-store result, rejecting on the first unparsable row.
func collectListedCollectibles(rows pgx.Rows) assets.ListStoreResult {
	values := make([]assets.Collectible, 0)
	for rows.Next() {
		parsed := scanCollectible(rows)
		accepted, matched := parsed.(collectibleParsed)
		if !matched {
			return assets.ListStoreRejected{Reason: parsed.(collectibleParseRejected).reason}
		}
		values = append(values, accepted.value)
	}
	if err := rows.Err(); err != nil {
		return assets.ListStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "read collectibles failed")}
	}
	return assets.ListStoreListed{Values: values}
}

func (store CollectibleStore) FundCollectibleReward(ctx context.Context, command assets.FundRewardStoreCommand) assets.FundRewardResult {
	tx, err := store.pool.Begin(ctx)
	if err != nil {
		return assets.FundRewardRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "begin fund collectible reward transaction failed")}
	}
	defer func() { _ = tx.Rollback(ctx) }()

	collectibleResult := lockCollectible(ctx, tx, command.CollectibleID)
	collectible, collectibleMatched := collectibleResult.(collectibleParsed)
	if !collectibleMatched {
		return assets.FundRewardRejected{Reason: collectibleResult.(collectibleParseRejected).reason}
	}
	if collectible.value.OwnerKind != assets.CollectibleOwnerKindUser || collectible.value.OwnerID != command.FunderUserID.String() {
		return assets.FundRewardRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "only the collectible owner can fund a task with it")}
	}
	if collectible.value.State != assets.CollectibleStateMinted {
		return assets.FundRewardRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "collectible is not available to escrow")}
	}
	if denied, matched := assets.AllowsRewardPayout(collectible.value.Policy).(assets.RewardDenied); matched {
		return assets.FundRewardRejected{Reason: denied.Reason}
	}

	taskResult := lockTaskOwnedBy(ctx, tx, command.TaskID, command.FunderUserID, "fund")
	taskRow, taskMatched := taskResult.(taskLocked)
	if !taskMatched {
		return assets.FundRewardRejected{Reason: taskResult.(taskLockRejected).reason}
	}
	if taskRow.state != "draft" {
		return assets.FundRewardRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "only draft tasks can be funded")}
	}

	rewardIDResult := core.NewTaskCollectibleRewardID()
	rewardID, rewardIDMatched := rewardIDResult.(core.TaskCollectibleRewardIDCreated)
	if !rewardIDMatched {
		return assets.FundRewardRejected{Reason: rewardIDResult.(core.TaskCollectibleRewardIDRejected).Reason}
	}

	if _, err := tx.Exec(ctx, "update collectibles set state = 'escrowed', state_recorded_at = now() where id = $1", command.CollectibleID.String()); err != nil {
		return assets.FundRewardRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "escrow collectible failed")}
	}
	if _, err := tx.Exec(ctx, `
		insert into task_collectible_rewards (id, task_id, collectible_id, funder_user_id, state)
		values ($1, $2, $3, $4, 'held')
	`, rewardID.Value.String(), command.TaskID.String(), command.CollectibleID.String(), command.FunderUserID.String()); err != nil {
		if isUniqueViolation(err) {
			return assets.FundRewardRejected{Reason: core.NewDomainError(core.ErrorCodeConflict, "collectible is already escrowed on this task")}
		}
		return assets.FundRewardRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "insert collectible reward failed")}
	}

	if err := tx.Commit(ctx); err != nil {
		return assets.FundRewardRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "commit fund collectible reward transaction failed")}
	}

	awarded := collectible.value
	awarded.State = assets.CollectibleStateEscrowed
	return assets.RewardFunded{Value: awarded}
}

func (store CollectibleStore) RefundCollectibleReward(ctx context.Context, command assets.RefundRewardStoreCommand) assets.RefundRewardResult {
	tx, err := store.pool.Begin(ctx)
	if err != nil {
		return assets.RefundRewardRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "begin refund collectible reward transaction failed")}
	}
	defer func() { _ = tx.Rollback(ctx) }()

	taskResult := lockTaskOwnedBy(ctx, tx, command.TaskID, command.RequesterUserID, "refund")
	taskRow, taskMatched := taskResult.(taskLocked)
	if !taskMatched {
		return assets.RefundRewardRejected{Reason: taskResult.(taskLockRejected).reason}
	}
	if taskRow.state != "draft" && taskRow.state != "open" {
		return assets.RefundRewardRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "only draft or open tasks can be refunded")}
	}
	if taskRow.rewardKind == "bundle" {
		return assets.RefundRewardRejected{Reason: core.NewDomainError(core.ErrorCodeConflict, "bundled rewards must be refunded together")}
	}

	heldIDs, scanRejected := heldCollectibleIDs(ctx, tx, command.TaskID)
	if scanRejected != nil {
		return assets.RefundRewardRejected{Reason: *scanRejected}
	}
	if len(heldIDs) == 0 {
		return assets.RefundRewardRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "task has no collectible reward to refund")}
	}

	refunded := make([]assets.Collectible, 0, len(heldIDs))
	for _, rawCollectibleID := range heldIDs {
		if _, err := tx.Exec(ctx, "update collectibles set state = 'minted', state_recorded_at = now() where id = $1", rawCollectibleID); err != nil {
			return assets.RefundRewardRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "return collectible failed")}
		}
		collectibleResult := findCollectible(ctx, tx, rawCollectibleID)
		collectible, matched := collectibleResult.(collectibleParsed)
		if !matched {
			return assets.RefundRewardRejected{Reason: collectibleResult.(collectibleParseRejected).reason}
		}
		refunded = append(refunded, collectible.value)
	}

	if _, err := tx.Exec(ctx, "update task_collectible_rewards set state = 'refunded', state_recorded_at = now() where task_id = $1 and state = 'held'", command.TaskID.String()); err != nil {
		return assets.RefundRewardRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "update collectible reward failed")}
	}
	if _, err := tx.Exec(ctx, "update tasks set state = 'cancelled', state_recorded_at = now() where id = $1", command.TaskID.String()); err != nil {
		return assets.RefundRewardRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "cancel task failed")}
	}

	if err := tx.Commit(ctx); err != nil {
		return assets.RefundRewardRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "commit refund collectible reward transaction failed")}
	}
	return assets.RewardRefunded{Values: refunded}
}

// heldCollectibleIDs returns the raw collectible IDs of every held reward on the
// task, locking those reward rows. It rejects when a non-held, non-refunded
// reward row exists so partially-released tasks cannot be silently refunded.
func heldCollectibleIDs(ctx context.Context, tx pgx.Tx, taskID core.TaskID) ([]string, *core.DomainError) {
	rows, err := tx.Query(ctx, "select collectible_id::text, state from task_collectible_rewards where task_id = $1 order by created_at, id for update", taskID.String())
	if err != nil {
		reason := core.NewDomainError(core.ErrorCodeInvalidState, "read collectible reward failed")
		return nil, &reason
	}
	defer rows.Close()

	ids := make([]string, 0)
	for rows.Next() {
		var rawCollectibleID string
		var rewardState string
		if err := rows.Scan(&rawCollectibleID, &rewardState); err != nil {
			reason := core.NewDomainError(core.ErrorCodeInvalidState, "scan collectible reward failed")
			return nil, &reason
		}
		if rewardState == "refunded" {
			continue
		}
		if rewardState != "held" {
			reason := core.NewDomainError(core.ErrorCodeInvalidState, "collectible reward is not held")
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

type collectibleParseResult interface {
	collectibleParseResult()
}

type collectibleParsed struct {
	value assets.Collectible
}

type collectibleParseRejected struct {
	reason core.DomainError
}

func (collectibleParsed) collectibleParseResult() {}

func (collectibleParseRejected) collectibleParseResult() {}

func (store CollectibleStore) GiftCollectible(ctx context.Context, command assets.GiftStoreCommand) assets.GiftResult {
	tx, err := store.pool.Begin(ctx)
	if err != nil {
		return assets.GiftRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "begin gift collectible transaction failed")}
	}
	defer func() { _ = tx.Rollback(ctx) }()

	collectibleResult := lockCollectible(ctx, tx, command.CollectibleID)
	collectible, collectibleMatched := collectibleResult.(collectibleParsed)
	if !collectibleMatched {
		// Use a uniform message so a known-but-unowned id is not distinguishable
		// from a non-existent one (a minor existence oracle on the tip path).
		return assets.GiftRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "collectible is not available to tip")}
	}
	// Idempotent replay: if the collectible already belongs to the recipient, a
	// retried accept (after a lost response) treats the gift as already done.
	if collectible.value.OwnerKind == assets.CollectibleOwnerKindUser && collectible.value.OwnerID == command.ToUserID.String() {
		return assets.CollectibleGifted{Value: collectible.value}
	}
	if collectible.value.OwnerKind != assets.CollectibleOwnerKindUser || collectible.value.OwnerID != command.FromUserID.String() {
		return assets.GiftRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "collectible is not available to tip")}
	}
	if collectible.value.State != assets.CollectibleStateMinted {
		return assets.GiftRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "collectible is not available to tip")}
	}
	if denied, matched := assets.AllowsTip(collectible.value.Policy).(assets.RewardDenied); matched {
		return assets.GiftRejected{Reason: denied.Reason}
	}

	if _, err := tx.Exec(ctx, "update collectibles set owner_user_id = $2, owner_kind = 'user', state_recorded_at = now() where id = $1", command.CollectibleID.String(), command.ToUserID.String()); err != nil {
		return assets.GiftRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "transfer collectible failed")}
	}
	if err := tx.Commit(ctx); err != nil {
		return assets.GiftRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "commit gift collectible failed")}
	}

	gifted := collectible.value
	gifted.OwnerKind = assets.CollectibleOwnerKindUser
	gifted.OwnerID = command.ToUserID.String()
	return assets.CollectibleGifted{Value: gifted}
}

func lockCollectible(ctx context.Context, tx pgx.Tx, collectibleID core.CollectibleID) collectibleParseResult {
	rows, err := tx.Query(ctx, "select id::text, name, kind, state, transfer_policy, owner_user_id::text, owner_kind, art from collectibles where id = $1 for update", collectibleID.String())
	if err != nil {
		return collectibleParseRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "lock collectible failed")}
	}
	defer rows.Close()
	if !rows.Next() {
		return collectibleParseRejected{reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "collectible was not found")}
	}
	return scanCollectible(rows)
}

func findCollectible(ctx context.Context, tx pgx.Tx, rawCollectibleID string) collectibleParseResult {
	rows, err := tx.Query(ctx, "select id::text, name, kind, state, transfer_policy, owner_user_id::text, owner_kind, art from collectibles where id = $1", rawCollectibleID)
	if err != nil {
		return collectibleParseRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "read collectible failed")}
	}
	defer rows.Close()
	if !rows.Next() {
		return collectibleParseRejected{reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "collectible was not found")}
	}
	return scanCollectible(rows)
}

func scanCollectible(rows pgx.Rows) collectibleParseResult {
	var rawID string
	var rawName string
	var rawKind string
	var rawState string
	var rawPolicy string
	var rawOwner string
	var rawOwnerKind string
	var rawArt string
	if err := rows.Scan(&rawID, &rawName, &rawKind, &rawState, &rawPolicy, &rawOwner, &rawOwnerKind, &rawArt); err != nil {
		return collectibleParseRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "scan collectible failed")}
	}
	return parseCollectible(rawID, rawName, rawKind, rawState, rawPolicy, rawOwner, rawOwnerKind, rawArt)
}

func parseCollectible(rawID string, rawName string, rawKind string, rawState string, rawPolicy string, rawOwner string, rawOwnerKind string, rawArt string) collectibleParseResult {
	idResult := core.ParseCollectibleID(rawID)
	collectibleID, idMatched := idResult.(core.CollectibleIDCreated)
	if !idMatched {
		return collectibleParseRejected{reason: idResult.(core.CollectibleIDRejected).Reason}
	}
	nameResult := assets.NewCollectibleName(rawName)
	name, nameMatched := nameResult.(assets.CollectibleNameAccepted)
	if !nameMatched {
		return collectibleParseRejected{reason: nameResult.(assets.CollectibleNameRejected).Reason}
	}
	kindResult := assets.ParseCollectibleKind(rawKind)
	kind, kindMatched := kindResult.(assets.CollectibleKindAccepted)
	if !kindMatched {
		return collectibleParseRejected{reason: kindResult.(assets.CollectibleKindRejected).Reason}
	}
	stateResult := assets.ParseCollectibleState(rawState)
	state, stateMatched := stateResult.(assets.CollectibleStateAccepted)
	if !stateMatched {
		return collectibleParseRejected{reason: stateResult.(assets.CollectibleStateRejected).Reason}
	}
	policyResult := assets.ParseTransferPolicy(rawPolicy)
	policy, policyMatched := policyResult.(assets.TransferPolicyAccepted)
	if !policyMatched {
		return collectibleParseRejected{reason: policyResult.(assets.TransferPolicyRejected).Reason}
	}
	if !assets.ValidCollectibleOwnerKind(rawOwnerKind) {
		return collectibleParseRejected{reason: core.NewDomainError(core.ErrorCodeInvalidEnum, "collectible owner kind is invalid")}
	}
	return collectibleParsed{value: assets.Collectible{
		ID:        collectibleID.Value,
		Name:      name.Value,
		Kind:      kind.Value,
		State:     state.Value,
		Policy:    policy.Value,
		OwnerKind: rawOwnerKind,
		OwnerID:   rawOwner,
		Art:       rawArt,
	}}
}
