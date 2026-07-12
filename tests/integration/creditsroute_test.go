//go:build integration

package integration_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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

	request := func() *http.Request {
		req := httptest.NewRequest("GET", "/api/credits/balance", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		return req
	}

	bridged, direct := serveRouteBothWays(t, ctx, pool, request)
	assertBridgeMatchesNative(t, bridged, direct)
	// The bridge read the real signup-grant balance through the ledger service.
	if !strings.Contains(string(bridged.Body), "100") {
		t.Errorf("bridge body did not contain the 100-credit signup balance: %q", bridged.Body)
	}
}
