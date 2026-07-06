package wasmdemo

import (
	"context"
	"testing"

	"github.com/e6qu/sharecrop/internal/assets"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/ledger"
	"github.com/e6qu/sharecrop/internal/org"
)

func testCollectibleName(t *testing.T, raw string) assets.CollectibleName {
	t.Helper()
	result := assets.NewCollectibleName(raw)
	accepted, matched := result.(assets.CollectibleNameAccepted)
	if !matched {
		t.Fatalf("new collectible name %q failed", raw)
	}
	return accepted.Value
}

func newAssetsTestEnv(t *testing.T) (assets.Service, BrowserStorage, InteractionIDSource) {
	t.Helper()
	storage := newTestBrowserStorage()
	ids := &counterLedgerIDs{}
	return assets.NewService(NewAssetBrowserStore(storage, ids)), storage, ids
}

// activateOrgMembership writes a minimal active membership record directly,
// bypassing org.Service, so asset-store tests can set up the org-membership
// preconditions isActiveOrgMember checks without depending on org.Service.
func activateOrgMembership(t *testing.T, storage BrowserStorage, organizationID string, userID string) {
	t.Helper()
	membershipID := "membership-" + organizationID + "-" + userID
	membership := storedMembership{ID: membershipID, OrganizationID: organizationID, UserID: userID, Status: org.MembershipStatusActive.String(), Roles: []string{"member"}}
	if !putStoredMembershipJSON(storage, orgMembershipKey(membershipID), membership) {
		t.Fatalf("write active membership failed")
	}
	if !putStorageString(storage, orgActiveMembershipKey(organizationID, userID), membershipID) {
		t.Fatalf("write active membership pointer failed")
	}
}

func TestAssetBrowserStoreMintAndList(t *testing.T) {
	service, _, _ := newAssetsTestEnv(t)
	ctx := context.Background()
	owner := testUserID(t, "owner")

	mintResult := service.Mint(ctx, assets.CollectibleOwnerKindUser, owner.String(), "", testCollectibleName(t, "Golden Badge"), assets.CollectibleKindBadge, assets.TransferPolicyTransferableBetweenUsers, "art-1")
	minted, matched := mintResult.(assets.CollectibleMinted)
	if !matched {
		t.Fatalf("mint: want CollectibleMinted, got %#v", mintResult)
	}
	if minted.Value.State != assets.CollectibleStateMinted {
		t.Fatalf("minted state = %v, want minted", minted.Value.State)
	}

	listResult := service.ListCollectibles(ctx, owner, testPage(t, 10, 0))
	listed, matched := listResult.(assets.CollectiblesListed)
	if !matched {
		t.Fatalf("list: want CollectiblesListed, got %#v", listResult)
	}
	if len(listed.Values) != 1 || listed.Values[0].ID != minted.Value.ID {
		t.Fatalf("listed collectibles = %+v, want just %v", listed.Values, minted.Value.ID)
	}
}

func TestAssetBrowserStoreFundRewardFlipsTaskRewardKind(t *testing.T) {
	service, storage, _ := newAssetsTestEnv(t)
	ctx := context.Background()
	owner := testUserID(t, "owner")

	minted := service.Mint(ctx, assets.CollectibleOwnerKindUser, owner.String(), "", testCollectibleName(t, "Golden Badge"), assets.CollectibleKindBadge, assets.TransferPolicyTransferableBetweenUsers, "art-1").(assets.CollectibleMinted)

	taskID := core.NewTaskID().(core.TaskIDCreated).Value
	seedDraftTask(t, storage, taskID.String(), owner.String(), "none", 0)

	fundResult := service.FundReward(ctx, owner, taskID, minted.Value.ID)
	funded, matched := fundResult.(assets.RewardFunded)
	if !matched {
		t.Fatalf("fund reward: want RewardFunded, got %#v", fundResult)
	}
	if funded.Value.State != assets.CollectibleStateEscrowed {
		t.Fatalf("funded collectible state = %v, want escrowed", funded.Value.State)
	}

	record, found, _ := loadStoredTaskRecord(storage, taskID.String())
	if !found || record.RewardKind != "collectible" {
		t.Fatalf("task after funding = %+v, want reward_kind=collectible", record)
	}
}

func TestAssetBrowserStoreFundRewardRejectsNonOwner(t *testing.T) {
	service, storage, _ := newAssetsTestEnv(t)
	ctx := context.Background()
	owner := testUserID(t, "owner")
	other := testUserID(t, "other")

	minted := service.Mint(ctx, assets.CollectibleOwnerKindUser, owner.String(), "", testCollectibleName(t, "Golden Badge"), assets.CollectibleKindBadge, assets.TransferPolicyTransferableBetweenUsers, "art-1").(assets.CollectibleMinted)
	taskID := core.NewTaskID().(core.TaskIDCreated).Value
	seedDraftTask(t, storage, taskID.String(), owner.String(), "none", 0)

	result := service.FundReward(ctx, other, taskID, minted.Value.ID)
	if _, matched := result.(assets.FundRewardRejected); !matched {
		t.Fatalf("fund with someone else's collectible: want FundRewardRejected, got %#v", result)
	}
}

func TestAssetBrowserStoreRefundRewardReturnsCollectibleAndCancelsTask(t *testing.T) {
	service, storage, _ := newAssetsTestEnv(t)
	ctx := context.Background()
	owner := testUserID(t, "owner")

	minted := service.Mint(ctx, assets.CollectibleOwnerKindUser, owner.String(), "", testCollectibleName(t, "Golden Badge"), assets.CollectibleKindBadge, assets.TransferPolicyTransferableBetweenUsers, "art-1").(assets.CollectibleMinted)
	taskID := core.NewTaskID().(core.TaskIDCreated).Value
	seedDraftTask(t, storage, taskID.String(), owner.String(), "none", 0)
	service.FundReward(ctx, owner, taskID, minted.Value.ID)

	refundResult := service.RefundReward(ctx, owner, taskID)
	refunded, matched := refundResult.(assets.RewardRefunded)
	if !matched {
		t.Fatalf("refund: want RewardRefunded, got %#v", refundResult)
	}
	if len(refunded.Values) != 1 || refunded.Values[0].State != assets.CollectibleStateMinted {
		t.Fatalf("refunded collectibles = %+v, want one minted collectible", refunded.Values)
	}

	record, found, _ := loadStoredTaskRecord(storage, taskID.String())
	if !found || record.State != "cancelled" {
		t.Fatalf("task after refund = %+v, want cancelled", record)
	}
}

func TestAssetBrowserStorePayOutHeldCollectibleRewardMovesOwnerIndex(t *testing.T) {
	storage := newTestBrowserStorage()
	ids := &counterLedgerIDs{}
	store := NewAssetBrowserStore(storage, ids)
	service := assets.NewService(store)
	ctx := context.Background()
	requester := testUserID(t, "requester")
	worker := testUserID(t, "worker")

	minted := service.Mint(ctx, assets.CollectibleOwnerKindUser, requester.String(), "", testCollectibleName(t, "Golden Badge"), assets.CollectibleKindBadge, assets.TransferPolicyTransferableBetweenUsers, "art-1").(assets.CollectibleMinted)
	taskID := core.NewTaskID().(core.TaskIDCreated).Value
	seedDraftTask(t, storage, taskID.String(), requester.String(), "none", 0)
	service.FundReward(ctx, requester, taskID, minted.Value.ID)

	awarded, err := store.payOutHeldCollectibleReward(taskID.String(), worker.String())
	if err != nil || len(awarded) != 1 || awarded[0] != minted.Value.ID {
		t.Fatalf("pay out held collectible reward = %+v, %+v, want [%v], nil", awarded, err, minted.Value.ID)
	}

	requesterListResult := service.ListCollectibles(ctx, requester, testPage(t, 10, 0)).(assets.CollectiblesListed)
	if len(requesterListResult.Values) != 0 {
		t.Fatalf("requester's collectibles after payout = %+v, want none", requesterListResult.Values)
	}
	workerListResult := service.ListCollectibles(ctx, worker, testPage(t, 10, 0)).(assets.CollectiblesListed)
	if len(workerListResult.Values) != 1 || workerListResult.Values[0].ID != minted.Value.ID {
		t.Fatalf("worker's collectibles after payout = %+v, want just %v", workerListResult.Values, minted.Value.ID)
	}
}

func TestAssetBrowserStoreCreditRefundOnBundleTaskReleasesCollectible(t *testing.T) {
	storage := newTestBrowserStorage()
	ids := &counterLedgerIDs{}
	assetService := assets.NewService(NewAssetBrowserStore(storage, ids))
	ledgerService := ledger.NewService(NewLedgerBrowserStore(storage, ids))
	ctx := context.Background()
	owner := testUserID(t, "owner")

	minted := assetService.Mint(ctx, assets.CollectibleOwnerKindUser, owner.String(), "", testCollectibleName(t, "Golden Badge"), assets.CollectibleKindBadge, assets.TransferPolicyTransferableBetweenUsers, "art-1").(assets.CollectibleMinted)
	taskID := core.NewTaskID().(core.TaskIDCreated).Value
	seedDraftTask(t, storage, taskID.String(), owner.String(), "none", 0)
	NewAuthBrowserStore(storage, ids).insertSignupGrant("user", owner.String())

	assetService.FundReward(ctx, owner, taskID, minted.Value.ID)
	fundResult := ledgerService.FundTask(ctx, owner, taskID, testCreditAmount(t, 20), testIdempotencyKey(t, "fund-1"))
	if _, matched := fundResult.(ledger.TaskFunded); !matched {
		t.Fatalf("fund credit half of bundle: want TaskFunded, got %#v", fundResult)
	}
	record, _, _ := loadStoredTaskRecord(storage, taskID.String())
	if record.RewardKind != "bundle" {
		t.Fatalf("task reward kind = %q, want bundle", record.RewardKind)
	}

	// A credit refund on a bundle task must release the collectible half too,
	// not strand it in escrow.
	refundResult := ledgerService.RefundTask(ctx, owner, taskID, testIdempotencyKey(t, "refund-1"))
	if _, matched := refundResult.(ledger.TaskRefunded); !matched {
		t.Fatalf("refund credit half of bundle: want TaskRefunded, got %#v", refundResult)
	}

	collectibleResult := assetService.ListCollectibles(ctx, owner, testPage(t, 10, 0))
	listed := collectibleResult.(assets.CollectiblesListed)
	if len(listed.Values) != 1 || listed.Values[0].State != assets.CollectibleStateMinted {
		t.Fatalf("collectible after bundle credit refund = %+v, want one minted collectible (released)", listed.Values)
	}
}

func TestAssetBrowserStoreGiftCollectible(t *testing.T) {
	service, _, _ := newAssetsTestEnv(t)
	ctx := context.Background()
	from := testUserID(t, "from")
	to := testUserID(t, "to")

	minted := service.Mint(ctx, assets.CollectibleOwnerKindUser, from.String(), "", testCollectibleName(t, "Golden Badge"), assets.CollectibleKindBadge, assets.TransferPolicyTransferableBetweenUsers, "art-1").(assets.CollectibleMinted)

	giftResult := service.GiftCollectible(ctx, from, to, minted.Value.ID)
	gifted, matched := giftResult.(assets.CollectibleGifted)
	if !matched {
		t.Fatalf("gift: want CollectibleGifted, got %#v", giftResult)
	}
	if gifted.Value.OwnerID != to.String() {
		t.Fatalf("gifted owner = %q, want %q", gifted.Value.OwnerID, to.String())
	}

	fromListResult := service.ListCollectibles(ctx, from, testPage(t, 10, 0)).(assets.CollectiblesListed)
	if len(fromListResult.Values) != 0 {
		t.Fatalf("sender's collectibles after gift = %+v, want none", fromListResult.Values)
	}
	toListResult := service.ListCollectibles(ctx, to, testPage(t, 10, 0)).(assets.CollectiblesListed)
	if len(toListResult.Values) != 1 || toListResult.Values[0].ID != minted.Value.ID {
		t.Fatalf("recipient's collectibles after gift = %+v, want just %v", toListResult.Values, minted.Value.ID)
	}
}

func TestAssetBrowserStoreAwardOrganizationCollectible(t *testing.T) {
	service, storage, _ := newAssetsTestEnv(t)
	ctx := context.Background()
	organizationID := testOrganizationID(t, "award-org")
	recipient := testUserID(t, "recipient")

	minted := service.Mint(ctx, assets.CollectibleOwnerKindOrganization, "", organizationID.String(), testCollectibleName(t, "Org Trophy"), assets.CollectibleKindUnique, assets.TransferPolicyTransferableWithinOrg, "art-1").(assets.CollectibleMinted)

	rejectedResult := service.AwardOrganizationCollectible(ctx, organizationID, minted.Value.ID, recipient)
	if _, matched := rejectedResult.(assets.GiftRejected); !matched {
		t.Fatalf("award to non-member: want GiftRejected, got %#v", rejectedResult)
	}

	activateOrgMembership(t, storage, organizationID.String(), recipient.String())
	awardResult := service.AwardOrganizationCollectible(ctx, organizationID, minted.Value.ID, recipient)
	awarded, matched := awardResult.(assets.CollectibleGifted)
	if !matched {
		t.Fatalf("award to active member: want CollectibleGifted, got %#v", awardResult)
	}
	if awarded.Value.OwnerID != recipient.String() {
		t.Fatalf("awarded owner = %q, want %q", awarded.Value.OwnerID, recipient.String())
	}

	recipientListResult := service.ListCollectibles(ctx, recipient, testPage(t, 10, 0)).(assets.CollectiblesListed)
	if len(recipientListResult.Values) != 1 || recipientListResult.Values[0].ID != minted.Value.ID {
		t.Fatalf("recipient's collectibles after award = %+v, want just %v", recipientListResult.Values, minted.Value.ID)
	}
}

func TestAssetBrowserStoreGiftWithinOrgRequiresBothActiveMembers(t *testing.T) {
	service, storage, _ := newAssetsTestEnv(t)
	ctx := context.Background()
	organizationID := testOrganizationID(t, "gift-org")
	from := testUserID(t, "from")
	to := testUserID(t, "to")

	minted := service.Mint(ctx, assets.CollectibleOwnerKindUser, from.String(), organizationID.String(), testCollectibleName(t, "Team Badge"), assets.CollectibleKindBadge, assets.TransferPolicyTransferableWithinOrg, "art-1").(assets.CollectibleMinted)

	// Neither party is an org member yet: rejected.
	rejectedResult := service.GiftCollectible(ctx, from, to, minted.Value.ID)
	if _, matched := rejectedResult.(assets.GiftRejected); !matched {
		t.Fatalf("gift with no org members: want GiftRejected, got %#v", rejectedResult)
	}

	activateOrgMembership(t, storage, organizationID.String(), from.String())
	stillRejectedResult := service.GiftCollectible(ctx, from, to, minted.Value.ID)
	if _, matched := stillRejectedResult.(assets.GiftRejected); !matched {
		t.Fatalf("gift with only sender as org member: want GiftRejected, got %#v", stillRejectedResult)
	}

	activateOrgMembership(t, storage, organizationID.String(), to.String())
	acceptedResult := service.GiftCollectible(ctx, from, to, minted.Value.ID)
	if _, matched := acceptedResult.(assets.CollectibleGifted); !matched {
		t.Fatalf("gift with both parties as org members: want CollectibleGifted, got %#v", acceptedResult)
	}
}

func TestAssetBrowserStoreGiftRejectsNonTransferablePolicy(t *testing.T) {
	service, _, _ := newAssetsTestEnv(t)
	ctx := context.Background()
	from := testUserID(t, "from")
	to := testUserID(t, "to")

	minted := service.Mint(ctx, assets.CollectibleOwnerKindUser, from.String(), "", testCollectibleName(t, "Payout-Only Badge"), assets.CollectibleKindBadge, assets.TransferPolicyNonTransferableExceptPayout, "art-1").(assets.CollectibleMinted)

	giftResult := service.GiftCollectible(ctx, from, to, minted.Value.ID)
	if _, matched := giftResult.(assets.GiftRejected); !matched {
		t.Fatalf("gift a non-transferable-except-payout collectible: want GiftRejected, got %#v", giftResult)
	}
}
