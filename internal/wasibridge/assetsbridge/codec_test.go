package assetsbridge

import (
	"testing"

	"github.com/e6qu/sharecrop/internal/assets"
	"github.com/e6qu/sharecrop/internal/assets/assetstest"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/event"
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
		Catalog:        assets.NoCatalogRef{},
		Edition:        assets.NoEditionNumber{},
		Issuer:         assets.NoIssuer{},
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
	created, createMatched := mustDecodeCreate(t, encodeCreateResult(assets.CreateStoreAccepted{Value: collectible})).(assets.CreateStoreAccepted)
	if !createMatched {
		t.Errorf("create accepted did not round-trip")
	} else {
		assertCollectibleEqual(t, created.Value, collectible)
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

func TestReleaseRoundTrips(t *testing.T) {
	collectible := sampleCollectible(t)

	// Release command (collectible id + the collectible_released draft).
	draftCreated, draftMatched := event.NewDraft(event.KindCollectibleReleased, event.ActorSystem{}, event.NoSubjectRefs(), event.EmptyMetadata(), event.NewRecipients()).(event.DraftCreated)
	if !draftMatched {
		t.Fatalf("release draft rejected")
	}
	command := assets.ReleaseCollectibleStoreCommand{CollectibleID: collectible.ID, Draft: draftCreated.Value}
	decodedCommand, err := decodeReleaseCollectibleCommand(encodeReleaseCollectibleCommand(command))
	if err != nil {
		t.Fatalf("decode release command: %v", err)
	}
	if decodedCommand.CollectibleID != collectible.ID || decodedCommand.Draft.Kind != event.KindCollectibleReleased {
		t.Errorf("release command did not round-trip")
	}

	// Release result (released/rejected).
	released, err := decodeReleaseResult(encodeReleaseResult(assets.CollectibleReleased{Value: collectible}))
	if err != nil {
		t.Fatalf("decode released: %v", err)
	}
	if typed, matched := released.(assets.CollectibleReleased); !matched {
		t.Errorf("released result = %T", released)
	} else {
		assertCollectibleEqual(t, typed.Value, collectible)
	}
	rejected, err := decodeReleaseResult(encodeReleaseResult(assets.ReleaseRejected{Reason: core.NewDomainError(core.ErrorCodeConflict, "another live instance exists")}))
	if err != nil {
		t.Fatalf("decode release rejection: %v", err)
	}
	if typed, matched := rejected.(assets.ReleaseRejected); !matched || typed.Reason.Code() != core.ErrorCodeConflict {
		t.Errorf("release rejection did not round-trip: %T", rejected)
	}
}

func mustDecodeCreate(t *testing.T, wire collectibleResultWire) assets.CreateStoreResult {
	t.Helper()
	result, err := decodeCreateResult(wire)
	if err != nil {
		t.Fatalf("decode create: %v", err)
	}
	return result
}
