//go:build integration

package integration_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/e6qu/sharecrop/internal/db"
	"github.com/e6qu/sharecrop/internal/wasibridge/httpbridge"
	"github.com/e6qu/sharecrop/internal/wasibridge/rpc"
	"github.com/e6qu/sharecrop/internal/wasibridge/storehost"
)

// TestPoolHandlerServesConcurrentRoutes drives the exact production path - the
// pooled app host: rpc.Pool over the app guest (the full mux) behind
// httpbridge.Handler, with the store dispatched to real Postgres. Many
// concurrent authenticated GET /api/notifications requests, one per recipient
// with a distinct seeded notification, run through a pool of fewer guest
// instances than there are requests. Each response must be 200 and contain that
// recipient's own notification, proving the pooled HTTP path serves concurrent
// requests without cross-talk.
func TestPoolHandlerServesConcurrentRoutes(t *testing.T) {
	ctx := context.Background()
	pool := newPool(t)
	notificationStore := db.NewNotificationStore(pool)
	secret := appRouteSecret

	const recipients = 16
	const poolSize = 4

	type client struct {
		token    string
		notifyID string
	}
	clients := make([]client, recipients)
	for i := range clients {
		recipient := createUser(t, pool, fmt.Sprintf("poolroute-recipient-%d", i))
		actor := createUser(t, pool, fmt.Sprintf("poolroute-actor-%d", i))
		seeded := seedNotification(t, ctx, notificationStore, recipient, actor)
		clients[i] = client{token: mintAccessToken(t, secret, recipient), notifyID: seeded.ID.String()}
	}

	guestWASM, err := compileWASIGuest(t, "github.com/e6qu/sharecrop/cmd/sharecrop-wasi-app-guest")
	if err != nil {
		t.Fatalf("compile app guest: %v", err)
	}
	rpcPool, err := rpc.NewPool(ctx, guestWASM, storehost.Dispatcher(pool), poolSize)
	if err != nil {
		t.Fatalf("new pool: %v", err)
	}
	rpcPool.WithGuestEnv(map[string]string{"SHARECROP_ACCESS_TOKEN_SECRET": secret})
	t.Cleanup(func() { _ = rpcPool.Close(ctx) })

	handler := httpbridge.Handler(rpcPool)

	var wg sync.WaitGroup
	failures := make(chan string, recipients)
	for i := range clients {
		wg.Add(1)
		go func(c client) {
			defer wg.Done()
			req := httptest.NewRequest("GET", "/api/notifications?limit=50&offset=0", nil)
			req.Header.Set("Authorization", "Bearer "+c.token)
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, req)
			if recorder.Code != http.StatusOK {
				failures <- fmt.Sprintf("notify %s: status %d, want 200 (body %q)", c.notifyID, recorder.Code, recorder.Body.String())
				return
			}
			if !strings.Contains(recorder.Body.String(), c.notifyID) {
				failures <- fmt.Sprintf("notify %s: response did not contain the recipient's own notification (cross-talk): %q", c.notifyID, recorder.Body.String())
			}
		}(clients[i])
	}
	wg.Wait()
	close(failures)

	for message := range failures {
		t.Error(message)
	}
}
