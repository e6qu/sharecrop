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
	"github.com/e6qu/sharecrop/internal/wasibridge/appmux"
	"github.com/e6qu/sharecrop/internal/wasibridge/rpc"
	"github.com/e6qu/sharecrop/internal/wasibridge/storehost"
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
	notificationStore := db.NewNotificationStore(pool)

	userID := newUserID(t)
	email := mustAuthEmail(t, "authroute-"+userID.String()+"@example.com")
	if _, matched := authStore.CreateUserCredential(ctx, userID, email, mustAuthPasswordHash(t)).(auth.StoreUserAccepted); !matched {
		t.Fatalf("seed credential rejected")
	}
	token := mintAccessToken(t, appRouteSecret, userID)
	secret := requireAccessTokenSecret(t, appRouteSecret)

	target := "/api/users?query=" + url.QueryEscape(email.String())
	request := func() *http.Request {
		req := httptest.NewRequest("GET", target, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		return req
	}

	// Native: the same mux the guest builds, over the real db stores.
	nativeMux := appmux.New(secret, authStore, notificationStore)
	direct := serveDirect(nativeMux, request())

	// Bridge: the app guest, with auth.* dispatched to the same db store.
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

	if direct.Status != 200 {
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
	if !strings.Contains(string(bridged.Body), userID.String()) {
		t.Errorf("bridge body did not contain the seeded user %s: %q", userID, bridged.Body)
	}
}
