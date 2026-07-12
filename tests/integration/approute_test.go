//go:build integration

package integration_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/db"
	"github.com/e6qu/sharecrop/internal/wasibridge/appmux"
	"github.com/e6qu/sharecrop/internal/wasibridge/rpc"
	"github.com/e6qu/sharecrop/internal/wasibridge/storehost"
	"github.com/jackc/pgx/v5/pgxpool"
)

// appmuxStores builds the full set of appmux stores over the db pool, so the
// native mux in the route tests is assembled exactly like the guest's - every
// domain service backed by a real Postgres store.
func appmuxStores(pool *pgxpool.Pool) appmux.Stores {
	return appmux.Stores{
		Auth:          db.NewAuthStore(pool),
		Notification:  db.NewNotificationStore(pool),
		Organization:  db.NewOrgStore(pool),
		Task:          db.NewTaskStore(pool),
		Submission:    db.NewSubmissionStore(pool),
		Ledger:        db.NewLedgerStore(pool),
		Agent:         db.NewAgentStore(pool),
		OrgCredential: db.NewOrgCredentialStore(pool),
		Assets:        db.NewCollectibleStore(pool),
		Audit:         db.NewAuditStore(pool),
	}
}

const appRouteSecret = "01234567890123456789012345678901"

// TestAppRouteEndToEndThroughGuest is the Phase 3 + Phase 4 milestone: a real
// authenticated, store-touching route (GET /api/notifications) served entirely
// by the wasip1 guest - the stateless access-token verifier checks the bearer
// token in-guest, and the notification read is bridged back to the host and
// hits real Postgres. The response must be byte-identical to the same mux run
// in-process against the same store, and must actually contain the seeded
// notification (so we know the bridge read real data, not an empty list).
func TestAppRouteEndToEndThroughGuest(t *testing.T) {
	ctx := context.Background()
	pool := newPool(t)
	notificationStore := db.NewNotificationStore(pool)

	recipient := createUser(t, pool, "app-recipient")
	actor := createUser(t, pool, "app-actor")
	seeded := seedNotification(t, ctx, notificationStore, recipient, actor)
	token := mintAccessToken(t, appRouteSecret, recipient)

	secret := requireAccessTokenSecret(t, appRouteSecret)

	// Native: the same mux the guest builds, over the real db stores, in-process.
	nativeMux := appmux.New(secret, appmuxStores(pool))
	direct := serveDirect(nativeMux, authedRequest(token))

	// Bridge: the app guest, with every store dispatched to the same db pool and
	// the secret handed to the guest via WASI env.
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

	bridged := serveThroughBridge(t, ctx, host, authedRequest(token))

	if direct.Status != http.StatusOK {
		t.Fatalf("native status = %d, want 200 (body %q)", direct.Status, direct.Body)
	}
	if bridged.Status != direct.Status {
		t.Errorf("status: bridge %d, direct %d", bridged.Status, direct.Status)
	}
	if bridged.Header.Get("Content-Type") != direct.Header.Get("Content-Type") {
		t.Errorf("content-type: bridge %q, direct %q",
			bridged.Header.Get("Content-Type"), direct.Header.Get("Content-Type"))
	}
	if string(bridged.Body) != string(direct.Body) {
		t.Errorf("body: bridge %q, direct %q", bridged.Body, direct.Body)
	}
	// The bridge actually read the seeded row, not an empty list.
	if !strings.Contains(string(bridged.Body), seeded.ID.String()) {
		t.Errorf("bridge body did not contain the seeded notification %s: %q", seeded.ID, bridged.Body)
	}
}

func authedRequest(token string) *http.Request {
	req := httptest.NewRequest("GET", "/api/notifications?limit=50&offset=0", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	return req
}

func mintAccessToken(t *testing.T, secret string, userID core.UserID) string {
	t.Helper()
	signed, matched := auth.SignAccessToken(requireAccessTokenSecret(t, secret), auth.UserSubject{ID: userID}, time.Now().UTC()).(auth.AccessTokenAccepted)
	if !matched {
		t.Fatalf("sign access token rejected")
	}
	return signed.Value.String()
}

func requireAccessTokenSecret(t *testing.T, raw string) auth.AccessTokenSecret {
	t.Helper()
	accepted, matched := auth.NewAccessTokenSecret(raw).(auth.AccessTokenSecretAccepted)
	if !matched {
		t.Fatalf("access token secret rejected")
	}
	return accepted.Value
}
