package assets

import (
	"context"
	"testing"

	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/event"
	"github.com/e6qu/sharecrop/internal/event/eventtest"
)

func TestParseCollectibleKindRoundTrips(t *testing.T) {
	for _, kind := range []CollectibleKind{CollectibleKindUnique, CollectibleKindEdition, CollectibleKindBadge} {
		accepted, matched := ParseCollectibleKind(kind.String()).(CollectibleKindAccepted)
		if !matched || accepted.Value != kind {
			t.Fatalf("ParseCollectibleKind(%q) did not round trip", kind.String())
		}
	}
}

func TestParseCollectibleStateRoundTrips(t *testing.T) {
	for _, state := range []CollectibleState{CollectibleStateMinted, CollectibleStateEscrowed, CollectibleStateAwarded} {
		accepted, matched := ParseCollectibleState(state.String()).(CollectibleStateAccepted)
		if !matched || accepted.Value != state {
			t.Fatalf("ParseCollectibleState(%q) did not round trip", state.String())
		}
	}
}

func TestTransferPolicyRewardChecks(t *testing.T) {
	allowed := []TransferPolicy{
		TransferPolicyNonTransferableExceptPayout,
		TransferPolicyTransferableBetweenUsers,
		TransferPolicyTransferableWithinOrg,
	}
	for _, policy := range allowed {
		if _, matched := AllowsRewardPayout(policy).(RewardAllowed); !matched {
			t.Fatalf("policy %q should allow reward payout", policy.String())
		}
	}
	if _, matched := AllowsRewardPayout(TransferPolicyIssuerControlled).(RewardDenied); !matched {
		t.Fatalf("issuer-controlled policy should deny reward payout")
	}
}

func TestNewCollectibleNameRejectsBlank(t *testing.T) {
	if _, matched := NewCollectibleName("   ").(CollectibleNameRejected); !matched {
		t.Fatalf("blank collectible name was accepted")
	}
}

func TestServiceMintCreatesMintedCollectible(t *testing.T) {
	store := &memoryStore{}
	service := NewService(store, eventtest.NewRecorder())
	minted, matched := service.Mint(context.Background(), newUserID(t), CollectibleOwnerKindUser, newUserID(t).String(), "", name(t, "Gold badge"), CollectibleKindBadge, TransferPolicyNonTransferableExceptPayout, "golden-sickle").(CollectibleMinted)
	if !matched {
		t.Fatalf("mint was rejected")
	}
	if minted.Value.State != CollectibleStateMinted {
		t.Fatalf("minted state = %q, want minted", minted.Value.State.String())
	}
	if len(store.created) != 1 {
		t.Fatalf("store create count = %d, want 1", len(store.created))
	}
}

func TestServiceMintScopesOrganizationOwnedCollectible(t *testing.T) {
	store := &memoryStore{}
	service := NewService(store, eventtest.NewRecorder())
	organizationID := newOrganizationID(t).String()

	minted, matched := service.Mint(context.Background(), newUserID(t), CollectibleOwnerKindOrganization, organizationID, "", name(t, "Org badge"), CollectibleKindBadge, TransferPolicyTransferableWithinOrg, "harvest-star").(CollectibleMinted)
	if !matched {
		t.Fatalf("mint was rejected")
	}
	if minted.Value.OrganizationID != organizationID {
		t.Fatalf("organization id = %q, want %q", minted.Value.OrganizationID, organizationID)
	}
}

type memoryStore struct {
	created []Collectible
	// events, when set, captures the drafts the store "recorded" inside its
	// mutations, so emission tests can assert on them.
	events *eventtest.CapturingStore
}

// record captures a recorded draft when a sink is wired; nil-safe for tests
// that assert nothing about emissions.
func (store *memoryStore) record(draft event.Draft) {
	if store.events != nil {
		store.events.Record(draft)
	}
}

func (store *memoryStore) CreateCollectible(_ context.Context, collectible Collectible) CreateStoreResult {
	store.created = append(store.created, collectible)
	return CreateStoreAccepted{Value: collectible}
}

func (store *memoryStore) ListCollectibles(_ context.Context, _ core.UserID, _ core.Page) ListStoreResult {
	return ListStoreListed{Values: store.created}
}

func (store *memoryStore) ListCollectiblesByOwner(_ context.Context, _ string, _ string, _ core.Page) ListStoreResult {
	return ListStoreListed{Values: store.created}
}

func (store *memoryStore) FundCollectibleReward(_ context.Context, _ FundRewardStoreCommand) FundRewardResult {
	return FundRewardRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "unused")}
}

func (store *memoryStore) RefundCollectibleReward(_ context.Context, _ RefundRewardStoreCommand) RefundRewardResult {
	return RefundRewardRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "unused")}
}

func (store *memoryStore) GiftCollectible(_ context.Context, command GiftStoreCommand) GiftResult {
	store.record(command.Draft)
	return CollectibleGifted{Value: Collectible{ID: command.CollectibleID, OwnerKind: CollectibleOwnerKindUser, OwnerID: command.ToUserID.String()}, RecordedEvents: []event.Draft{command.Draft}}
}

func (store *memoryStore) AwardOrganizationCollectible(_ context.Context, command AwardOrganizationCollectibleStoreCommand) GiftResult {
	store.record(command.Draft)
	return CollectibleGifted{Value: Collectible{ID: command.CollectibleID, OwnerKind: CollectibleOwnerKindUser, OwnerID: command.RecipientUserID.String()}, RecordedEvents: []event.Draft{command.Draft}}
}

func (store *memoryStore) TaskHeldCollectibles(_ context.Context, _ core.TaskID) TaskHeldCollectiblesResult {
	return TaskHeldCollectiblesFound{IDs: nil}
}

func (store *memoryStore) ListCatalogEntries(_ context.Context) CatalogListResult {
	return CatalogListed{Values: []CatalogListing{}}
}

func (store *memoryStore) AddCatalogEntry(_ context.Context, entry CatalogEntry) CatalogMutationResult {
	return CatalogMutated{Value: entry}
}

func (store *memoryStore) WithdrawCatalogEntry(_ context.Context, _ CatalogSlug) CatalogMutationResult {
	return CatalogMutationRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "unused")}
}

func (store *memoryStore) DeleteCatalogEntry(_ context.Context, _ CatalogSlug) CatalogMutationResult {
	return CatalogMutationRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "unused")}
}

func (store *memoryStore) AwardFromCatalog(_ context.Context, command AwardFromCatalogStoreCommand) CreateStoreResult {
	return CreateStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "unused")}
}

func (store *memoryStore) WithdrawCollectible(_ context.Context, command WithdrawCollectibleStoreCommand) WithdrawResult {
	store.record(command.Draft)
	return CollectibleWithdrawn{Value: Collectible{ID: command.CollectibleID, State: CollectibleStateWithdrawn, Catalog: NoCatalogRef{}, Edition: NoEditionNumber{}, Issuer: NoIssuer{}}, RecordedEvents: []event.Draft{command.Draft}}
}

func (store *memoryStore) DeleteWithdrawnCollectible(_ context.Context, _ core.CollectibleID) DeleteCollectibleResult {
	return DeleteCollectibleRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "unused")}
}

func (store *memoryStore) TransferCollectibleToOrganization(_ context.Context, command TransferToOrganizationStoreCommand) GiftResult {
	store.record(command.Draft)
	return CollectibleGifted{Value: Collectible{ID: command.CollectibleID, OwnerKind: CollectibleOwnerKindOrganization, OwnerID: command.OrganizationID.String(), Catalog: NoCatalogRef{}, Edition: NoEditionNumber{}, Issuer: NoIssuer{}}, RecordedEvents: []event.Draft{command.Draft}}
}

func (store *memoryStore) TransferCollectibleFromOrganization(_ context.Context, command TransferFromOrganizationStoreCommand) GiftResult {
	store.record(command.Draft)
	return CollectibleGifted{Value: Collectible{ID: command.CollectibleID, OwnerKind: CollectibleOwnerKindUser, OwnerID: command.ToUserID.String(), Catalog: NoCatalogRef{}, Edition: NoEditionNumber{}, Issuer: NoIssuer{}}, RecordedEvents: []event.Draft{command.Draft}}
}

func name(t *testing.T, raw string) CollectibleName {
	t.Helper()
	accepted, matched := NewCollectibleName(raw).(CollectibleNameAccepted)
	if !matched {
		t.Fatalf("collectible name rejected")
	}
	return accepted.Value
}

func newUserID(t *testing.T) core.UserID {
	t.Helper()
	created, matched := core.NewUserID().(core.UserIDCreated)
	if !matched {
		t.Fatalf("user id rejected")
	}
	return created.Value
}

func newOrganizationID(t *testing.T) core.OrganizationID {
	t.Helper()
	created, matched := core.NewOrganizationID().(core.OrganizationIDCreated)
	if !matched {
		t.Fatalf("organization id rejected")
	}
	return created.Value
}
