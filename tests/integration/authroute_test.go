//go:build integration

package integration_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/db"
)

// TestAuthRouteEndToEndThroughGuest proves an auth-store-touching route runs
// end to end through the guest: GET /api/users reads the auth store's directory
// via the auth service, which the guest builds over the bridged auth GuestStore.
// The response must be byte-identical to the same mux run in-process over the
// real store, and must contain the seeded user.
func TestAuthRouteEndToEndThroughGuest(t *testing.T) {
	ctx := context.Background()
	pool := newPool(t)
	authStore := db.NewAuthStore(pool)

	userID := newUserID(t)
	email := mustAuthEmail(t, "authroute-"+userID.String()+"@example.com")
	if _, matched := authStore.CreateUserCredential(ctx, userID, email, mustAuthPasswordHash(t)).(auth.StoreUserAccepted); !matched {
		t.Fatalf("seed credential rejected")
	}
	token := mintAccessToken(t, appRouteSecret, userID)

	target := "/api/users?query=" + url.QueryEscape(email.String())
	request := func() *http.Request {
		req := httptest.NewRequest("GET", target, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		return req
	}

	bridged, direct := serveRouteBothWays(t, ctx, pool, request)
	assertBridgeMatchesNative(t, bridged, direct)
	if !strings.Contains(string(bridged.Body), userID.String()) {
		t.Errorf("bridge body did not contain the seeded user %s: %q", userID, bridged.Body)
	}
}
