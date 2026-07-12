//go:build integration

package integration_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/e6qu/sharecrop/internal/db"
	httpserver "github.com/e6qu/sharecrop/internal/http"
)

// TestSavedViewRouteEndToEndThroughGuest proves the bridged RuntimeState service
// works over a real route: GET /api/saved-queue-views reads the saved-queue-view
// store via the service the guest now builds over the bridged GuestStore (not a
// per-instance in-memory copy). The response must be 200, byte-identical to the
// native mux, and contain a view seeded directly in Postgres - so we know the
// guest read shared db state through the bridge.
func TestSavedViewRouteEndToEndThroughGuest(t *testing.T) {
	ctx := context.Background()
	pool := newPool(t)

	owner := createUser(t, pool, "savedview-route-owner")
	if _, matched := db.NewSavedQueueViewStore(pool).Upsert(ctx, httpserver.SavedQueueView{
		UserID: owner,
		Scope:  "team_work",
		Name:   "Bridged view",
		Sort:   "newest",
	}).(httpserver.SavedQueueViewSaved); !matched {
		t.Fatalf("seed saved view rejected")
	}
	token := mintAccessToken(t, appRouteSecret, owner)

	request := func() *http.Request {
		req := httptest.NewRequest("GET", "/api/saved-queue-views?scope=team_work", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		return req
	}

	bridged, direct := serveRouteBothWays(t, ctx, pool, request)
	assertBridgeMatchesNative(t, bridged, direct)
	if !strings.Contains(string(bridged.Body), "Bridged view") {
		t.Errorf("bridge body did not contain the seeded saved view: %q", bridged.Body)
	}
}
