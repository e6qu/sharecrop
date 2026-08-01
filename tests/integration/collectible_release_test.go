//go:build integration

package integration_test

import (
	"context"
	"testing"

	"github.com/e6qu/sharecrop/internal/assets"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/event"
	"github.com/jackc/pgx/v5/pgxpool"
)

// setUserDisplayName gives a fixture user a stable display name so owner
// labels can be asserted literally.
func setUserDisplayName(t *testing.T, pool *pgxpool.Pool, userID core.UserID, name string) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), "update users set display_name = $2 where id = $1", userID.String(), name); err != nil {
		t.Fatalf("set display name: %v", err)
	}
}

// createTeamForOrganization inserts a team fixture owned by an organization.
func createTeamForOrganization(t *testing.T, pool *pgxpool.Pool, organizationID core.OrganizationID, name string, creator core.UserID) core.TeamID {
	t.Helper()
	teamID, matched := core.NewTeamID().(core.TeamIDCreated)
	if !matched {
		t.Fatalf("team id rejected")
	}
	if _, err := pool.Exec(context.Background(), `
		insert into teams (id, name, owner_kind, organization_id, created_by_user_id)
		values ($1, $2, 'organization', $3, $4)
	`, teamID.Value.String(), name, organizationID.String(), creator.String()); err != nil {
		t.Fatalf("insert team: %v", err)
	}
	return teamID.Value
}

// TestCatalogEntryWithdrawReleaseAwardLifecycle covers the entry release
// path: a withdrawn entry refuses awards, releasing it makes it awardable
// again, and releasing an entry that is not withdrawn is refused.
func TestCatalogEntryWithdrawReleaseAwardLifecycle(t *testing.T) {
	pool := newPool(t)
	service := newAssetService(pool)
	admin := createUser(t, pool, "entry-release-admin")
	holder := createUser(t, pool, "entry-release-holder")

	slug := uniqueTestSlug(t, "test-release-badge")
	if _, matched := service.AddCatalogEntry(context.Background(), testCatalogEntry(t, slug, assets.CollectibleKindBadge, assets.NoEditionCap{})).(assets.CatalogMutated); !matched {
		t.Fatalf("add catalog entry rejected")
	}

	// Releasing an available entry is refused with a conflict.
	releaseAvailable := service.ReleaseCatalogEntry(context.Background(), mustCatalogSlug(t, slug))
	rejected, rejectedMatched := releaseAvailable.(assets.CatalogMutationRejected)
	if !rejectedMatched || rejected.Reason.Code() != core.ErrorCodeConflict {
		t.Fatalf("release of an available entry = %#v, want a conflict rejection", releaseAvailable)
	}

	if _, matched := service.WithdrawCatalogEntry(context.Background(), mustCatalogSlug(t, slug)).(assets.CatalogMutated); !matched {
		t.Fatalf("withdraw catalog entry rejected")
	}
	if _, awardRejected := service.AwardFromCatalog(context.Background(), admin, slug, assets.CollectibleOwnerKindUser, holder.String(), "").(assets.MintRejected); !awardRejected {
		t.Fatalf("withdrawn entry was still awardable")
	}

	released, releasedMatched := service.ReleaseCatalogEntry(context.Background(), mustCatalogSlug(t, slug)).(assets.CatalogMutated)
	if !releasedMatched {
		t.Fatalf("release catalog entry rejected")
	}
	if released.Value.State != assets.CatalogEntryStateAvailable {
		t.Fatalf("released entry state = %s, want available", released.Value.State.String())
	}
	if _, matched := service.AwardFromCatalog(context.Background(), admin, slug, assets.CollectibleOwnerKindUser, holder.String(), "").(assets.CollectibleMinted); !matched {
		t.Fatalf("award after entry release rejected")
	}
}

// TestReleaseCollectibleReturnsInstanceToHolder covers the instance release
// path: withdraw -> release restores the minted state with the owner
// unchanged, notifies the holder through collectible_released, and lifts the
// withdrawn-instance guards (the instance is transferable again).
func TestReleaseCollectibleReturnsInstanceToHolder(t *testing.T) {
	pool := newPool(t)
	service := newAssetService(pool)
	admin := createUser(t, pool, "release-admin")
	holder := createUser(t, pool, "release-holder")
	friend := createUser(t, pool, "release-friend")

	awarded, awardMatched := service.AwardFromCatalog(context.Background(), admin, "harvest-star", assets.CollectibleOwnerKindUser, holder.String(), "").(assets.CollectibleMinted)
	if !awardMatched {
		t.Fatalf("award rejected")
	}

	// Releasing a live (never-withdrawn) instance is refused with a conflict.
	liveRelease := service.ReleaseCollectible(context.Background(), admin, awarded.Value.ID)
	if liveRejected, matched := liveRelease.(assets.ReleaseRejected); !matched || liveRejected.Reason.Code() != core.ErrorCodeConflict {
		t.Fatalf("release of a live instance = %#v, want a conflict rejection", liveRelease)
	}

	if _, matched := service.WithdrawCollectible(context.Background(), admin, awarded.Value.ID).(assets.CollectibleWithdrawn); !matched {
		t.Fatalf("withdraw rejected")
	}

	released, releasedMatched := service.ReleaseCollectible(context.Background(), admin, awarded.Value.ID).(assets.CollectibleReleased)
	if !releasedMatched {
		t.Fatalf("release rejected")
	}
	if released.Value.State != assets.CollectibleStateMinted {
		t.Fatalf("released state = %s, want minted", released.Value.State.String())
	}
	if released.Value.OwnerKind != assets.CollectibleOwnerKindUser || released.Value.OwnerID != holder.String() {
		t.Fatalf("released owner = %s/%s, want the original holder", released.Value.OwnerKind, released.Value.OwnerID)
	}
	if len(released.RecordedEvents) != 1 || released.RecordedEvents[0].Kind != event.KindCollectibleReleased {
		t.Fatalf("recorded events = %#v, want one collectible_released", released.RecordedEvents)
	}
	if got := countNotificationsForEvent(t, pool, released.RecordedEvents[0].ID, holder); got != 1 {
		t.Fatalf("holder notifications = %d, want 1", got)
	}

	// The instance is back in the holder's inventory as minted.
	page := requirePage(t, 50, 0)
	listed, listedMatched := service.ListCollectibles(context.Background(), holder, page).(assets.CollectiblesListed)
	if !listedMatched {
		t.Fatalf("list rejected")
	}
	found := false
	for _, value := range listed.Values {
		if value.ID == awarded.Value.ID {
			found = true
			if value.State != assets.CollectibleStateMinted {
				t.Fatalf("listed state = %s, want minted", value.State.String())
			}
		}
	}
	if !found {
		t.Fatalf("released instance missing from the holder's inventory")
	}

	// A second release is refused (the instance is no longer withdrawn).
	if _, doubleRejected := service.ReleaseCollectible(context.Background(), admin, awarded.Value.ID).(assets.ReleaseRejected); !doubleRejected {
		t.Fatalf("double release accepted")
	}
	// The withdrawn-instance guards are gone: the holder can tip it onward.
	if _, matched := service.GiftCollectible(context.Background(), holder, friend, awarded.Value.ID).(assets.CollectibleGifted); !matched {
		t.Fatalf("released instance was not transferable")
	}
}

// TestReleaseUniqueConflictWhenSlotWasReminted covers the re-validated cap:
// withdrawing a unique frees the slot, awarding a replacement occupies it,
// and releasing the old instance is then refused with a conflict — while
// deleting the old withdrawn instance still works.
func TestReleaseUniqueConflictWhenSlotWasReminted(t *testing.T) {
	pool := newPool(t)
	service := newAssetService(pool)
	admin := createUser(t, pool, "unique-release-admin")
	holder := createUser(t, pool, "unique-release-holder")

	slug := uniqueTestSlug(t, "test-release-unique")
	if _, matched := service.AddCatalogEntry(context.Background(), testCatalogEntry(t, slug, assets.CollectibleKindUnique, assets.EditionCapOf{Limit: 1})).(assets.CatalogMutated); !matched {
		t.Fatalf("add unique entry rejected")
	}
	first, firstMatched := service.AwardFromCatalog(context.Background(), admin, slug, assets.CollectibleOwnerKindUser, holder.String(), "").(assets.CollectibleMinted)
	if !firstMatched {
		t.Fatalf("first unique award rejected")
	}
	if _, matched := service.WithdrawCollectible(context.Background(), admin, first.Value.ID).(assets.CollectibleWithdrawn); !matched {
		t.Fatalf("withdraw first unique rejected")
	}
	second, secondMatched := service.AwardFromCatalog(context.Background(), admin, slug, assets.CollectibleOwnerKindUser, holder.String(), "").(assets.CollectibleMinted)
	if !secondMatched {
		t.Fatalf("replacement unique award rejected")
	}

	// The old instance cannot come back while the replacement lives.
	conflicted := service.ReleaseCollectible(context.Background(), admin, first.Value.ID)
	conflictRejected, conflictMatched := conflicted.(assets.ReleaseRejected)
	if !conflictMatched || conflictRejected.Reason.Code() != core.ErrorCodeConflict {
		t.Fatalf("release with an occupied unique slot = %#v, want a conflict rejection", conflicted)
	}
	// Deleting the stuck withdrawn instance is the way out.
	if _, matched := service.DeleteWithdrawnCollectible(context.Background(), first.Value.ID).(assets.CollectibleDeleted); !matched {
		t.Fatalf("delete of the old withdrawn unique rejected")
	}
	// Sanity: the replacement is still the live instance.
	if _, rejected := service.AwardFromCatalog(context.Background(), admin, slug, assets.CollectibleOwnerKindUser, holder.String(), "").(assets.MintRejected); !rejected {
		t.Fatalf("unique slot was double-occupied after delete")
	}
	// Withdrawing the replacement frees the slot for a future release again.
	if _, matched := service.WithdrawCollectible(context.Background(), admin, second.Value.ID).(assets.CollectibleWithdrawn); !matched {
		t.Fatalf("withdraw replacement rejected")
	}
	if _, matched := service.ReleaseCollectible(context.Background(), admin, second.Value.ID).(assets.CollectibleReleased); !matched {
		t.Fatalf("release with a free slot rejected")
	}
}

// TestDeleteCatalogEntryRefusedWhileWithdrawnInstancesRemain pins the delete
// rule: an entry with only withdrawn instances still cannot be deleted —
// withdrawn instances carry the entry's provenance and can be released — and
// deletes cleanly once the instances are hard-deleted.
func TestDeleteCatalogEntryRefusedWhileWithdrawnInstancesRemain(t *testing.T) {
	pool := newPool(t)
	service := newAssetService(pool)
	admin := createUser(t, pool, "entry-delete-admin")
	holder := createUser(t, pool, "entry-delete-holder")

	slug := uniqueTestSlug(t, "test-delete-badge")
	if _, matched := service.AddCatalogEntry(context.Background(), testCatalogEntry(t, slug, assets.CollectibleKindBadge, assets.NoEditionCap{})).(assets.CatalogMutated); !matched {
		t.Fatalf("add catalog entry rejected")
	}
	awarded, awardMatched := service.AwardFromCatalog(context.Background(), admin, slug, assets.CollectibleOwnerKindUser, holder.String(), "").(assets.CollectibleMinted)
	if !awardMatched {
		t.Fatalf("award rejected")
	}
	if _, matched := service.WithdrawCatalogEntry(context.Background(), mustCatalogSlug(t, slug)).(assets.CatalogMutated); !matched {
		t.Fatalf("withdraw entry rejected")
	}
	if _, matched := service.WithdrawCollectible(context.Background(), admin, awarded.Value.ID).(assets.CollectibleWithdrawn); !matched {
		t.Fatalf("withdraw instance rejected")
	}

	// Only a withdrawn instance remains — deletion is still refused.
	blocked := service.DeleteCatalogEntry(context.Background(), mustCatalogSlug(t, slug))
	blockedRejected, blockedMatched := blocked.(assets.CatalogMutationRejected)
	if !blockedMatched || blockedRejected.Reason.Code() != core.ErrorCodeConflict {
		t.Fatalf("delete with a withdrawn instance = %#v, want a conflict rejection", blocked)
	}

	if _, matched := service.DeleteWithdrawnCollectible(context.Background(), awarded.Value.ID).(assets.CollectibleDeleted); !matched {
		t.Fatalf("delete withdrawn instance rejected")
	}
	if _, matched := service.DeleteCatalogEntry(context.Background(), mustCatalogSlug(t, slug)).(assets.CatalogMutated); !matched {
		t.Fatalf("delete entry rejected after instances were removed")
	}
}

// TestCollectibleOwnerLabels resolves the owner display label for all three
// owner kinds on the list read paths: user display name, organization name,
// and team name.
func TestCollectibleOwnerLabels(t *testing.T) {
	pool := newPool(t)
	service := newAssetService(pool)
	admin := createUser(t, pool, "owner-label-admin")
	holder := createUser(t, pool, "owner-label-holder")
	setUserDisplayName(t, pool, holder, "Mara Fields")
	organizationID := createOrganization(t, pool, "owner-label")
	teamCreator := createUser(t, pool, "owner-label-team-creator")
	teamID := createTeamForOrganization(t, pool, organizationID, "Harvest Crew", teamCreator)
	page := requirePage(t, 50, 0)

	userAward := service.AwardFromCatalog(context.Background(), admin, "harvest-star", assets.CollectibleOwnerKindUser, holder.String(), "").(assets.CollectibleMinted)
	orgAward := service.AwardFromCatalog(context.Background(), admin, "harvest-star", assets.CollectibleOwnerKindOrganization, organizationID.String(), "").(assets.CollectibleMinted)
	teamAward := service.AwardFromCatalog(context.Background(), admin, "harvest-star", assets.CollectibleOwnerKindTeam, teamID.String(), "").(assets.CollectibleMinted)

	assertOwnerLabel := func(ownerKind string, ownerID string, collectibleID core.CollectibleID, want string) {
		t.Helper()
		listed, matched := service.ListByOwner(context.Background(), ownerKind, ownerID, page).(assets.CollectiblesListed)
		if !matched {
			t.Fatalf("list by owner (%s) rejected", ownerKind)
		}
		for _, value := range listed.Values {
			if value.ID == collectibleID {
				if value.OwnerDisplayName.String() != want {
					t.Fatalf("%s owner label = %q, want %q", ownerKind, value.OwnerDisplayName.String(), want)
				}
				return
			}
		}
		t.Fatalf("collectible missing from %s owner listing", ownerKind)
	}

	assertOwnerLabel(assets.CollectibleOwnerKindUser, holder.String(), userAward.Value.ID, "Mara Fields")
	assertOwnerLabel(assets.CollectibleOwnerKindOrganization, organizationID.String(), orgAward.Value.ID, "owner-label org")
	assertOwnerLabel(assets.CollectibleOwnerKindTeam, teamID.String(), teamAward.Value.ID, "Harvest Crew")

	// The user-inventory list resolves the same label.
	inventory, inventoryMatched := service.ListCollectibles(context.Background(), holder, page).(assets.CollectiblesListed)
	if !inventoryMatched {
		t.Fatalf("inventory list rejected")
	}
	for _, value := range inventory.Values {
		if value.ID == userAward.Value.ID && value.OwnerDisplayName.String() != "Mara Fields" {
			t.Fatalf("inventory owner label = %q, want Mara Fields", value.OwnerDisplayName.String())
		}
	}
}

// TestCatalogOwnershipShapes covers the catalog listing's ownership columns:
// a unique entry reports its live holder's label, an edition run reports the
// distinct live-owner count, and unminted entries report empty ownership.
func TestCatalogOwnershipShapes(t *testing.T) {
	pool := newPool(t)
	service := newAssetService(pool)
	admin := createUser(t, pool, "catalog-owner-admin")
	firstHolder := createUser(t, pool, "catalog-owner-first")
	secondHolder := createUser(t, pool, "catalog-owner-second")
	setUserDisplayName(t, pool, firstHolder, "Reza Amani")

	uniqueSlug := uniqueTestSlug(t, "test-owner-unique")
	if _, matched := service.AddCatalogEntry(context.Background(), testCatalogEntry(t, uniqueSlug, assets.CollectibleKindUnique, assets.EditionCapOf{Limit: 1})).(assets.CatalogMutated); !matched {
		t.Fatalf("add unique entry rejected")
	}
	editionSlug := uniqueTestSlug(t, "test-owner-edition")
	if _, matched := service.AddCatalogEntry(context.Background(), testCatalogEntry(t, editionSlug, assets.CollectibleKindEdition, assets.EditionCapOf{Limit: 5})).(assets.CatalogMutated); !matched {
		t.Fatalf("add edition entry rejected")
	}
	emptySlug := uniqueTestSlug(t, "test-owner-empty")
	if _, matched := service.AddCatalogEntry(context.Background(), testCatalogEntry(t, emptySlug, assets.CollectibleKindUnique, assets.EditionCapOf{Limit: 1})).(assets.CatalogMutated); !matched {
		t.Fatalf("add empty unique entry rejected")
	}

	if _, matched := service.AwardFromCatalog(context.Background(), admin, uniqueSlug, assets.CollectibleOwnerKindUser, firstHolder.String(), "").(assets.CollectibleMinted); !matched {
		t.Fatalf("unique award rejected")
	}
	for _, holder := range []core.UserID{firstHolder, firstHolder, secondHolder} {
		if _, matched := service.AwardFromCatalog(context.Background(), admin, editionSlug, assets.CollectibleOwnerKindUser, holder.String(), "").(assets.CollectibleMinted); !matched {
			t.Fatalf("edition award rejected")
		}
	}

	listed, listedMatched := service.ListCatalog(context.Background()).(assets.CatalogListed)
	if !listedMatched {
		t.Fatalf("catalog list rejected")
	}
	listings := map[string]assets.CatalogListing{}
	for _, listing := range listed.Values {
		listings[listing.Entry.Slug.String()] = listing
	}

	unique := listings[uniqueSlug]
	if unique.OwnerDisplayName.String() != "Reza Amani" {
		t.Fatalf("unique owner label = %q, want Reza Amani", unique.OwnerDisplayName.String())
	}
	if unique.LiveInstanceCount != 1 || unique.LiveOwnerCount != 1 {
		t.Fatalf("unique counts = %d/%d, want 1/1", unique.LiveInstanceCount, unique.LiveOwnerCount)
	}

	edition := listings[editionSlug]
	if edition.OwnerDisplayName.String() != "" {
		t.Fatalf("edition owner label = %q, want empty", edition.OwnerDisplayName.String())
	}
	if edition.LiveInstanceCount != 3 || edition.LiveOwnerCount != 2 {
		t.Fatalf("edition counts = %d/%d, want 3 live across 2 owners", edition.LiveInstanceCount, edition.LiveOwnerCount)
	}

	empty := listings[emptySlug]
	if empty.OwnerDisplayName.String() != "" || empty.LiveInstanceCount != 0 || empty.LiveOwnerCount != 0 {
		t.Fatalf("unminted entry ownership = %q %d/%d, want empty", empty.OwnerDisplayName.String(), empty.LiveInstanceCount, empty.LiveOwnerCount)
	}
}
