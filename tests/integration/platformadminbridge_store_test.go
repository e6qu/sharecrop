//go:build integration

package integration_test

import (
	"context"
	"testing"

	"github.com/e6qu/sharecrop/internal/db"
	httpserver "github.com/e6qu/sharecrop/internal/http"
	"github.com/e6qu/sharecrop/internal/wasibridge/platformadminbridge"
	"github.com/e6qu/sharecrop/internal/wasibridge/rpc"
)

// TestPlatformAdminBridgeDualRun exercises the platform-admin service - an
// internal/http RuntimeState service - through the compiled wasip1 guest + host
// bridge: grant an admin (Grant takes two user ids, which the generator
// disambiguates), then check and list through the bridge, matching a direct db
// call.
func TestPlatformAdminBridgeDualRun(t *testing.T) {
	ctx := context.Background()
	pool := newPool(t)
	dbStore := db.NewPlatformAdminStore(pool, map[string]bool{})

	guestWASM, err := compileWASIGuest(t, "github.com/e6qu/sharecrop/cmd/sharecrop-wasi-store-guest")
	if err != nil {
		t.Fatalf("compile store guest: %v", err)
	}
	host, err := rpc.NewHost(ctx, guestWASM, func(ctx context.Context, method string, args []byte) ([]byte, error) {
		return platformadminbridge.Dispatch(ctx, dbStore, method, args)
	})
	if err != nil {
		t.Fatalf("new host: %v", err)
	}
	t.Cleanup(func() { _ = host.Close(ctx) })
	bridgeStore := platformadminbridge.NewGuestStore(func(method string, args []byte) ([]byte, error) {
		return host.Call(ctx, method, args)
	})

	granter := createUser(t, pool, "platformadmin-granter")
	grantee := createUser(t, pool, "platformadmin-grantee")
	page := requirePage(t, 50, 0)

	t.Run("grant through the bridge makes the user an admin", func(t *testing.T) {
		if _, matched := bridgeStore.Grant(ctx, grantee, granter).(httpserver.PlatformAdminSaved); !matched {
			t.Fatalf("bridge Grant did not save")
		}
		if _, matched := bridgeStore.IsAdmin(ctx, grantee).(httpserver.PlatformAdminAllowed); !matched {
			t.Errorf("bridge IsAdmin did not allow the granted user")
		}
		if _, matched := bridgeStore.IsAdmin(ctx, granter).(httpserver.PlatformAdminDenied); !matched {
			t.Errorf("bridge IsAdmin allowed a non-admin")
		}
	})

	t.Run("list matches a direct call", func(t *testing.T) {
		viaBridge := requireAdminsListed(t, bridgeStore.List(ctx, page))
		direct := requireAdminsListed(t, dbStore.List(ctx, page))
		if len(viaBridge) != len(direct) || len(viaBridge) != 1 {
			t.Fatalf("admin counts: bridge %d, direct %d, want 1", len(viaBridge), len(direct))
		}
		if viaBridge[0].UserID != direct[0].UserID || viaBridge[0].UserID != grantee {
			t.Errorf("listed admin = %s, want %s", viaBridge[0].UserID, grantee)
		}
		if viaBridge[0].Source != direct[0].Source {
			t.Errorf("admin source: bridge %q, direct %q", viaBridge[0].Source, direct[0].Source)
		}
	})
}

func requireAdminsListed(t *testing.T, result httpserver.PlatformAdminListResult) []httpserver.PlatformAdminRecord {
	t.Helper()
	listed, matched := result.(httpserver.PlatformAdminsListed)
	if !matched {
		t.Fatalf("list result = %T, want listed", result)
	}
	return listed.Values
}
