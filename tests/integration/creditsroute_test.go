//go:build integration

package integration_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/e6qu/sharecrop/internal/wasibridge/appmux"
	"github.com/e6qu/sharecrop/internal/wasibridge/rpc"
	"github.com/e6qu/sharecrop/internal/wasibridge/storehost"
)

// TestCreditsRouteEndToEndThroughGuest proves a route backed by a service other
// than auth/notification runs end to end through the full-graph guest:
// GET /api/credits/balance reads the ledger balance via the ledger service,
// which the guest builds over the bridged ledger GuestStore. A freshly created
// user holds the 100-credit signup grant, so the response must be 200, byte-
// identical to the native mux, and contain that balance.
func TestCreditsRouteEndToEndThroughGuest(t *testing.T) {
	ctx := context.Background()
	pool := newPool(t)

	user := createUser(t, pool, "credits-user")
	token := mintAccessToken(t, appRouteSecret, user)
	secret := requireAccessTokenSecret(t, appRouteSecret)

	request := func() *http.Request {
		req := httptest.NewRequest("GET", "/api/credits/balance", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		return req
	}

	// Native: the same mux the guest builds, over the real db stores.
	nativeMux := appmux.New(secret, appmuxStores(pool))
	direct := serveDirect(nativeMux, request())

	// Bridge: the full-graph app guest, every store dispatched to the same pool.
	guestWASM, err := compileWASIGuest(t, "github.com/e6qu/sharecrop/cmd/sharecrop-wasi-app-guest")
	if err != nil {
		t.Fatalf("compile app guest: %v", err)
	}
	host, err := rpc.NewHost(ctx, guestWASM, storehost.Dispatcher(pool))
	if err != nil {
		t.Fatalf("new host: %v", err)
	}
	host.WithGuestEnv(map[string]string{"SHARECROP_ACCESS_TOKEN_SECRET": appRouteSecret})
	t.Cleanup(func() { _ = host.Close(ctx) })

	bridged := serveThroughBridge(t, ctx, host, request())

	if direct.Status != http.StatusOK {
		t.Fatalf("native status = %d, want 200 (body %q)", direct.Status, direct.Body)
	}
	if bridged.Status != direct.Status {
		t.Errorf("status: bridge %d, direct %d", bridged.Status, direct.Status)
	}
	if bridged.Header.Get("Content-Type") != direct.Header.Get("Content-Type") {
		t.Errorf("content-type: bridge %q, direct %q", bridged.Header.Get("Content-Type"), direct.Header.Get("Content-Type"))
	}
	if string(bridged.Body) != string(direct.Body) {
		t.Errorf("body: bridge %q, direct %q", bridged.Body, direct.Body)
	}
	// The bridge read the real signup-grant balance through the ledger service.
	if !strings.Contains(string(bridged.Body), "100") {
		t.Errorf("bridge body did not contain the 100-credit signup balance: %q", bridged.Body)
	}
}
