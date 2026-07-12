//go:build integration

package integration_test

import (
	"context"
	"sort"
	"strings"
	"testing"

	"github.com/e6qu/sharecrop/internal/core"
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

	// db-checks shares one Postgres across every integration test, and List
	// returns all active platform admins globally, so a leaked active grant would
	// break other tests' counts. Revoke at the end (after the subtests below run).
	t.Cleanup(func() { _ = dbStore.Revoke(context.Background(), grantee) })

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

	t.Run("list matches a direct call and contains the granted admin", func(t *testing.T) {
		viaBridge := requireAdminsListed(t, bridgeStore.List(ctx, page))
		direct := requireAdminsListed(t, dbStore.List(ctx, page))
		if adminsKey(viaBridge) != adminsKey(direct) {
			t.Errorf("admin list: bridge %s, direct %s", adminsKey(viaBridge), adminsKey(direct))
		}
		if !containsAdmin(viaBridge, grantee) {
			t.Errorf("bridge list did not contain the granted admin %s", grantee)
		}
	})
}

func adminsKey(records []httpserver.PlatformAdminRecord) string {
	ids := make([]string, 0, len(records))
	for index := range records {
		ids = append(ids, records[index].UserID.String()+":"+records[index].Source)
	}
	sort.Strings(ids)
	return strings.Join(ids, ",")
}

func containsAdmin(records []httpserver.PlatformAdminRecord, userID core.UserID) bool {
	for index := range records {
		if records[index].UserID == userID {
			return true
		}
	}
	return false
}

func requireAdminsListed(t *testing.T, result httpserver.PlatformAdminListResult) []httpserver.PlatformAdminRecord {
	t.Helper()
	listed, matched := result.(httpserver.PlatformAdminsListed)
	if !matched {
		t.Fatalf("list result = %T, want listed", result)
	}
	return listed.Values
}
