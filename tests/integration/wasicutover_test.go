//go:build integration

package integration_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/e6qu/sharecrop/internal/db"
	"github.com/e6qu/sharecrop/internal/wasibridge/httpbridge"
	"github.com/e6qu/sharecrop/internal/wasibridge/rpc"
	"github.com/e6qu/sharecrop/internal/wasibridge/storehost"
)

// TestWASICutoverServesStaticAndGuest mirrors the handler the production cutover
// (cmd/sharecrop serve with SHARECROP_WASI_GUEST set) assembles: a pool of the
// app guest serves the dynamic routes over the bridged stores, while static
// assets and the SPA shell are served host-side. It checks both halves: GET /
// returns the host's SPA shell, and GET /api/notifications is handled by the
// guest and returns real Postgres data.
func TestWASICutoverServesStaticAndGuest(t *testing.T) {
	ctx := context.Background()
	pool := newPool(t)
	notificationStore := db.NewNotificationStore(pool)

	recipient := createUser(t, pool, "cutover-recipient")
	actor := createUser(t, pool, "cutover-actor")
	seeded := seedNotification(t, ctx, notificationStore, recipient, actor)
	token := mintAccessToken(t, appRouteSecret, recipient)

	guestWASM, err := compileWASIGuest(t, "github.com/e6qu/sharecrop/cmd/sharecrop-wasi-app-guest")
	if err != nil {
		t.Fatalf("compile app guest: %v", err)
	}
	guestPool, err := rpc.NewPool(ctx, guestWASM, storehost.Dispatcher(pool), 3)
	if err != nil {
		t.Fatalf("new pool: %v", err)
	}
	guestPool.WithGuestEnv(map[string]string{"SHARECROP_ACCESS_TOKEN_SECRET": appRouteSecret})
	t.Cleanup(func() { _ = guestPool.Close(ctx) })
	guest := httpbridge.Handler(guestPool)

	const shell = "<html><body>sharecrop app shell</body></html>"
	// Assets sit at the FS root; the /static/ prefix is stripped before lookup,
	// matching how web.StaticFiles() is served.
	staticFiles := fstest.MapFS{
		"index.html": &fstest.MapFile{Data: []byte(shell)},
		"app.js":     &fstest.MapFile{Data: []byte("console.log('app')")},
	}

	// The same split cmd/sharecrop's serveThroughWASIGuest builds.
	mux := http.NewServeMux()
	mux.Handle("/api/", guest)
	mux.Handle("/mcp", guest)
	mux.Handle("/healthz", guest)
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFiles))))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		data, _ := staticFiles.ReadFile("index.html")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(data)
	})

	t.Run("the SPA shell and static assets are served host-side", func(t *testing.T) {
		for _, tc := range []struct{ path, want string }{
			{"/", shell},
			{"/some/deep/link", shell},
			{"/static/app.js", "console.log('app')"},
		} {
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, httptest.NewRequest("GET", tc.path, nil))
			if rec.Code != http.StatusOK {
				t.Errorf("GET %s = %d, want 200", tc.path, rec.Code)
			}
			if !strings.Contains(rec.Body.String(), tc.want) {
				t.Errorf("GET %s body = %q, want to contain %q", tc.path, rec.Body.String(), tc.want)
			}
		}
	})

	t.Run("dynamic routes are handled by the guest against Postgres", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/notifications?limit=50&offset=0", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("GET /api/notifications = %d, want 200 (body %q)", rec.Code, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), seeded.ID.String()) {
			t.Errorf("guest response did not contain the seeded notification %s: %q", seeded.ID, rec.Body.String())
		}
	})
}
