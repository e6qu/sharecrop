package db

import (
	"context"

	"github.com/e6qu/sharecrop/internal/assets"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/org"
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
		insert into collectibles (id, name, kind, state, transfer_policy, owner_user_id, owner_kind, organization_id, art)
		values ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, collectible.ID.String(), collectible.Name.String(), collectible.Kind.String(), collectible.State.String(), collectible.Policy.String(), collectible.OwnerID, collectible.OwnerKind, nullableText(collectible.OrganizationID), collectible.Art)
	if err != nil {
		return assets.CreateStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "insert collectible failed")}
	}
	return assets.CreateStoreAccepted{}
}

func (store CollectibleStore) ListCollectibles(ctx context.Context, owner core.UserID, page core.Page) assets.ListStoreResult {
	rows, err := store.pool.Query(ctx, `
		select id::text, name, kind, state, transfer_policy, owner_user_id::text, owner_kind, coalesce(organization_id::text, ''), art
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
		select id::text, name, kind, state, transfer_policy, owner_user_id::text, owner_kind, coalesce(organization_id::text, ''), art
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

	// A draft task is always fundable with a collectible by its creator,
	// regardless of the reward kind it was created with: none -> collectible,
	// credit -> bundle. A task already declaring a collectible component
	// (collectible/bundle) keeps its reward kind as-is.
	var rewardKindUpdate string
	switch taskRow.rewardKind {
	case "none":
		rewardKindUpdate = "collectible"
	case "credit":
		rewardKindUpdate = "bundle"
	}
	if rewardKindUpdate != "" {
		if _, err := tx.Exec(ctx, "update tasks set reward_kind = $1 where id = $2", rewardKindUpdate, command.TaskID.String()); err != nil {
			return assets.FundRewardRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "update task reward kind failed")}
		}
	}

	if _, err := tx.Exec(ctx, "update collectibles set state = 'escrowed', state_recorded_at = now() where id = $1", command.CollectibleID.String()); err != nil {
		return assets.FundRewardRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "escrow collectible failed")}
	}
	if _, err := tx.Exec(ctx, `
		insert into task_fund_collectibles (task_id, collectible_id)
		values ($1, $2)
	`, command.TaskID.String(), command.CollectibleID.String()); err != nil {
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

	if reason := lockTaskRefundable(ctx, tx, command.TaskID, command.RequesterUserID); reason != nil {
		return assets.RefundRewardRejected{Reason: *reason}
	}
	var rewardKind string
	if err := tx.QueryRow(ctx, "select reward_kind from tasks where id = $1", command.TaskID.String()).Scan(&rewardKind); err != nil {
		return assets.RefundRewardRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "read task reward kind failed")}
	}
	if rewardKind == "bundle" {
		return assets.RefundRewardRejected{Reason: core.NewDomainError(core.ErrorCodeConflict, "bundled rewards must be refunded together")}
	}

	heldIDs, scanRejected := heldFundCollectibleIDs(ctx, tx, command.TaskID, true)
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

	if _, err := tx.Exec(ctx, "delete from task_fund_collectibles where task_id = $1", command.TaskID.String()); err != nil {
		return assets.RefundRewardRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "clear collectible reward failed")}
	}
	if _, err := tx.Exec(ctx, "update tasks set state = 'cancelled', state_recorded_at = now() where id = $1", command.TaskID.String()); err != nil {
		return assets.RefundRewardRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "cancel task failed")}
	}

	if err := tx.Commit(ctx); err != nil {
		return assets.RefundRewardRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "commit refund collectible reward transaction failed")}
	}
	return assets.RewardRefunded{Values: refunded}
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
	if collectible.value.Policy == assets.TransferPolicyTransferableWithinOrg {
		if reason, rejected := requireSharedCollectibleOrganization(ctx, tx, collectible.value, command.FromUserID, command.ToUserID); rejected {
			return assets.GiftRejected{Reason: reason}
		}
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

func (store CollectibleStore) AwardOrganizationCollectible(ctx context.Context, command assets.AwardOrganizationCollectibleStoreCommand) assets.GiftResult {
	tx, err := store.pool.Begin(ctx)
	if err != nil {
		return assets.GiftRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "begin award organization collectible transaction failed")}
	}
	defer func() { _ = tx.Rollback(ctx) }()

	collectibleResult := lockCollectible(ctx, tx, command.CollectibleID)
	collectible, collectibleMatched := collectibleResult.(collectibleParsed)
	if !collectibleMatched {
		return assets.GiftRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "collectible is not available to award")}
	}
	if collectible.value.OwnerKind != assets.CollectibleOwnerKindOrganization || collectible.value.OrganizationID != command.OrganizationID.String() {
		return assets.GiftRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "collectible does not belong to this organization")}
	}
	if collectible.value.State != assets.CollectibleStateMinted {
		return assets.GiftRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "collectible is not available to award")}
	}

	var isActiveMember bool
	if err := tx.QueryRow(ctx, `
		select exists(
			select 1 from organization_memberships
			where organization_id = $1 and user_id = $2 and status = $3
		)
	`, command.OrganizationID.String(), command.RecipientUserID.String(), org.MembershipStatusActive.String()).Scan(&isActiveMember); err != nil {
		return assets.GiftRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "check organization membership failed")}
	}
	if !isActiveMember {
		return assets.GiftRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "recipient is not an active member of this organization")}
	}

	if _, err := tx.Exec(ctx, "update collectibles set owner_user_id = $2, owner_kind = 'user', organization_id = null, state_recorded_at = now() where id = $1", command.CollectibleID.String(), command.RecipientUserID.String()); err != nil {
		return assets.GiftRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "transfer organization collectible failed")}
	}
	if err := tx.Commit(ctx); err != nil {
		return assets.GiftRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "commit award organization collectible failed")}
	}

	awarded := collectible.value
	awarded.OwnerKind = assets.CollectibleOwnerKindUser
	awarded.OwnerID = command.RecipientUserID.String()
	awarded.OrganizationID = ""
	return assets.CollectibleGifted{Value: awarded}
}

func requireSharedCollectibleOrganization(ctx context.Context, tx pgx.Tx, collectible assets.Collectible, from core.UserID, to core.UserID) (core.DomainError, bool) {
	if collectible.OrganizationID == "" {
		return core.NewDomainError(core.ErrorCodeInvalidState, "within-organization collectible has no organization"), true
	}
	for _, userID := range []core.UserID{from, to} {
		var active bool
		if err := tx.QueryRow(ctx, `
			select exists(
				select 1
				from organization_memberships
				where organization_id = $1
					and user_id = $2
					and status = 'active'
			)
		`, collectible.OrganizationID, userID.String()).Scan(&active); err != nil {
			return core.NewDomainError(core.ErrorCodeInvalidState, "check collectible organization membership failed"), true
		}
		if !active {
			return core.NewDomainError(core.ErrorCodePermissionDenied, "within-organization collectible can only be tipped between organization members"), true
		}
	}
	return core.DomainError{}, false
}

func lockCollectible(ctx context.Context, tx pgx.Tx, collectibleID core.CollectibleID) collectibleParseResult {
	rows, err := tx.Query(ctx, "select id::text, name, kind, state, transfer_policy, owner_user_id::text, owner_kind, coalesce(organization_id::text, ''), art from collectibles where id = $1 for update", collectibleID.String())
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
	rows, err := tx.Query(ctx, "select id::text, name, kind, state, transfer_policy, owner_user_id::text, owner_kind, coalesce(organization_id::text, ''), art from collectibles where id = $1", rawCollectibleID)
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
	var rawOrganizationID string
	var rawArt string
	if err := rows.Scan(&rawID, &rawName, &rawKind, &rawState, &rawPolicy, &rawOwner, &rawOwnerKind, &rawOrganizationID, &rawArt); err != nil {
		return collectibleParseRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "scan collectible failed")}
	}
	return parseCollectible(rawID, rawName, rawKind, rawState, rawPolicy, rawOwner, rawOwnerKind, rawOrganizationID, rawArt)
}

func parseCollectible(rawID string, rawName string, rawKind string, rawState string, rawPolicy string, rawOwner string, rawOwnerKind string, rawOrganizationID string, rawArt string) collectibleParseResult {
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
	if rawOrganizationID != "" {
		if _, matched := core.ParseOrganizationID(rawOrganizationID).(core.OrganizationIDCreated); !matched {
			return collectibleParseRejected{reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "collectible organization is invalid")}
		}
	}
	return collectibleParsed{value: assets.Collectible{
		ID:             collectibleID.Value,
		Name:           name.Value,
		Kind:           kind.Value,
		State:          state.Value,
		Policy:         policy.Value,
		OwnerKind:      rawOwnerKind,
		OwnerID:        rawOwner,
		OrganizationID: rawOrganizationID,
		Art:            rawArt,
	}}
}

func nullableText(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
