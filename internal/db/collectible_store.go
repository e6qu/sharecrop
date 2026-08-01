package db

import (
	"context"

	"github.com/e6qu/sharecrop/internal/assets"
	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/event"
	"github.com/e6qu/sharecrop/internal/org"
	"github.com/jackc/pgx/v5/pgxpool"
)

// collectibleColumns is the shared collectible row projection (including the
// provenance columns added by migration 000051).
const collectibleColumns = `id::text, name, kind, state, transfer_policy, owner_user_id::text, owner_kind,
	coalesce(organization_id::text, ''), art, coalesce(catalog_slug, ''), edition_number, coalesce(issued_by_user_id::text, '')`

// collectibleListColumns is collectibleColumns qualified for the left join
// that resolves the issuer's display name on the list read paths (mutation
// paths keep the unjoined projection so their FOR UPDATE locks stay simple),
// plus the owner's display label resolved per owner kind.
var collectibleListColumns = `collectibles.id::text, collectibles.name, collectibles.kind, collectibles.state,
	collectibles.transfer_policy, collectibles.owner_user_id::text, collectibles.owner_kind,
	coalesce(collectibles.organization_id::text, ''), collectibles.art, coalesce(collectibles.catalog_slug, ''),
	collectibles.edition_number, coalesce(collectibles.issued_by_user_id::text, ''),
	coalesce(coalesce(nullif(users.display_name, ''), split_part(users.email, '@', 1)), ''),
	` + collectibleOwnerLabelSQL("collectibles")

// collectibleIssuerJoin joins the issuer's user row for display-name
// resolution; issuerless (pre-provenance) rows keep an empty name.
const collectibleIssuerJoin = "left join users on users.id = collectibles.issued_by_user_id"

// collectibleOwnerLabelSQL resolves the owner's display label for one
// collectibles row alias: the user's display name (the standard
// displayNameSQL fallback chain), the organization's name, or the team's
// name, per owner_kind. An owner row that no longer exists yields an empty
// label.
func collectibleOwnerLabelSQL(alias string) string {
	return `case ` + alias + `.owner_kind
		when 'user' then coalesce((select ` + displayNameSQL("owner_users") + ` from users owner_users where owner_users.id = ` + alias + `.owner_user_id), '')
		when 'organization' then coalesce((select owner_orgs.name from organizations owner_orgs where owner_orgs.id = ` + alias + `.owner_user_id), '')
		when 'team' then coalesce((select owner_teams.name from teams owner_teams where owner_teams.id = ` + alias + `.owner_user_id), '')
		else ''
	end`
}

type CollectibleStore struct {
	db Beginner
}

func NewCollectibleStore(pool *pgxpool.Pool) CollectibleStore {
	return NewCollectibleStoreFromHandle(NewPGX(pool))
}

func NewCollectibleStoreFromHandle(handle Beginner) CollectibleStore {
	return CollectibleStore{db: handle}
}

// CreateCollectible mints a custom collectible, enforcing the kind
// semantics for custom mints inside one transaction: at most one live unique
// per (issuer, name), and edition numbering per (issuer, name). The partial
// unique indexes from migration 000051 back the application checks.
func (store CollectibleStore) CreateCollectible(ctx context.Context, collectible assets.Collectible) assets.CreateStoreResult {
	tx, err := store.db.Begin(ctx)
	if err != nil {
		return assets.CreateStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "begin mint collectible transaction failed")}
	}
	defer func() { _ = tx.Rollback(ctx) }()

	issuer := ""
	if issuedBy, matched := collectible.Issuer.(assets.IssuedBy); matched {
		issuer = issuedBy.ID.String()
	}

	created := collectible
	created.Edition = assets.NoEditionNumber{}
	var editionNumber *int64
	if issuer != "" {
		switch collectible.Kind {
		case assets.CollectibleKindUnique:
			var liveExists bool
			if err := tx.QueryRow(ctx, `
				select exists(
					select 1 from collectibles
					where kind = 'unique' and catalog_slug is null
					and issued_by_user_id = $1 and name = $2 and state <> 'withdrawn'
				)
			`, issuer, collectible.Name.String()).Scan(&liveExists); err != nil {
				return assets.CreateStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "check unique collectible failed")}
			}
			if liveExists {
				return assets.CreateStoreRejected{Reason: core.NewDomainError(core.ErrorCodeConflict, "a live instance of this unique collectible already exists")}
			}
		case assets.CollectibleKindEdition:
			var next int64
			if err := tx.QueryRow(ctx, `
				select coalesce(max(edition_number), 0) + 1 from collectibles
				where catalog_slug is null and issued_by_user_id = $1 and name = $2
			`, issuer, collectible.Name.String()).Scan(&next); err != nil {
				return assets.CreateStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "number collectible edition failed")}
			}
			editionNumber = &next
			created.Edition = assets.EditionNumbered{Number: next}
		}
	}

	_, err = tx.Exec(ctx, `
		insert into collectibles (id, name, kind, state, transfer_policy, owner_user_id, owner_kind, organization_id, art, catalog_slug, edition_number, issued_by_user_id)
		values ($1, $2, $3, $4, $5, $6, $7, $8, $9, null, $10, $11)
	`, collectible.ID.String(), collectible.Name.String(), collectible.Kind.String(), collectible.State.String(), collectible.Policy.String(), collectible.OwnerID, collectible.OwnerKind, nullableText(collectible.OrganizationID), collectible.Art, editionNumber, nullableText(issuer))
	if err != nil {
		if isUniqueViolation(err) {
			return assets.CreateStoreRejected{Reason: core.NewDomainError(core.ErrorCodeConflict, "collectible uniqueness constraint was violated")}
		}
		return assets.CreateStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "insert collectible failed")}
	}
	if err := tx.Commit(ctx); err != nil {
		return assets.CreateStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "commit mint collectible transaction failed")}
	}
	return assets.CreateStoreAccepted{Value: created}
}

func (store CollectibleStore) ListCollectibles(ctx context.Context, owner core.UserID, page core.Page) assets.ListStoreResult {
	rows, err := store.db.Query(ctx, `
		select `+collectibleListColumns+`
		from collectibles
		`+collectibleIssuerJoin+`
		where collectibles.owner_user_id = $1 and collectibles.owner_kind = 'user'
		order by collectibles.created_at desc, collectibles.id
		limit $2 offset $3
	`, owner.String(), page.Limit(), page.Offset())
	if err != nil {
		return assets.ListStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "list collectibles failed")}
	}
	defer rows.Close()
	return collectListedCollectibles(rows)
}

func (store CollectibleStore) TaskHeldCollectibles(ctx context.Context, taskID core.TaskID) assets.TaskHeldCollectiblesResult {
	rows, err := store.db.Query(ctx,
		"select collectible_id::text from task_fund_collectibles where task_id = $1 order by collectible_id",
		taskID.String())
	if err != nil {
		return assets.TaskHeldCollectiblesRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "list held task collectibles failed")}
	}
	defer rows.Close()

	ids := make([]core.CollectibleID, 0)
	for rows.Next() {
		var rawID string
		if err := rows.Scan(&rawID); err != nil {
			return assets.TaskHeldCollectiblesRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "scan held task collectible failed")}
		}
		idResult := core.ParseCollectibleID(rawID)
		idCreated, matched := idResult.(core.CollectibleIDCreated)
		if !matched {
			return assets.TaskHeldCollectiblesRejected{Reason: idResult.(core.CollectibleIDRejected).Reason}
		}
		ids = append(ids, idCreated.Value)
	}
	if rows.Err() != nil {
		return assets.TaskHeldCollectiblesRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "read held task collectibles failed")}
	}
	return assets.TaskHeldCollectiblesFound{IDs: ids}
}

func (store CollectibleStore) ListCollectiblesByOwner(ctx context.Context, ownerKind string, ownerID string, page core.Page) assets.ListStoreResult {
	rows, err := store.db.Query(ctx, `
		select `+collectibleListColumns+`
		from collectibles
		`+collectibleIssuerJoin+`
		where collectibles.owner_user_id = $1 and collectibles.owner_kind = $2
		order by collectibles.created_at desc, collectibles.id
		limit $3 offset $4
	`, ownerID, ownerKind, page.Limit(), page.Offset())
	if err != nil {
		return assets.ListStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "list collectibles failed")}
	}
	defer rows.Close()
	return collectListedCollectibles(rows)
}

// collectListedCollectibles scans every row of a collectibles list query
// (projected with the trailing issuer display-name column) into the
// listed-store result, rejecting on the first unparsable row.
func collectListedCollectibles(rows Rows) assets.ListStoreResult {
	values := make([]assets.Collectible, 0)
	for rows.Next() {
		parsed := scanCollectibleWithIssuerName(rows)
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
	tx, err := store.db.Begin(ctx)
	if err != nil {
		return assets.FundRewardRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "begin fund collectible reward transaction failed")}
	}
	defer func() { _ = tx.Rollback(ctx) }()

	collectible, escrowableReason := requireEscrowableCollectible(ctx, tx, command.CollectibleID, command.FunderUserID)
	if escrowableReason != nil {
		return assets.FundRewardRejected{Reason: *escrowableReason}
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

	if holdReason := holdCollectibleForTask(ctx, tx, command.TaskID, command.CollectibleID); holdReason != nil {
		return assets.FundRewardRejected{Reason: *holdReason}
	}

	if err := tx.Commit(ctx); err != nil {
		return assets.FundRewardRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "commit fund collectible reward transaction failed")}
	}

	awarded := collectible
	awarded.State = assets.CollectibleStateEscrowed
	return assets.RewardFunded{Value: awarded}
}

// requireEscrowableCollectible locks a collectible and validates that funder
// can escrow it as a task reward: funder must own it as a user, it must be
// available (minted), and its transfer policy must allow reward payout. It is
// shared by the post-create funding path (FundCollectibleReward) and the
// create-with-escrow path (TaskStore.CreateTask), so both enforce the same
// per-collectible rules inside their transactions.
func requireEscrowableCollectible(ctx context.Context, tx Tx, collectibleID core.CollectibleID, funder core.UserID) (assets.Collectible, *core.DomainError) {
	collectibleResult := lockCollectible(ctx, tx, collectibleID)
	collectible, collectibleMatched := collectibleResult.(collectibleParsed)
	if !collectibleMatched {
		reason := collectibleResult.(collectibleParseRejected).reason
		return assets.Collectible{}, &reason
	}
	if collectible.value.OwnerKind != assets.CollectibleOwnerKindUser || collectible.value.OwnerID != funder.String() {
		reason := core.NewDomainError(core.ErrorCodeInvalidArgument, "only the collectible owner can fund a task with it")
		return assets.Collectible{}, &reason
	}
	if collectible.value.State != assets.CollectibleStateMinted {
		reason := core.NewDomainError(core.ErrorCodeInvalidState, "collectible is not available to escrow")
		return assets.Collectible{}, &reason
	}
	if denied, matched := assets.AllowsRewardPayout(collectible.value.Policy).(assets.RewardDenied); matched {
		reason := denied.Reason
		return assets.Collectible{}, &reason
	}
	return collectible.value, nil
}

// holdCollectibleForTask moves a validated collectible into escrow and records
// the task's hold on it. Shared by FundCollectibleReward and
// TaskStore.CreateTask (see requireEscrowableCollectible).
func holdCollectibleForTask(ctx context.Context, tx Tx, taskID core.TaskID, collectibleID core.CollectibleID) *core.DomainError {
	if _, err := tx.Exec(ctx, "update collectibles set state = 'escrowed', state_recorded_at = now() where id = $1", collectibleID.String()); err != nil {
		reason := core.NewDomainError(core.ErrorCodeInvalidState, "escrow collectible failed")
		return &reason
	}
	if _, err := tx.Exec(ctx, `
		insert into task_fund_collectibles (task_id, collectible_id)
		values ($1, $2)
	`, taskID.String(), collectibleID.String()); err != nil {
		if isUniqueViolation(err) {
			reason := core.NewDomainError(core.ErrorCodeConflict, "collectible is already escrowed on this task")
			return &reason
		}
		reason := core.NewDomainError(core.ErrorCodeInvalidState, "insert collectible reward failed")
		return &reason
	}
	return nil
}

func (store CollectibleStore) RefundCollectibleReward(ctx context.Context, command assets.RefundRewardStoreCommand) assets.RefundRewardResult {
	tx, err := store.db.Begin(ctx)
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

	// A collectible-reward refund cancels the task, so release the worker's
	// reservation too (same as the cancel and credit-refund paths).
	if reason := releaseReservationsOnCancel(ctx, tx, command.TaskID); reason != nil {
		return assets.RefundRewardRejected{Reason: *reason}
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
	tx, err := store.db.Begin(ctx)
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
	// retried accept (after a lost response) treats the gift as already done
	// and records no new event.
	if collectible.value.OwnerKind == assets.CollectibleOwnerKindUser && collectible.value.OwnerID == command.ToUserID.String() {
		return assets.CollectibleGifted{Value: collectible.value, RecordedEvents: []event.Draft{}}
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
	if err := recordEventDraftInTx(ctx, tx, command.Draft); err != nil {
		return assets.GiftRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "record collectible gift event failed")}
	}
	if err := tx.Commit(ctx); err != nil {
		return assets.GiftRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "commit gift collectible failed")}
	}

	gifted := collectible.value
	gifted.OwnerKind = assets.CollectibleOwnerKindUser
	gifted.OwnerID = command.ToUserID.String()
	return assets.CollectibleGifted{Value: gifted, RecordedEvents: []event.Draft{command.Draft}}
}

func (store CollectibleStore) AwardOrganizationCollectible(ctx context.Context, command assets.AwardOrganizationCollectibleStoreCommand) assets.GiftResult {
	tx, err := store.db.Begin(ctx)
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
	if err := recordEventDraftInTx(ctx, tx, command.Draft); err != nil {
		return assets.GiftRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "record collectible award event failed")}
	}
	if err := tx.Commit(ctx); err != nil {
		return assets.GiftRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "commit award organization collectible failed")}
	}

	awarded := collectible.value
	awarded.OwnerKind = assets.CollectibleOwnerKindUser
	awarded.OwnerID = command.RecipientUserID.String()
	awarded.OrganizationID = ""
	return assets.CollectibleGifted{Value: awarded, RecordedEvents: []event.Draft{command.Draft}}
}

func requireSharedCollectibleOrganization(ctx context.Context, tx Tx, collectible assets.Collectible, from core.UserID, to core.UserID) (core.DomainError, bool) {
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

func lockCollectible(ctx context.Context, tx Tx, collectibleID core.CollectibleID) collectibleParseResult {
	rows, err := tx.Query(ctx, "select "+collectibleColumns+" from collectibles where id = $1 for update", collectibleID.String())
	if err != nil {
		return collectibleParseRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "lock collectible failed")}
	}
	defer rows.Close()
	if !rows.Next() {
		return collectibleParseRejected{reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "collectible was not found")}
	}
	return scanCollectible(rows)
}

func findCollectible(ctx context.Context, tx Tx, rawCollectibleID string) collectibleParseResult {
	rows, err := tx.Query(ctx, "select "+collectibleColumns+" from collectibles where id = $1", rawCollectibleID)
	if err != nil {
		return collectibleParseRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "read collectible failed")}
	}
	defer rows.Close()
	if !rows.Next() {
		return collectibleParseRejected{reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "collectible was not found")}
	}
	return scanCollectible(rows)
}

func scanCollectible(rows Rows) collectibleParseResult {
	var rawID string
	var rawName string
	var rawKind string
	var rawState string
	var rawPolicy string
	var rawOwner string
	var rawOwnerKind string
	var rawOrganizationID string
	var rawArt string
	var rawCatalogSlug string
	var editionNumber *int64
	var rawIssuer string
	if err := rows.Scan(&rawID, &rawName, &rawKind, &rawState, &rawPolicy, &rawOwner, &rawOwnerKind, &rawOrganizationID, &rawArt, &rawCatalogSlug, &editionNumber, &rawIssuer); err != nil {
		return collectibleParseRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "scan collectible failed")}
	}
	return parseCollectible(rawID, rawName, rawKind, rawState, rawPolicy, rawOwner, rawOwnerKind, rawOrganizationID, rawArt, rawCatalogSlug, editionNumber, rawIssuer)
}

// scanCollectibleWithIssuerName scans a list-projection row (the collectible
// columns plus the resolved issuer display name and owner display label).
func scanCollectibleWithIssuerName(rows Rows) collectibleParseResult {
	var rawID string
	var rawName string
	var rawKind string
	var rawState string
	var rawPolicy string
	var rawOwner string
	var rawOwnerKind string
	var rawOrganizationID string
	var rawArt string
	var rawCatalogSlug string
	var editionNumber *int64
	var rawIssuer string
	var rawIssuerName string
	var rawOwnerName string
	if err := rows.Scan(&rawID, &rawName, &rawKind, &rawState, &rawPolicy, &rawOwner, &rawOwnerKind, &rawOrganizationID, &rawArt, &rawCatalogSlug, &editionNumber, &rawIssuer, &rawIssuerName, &rawOwnerName); err != nil {
		return collectibleParseRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "scan collectible failed")}
	}
	parsed := parseCollectible(rawID, rawName, rawKind, rawState, rawPolicy, rawOwner, rawOwnerKind, rawOrganizationID, rawArt, rawCatalogSlug, editionNumber, rawIssuer)
	accepted, matched := parsed.(collectibleParsed)
	if !matched {
		return parsed
	}
	if rawIssuerName != "" {
		nameResult := auth.NewDisplayName(rawIssuerName)
		name, nameMatched := nameResult.(auth.DisplayNameAccepted)
		if !nameMatched {
			return collectibleParseRejected{reason: nameResult.(auth.DisplayNameRejected).Reason}
		}
		accepted.value.IssuerDisplayName = name.Value
	}
	if rawOwnerName != "" {
		nameResult := auth.NewDisplayName(rawOwnerName)
		name, nameMatched := nameResult.(auth.DisplayNameAccepted)
		if !nameMatched {
			return collectibleParseRejected{reason: nameResult.(auth.DisplayNameRejected).Reason}
		}
		accepted.value.OwnerDisplayName = name.Value
	}
	return collectibleParsed{value: accepted.value}
}

func parseCollectible(rawID string, rawName string, rawKind string, rawState string, rawPolicy string, rawOwner string, rawOwnerKind string, rawOrganizationID string, rawArt string, rawCatalogSlug string, editionNumber *int64, rawIssuer string) collectibleParseResult {
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
	var catalogRef assets.CatalogRef = assets.NoCatalogRef{}
	if rawCatalogSlug != "" {
		slugResult := assets.NewCatalogSlug(rawCatalogSlug)
		slug, slugMatched := slugResult.(assets.CatalogSlugAccepted)
		if !slugMatched {
			return collectibleParseRejected{reason: slugResult.(assets.CatalogSlugRejected).Reason}
		}
		catalogRef = assets.FromCatalog{Slug: slug.Value}
	}
	var editionRef assets.EditionRef = assets.NoEditionNumber{}
	if editionNumber != nil {
		editionRef = assets.EditionNumbered{Number: *editionNumber}
	}
	var issuerRef assets.IssuerRef = assets.NoIssuer{}
	if rawIssuer != "" {
		issuerResult := core.ParseUserID(rawIssuer)
		issuerCreated, issuerMatched := issuerResult.(core.UserIDCreated)
		if !issuerMatched {
			return collectibleParseRejected{reason: issuerResult.(core.UserIDRejected).Reason}
		}
		issuerRef = assets.IssuedBy{ID: issuerCreated.Value}
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
		Catalog:        catalogRef,
		Edition:        editionRef,
		Issuer:         issuerRef,
	}}
}

func nullableText(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

// catalogEntryColumns is the shared catalog entry projection.
const catalogEntryColumns = "slug, name, kind, transfer_policy, art, state, max_editions"

type catalogEntryParseResult interface {
	catalogEntryParseResult()
}

type catalogEntryParsed struct {
	value assets.CatalogEntry
}

type catalogEntryParseRejected struct {
	reason core.DomainError
}

func (catalogEntryParsed) catalogEntryParseResult() {}

func (catalogEntryParseRejected) catalogEntryParseResult() {}

func scanCatalogEntry(rows Rows) catalogEntryParseResult {
	var rawSlug, rawName, rawKind, rawPolicy, rawArt, rawState string
	var maxEditions *int64
	if err := rows.Scan(&rawSlug, &rawName, &rawKind, &rawPolicy, &rawArt, &rawState, &maxEditions); err != nil {
		return catalogEntryParseRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "scan catalog entry failed")}
	}
	return parseCatalogEntry(rawSlug, rawName, rawKind, rawPolicy, rawArt, rawState, maxEditions)
}

func parseCatalogEntry(rawSlug string, rawName string, rawKind string, rawPolicy string, rawArt string, rawState string, maxEditions *int64) catalogEntryParseResult {
	slugResult := assets.NewCatalogSlug(rawSlug)
	slug, slugMatched := slugResult.(assets.CatalogSlugAccepted)
	if !slugMatched {
		return catalogEntryParseRejected{reason: slugResult.(assets.CatalogSlugRejected).Reason}
	}
	nameResult := assets.NewCollectibleName(rawName)
	name, nameMatched := nameResult.(assets.CollectibleNameAccepted)
	if !nameMatched {
		return catalogEntryParseRejected{reason: nameResult.(assets.CollectibleNameRejected).Reason}
	}
	kindResult := assets.ParseCollectibleKind(rawKind)
	kind, kindMatched := kindResult.(assets.CollectibleKindAccepted)
	if !kindMatched {
		return catalogEntryParseRejected{reason: kindResult.(assets.CollectibleKindRejected).Reason}
	}
	policyResult := assets.ParseTransferPolicy(rawPolicy)
	policy, policyMatched := policyResult.(assets.TransferPolicyAccepted)
	if !policyMatched {
		return catalogEntryParseRejected{reason: policyResult.(assets.TransferPolicyRejected).Reason}
	}
	stateResult := assets.ParseCatalogEntryState(rawState)
	state, stateMatched := stateResult.(assets.CatalogEntryStateAccepted)
	if !stateMatched {
		return catalogEntryParseRejected{reason: stateResult.(assets.CatalogEntryStateRejected).Reason}
	}
	var cap assets.EditionCap = assets.NoEditionCap{}
	if maxEditions != nil {
		cap = assets.EditionCapOf{Limit: *maxEditions}
	}
	return catalogEntryParsed{value: assets.CatalogEntry{
		Slug:   slug.Value,
		Name:   name.Value,
		Kind:   kind.Value,
		Policy: policy.Value,
		Art:    rawArt,
		State:  state.Value,
		Cap:    cap,
	}}
}

func (store CollectibleStore) ListCatalogEntries(ctx context.Context) assets.CatalogListResult {
	// The correlated subqueries report each entry's circulation in the same
	// read, without a per-entry round trip: the live (non-withdrawn) instance
	// count, the distinct owners of those instances, and — for unique entries
	// — the display label of the live instance's holder.
	rows, err := store.db.Query(ctx, `
		select `+catalogEntryColumns+`,
			(select count(*) from collectibles where collectibles.catalog_slug = collectible_catalog_entries.slug and collectibles.state <> 'withdrawn') as live_instances,
			(select count(distinct collectibles.owner_kind || ':' || collectibles.owner_user_id::text) from collectibles where collectibles.catalog_slug = collectible_catalog_entries.slug and collectibles.state <> 'withdrawn') as live_owners,
			case when collectible_catalog_entries.kind = 'unique' then coalesce((
				select `+collectibleOwnerLabelSQL("live_uniques")+`
				from collectibles live_uniques
				where live_uniques.catalog_slug = collectible_catalog_entries.slug and live_uniques.state <> 'withdrawn'
				limit 1
			), '') else '' end as owner_label
		from collectible_catalog_entries
		order by created_at asc, slug asc
	`)
	if err != nil {
		return assets.CatalogListRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "list catalog entries failed")}
	}
	defer rows.Close()
	values := make([]assets.CatalogListing, 0)
	for rows.Next() {
		parsed := scanCatalogListing(rows)
		accepted, matched := parsed.(catalogListingParsed)
		if !matched {
			return assets.CatalogListRejected{Reason: parsed.(catalogListingParseRejected).reason}
		}
		values = append(values, accepted.value)
	}
	if rows.Err() != nil {
		return assets.CatalogListRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "read catalog entries failed")}
	}
	return assets.CatalogListed{Values: values}
}

type catalogListingParseResult interface {
	catalogListingParseResult()
}

type catalogListingParsed struct {
	value assets.CatalogListing
}

type catalogListingParseRejected struct {
	reason core.DomainError
}

func (catalogListingParsed) catalogListingParseResult() {}

func (catalogListingParseRejected) catalogListingParseResult() {}

// scanCatalogListing scans one catalog row projected with the trailing
// live-instance count, live-owner count, and unique-owner label columns.
func scanCatalogListing(rows Rows) catalogListingParseResult {
	var rawSlug, rawName, rawKind, rawPolicy, rawArt, rawState string
	var maxEditions *int64
	var liveInstances int64
	var liveOwners int64
	var rawOwnerLabel string
	if err := rows.Scan(&rawSlug, &rawName, &rawKind, &rawPolicy, &rawArt, &rawState, &maxEditions, &liveInstances, &liveOwners, &rawOwnerLabel); err != nil {
		return catalogListingParseRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "scan catalog listing failed")}
	}
	parsed := parseCatalogEntry(rawSlug, rawName, rawKind, rawPolicy, rawArt, rawState, maxEditions)
	entry, matched := parsed.(catalogEntryParsed)
	if !matched {
		return catalogListingParseRejected{reason: parsed.(catalogEntryParseRejected).reason}
	}
	var ownerLabel auth.DisplayName
	if rawOwnerLabel != "" {
		labelResult := auth.NewDisplayName(rawOwnerLabel)
		label, labelMatched := labelResult.(auth.DisplayNameAccepted)
		if !labelMatched {
			return catalogListingParseRejected{reason: labelResult.(auth.DisplayNameRejected).Reason}
		}
		ownerLabel = label.Value
	}
	return catalogListingParsed{value: assets.CatalogListing{
		Entry:             entry.value,
		LiveInstanceCount: liveInstances,
		LiveOwnerCount:    liveOwners,
		OwnerDisplayName:  ownerLabel,
	}}
}

func maxEditionsColumn(cap assets.EditionCap) *int64 {
	if capped, matched := cap.(assets.EditionCapOf); matched {
		limit := capped.Limit
		return &limit
	}
	return nil
}

func (store CollectibleStore) AddCatalogEntry(ctx context.Context, entry assets.CatalogEntry) assets.CatalogMutationResult {
	_, err := store.db.Exec(ctx, `
		insert into collectible_catalog_entries (slug, name, kind, transfer_policy, art, state, max_editions)
		values ($1, $2, $3, $4, $5, $6, $7)
	`, entry.Slug.String(), entry.Name.String(), entry.Kind.String(), entry.Policy.String(), entry.Art, entry.State.String(), maxEditionsColumn(entry.Cap))
	if err != nil {
		if isUniqueViolation(err) {
			return assets.CatalogMutationRejected{Reason: core.NewDomainError(core.ErrorCodeConflict, "catalog slug already exists")}
		}
		return assets.CatalogMutationRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "insert catalog entry failed")}
	}
	return assets.CatalogMutated{Value: entry}
}

func (store CollectibleStore) findCatalogEntry(ctx context.Context, querier Querier, slug assets.CatalogSlug, forUpdate bool) catalogEntryParseResult {
	query := "select " + catalogEntryColumns + " from collectible_catalog_entries where slug = $1"
	if forUpdate {
		query += " for update"
	}
	rows, err := querier.Query(ctx, query, slug.String())
	if err != nil {
		return catalogEntryParseRejected{reason: core.NewDomainError(core.ErrorCodeInvalidState, "read catalog entry failed")}
	}
	defer rows.Close()
	if !rows.Next() {
		return catalogEntryParseRejected{reason: core.NewDomainError(core.ErrorCodeNotFound, "catalog entry was not found")}
	}
	return scanCatalogEntry(rows)
}

// WithdrawCatalogEntry marks an available entry withdrawn: it can no longer
// be awarded, while existing instances stay untouched.
func (store CollectibleStore) WithdrawCatalogEntry(ctx context.Context, slug assets.CatalogSlug) assets.CatalogMutationResult {
	tag, err := store.db.Exec(ctx, `
		update collectible_catalog_entries
		set state = 'withdrawn', state_recorded_at = now()
		where slug = $1 and state = 'available'
	`, slug.String())
	if err != nil {
		return assets.CatalogMutationRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "withdraw catalog entry failed")}
	}
	if tag != 1 {
		return assets.CatalogMutationRejected{Reason: core.NewDomainError(core.ErrorCodeConflict, "catalog entry is not available to withdraw")}
	}
	result := store.findCatalogEntry(ctx, store.db, slug, false)
	entry, matched := result.(catalogEntryParsed)
	if !matched {
		return assets.CatalogMutationRejected{Reason: result.(catalogEntryParseRejected).reason}
	}
	return assets.CatalogMutated{Value: entry.value}
}

// ReleaseCatalogEntry returns a withdrawn entry to the available state so it
// can be awarded again. An entry that is not withdrawn (available, or
// nonexistent) is refused with a conflict, mirroring WithdrawCatalogEntry.
func (store CollectibleStore) ReleaseCatalogEntry(ctx context.Context, slug assets.CatalogSlug) assets.CatalogMutationResult {
	tag, err := store.db.Exec(ctx, `
		update collectible_catalog_entries
		set state = 'available', state_recorded_at = now()
		where slug = $1 and state = 'withdrawn'
	`, slug.String())
	if err != nil {
		return assets.CatalogMutationRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "release catalog entry failed")}
	}
	if tag != 1 {
		return assets.CatalogMutationRejected{Reason: core.NewDomainError(core.ErrorCodeConflict, "catalog entry is not withdrawn and cannot be released")}
	}
	result := store.findCatalogEntry(ctx, store.db, slug, false)
	entry, matched := result.(catalogEntryParsed)
	if !matched {
		return assets.CatalogMutationRejected{Reason: result.(catalogEntryParseRejected).reason}
	}
	return assets.CatalogMutated{Value: entry.value}
}

// DeleteCatalogEntry removes a withdrawn entry that no instance references —
// live or withdrawn. Withdrawn instances still carry the entry's provenance
// (and can be released back into circulation), so deleting a history-bearing
// entry is refused; delete the withdrawn instances first.
func (store CollectibleStore) DeleteCatalogEntry(ctx context.Context, slug assets.CatalogSlug) assets.CatalogMutationResult {
	tx, err := store.db.Begin(ctx)
	if err != nil {
		return assets.CatalogMutationRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "begin delete catalog entry transaction failed")}
	}
	defer func() { _ = tx.Rollback(ctx) }()

	result := store.findCatalogEntry(ctx, tx, slug, true)
	entry, matched := result.(catalogEntryParsed)
	if !matched {
		return assets.CatalogMutationRejected{Reason: result.(catalogEntryParseRejected).reason}
	}
	if entry.value.State != assets.CatalogEntryStateWithdrawn {
		return assets.CatalogMutationRejected{Reason: core.NewDomainError(core.ErrorCodeConflict, "only withdrawn catalog entries can be deleted")}
	}
	var hasInstances bool
	if err := tx.QueryRow(ctx, `
		select exists(select 1 from collectibles where catalog_slug = $1)
	`, slug.String()).Scan(&hasInstances); err != nil {
		return assets.CatalogMutationRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "check catalog entry instances failed")}
	}
	if hasInstances {
		return assets.CatalogMutationRejected{Reason: core.NewDomainError(core.ErrorCodeConflict, "catalog entry still has instances (live or withdrawn)")}
	}
	if _, err := tx.Exec(ctx, "delete from collectible_catalog_entries where slug = $1", slug.String()); err != nil {
		return assets.CatalogMutationRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "delete catalog entry failed")}
	}
	if err := tx.Commit(ctx); err != nil {
		return assets.CatalogMutationRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "commit delete catalog entry failed")}
	}
	return assets.CatalogMutated{Value: entry.value}
}

// AwardFromCatalog mints one instance of an available catalog entry inside a
// transaction that locks the entry row, so uniqueness checks and edition
// numbering serialize per entry (the partial unique indexes back them up).
func (store CollectibleStore) AwardFromCatalog(ctx context.Context, command assets.AwardFromCatalogStoreCommand) assets.CreateStoreResult {
	tx, err := store.db.Begin(ctx)
	if err != nil {
		return assets.CreateStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "begin award collectible transaction failed")}
	}
	defer func() { _ = tx.Rollback(ctx) }()

	entryResult := store.findCatalogEntry(ctx, tx, command.Slug, true)
	entry, entryMatched := entryResult.(catalogEntryParsed)
	if !entryMatched {
		return assets.CreateStoreRejected{Reason: entryResult.(catalogEntryParseRejected).reason}
	}
	if entry.value.State != assets.CatalogEntryStateAvailable {
		return assets.CreateStoreRejected{Reason: core.NewDomainError(core.ErrorCodeConflict, "catalog entry is withdrawn and can no longer be awarded")}
	}

	var editionRef assets.EditionRef = assets.NoEditionNumber{}
	var editionNumber *int64
	switch entry.value.Kind {
	case assets.CollectibleKindUnique:
		var liveExists bool
		if err := tx.QueryRow(ctx, `
			select exists(select 1 from collectibles where catalog_slug = $1 and state <> 'withdrawn')
		`, command.Slug.String()).Scan(&liveExists); err != nil {
			return assets.CreateStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "check unique catalog instance failed")}
		}
		if liveExists {
			return assets.CreateStoreRejected{Reason: core.NewDomainError(core.ErrorCodeConflict, "unique collectible already has a live instance")}
		}
	case assets.CollectibleKindEdition:
		var next int64
		if err := tx.QueryRow(ctx, `
			select coalesce(max(edition_number), 0) + 1 from collectibles where catalog_slug = $1
		`, command.Slug.String()).Scan(&next); err != nil {
			return assets.CreateStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "number catalog edition failed")}
		}
		if capped, matched := entry.value.Cap.(assets.EditionCapOf); matched && next > capped.Limit {
			return assets.CreateStoreRejected{Reason: core.NewDomainError(core.ErrorCodeConflict, "edition cap reached; no further instances can be minted")}
		}
		editionNumber = &next
		editionRef = assets.EditionNumbered{Number: next}
	}

	if _, err := tx.Exec(ctx, `
		insert into collectibles (id, name, kind, state, transfer_policy, owner_user_id, owner_kind, organization_id, art, catalog_slug, edition_number, issued_by_user_id)
		values ($1, $2, $3, 'minted', $4, $5, $6, $7, $8, $9, $10, $11)
	`, command.NewID.String(), entry.value.Name.String(), entry.value.Kind.String(), entry.value.Policy.String(),
		command.OwnerID, command.OwnerKind, nullableText(command.OrganizationID), entry.value.Art,
		command.Slug.String(), editionNumber, command.Issuer.String()); err != nil {
		if isUniqueViolation(err) {
			return assets.CreateStoreRejected{Reason: core.NewDomainError(core.ErrorCodeConflict, "collectible uniqueness constraint was violated")}
		}
		return assets.CreateStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "insert awarded collectible failed")}
	}
	if err := tx.Commit(ctx); err != nil {
		return assets.CreateStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "commit award collectible transaction failed")}
	}
	return assets.CreateStoreAccepted{Value: assets.Collectible{
		ID:             command.NewID,
		Name:           entry.value.Name,
		Kind:           entry.value.Kind,
		State:          assets.CollectibleStateMinted,
		Policy:         entry.value.Policy,
		OwnerKind:      command.OwnerKind,
		OwnerID:        command.OwnerID,
		OrganizationID: command.OrganizationID,
		Art:            entry.value.Art,
		Catalog:        assets.FromCatalog{Slug: command.Slug},
		Edition:        editionRef,
		Issuer:         assets.IssuedBy{ID: command.Issuer},
	}}
}

// WithdrawCollectible removes a catalog-minted instance from its holder
// (state -> withdrawn). Escrowed instances are refused (a task reward
// references them); the former holder is merged into the event draft's
// recipients inside the transaction.
func (store CollectibleStore) WithdrawCollectible(ctx context.Context, command assets.WithdrawCollectibleStoreCommand) assets.WithdrawResult {
	tx, err := store.db.Begin(ctx)
	if err != nil {
		return assets.WithdrawRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "begin withdraw collectible transaction failed")}
	}
	defer func() { _ = tx.Rollback(ctx) }()

	collectibleResult := lockCollectible(ctx, tx, command.CollectibleID)
	collectible, collectibleMatched := collectibleResult.(collectibleParsed)
	if !collectibleMatched {
		return assets.WithdrawRejected{Reason: collectibleResult.(collectibleParseRejected).reason}
	}
	if _, fromCatalog := collectible.value.Catalog.(assets.FromCatalog); !fromCatalog {
		return assets.WithdrawRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "only catalog collectibles can be withdrawn")}
	}
	if collectible.value.State == assets.CollectibleStateEscrowed {
		return assets.WithdrawRejected{Reason: core.NewDomainError(core.ErrorCodeConflict, "collectible is held as a task reward and cannot be withdrawn")}
	}
	if collectible.value.State != assets.CollectibleStateMinted {
		return assets.WithdrawRejected{Reason: core.NewDomainError(core.ErrorCodeConflict, "collectible is not withdrawable in its current state")}
	}

	if _, err := tx.Exec(ctx, "update collectibles set state = 'withdrawn', state_recorded_at = now() where id = $1", command.CollectibleID.String()); err != nil {
		return assets.WithdrawRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "withdraw collectible failed")}
	}

	draft := command.Draft
	if collectible.value.OwnerKind == assets.CollectibleOwnerKindUser {
		holderResult := core.ParseUserID(collectible.value.OwnerID)
		holder, holderMatched := holderResult.(core.UserIDCreated)
		if !holderMatched {
			return assets.WithdrawRejected{Reason: holderResult.(core.UserIDRejected).Reason}
		}
		draft = draft.WithRecipients(holder.Value)
	}
	if err := recordEventDraftInTx(ctx, tx, draft); err != nil {
		return assets.WithdrawRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "record collectible withdrawn event failed")}
	}
	if err := tx.Commit(ctx); err != nil {
		return assets.WithdrawRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "commit withdraw collectible failed")}
	}
	withdrawn := collectible.value
	withdrawn.State = assets.CollectibleStateWithdrawn
	return assets.CollectibleWithdrawn{Value: withdrawn, RecordedEvents: []event.Draft{draft}}
}

// ReleaseCollectible returns a withdrawn instance to its holder's inventory
// (state -> minted, owner unchanged). The catalog entry row is locked first
// so the uniqueness re-check serializes against AwardFromCatalog: a unique
// whose live slot was re-minted while this instance was withdrawn is refused
// with a conflict, and the partial unique index on live instances backstops
// the check (a unique violation on the state flip is translated into the
// same conflict). The holder is merged into the event draft's recipients
// inside the transaction.
func (store CollectibleStore) ReleaseCollectible(ctx context.Context, command assets.ReleaseCollectibleStoreCommand) assets.ReleaseResult {
	tx, err := store.db.Begin(ctx)
	if err != nil {
		return assets.ReleaseRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "begin release collectible transaction failed")}
	}
	defer func() { _ = tx.Rollback(ctx) }()

	collectibleResult := lockCollectible(ctx, tx, command.CollectibleID)
	collectible, collectibleMatched := collectibleResult.(collectibleParsed)
	if !collectibleMatched {
		return assets.ReleaseRejected{Reason: collectibleResult.(collectibleParseRejected).reason}
	}
	if collectible.value.State != assets.CollectibleStateWithdrawn {
		return assets.ReleaseRejected{Reason: core.NewDomainError(core.ErrorCodeConflict, "only withdrawn collectibles can be released")}
	}
	fromCatalog, catalogMatched := collectible.value.Catalog.(assets.FromCatalog)
	if !catalogMatched {
		// Unreachable through the current lifecycle (only catalog instances
		// can be withdrawn), kept as a guard for pre-provenance data.
		return assets.ReleaseRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "collectible has no catalog entry to validate against")}
	}

	// Lock the entry row so the uniqueness re-check serializes with awards.
	entryResult := store.findCatalogEntry(ctx, tx, fromCatalog.Slug, true)
	entry, entryMatched := entryResult.(catalogEntryParsed)
	if !entryMatched {
		return assets.ReleaseRejected{Reason: entryResult.(catalogEntryParseRejected).reason}
	}
	if entry.value.Kind == assets.CollectibleKindUnique {
		var liveExists bool
		if err := tx.QueryRow(ctx, `
			select exists(select 1 from collectibles where catalog_slug = $1 and state <> 'withdrawn')
		`, fromCatalog.Slug.String()).Scan(&liveExists); err != nil {
			return assets.ReleaseRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "check unique catalog instance failed")}
		}
		if liveExists {
			return assets.ReleaseRejected{Reason: core.NewDomainError(core.ErrorCodeConflict, "another live instance of this unique collectible exists")}
		}
	}

	if _, err := tx.Exec(ctx, "update collectibles set state = 'minted', state_recorded_at = now() where id = $1", command.CollectibleID.String()); err != nil {
		if isUniqueViolation(err) {
			return assets.ReleaseRejected{Reason: core.NewDomainError(core.ErrorCodeConflict, "another live instance of this unique collectible exists")}
		}
		return assets.ReleaseRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "release collectible failed")}
	}

	draft := command.Draft
	if collectible.value.OwnerKind == assets.CollectibleOwnerKindUser {
		holderResult := core.ParseUserID(collectible.value.OwnerID)
		holder, holderMatched := holderResult.(core.UserIDCreated)
		if !holderMatched {
			return assets.ReleaseRejected{Reason: holderResult.(core.UserIDRejected).Reason}
		}
		draft = draft.WithRecipients(holder.Value)
	}
	if err := recordEventDraftInTx(ctx, tx, draft); err != nil {
		return assets.ReleaseRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "record collectible released event failed")}
	}
	if err := tx.Commit(ctx); err != nil {
		return assets.ReleaseRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "commit release collectible failed")}
	}
	released := collectible.value
	released.State = assets.CollectibleStateMinted
	return assets.CollectibleReleased{Value: released, RecordedEvents: []event.Draft{draft}}
}

// DeleteWithdrawnCollectible hard-deletes a withdrawn instance after
// verifying no task escrow references it.
func (store CollectibleStore) DeleteWithdrawnCollectible(ctx context.Context, collectibleID core.CollectibleID) assets.DeleteCollectibleResult {
	tx, err := store.db.Begin(ctx)
	if err != nil {
		return assets.DeleteCollectibleRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "begin delete collectible transaction failed")}
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var escrowed bool
	if err := tx.QueryRow(ctx, "select exists(select 1 from task_fund_collectibles where collectible_id = $1)", collectibleID.String()).Scan(&escrowed); err != nil {
		return assets.DeleteCollectibleRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "check collectible escrow reference failed")}
	}
	if escrowed {
		return assets.DeleteCollectibleRejected{Reason: core.NewDomainError(core.ErrorCodeConflict, "collectible is referenced by a task reward escrow")}
	}
	tag, err := tx.Exec(ctx, "delete from collectibles where id = $1 and state = 'withdrawn'", collectibleID.String())
	if err != nil {
		return assets.DeleteCollectibleRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "delete collectible failed")}
	}
	if tag != 1 {
		return assets.DeleteCollectibleRejected{Reason: core.NewDomainError(core.ErrorCodeConflict, "only withdrawn collectibles can be deleted")}
	}
	if err := tx.Commit(ctx); err != nil {
		return assets.DeleteCollectibleRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "commit delete collectible failed")}
	}
	return assets.CollectibleDeleted{}
}

// TransferCollectibleToOrganization donates a user-owned collectible to an
// organization: ownership, availability, and transfer policy are enforced
// here; a within-organization policy additionally requires the collectible's
// organization to be the receiving one and the sender to be an active
// member.
func (store CollectibleStore) TransferCollectibleToOrganization(ctx context.Context, command assets.TransferToOrganizationStoreCommand) assets.GiftResult {
	tx, err := store.db.Begin(ctx)
	if err != nil {
		return assets.GiftRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "begin transfer collectible transaction failed")}
	}
	defer func() { _ = tx.Rollback(ctx) }()

	collectibleResult := lockCollectible(ctx, tx, command.CollectibleID)
	collectible, collectibleMatched := collectibleResult.(collectibleParsed)
	if !collectibleMatched {
		return assets.GiftRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "collectible is not available to transfer")}
	}
	if collectible.value.OwnerKind != assets.CollectibleOwnerKindUser || collectible.value.OwnerID != command.FromUserID.String() {
		return assets.GiftRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "collectible is not available to transfer")}
	}
	if collectible.value.State != assets.CollectibleStateMinted {
		return assets.GiftRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "collectible is not available to transfer")}
	}
	if denied, matched := assets.AllowsTransfer(collectible.value.Policy).(assets.RewardDenied); matched {
		return assets.GiftRejected{Reason: denied.Reason}
	}
	if collectible.value.Policy == assets.TransferPolicyTransferableWithinOrg {
		if collectible.value.OrganizationID != command.OrganizationID.String() {
			return assets.GiftRejected{Reason: core.NewDomainError(core.ErrorCodePermissionDenied, "within-organization collectible can only be donated to its own organization")}
		}
		if reason, rejected := requireSharedCollectibleOrganization(ctx, tx, collectible.value, command.FromUserID, command.FromUserID); rejected {
			return assets.GiftRejected{Reason: reason}
		}
	}

	if _, err := tx.Exec(ctx, "update collectibles set owner_user_id = $2, owner_kind = 'organization', organization_id = $2, state_recorded_at = now() where id = $1", command.CollectibleID.String(), command.OrganizationID.String()); err != nil {
		return assets.GiftRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "transfer collectible failed")}
	}
	if err := recordEventDraftInTx(ctx, tx, command.Draft); err != nil {
		return assets.GiftRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "record collectible transfer event failed")}
	}
	if err := tx.Commit(ctx); err != nil {
		return assets.GiftRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "commit transfer collectible failed")}
	}
	transferred := collectible.value
	transferred.OwnerKind = assets.CollectibleOwnerKindOrganization
	transferred.OwnerID = command.OrganizationID.String()
	transferred.OrganizationID = command.OrganizationID.String()
	return assets.CollectibleGifted{Value: transferred, RecordedEvents: []event.Draft{command.Draft}}
}

// TransferCollectibleFromOrganization moves an organization-owned
// collectible to a user, authorized in-transaction by the acting member's
// manage_collectibles permission.
func (store CollectibleStore) TransferCollectibleFromOrganization(ctx context.Context, command assets.TransferFromOrganizationStoreCommand) assets.GiftResult {
	tx, err := store.db.Begin(ctx)
	if err != nil {
		return assets.GiftRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "begin transfer collectible transaction failed")}
	}
	defer func() { _ = tx.Rollback(ctx) }()

	check := organizationMemberPermission(ctx, tx, command.OrganizationID.String(), command.ActingUserID, org.PermissionManageCollectibles)
	if _, denied := check.(org.PermissionDenied); denied {
		return assets.GiftRejected{Reason: core.NewDomainError(core.ErrorCodePermissionDenied, "transferring organization collectibles requires the manage_collectibles permission")}
	}

	collectibleResult := lockCollectible(ctx, tx, command.CollectibleID)
	collectible, collectibleMatched := collectibleResult.(collectibleParsed)
	if !collectibleMatched {
		return assets.GiftRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "collectible is not available to transfer")}
	}
	if collectible.value.OwnerKind != assets.CollectibleOwnerKindOrganization || collectible.value.OrganizationID != command.OrganizationID.String() {
		return assets.GiftRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "collectible does not belong to this organization")}
	}
	if collectible.value.State != assets.CollectibleStateMinted {
		return assets.GiftRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "collectible is not available to transfer")}
	}
	if denied, matched := assets.AllowsTransfer(collectible.value.Policy).(assets.RewardDenied); matched {
		return assets.GiftRejected{Reason: denied.Reason}
	}

	// Mirror AwardOrganizationCollectible: leaving the organization clears
	// the organization scope column.
	if _, err := tx.Exec(ctx, "update collectibles set owner_user_id = $2, owner_kind = 'user', organization_id = null, state_recorded_at = now() where id = $1", command.CollectibleID.String(), command.ToUserID.String()); err != nil {
		return assets.GiftRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "transfer collectible failed")}
	}
	if err := recordEventDraftInTx(ctx, tx, command.Draft); err != nil {
		return assets.GiftRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "record collectible transfer event failed")}
	}
	if err := tx.Commit(ctx); err != nil {
		return assets.GiftRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "commit transfer collectible failed")}
	}
	transferred := collectible.value
	transferred.OwnerKind = assets.CollectibleOwnerKindUser
	transferred.OwnerID = command.ToUserID.String()
	transferred.OrganizationID = ""
	return assets.CollectibleGifted{Value: transferred, RecordedEvents: []event.Draft{command.Draft}}
}
