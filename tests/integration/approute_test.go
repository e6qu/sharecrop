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
	"github.com/jackc/pgx/v5/pgxpool"
)

// appmuxStores builds the full set of appmux stores over the db pool, so the
// native mux in the route tests is assembled exactly like the guest's - every
// domain service backed by a real Postgres store.
func appmuxStores(pool *pgxpool.Pool) appmux.Stores {
	return appmux.Stores{
		Auth:            db.NewAuthStore(pool),
		Notification:    db.NewNotificationStore(pool),
		Organization:    db.NewOrgStore(pool),
		Task:            db.NewTaskStore(pool),
		Submission:      db.NewSubmissionStore(pool),
		Ledger:          db.NewLedgerStore(pool),
		Agent:           db.NewAgentStore(pool),
		OrgCredential:   db.NewOrgCredentialStore(pool),
		Assets:          db.NewCollectibleStore(pool),
		Audit:           db.NewAuditStore(pool),
		SavedQueueViews: db.NewSavedQueueViewStore(pool),
		PlatformAdmins:  db.NewPlatformAdminStore(pool, map[string]bool{}),
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

	bridged, direct := serveRouteBothWays(t, ctx, pool, func() *http.Request { return authedRequest(token) })
	assertBridgeMatchesNative(t, bridged, direct)
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
