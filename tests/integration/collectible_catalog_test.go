//go:build integration

package integration_test

import (
	"context"
	"strings"
	"testing"

	"github.com/e6qu/sharecrop/internal/assets"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/db"
	"github.com/e6qu/sharecrop/internal/event"
	"github.com/jackc/pgx/v5/pgxpool"
)

func newAssetService(pool *pgxpool.Pool) assets.Service {
	return assets.NewService(db.NewCollectibleStore(pool), newDBRecorder(pool))
}

func mustCatalogSlug(t *testing.T, raw string) assets.CatalogSlug {
	t.Helper()
	accepted, matched := assets.NewCatalogSlug(raw).(assets.CatalogSlugAccepted)
	if !matched {
		t.Fatalf("catalog slug %q rejected", raw)
	}
	return accepted.Value
}

func uniqueTestSlug(t *testing.T, prefix string) string {
	t.Helper()
	id, matched := core.NewCollectibleID().(core.CollectibleIDCreated)
	if !matched {
		t.Fatalf("id rejected")
	}
	return prefix + "-" + strings.ToLower(id.Value.String())
}

func testCatalogEntry(t *testing.T, slug string, kind assets.CollectibleKind, cap assets.EditionCap) assets.CatalogEntry {
	t.Helper()
	name, matched := assets.NewCollectibleName("Test " + slug).(assets.CollectibleNameAccepted)
	if !matched {
		t.Fatalf("name rejected")
	}
	return assets.CatalogEntry{
		Slug:   mustCatalogSlug(t, slug),
		Name:   name.Value,
		Kind:   kind,
		Policy: assets.TransferPolicyTransferableBetweenUsers,
		Art:    "pumpkin",
		State:  assets.CatalogEntryStateAvailable,
		Cap:    cap,
	}
}

// TestCatalogSeededFromMigration proves the DB-backed catalog carries the 25
// default entries with their kinds and caps.
func TestCatalogSeededFromMigration(t *testing.T) {
	pool := newPool(t)
	service := newAssetService(pool)
	listed, matched := service.ListCatalog(context.Background()).(assets.CatalogListed)
	if !matched {
		t.Fatalf("catalog list rejected")
	}
	bySlug := map[string]assets.CatalogEntry{}
	for _, listing := range listed.Values {
		bySlug[listing.Entry.Slug.String()] = listing.Entry
	}
	for _, slug := range assets.SpriteSlugs {
		if _, found := bySlug[slug]; !found {
			t.Fatalf("catalog is missing seeded entry %q", slug)
		}
	}
	badge := bySlug["harvest-star"]
	if badge.Kind != assets.CollectibleKindBadge {
		t.Fatalf("harvest-star kind = %s, want badge", badge.Kind.String())
	}
	if _, uncapped := badge.Cap.(assets.NoEditionCap); !uncapped {
		t.Fatalf("badge cap = %#v, want uncapped", badge.Cap)
	}
	edition := bySlug["golden-egg"]
	if capped, matched := edition.Cap.(assets.EditionCapOf); !matched || capped.Limit != 100 {
		t.Fatalf("edition cap = %#v, want 100", edition.Cap)
	}
	unique := bySlug["cornucopia"]
	if capped, matched := unique.Cap.(assets.EditionCapOf); !matched || capped.Limit != 1 {
		t.Fatalf("unique cap = %#v, want 1", unique.Cap)
	}
}

// TestCatalogEntryLifecycle walks add -> award -> withdraw -> delete: a
// withdrawn entry can no longer be awarded, cannot be deleted while a live
// instance references it, and deletes cleanly once the instance is withdrawn
// and removed.
func TestCatalogEntryLifecycle(t *testing.T) {
	pool := newPool(t)
	service := newAssetService(pool)
	admin := createUser(t, pool, "catalog-admin")
	holder := createUser(t, pool, "catalog-holder")

	slug := uniqueTestSlug(t, "test-badge")
	added, addedMatched := service.AddCatalogEntry(context.Background(), testCatalogEntry(t, slug, assets.CollectibleKindBadge, assets.NoEditionCap{})).(assets.CatalogMutated)
	if !addedMatched {
		t.Fatalf("add catalog entry rejected")
	}
	if added.Value.Slug.String() != slug {
		t.Fatalf("added slug = %q", added.Value.Slug.String())
	}

	// Duplicate slugs are refused.
	if _, rejected := service.AddCatalogEntry(context.Background(), testCatalogEntry(t, slug, assets.CollectibleKindBadge, assets.NoEditionCap{})).(assets.CatalogMutationRejected); !rejected {
		t.Fatalf("duplicate catalog slug was accepted")
	}
	// Unknown art is refused.
	unknownArt := testCatalogEntry(t, uniqueTestSlug(t, "test-art"), assets.CollectibleKindBadge, assets.NoEditionCap{})
	unknownArt.Art = "not-a-sprite"
	if _, rejected := service.AddCatalogEntry(context.Background(), unknownArt).(assets.CatalogMutationRejected); !rejected {
		t.Fatalf("unknown art slug was accepted")
	}

	awarded, awardMatched := service.AwardFromCatalog(context.Background(), admin, slug, assets.CollectibleOwnerKindUser, holder.String(), "").(assets.CollectibleMinted)
	if !awardMatched {
		t.Fatalf("award from catalog rejected")
	}
	fromCatalog, hasCatalog := awarded.Value.Catalog.(assets.FromCatalog)
	if !hasCatalog || fromCatalog.Slug.String() != slug {
		t.Fatalf("awarded instance catalog ref = %#v", awarded.Value.Catalog)
	}

	// Withdraw the entry: no further awards.
	if _, matched := service.WithdrawCatalogEntry(context.Background(), mustCatalogSlug(t, slug)).(assets.CatalogMutated); !matched {
		t.Fatalf("withdraw catalog entry rejected")
	}
	if _, rejected := service.AwardFromCatalog(context.Background(), admin, slug, assets.CollectibleOwnerKindUser, holder.String(), "").(assets.MintRejected); !rejected {
		t.Fatalf("withdrawn entry was still awardable")
	}
	// Delete refused while the live instance exists.
	if _, rejected := service.DeleteCatalogEntry(context.Background(), mustCatalogSlug(t, slug)).(assets.CatalogMutationRejected); !rejected {
		t.Fatalf("catalog entry deleted while a live instance references it")
	}

	// Withdraw and hard-delete the instance, then the entry deletes.
	if _, matched := service.WithdrawCollectible(context.Background(), admin, awarded.Value.ID).(assets.CollectibleWithdrawn); !matched {
		t.Fatalf("withdraw collectible rejected")
	}
	if _, matched := service.DeleteWithdrawnCollectible(context.Background(), awarded.Value.ID).(assets.CollectibleDeleted); !matched {
		t.Fatalf("delete withdrawn collectible rejected")
	}
	if _, matched := service.DeleteCatalogEntry(context.Background(), mustCatalogSlug(t, slug)).(assets.CatalogMutated); !matched {
		t.Fatalf("delete catalog entry rejected after instances were removed")
	}
}

// TestCatalogKindSemantics enforces true uniqueness and edition caps at
// award time: one live unique instance ever (withdrawing frees the slot),
// numbered editions up to the cap.
func TestCatalogKindSemantics(t *testing.T) {
	pool := newPool(t)
	service := newAssetService(pool)
	admin := createUser(t, pool, "kind-admin")
	holder := createUser(t, pool, "kind-holder")

	// Unique: one live instance ever.
	uniqueSlug := uniqueTestSlug(t, "test-unique")
	if _, matched := service.AddCatalogEntry(context.Background(), testCatalogEntry(t, uniqueSlug, assets.CollectibleKindUnique, assets.EditionCapOf{Limit: 1})).(assets.CatalogMutated); !matched {
		t.Fatalf("add unique entry rejected")
	}
	first, firstMatched := service.AwardFromCatalog(context.Background(), admin, uniqueSlug, assets.CollectibleOwnerKindUser, holder.String(), "").(assets.CollectibleMinted)
	if !firstMatched {
		t.Fatalf("first unique award rejected")
	}
	if _, rejected := service.AwardFromCatalog(context.Background(), admin, uniqueSlug, assets.CollectibleOwnerKindUser, holder.String(), "").(assets.MintRejected); !rejected {
		t.Fatalf("second live unique instance was minted")
	}
	// Withdrawing the live instance frees the unique slot.
	if _, matched := service.WithdrawCollectible(context.Background(), admin, first.Value.ID).(assets.CollectibleWithdrawn); !matched {
		t.Fatalf("withdraw unique instance rejected")
	}
	if _, matched := service.AwardFromCatalog(context.Background(), admin, uniqueSlug, assets.CollectibleOwnerKindUser, holder.String(), "").(assets.CollectibleMinted); !matched {
		t.Fatalf("unique award after withdrawal rejected")
	}

	// Edition: numbered instances up to the cap.
	editionSlug := uniqueTestSlug(t, "test-edition")
	if _, matched := service.AddCatalogEntry(context.Background(), testCatalogEntry(t, editionSlug, assets.CollectibleKindEdition, assets.EditionCapOf{Limit: 2})).(assets.CatalogMutated); !matched {
		t.Fatalf("add edition entry rejected")
	}
	firstEdition := service.AwardFromCatalog(context.Background(), admin, editionSlug, assets.CollectibleOwnerKindUser, holder.String(), "").(assets.CollectibleMinted)
	secondEdition := service.AwardFromCatalog(context.Background(), admin, editionSlug, assets.CollectibleOwnerKindUser, holder.String(), "").(assets.CollectibleMinted)
	firstNumber := firstEdition.Value.Edition.(assets.EditionNumbered).Number
	secondNumber := secondEdition.Value.Edition.(assets.EditionNumbered).Number
	if firstNumber != 1 || secondNumber != 2 {
		t.Fatalf("edition numbers = %d, %d, want 1, 2", firstNumber, secondNumber)
	}
	if _, rejected := service.AwardFromCatalog(context.Background(), admin, editionSlug, assets.CollectibleOwnerKindUser, holder.String(), "").(assets.MintRejected); !rejected {
		t.Fatalf("edition cap was exceeded")
	}
}

// TestCustomMintUniqueness enforces the custom-mint semantics: one live
// unique per (issuer, name), and per-(issuer, name) edition numbering.
func TestCustomMintUniqueness(t *testing.T) {
	pool := newPool(t)
	service := newAssetService(pool)
	issuer := createUser(t, pool, "mint-issuer")
	otherIssuer := createUser(t, pool, "mint-other")

	name, _ := assets.NewCollectibleName("One Of A Kind " + issuer.String()).(assets.CollectibleNameAccepted)
	if _, matched := service.Mint(context.Background(), issuer, assets.CollectibleOwnerKindUser, issuer.String(), "", name.Value, assets.CollectibleKindUnique, assets.TransferPolicyTransferableBetweenUsers, "carrot").(assets.CollectibleMinted); !matched {
		t.Fatalf("first custom unique mint rejected")
	}
	if _, rejected := service.Mint(context.Background(), issuer, assets.CollectibleOwnerKindUser, issuer.String(), "", name.Value, assets.CollectibleKindUnique, assets.TransferPolicyTransferableBetweenUsers, "carrot").(assets.MintRejected); !rejected {
		t.Fatalf("second custom unique mint with the same issuer+name was accepted")
	}
	// A different issuer can mint a unique with the same name.
	if _, matched := service.Mint(context.Background(), otherIssuer, assets.CollectibleOwnerKindUser, otherIssuer.String(), "", name.Value, assets.CollectibleKindUnique, assets.TransferPolicyTransferableBetweenUsers, "carrot").(assets.CollectibleMinted); !matched {
		t.Fatalf("other issuer's unique mint rejected")
	}

	editionName, _ := assets.NewCollectibleName("Numbered Run " + issuer.String()).(assets.CollectibleNameAccepted)
	firstEdition := service.Mint(context.Background(), issuer, assets.CollectibleOwnerKindUser, issuer.String(), "", editionName.Value, assets.CollectibleKindEdition, assets.TransferPolicyTransferableBetweenUsers, "apple").(assets.CollectibleMinted)
	secondEdition := service.Mint(context.Background(), issuer, assets.CollectibleOwnerKindUser, issuer.String(), "", editionName.Value, assets.CollectibleKindEdition, assets.TransferPolicyTransferableBetweenUsers, "apple").(assets.CollectibleMinted)
	if firstEdition.Value.Edition.(assets.EditionNumbered).Number != 1 || secondEdition.Value.Edition.(assets.EditionNumbered).Number != 2 {
		t.Fatalf("custom edition numbering = %#v, %#v", firstEdition.Value.Edition, secondEdition.Value.Edition)
	}
}

// TestWithdrawCollectibleNotifiesHolderAndFreezesInstance covers the
// instance withdrawal lifecycle: the holder learns exactly once, the
// withdrawn instance leaves every movement path (tip, escrow), and only
// withdrawn instances can be hard-deleted.
func TestWithdrawCollectibleNotifiesHolderAndFreezesInstance(t *testing.T) {
	pool := newPool(t)
	service := newAssetService(pool)
	admin := createUser(t, pool, "withdraw-admin")
	holder := createUser(t, pool, "withdraw-holder")
	friend := createUser(t, pool, "withdraw-friend")

	awarded, awardMatched := service.AwardFromCatalog(context.Background(), admin, "harvest-star", assets.CollectibleOwnerKindUser, holder.String(), "").(assets.CollectibleMinted)
	if !awardMatched {
		t.Fatalf("award rejected")
	}

	// Deleting a live (non-withdrawn) instance is refused.
	if _, rejected := service.DeleteWithdrawnCollectible(context.Background(), awarded.Value.ID).(assets.DeleteCollectibleRejected); !rejected {
		t.Fatalf("live instance was hard-deleted")
	}

	withdrawn, withdrawnMatched := service.WithdrawCollectible(context.Background(), admin, awarded.Value.ID).(assets.CollectibleWithdrawn)
	if !withdrawnMatched {
		t.Fatalf("withdraw rejected")
	}
	if withdrawn.Value.State != assets.CollectibleStateWithdrawn {
		t.Fatalf("state = %s, want withdrawn", withdrawn.Value.State.String())
	}
	if len(withdrawn.RecordedEvents) != 1 {
		t.Fatalf("recorded events = %d, want 1", len(withdrawn.RecordedEvents))
	}
	draft := withdrawn.RecordedEvents[0]
	if draft.Kind != event.KindCollectibleWithdrawn {
		t.Fatalf("event kind = %s", draft.Kind.String())
	}
	if got := countNotificationsForEvent(t, pool, draft.ID, holder); got != 1 {
		t.Fatalf("holder notifications = %d, want 1", got)
	}
	// A second withdraw is refused (the instance is already withdrawn).
	if _, rejected := service.WithdrawCollectible(context.Background(), admin, awarded.Value.ID).(assets.WithdrawRejected); !rejected {
		t.Fatalf("double withdraw accepted")
	}
	// Withdrawn instances cannot be tipped onward.
	if _, rejected := service.GiftCollectible(context.Background(), holder, friend, awarded.Value.ID).(assets.GiftRejected); !rejected {
		t.Fatalf("withdrawn instance was tipped")
	}
	// And they no longer appear as minted for escrow (FundReward path).
	taskID := insertTaskWithRewardKind(t, pool, holder, "draft", "none")
	if _, rejected := service.FundReward(context.Background(), holder, taskID, awarded.Value.ID).(assets.FundRewardRejected); !rejected {
		t.Fatalf("withdrawn instance was escrowed")
	}
}

// TestCollectibleOrganizationTransfers covers user->org and org->user moves:
// donating requires a transferable policy, and moving out of the
// organization requires the acting member's manage_collectibles permission.
func TestCollectibleOrganizationTransfers(t *testing.T) {
	pool := newPool(t)
	service := newAssetService(pool)
	organizationID := createOrganization(t, pool, "transfer-org")
	adminMember := createUser(t, pool, "transfer-admin")
	plainMember := createUser(t, pool, "transfer-member")
	donor := createUser(t, pool, "transfer-donor")
	receiver := createUser(t, pool, "transfer-receiver")
	addOrganizationMember(t, pool, organizationID, adminMember, "admin")
	addOrganizationMember(t, pool, organizationID, plainMember, "member")

	name, _ := assets.NewCollectibleName("Donated Trophy " + donor.String()).(assets.CollectibleNameAccepted)
	minted := service.Mint(context.Background(), donor, assets.CollectibleOwnerKindUser, donor.String(), "", name.Value, assets.CollectibleKindBadge, assets.TransferPolicyTransferableBetweenUsers, "windmill").(assets.CollectibleMinted)

	donated, donatedMatched := service.TransferCollectibleToOrganization(context.Background(), donor, organizationID, minted.Value.ID).(assets.CollectibleGifted)
	if !donatedMatched {
		t.Fatalf("donation rejected: %#v", donated)
	}
	if donated.Value.OwnerKind != assets.CollectibleOwnerKindOrganization || donated.Value.OwnerID != organizationID.String() {
		t.Fatalf("donated owner = %s/%s", donated.Value.OwnerKind, donated.Value.OwnerID)
	}

	// A plain member cannot move it out of the organization.
	if _, rejected := service.TransferCollectibleFromOrganization(context.Background(), plainMember, organizationID, receiver, minted.Value.ID).(assets.GiftRejected); !rejected {
		t.Fatalf("plain member transferred an organization collectible")
	}
	// An admin (manage_collectibles) can.
	out, outMatched := service.TransferCollectibleFromOrganization(context.Background(), adminMember, organizationID, receiver, minted.Value.ID).(assets.CollectibleGifted)
	if !outMatched {
		t.Fatalf("admin transfer rejected")
	}
	if out.Value.OwnerKind != assets.CollectibleOwnerKindUser || out.Value.OwnerID != receiver.String() {
		t.Fatalf("transferred owner = %s/%s", out.Value.OwnerKind, out.Value.OwnerID)
	}

	// Non-transferable policies stay put: a payout-only mint cannot be
	// donated.
	lockedName, _ := assets.NewCollectibleName("Locked Medal " + donor.String()).(assets.CollectibleNameAccepted)
	locked := service.Mint(context.Background(), donor, assets.CollectibleOwnerKindUser, donor.String(), "", lockedName.Value, assets.CollectibleKindBadge, assets.TransferPolicyNonTransferableExceptPayout, "beehive").(assets.CollectibleMinted)
	if _, rejected := service.TransferCollectibleToOrganization(context.Background(), donor, organizationID, locked.Value.ID).(assets.GiftRejected); !rejected {
		t.Fatalf("non-transferable collectible was donated")
	}
}
