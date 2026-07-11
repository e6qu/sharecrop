package assetsbridge

import (
	"testing"

	"github.com/e6qu/sharecrop/internal/assets"
	"github.com/e6qu/sharecrop/internal/assets/assetstest"
	"github.com/e6qu/sharecrop/internal/core"
)

func sampleCollectible(t *testing.T) assets.Collectible {
	t.Helper()
	id, matched := core.NewCollectibleID().(core.CollectibleIDCreated)
	if !matched {
		t.Fatalf("collectible id rejected")
	}
	name, matched := assets.NewCollectibleName("Sample medal").(assets.CollectibleNameAccepted)
	if !matched {
		t.Fatalf("name rejected")
	}
	return assets.Collectible{
		ID:             id.Value,
		Name:           name.Value,
		Kind:           assets.CollectibleKindBadge,
		State:          assets.CollectibleStateMinted,
		Policy:         assets.TransferPolicyNonTransferableExceptPayout,
		OwnerKind:      assets.CollectibleOwnerKindUser,
		OwnerID:        "owner-123",
		OrganizationID: "",
		Art:            "art-url",
	}
}

func assertCollectibleEqual(t *testing.T, got, want assets.Collectible) {
	t.Helper()
	if diff := assetstest.CollectibleDiff(got, want); diff != "" {
		t.Errorf("collectible mismatch: %s", diff)
	}
}

func TestCollectibleRoundTrip(t *testing.T) {
	original := sampleCollectible(t)
	restored, err := decodeCollectible(encodeCollectible(original))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	assertCollectibleEqual(t, restored, original)
}

func TestResultRoundTrips(t *testing.T) {
	collectible := sampleCollectible(t)

	// Create (accept/reject).
	if _, matched := mustDecodeCreate(t, encodeCreateResult(assets.CreateStoreAccepted{})).(assets.CreateStoreAccepted); !matched {
		t.Errorf("create accepted did not round-trip")
	}

	// Fund (single collectible).
	funded, err := decodeFundRewardResult(encodeFundRewardResult(assets.RewardFunded{Value: collectible}))
	if err != nil {
		t.Fatalf("decode funded: %v", err)
	}
	if typed, matched := funded.(assets.RewardFunded); !matched {
		t.Fatalf("funded result = %T", funded)
	} else {
		assertCollectibleEqual(t, typed.Value, collectible)
	}

	// Gift (single collectible).
	gifted, err := decodeGiftResult(encodeGiftResult(assets.CollectibleGifted{Value: collectible}))
	if err != nil {
		t.Fatalf("decode gifted: %v", err)
	}
	if _, matched := gifted.(assets.CollectibleGifted); !matched {
		t.Errorf("gifted result = %T", gifted)
	}

	// Refund and list (collectible slice).
	refunded, err := decodeRefundRewardResult(encodeRefundRewardResult(assets.RewardRefunded{Values: []assets.Collectible{collectible}}))
	if err != nil {
		t.Fatalf("decode refunded: %v", err)
	}
	if typed, matched := refunded.(assets.RewardRefunded); !matched || len(typed.Values) != 1 {
		t.Errorf("refunded result = %T", refunded)
	}

	listed, err := decodeListResult(encodeListResult(assets.ListStoreListed{Values: []assets.Collectible{collectible}}))
	if err != nil {
		t.Fatalf("decode listed: %v", err)
	}
	if typed, matched := listed.(assets.ListStoreListed); !matched || len(typed.Values) != 1 {
		t.Errorf("listed result = %T", listed)
	} else {
		assertCollectibleEqual(t, typed.Values[0], collectible)
	}

	// TaskHeld (collectible-id slice).
	held, err := decodeTaskHeldResult(encodeTaskHeldResult(assets.TaskHeldCollectiblesFound{IDs: []core.CollectibleID{collectible.ID}}))
	if err != nil {
		t.Fatalf("decode held: %v", err)
	}
	if typed, matched := held.(assets.TaskHeldCollectiblesFound); !matched || len(typed.IDs) != 1 || typed.IDs[0] != collectible.ID {
		t.Errorf("task-held result did not round-trip: %T", held)
	}
}

func mustDecodeCreate(t *testing.T, wire acceptedRejectedWire) assets.CreateStoreResult {
	t.Helper()
	result, err := decodeCreateResult(wire)
	if err != nil {
		t.Fatalf("decode create: %v", err)
	}
	return result
}
