//go:build integration

package integration_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/e6qu/sharecrop/internal/wasibridge/httpbridge"
	"github.com/e6qu/sharecrop/internal/wasibridge/rpc"
	"github.com/e6qu/sharecrop/internal/wasibridge/storehost"
)

// TestGuestRateLimitsPerClientIP covers per-IP rate limiting over the
// production-default WASI guest. Unauthenticated endpoints are limited by the
// direct peer address (clientIP reads r.RemoteAddr), but the request bridge only
// carried method/target/header/body - not RemoteAddr - so in the guest every
// request looked like httptest's hardcoded placeholder and all clients shared a
// single bucket. A burst from one caller would then rate-limit login and
// registration for everyone. The bridge must forward RemoteAddr so each client
// gets its own bucket.
//
// The check bursts past the per-IP capacity from one address (proving the limit
// engages) and then, from a second address, requires a fresh allowance - which
// only holds if the two addresses map to separate buckets. Requests are fired
// concurrently against a cheap endpoint (refresh with no cookie: the rate-limit
// gate runs first, then it returns 401 without touching the database) so total
// throughput stays well above the bucket's 5/sec refill even under CI load,
// which a slow sequential burst cannot guarantee.
func TestGuestRateLimitsPerClientIP(t *testing.T) {
	ctx := context.Background()
	pool := newPool(t)

	guestWASM, err := compileWASIGuest(t, "github.com/e6qu/sharecrop/cmd/sharecrop-wasi-app-guest")
	if err != nil {
		t.Fatalf("compile app guest: %v", err)
	}
	guestPool, err := rpc.NewPool(ctx, guestWASM, storehost.Dispatcher(pool), 6)
	if err != nil {
		t.Fatalf("new pool: %v", err)
	}
	guestPool.WithGuestEnv(map[string]string{"SHARECROP_ACCESS_TOKEN_SECRET": appRouteSecret})
	t.Cleanup(func() { _ = guestPool.Close(ctx) })
	guest := httpbridge.Handler(guestPool)

	// allowedFromBurst fires n concurrent refresh requests from one peer address
	// and returns how many were allowed (not 429). The rate-limit gate runs
	// before the cookie check, so an allowed request is a 401 and a limited one
	// is a 429; nothing hits the database.
	allowedFromBurst := func(remoteAddr string, n int) int {
		var allowed int64
		var wg sync.WaitGroup
		for i := 0; i < n; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				req := httptest.NewRequest("POST", "/api/auth/refresh", nil)
				req.RemoteAddr = remoteAddr
				rec := httptest.NewRecorder()
				guest.ServeHTTP(rec, req)
				if rec.Code != http.StatusTooManyRequests {
					atomic.AddInt64(&allowed, 1)
				}
			}()
		}
		wg.Wait()
		return int(allowed)
	}

	// Address A bursts well past the capacity (20); with concurrent cheap requests
	// the bucket drains despite refill, so many are rejected.
	const burst = 60
	allowedA := allowedFromBurst("10.1.1.1:40000", burst)
	if allowedA >= burst {
		t.Fatalf("all %d requests from one address were allowed - the IP rate limit never engaged", burst)
	}

	// Address B has its own, untouched bucket. A small probe (far below the
	// capacity of 20) should be allowed in full. If RemoteAddr were dropped, B
	// would share A's freshly drained bucket and be limited to near zero. The
	// probe stays well under capacity so it is insensitive to load - the only way
	// it fails is a shared bucket.
	const probe = 5
	allowedB := allowedFromBurst("10.2.2.2:40000", probe)
	if allowedB < probe {
		t.Errorf("second client was allowed only %d/%d requests - clients are sharing one IP bucket (RemoteAddr not forwarded to the guest)", allowedB, probe)
	}
}
