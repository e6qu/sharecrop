//go:build integration

package integration_test

import (
	"context"
	"testing"

	"github.com/e6qu/sharecrop/internal/assets"
	"github.com/e6qu/sharecrop/internal/assets/assetstest"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/db"
	"github.com/e6qu/sharecrop/internal/wasibridge/assetsbridge"
	"github.com/e6qu/sharecrop/internal/wasibridge/rpc"
)

// TestAssetsBridgeDualRun exercises the collectibles store through both the
// direct-db path and the compiled wasip1 guest + host bridge: create, list,
// list-by-owner (the two-string-argument method), gift, and task-held.
func TestAssetsBridgeDualRun(t *testing.T) {
	ctx := context.Background()
	pool := newPool(t)
	dbStore := db.NewCollectibleStore(pool)

	guestWASM, err := compileWASIGuest(t, "github.com/e6qu/sharecrop/cmd/sharecrop-wasi-store-guest")
	if err != nil {
		t.Fatalf("compile store guest: %v", err)
	}
	host, err := rpc.NewHost(ctx, guestWASM, func(ctx context.Context, method string, args []byte) ([]byte, error) {
		return assetsbridge.Dispatch(ctx, dbStore, method, args)
	})
	if err != nil {
		t.Fatalf("new host: %v", err)
	}
	t.Cleanup(func() { _ = host.Close(ctx) })
	bridgeStore := assetsbridge.NewGuestStore(func(method string, args []byte) ([]byte, error) {
		return host.Call(ctx, method, args)
	})

	owner := createUser(t, pool, "assets-owner")
	recipient := createUser(t, pool, "assets-recipient")
	page := requirePage(t, 50, 0)

	collectible := buildCollectible(t, owner)

	t.Run("create through the bridge, then list matches a direct call", func(t *testing.T) {
		if _, matched := bridgeStore.CreateCollectible(ctx, collectible).(assets.CreateStoreAccepted); !matched {
			t.Fatalf("bridge CreateCollectible did not accept")
		}
		viaBridge := requireCollectiblesListed(t, bridgeStore.ListCollectibles(ctx, owner, page))
		direct := requireCollectiblesListed(t, dbStore.ListCollectibles(ctx, owner, page))
		assertCollectibleSetsEqual(t, viaBridge, direct)
		if !containsCollectible(viaBridge, collectible.ID) {
			t.Errorf("bridge list did not contain the created collectible")
		}
	})

	t.Run("list by owner (two string arguments) matches a direct call", func(t *testing.T) {
		viaBridge := requireCollectiblesListed(t, bridgeStore.ListCollectiblesByOwner(ctx, assets.CollectibleOwnerKindUser, owner.String(), page))
		direct := requireCollectiblesListed(t, dbStore.ListCollectiblesByOwner(ctx, assets.CollectibleOwnerKindUser, owner.String(), page))
		assertCollectibleSetsEqual(t, viaBridge, direct)
		if !containsCollectible(viaBridge, collectible.ID) {
			t.Errorf("bridge list-by-owner did not contain the created collectible")
		}
	})

	t.Run("gift through the bridge transfers ownership", func(t *testing.T) {
		gifted, matched := bridgeStore.GiftCollectible(ctx, assets.GiftStoreCommand{
			FromUserID:    owner,
			ToUserID:      recipient,
			CollectibleID: collectible.ID,
		}).(assets.CollectibleGifted)
		if !matched {
			t.Fatalf("bridge GiftCollectible did not report CollectibleGifted")
		}
		if gifted.Value.OwnerID != recipient.String() {
			t.Errorf("gifted owner = %q, want %s", gifted.Value.OwnerID, recipient)
		}
	})

	t.Run("task-held collectibles matches a direct call", func(t *testing.T) {
		taskID := newTaskID(t)
		viaBridge := requireTaskHeld(t, bridgeStore.TaskHeldCollectibles(ctx, taskID))
		direct := requireTaskHeld(t, dbStore.TaskHeldCollectibles(ctx, taskID))
		if len(viaBridge) != len(direct) {
			t.Errorf("task-held counts: bridge %d, direct %d", len(viaBridge), len(direct))
		}
	})
}

func buildCollectible(t *testing.T, owner core.UserID) assets.Collectible {
	t.Helper()
	id, matched := core.NewCollectibleID().(core.CollectibleIDCreated)
	if !matched {
		t.Fatalf("collectible id rejected")
	}
	name, matched := assets.NewCollectibleName("Bridge medal").(assets.CollectibleNameAccepted)
	if !matched {
		t.Fatalf("collectible name rejected")
	}
	return assets.Collectible{
		ID:        id.Value,
		Name:      name.Value,
		Kind:      assets.CollectibleKindBadge,
		State:     assets.CollectibleStateMinted,
		Policy:    assets.TransferPolicyTransferableBetweenUsers,
		OwnerKind: assets.CollectibleOwnerKindUser,
		OwnerID:   owner.String(),
	}
}

func requireCollectiblesListed(t *testing.T, result assets.ListStoreResult) []assets.Collectible {
	t.Helper()
	listed, matched := result.(assets.ListStoreListed)
	if !matched {
		t.Fatalf("list result = %T, want ListStoreListed", result)
	}
	return listed.Values
}

func requireTaskHeld(t *testing.T, result assets.TaskHeldCollectiblesResult) []core.CollectibleID {
	t.Helper()
	found, matched := result.(assets.TaskHeldCollectiblesFound)
	if !matched {
		t.Fatalf("task-held result = %T, want TaskHeldCollectiblesFound", result)
	}
	return found.IDs
}

func containsCollectible(values []assets.Collectible, id core.CollectibleID) bool {
	for index := range values {
		if values[index].ID == id {
			return true
		}
	}
	return false
}

func assertCollectibleSetsEqual(t *testing.T, got, want []assets.Collectible) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("collectible counts: bridge %d, direct %d", len(got), len(want))
	}
	for index := range want {
		if diff := assetstest.CollectibleDiff(got[index], want[index]); diff != "" {
			t.Errorf("collectible %d mismatch: %s", index, diff)
		}
	}
}
