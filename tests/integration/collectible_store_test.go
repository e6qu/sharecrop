//go:build integration

package integration_test

import (
	"context"
	"testing"

	"github.com/e6qu/sharecrop/internal/assets"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/db"
	"github.com/e6qu/sharecrop/internal/ledger"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestMultipleCollectiblesAwardedOnAccept(t *testing.T) {
	pool := newPool(t)
	collectibleStore := db.NewCollectibleStore(pool)
	ledgerStore := db.NewLedgerStore(pool)

	owner := createUser(t, pool, "multi-collectible-owner")
	worker := createUser(t, pool, "multi-collectible-worker")

	first := mintIntegrationCollectible(t, collectibleStore, owner, "First medal")
	second := mintIntegrationCollectible(t, collectibleStore, owner, "Second medal")

	taskID := insertCollectibleTask(t, pool, owner, "draft")

	fundCollectible(t, collectibleStore, owner, taskID, first)
	fundCollectible(t, collectibleStore, owner, taskID, second)

	// Both collectibles are escrowed and held against the same task.
	if escrowedCount := collectibleStateCount(t, pool, owner, "escrowed"); escrowedCount != 2 {
		t.Fatalf("escrowed collectible count = %d, want 2", escrowedCount)
	}
	if held := heldRewardCount(t, pool, taskID); held != 2 {
		t.Fatalf("held reward count = %d, want 2", held)
	}

	setTaskState(t, pool, taskID, "open")
	submissionID := insertSubmission(t, pool, taskID, worker)

	result := ledgerStore.AcceptSubmission(context.Background(), acceptCommand(t, owner, taskID, submissionID, "accept-multi-"+submissionID.String()))
	accepted, matched := result.(ledger.SubmissionAccepted)
	if !matched {
		t.Fatalf("accept result = %T (%s), want SubmissionAccepted", result, result.(ledger.AcceptRejected).Reason.Description())
	}
	payout, paid := accepted.Payout.(ledger.CollectiblePayout)
	if !paid {
		t.Fatalf("payout = %T, want CollectiblePayout", accepted.Payout)
	}
	if len(payout.CollectibleIDs) != 2 {
		t.Fatalf("awarded collectible count = %d, want 2", len(payout.CollectibleIDs))
	}
	if !containsCollectibleID(payout.CollectibleIDs, first) || !containsCollectibleID(payout.CollectibleIDs, second) {
		t.Fatalf("awarded collectibles = %v, want both %s and %s", payout.CollectibleIDs, first, second)
	}

	// Both collectibles now belong to the worker and are marked awarded.
	if ownerHeld := collectibleStateCount(t, pool, owner, "awarded"); ownerHeld != 0 {
		t.Fatalf("owner awarded collectible count = %d, want 0", ownerHeld)
	}
	if workerAwarded := collectibleStateCount(t, pool, worker, "awarded"); workerAwarded != 2 {
		t.Fatalf("worker awarded collectible count = %d, want 2", workerAwarded)
	}
}

func TestFundCollectibleRewardTransitionsRewardKind(t *testing.T) {
	pool := newPool(t)
	collectibleStore := db.NewCollectibleStore(pool)

	owner := createUser(t, pool, "collectible-fund-none")

	noneTaskID := insertTaskWithRewardKind(t, pool, owner, "draft", "none")
	noneCollectible := mintIntegrationCollectible(t, collectibleStore, owner, "None to collectible medal")
	fundCollectible(t, collectibleStore, owner, noneTaskID, noneCollectible)
	rewardKind, _ := taskRewardRow(t, pool, noneTaskID)
	if rewardKind != "collectible" {
		t.Fatalf("reward kind after funding a none-reward task with a collectible = %q, want collectible", rewardKind)
	}

	creditTaskID := insertTask(t, pool, owner, "draft", 15)
	creditCollectible := mintIntegrationCollectible(t, collectibleStore, owner, "Credit to bundle medal")
	fundCollectible(t, collectibleStore, owner, creditTaskID, creditCollectible)
	bundleRewardKind, bundleRewardAmount := taskRewardRow(t, pool, creditTaskID)
	if bundleRewardKind != "bundle" || bundleRewardAmount != 15 {
		t.Fatalf("reward row after funding a credit-reward task with a collectible = (%q, %d), want (bundle, 15)", bundleRewardKind, bundleRewardAmount)
	}
}

func TestMultipleCollectibleRefundReturnsAll(t *testing.T) {
	pool := newPool(t)
	collectibleStore := db.NewCollectibleStore(pool)

	owner := createUser(t, pool, "multi-refund-owner")

	first := mintIntegrationCollectible(t, collectibleStore, owner, "First refund medal")
	second := mintIntegrationCollectible(t, collectibleStore, owner, "Second refund medal")

	taskID := insertCollectibleTask(t, pool, owner, "draft")
	fundCollectible(t, collectibleStore, owner, taskID, first)
	fundCollectible(t, collectibleStore, owner, taskID, second)

	result := collectibleStore.RefundCollectibleReward(context.Background(), assets.RefundRewardStoreCommand{
		RequesterUserID: owner,
		TaskID:          taskID,
	})
	refunded, matched := result.(assets.RewardRefunded)
	if !matched {
		t.Fatalf("refund result = %T (%s), want RewardRefunded", result, result.(assets.RefundRewardRejected).Reason.Description())
	}
	if len(refunded.Values) != 2 {
		t.Fatalf("refunded collectible count = %d, want 2", len(refunded.Values))
	}

	if minted := collectibleStateCount(t, pool, owner, "minted"); minted != 2 {
		t.Fatalf("owner minted collectible count = %d, want 2", minted)
	}
	if held := heldRewardCount(t, pool, taskID); held != 0 {
		t.Fatalf("held reward count after refund = %d, want 0", held)
	}
}

func TestWithinOrganizationCollectibleTipRequiresSharedActiveMembership(t *testing.T) {
	pool := newPool(t)
	collectibleStore := db.NewCollectibleStore(pool)

	owner := createUser(t, pool, "org-tip-owner")
	worker := createUser(t, pool, "org-tip-worker")
	outsider := createUser(t, pool, "org-tip-outsider")
	organizationID := insertIntegrationOrganization(t, pool, owner)
	insertIntegrationMembership(t, pool, organizationID, owner)
	insertIntegrationMembership(t, pool, organizationID, worker)

	collectibleID := mintIntegrationScopedCollectible(t, collectibleStore, owner, organizationID, "Org medal")

	result := collectibleStore.GiftCollectible(context.Background(), assets.GiftStoreCommand{
		FromUserID:    owner,
		ToUserID:      worker,
		CollectibleID: collectibleID,
	})
	gifted, matched := result.(assets.CollectibleGifted)
	if !matched {
		t.Fatalf("gift result = %T (%s), want CollectibleGifted", result, result.(assets.GiftRejected).Reason.Description())
	}
	if gifted.Value.OwnerID != worker.String() {
		t.Fatalf("gift owner = %q, want worker", gifted.Value.OwnerID)
	}

	deniedCollectibleID := mintIntegrationScopedCollectible(t, collectibleStore, owner, organizationID, "Org medal denied")
	denied := collectibleStore.GiftCollectible(context.Background(), assets.GiftStoreCommand{
		FromUserID:    owner,
		ToUserID:      outsider,
		CollectibleID: deniedCollectibleID,
	})
	if _, matched := denied.(assets.GiftRejected); !matched {
		t.Fatalf("gift to outsider = %T, want GiftRejected", denied)
	}
}

func mintIntegrationCollectible(t *testing.T, store db.CollectibleStore, owner core.UserID, name string) core.CollectibleID {
	t.Helper()
	return mintIntegrationCollectibleWithPolicy(t, store, owner, "", name, assets.TransferPolicyNonTransferableExceptPayout)
}

func mintIntegrationScopedCollectible(t *testing.T, store db.CollectibleStore, owner core.UserID, organizationID core.OrganizationID, name string) core.CollectibleID {
	t.Helper()
	return mintIntegrationCollectibleWithPolicy(t, store, owner, organizationID.String(), name, assets.TransferPolicyTransferableWithinOrg)
}

func mintIntegrationCollectibleWithPolicy(t *testing.T, store db.CollectibleStore, owner core.UserID, organizationID string, name string, policy assets.TransferPolicy) core.CollectibleID {
	t.Helper()
	idCreated := core.NewCollectibleID().(core.CollectibleIDCreated)
	nameAccepted := assets.NewCollectibleName(name).(assets.CollectibleNameAccepted)
	collectible := assets.Collectible{
		ID:             idCreated.Value,
		Name:           nameAccepted.Value,
		Kind:           assets.CollectibleKindBadge,
		State:          assets.CollectibleStateMinted,
		Policy:         policy,
		OwnerKind:      assets.CollectibleOwnerKindUser,
		OwnerID:        owner.String(),
		OrganizationID: organizationID,
	}
	if _, matched := store.CreateCollectible(context.Background(), collectible).(assets.CreateStoreAccepted); !matched {
		t.Fatalf("create collectible rejected")
	}
	return idCreated.Value
}

func insertIntegrationOrganization(t *testing.T, pool *pgxpool.Pool, owner core.UserID) core.OrganizationID {
	t.Helper()
	organizationID := core.NewOrganizationID().(core.OrganizationIDCreated)
	_, err := pool.Exec(context.Background(), `
		insert into organizations (id, name, created_by_user_id)
		values ($1, 'Integration Org', $2)
	`, organizationID.Value.String(), owner.String())
	if err != nil {
		t.Fatalf("insert organization: %v", err)
	}
	return organizationID.Value
}

func insertIntegrationMembership(t *testing.T, pool *pgxpool.Pool, organizationID core.OrganizationID, userID core.UserID) {
	t.Helper()
	membershipID := core.NewOrganizationMembershipID().(core.OrganizationMembershipIDCreated)
	_, err := pool.Exec(context.Background(), `
		insert into organization_memberships (id, organization_id, user_id, status)
		values ($1, $2, $3, 'active')
	`, membershipID.Value.String(), organizationID.String(), userID.String())
	if err != nil {
		t.Fatalf("insert organization membership: %v", err)
	}
}

func fundCollectible(t *testing.T, store db.CollectibleStore, owner core.UserID, taskID core.TaskID, collectibleID core.CollectibleID) {
	t.Helper()
	result := store.FundCollectibleReward(context.Background(), assets.FundRewardStoreCommand{
		FunderUserID:  owner,
		TaskID:        taskID,
		CollectibleID: collectibleID,
	})
	if _, matched := result.(assets.RewardFunded); !matched {
		t.Fatalf("fund collectible rejected: %s", result.(assets.FundRewardRejected).Reason.Description())
	}
}

func insertCollectibleTask(t *testing.T, pool *pgxpool.Pool, owner core.UserID, state string) core.TaskID {
	t.Helper()
	taskID := newTaskID(t)
	_, err := pool.Exec(context.Background(), `
		insert into tasks (id, owner_kind, user_id, title, description, reward_kind, reward_credit_amount, state, response_schema_json, data_payload_kind, created_by_user_id)
		values ($1, 'user', $2, 'Collectible task', 'Collectible task description', 'collectible', null, $3, '{}'::jsonb, 'none', $2)
	`, taskID.String(), owner.String(), state)
	if err != nil {
		t.Fatalf("insert collectible task: %v", err)
	}
	return taskID
}

func collectibleStateCount(t *testing.T, pool *pgxpool.Pool, owner core.UserID, state string) int {
	t.Helper()
	var count int
	if err := pool.QueryRow(context.Background(), "select count(*) from collectibles where owner_user_id = $1 and state = $2", owner.String(), state).Scan(&count); err != nil {
		t.Fatalf("count collectibles: %v", err)
	}
	return count
}

func heldRewardCount(t *testing.T, pool *pgxpool.Pool, taskID core.TaskID) int {
	t.Helper()
	var count int
	if err := pool.QueryRow(context.Background(), "select count(*) from task_collectible_rewards where task_id = $1 and state = 'held'", taskID.String()).Scan(&count); err != nil {
		t.Fatalf("count held rewards: %v", err)
	}
	return count
}

func containsCollectibleID(ids []core.CollectibleID, target core.CollectibleID) bool {
	for index := range ids {
		if ids[index] == target {
			return true
		}
	}
	return false
}
