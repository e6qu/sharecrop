package assets

import (
	"context"
	"testing"

	"github.com/e6qu/sharecrop/internal/core"
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
	service := NewService(store)
	minted, matched := service.Mint(context.Background(), newUserID(t), name(t, "Gold badge"), CollectibleKindBadge, TransferPolicyNonTransferableExceptPayout).(CollectibleMinted)
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

type memoryStore struct {
	created []Collectible
}

func (store *memoryStore) CreateCollectible(_ context.Context, collectible Collectible) CreateStoreResult {
	store.created = append(store.created, collectible)
	return CreateStoreAccepted{}
}

func (store *memoryStore) ListCollectibles(_ context.Context, _ core.UserID, _ core.Page) ListStoreResult {
	return ListStoreListed{Values: store.created}
}

func (store *memoryStore) FundCollectibleReward(_ context.Context, _ FundRewardStoreCommand) FundRewardResult {
	return FundRewardRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "unused")}
}

func (store *memoryStore) RefundCollectibleReward(_ context.Context, _ RefundRewardStoreCommand) RefundRewardResult {
	return RefundRewardRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "unused")}
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
