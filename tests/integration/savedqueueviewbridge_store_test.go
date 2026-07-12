//go:build integration

package integration_test

import (
	"context"
	"testing"

	"github.com/e6qu/sharecrop/internal/db"
	httpserver "github.com/e6qu/sharecrop/internal/http"
	"github.com/e6qu/sharecrop/internal/wasibridge/rpc"
	"github.com/e6qu/sharecrop/internal/wasibridge/savedqueueviewbridge"
)

// TestSavedQueueViewBridgeDualRun exercises the saved-queue-view service - an
// internal/http RuntimeState service, not a domain store - through the compiled
// wasip1 guest + host bridge: upsert a view, then list it, checking the bridge's
// results match a direct db call. This is the pattern for bridging the remaining
// RuntimeState services so the pooled mux shares one Postgres store.
func TestSavedQueueViewBridgeDualRun(t *testing.T) {
	ctx := context.Background()
	pool := newPool(t)
	dbStore := db.NewSavedQueueViewStore(pool)

	guestWASM, err := compileWASIGuest(t, "github.com/e6qu/sharecrop/cmd/sharecrop-wasi-store-guest")
	if err != nil {
		t.Fatalf("compile store guest: %v", err)
	}
	host, err := rpc.NewHost(ctx, guestWASM, func(ctx context.Context, method string, args []byte) ([]byte, error) {
		return savedqueueviewbridge.Dispatch(ctx, dbStore, method, args)
	})
	if err != nil {
		t.Fatalf("new host: %v", err)
	}
	t.Cleanup(func() { _ = host.Close(ctx) })
	bridgeStore := savedqueueviewbridge.NewGuestStore(func(method string, args []byte) ([]byte, error) {
		return host.Call(ctx, method, args)
	})

	owner := createUser(t, pool, "savedview-owner")
	view := httpserver.SavedQueueView{
		UserID:      owner,
		Scope:       "team_work",
		Name:        "My open reviews",
		Query:       "review",
		StateFilter: "open",
		TypeFilter:  "code_review",
		Sort:        "newest",
	}

	t.Run("upsert through the bridge, then list matches a direct call", func(t *testing.T) {
		saved, matched := bridgeStore.Upsert(ctx, view).(httpserver.SavedQueueViewSaved)
		if !matched {
			t.Fatalf("bridge Upsert did not save the view")
		}
		if saved.Value.Name != view.Name || saved.Value.ID == "" {
			t.Errorf("saved view = %+v", saved.Value)
		}

		viaBridge := requireViewsListed(t, bridgeStore.List(ctx, owner, "team_work"))
		direct := requireViewsListed(t, dbStore.List(ctx, owner, "team_work"))
		if len(viaBridge) != len(direct) || len(viaBridge) != 1 {
			t.Fatalf("view counts: bridge %d, direct %d, want 1", len(viaBridge), len(direct))
		}
		if diff := savedViewDiff(viaBridge[0], direct[0]); diff != "" {
			t.Errorf("saved view mismatch: %s", diff)
		}
	})
}

func requireViewsListed(t *testing.T, result httpserver.SavedQueueViewsListResult) []httpserver.SavedQueueView {
	t.Helper()
	listed, matched := result.(httpserver.SavedQueueViewsListed)
	if !matched {
		t.Fatalf("list result = %T, want listed", result)
	}
	return listed.Values
}

func savedViewDiff(got, want httpserver.SavedQueueView) string {
	switch {
	case got.ID != want.ID:
		return "id"
	case got.UserID != want.UserID:
		return "user_id"
	case got.Scope != want.Scope:
		return "scope"
	case got.Name != want.Name:
		return "name"
	case got.Query != want.Query:
		return "query"
	case got.StateFilter != want.StateFilter:
		return "state_filter"
	case got.TypeFilter != want.TypeFilter:
		return "type_filter"
	case got.Sort != want.Sort:
		return "sort"
	default:
		return ""
	}
}
