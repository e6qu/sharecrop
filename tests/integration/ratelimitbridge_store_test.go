//go:build integration

package integration_test

import (
	"context"
	"testing"

	"github.com/e6qu/sharecrop/internal/db"
	httpserver "github.com/e6qu/sharecrop/internal/http"
	"github.com/e6qu/sharecrop/internal/wasibridge/ratelimitbridge"
	"github.com/e6qu/sharecrop/internal/wasibridge/rpc"
)

// TestRateLimitBridgeDualRun exercises the rate limiter - a hand-written bridge
// (Allow takes no context and returns a bare bool, so it isn't generated) -
// through the compiled wasip1 guest + host bridge. It drains one key's token
// bucket to prove Allow enforces the shared Postgres budget end to end, and
// checks StorageKind matches a direct call. A unique key keeps the bucket private
// so nothing contaminates the shared db-checks database.
func TestRateLimitBridgeDualRun(t *testing.T) {
	ctx := context.Background()
	pool := newPool(t)
	ipLimiter := db.NewRateLimiter(pool, "ip", httpserver.IPRateCapacity, httpserver.IPRateRefillPerSec)
	subjectLimiter := db.NewRateLimiter(pool, "subject", httpserver.MCPRateCapacity, httpserver.MCPRateRefillPerSec)

	guestWASM, err := compileWASIGuest(t, "github.com/e6qu/sharecrop/cmd/sharecrop-wasi-store-guest")
	if err != nil {
		t.Fatalf("compile store guest: %v", err)
	}
	host, err := rpc.NewHost(ctx, guestWASM, func(ctx context.Context, method string, args []byte) ([]byte, error) {
		return ratelimitbridge.Dispatch(ctx, ipLimiter, subjectLimiter, method, args)
	})
	if err != nil {
		t.Fatalf("new host: %v", err)
	}
	t.Cleanup(func() { _ = host.Close(ctx) })
	bridgeLimiter := ratelimitbridge.NewGuestRateLimiter(func(method string, args []byte) ([]byte, error) {
		return host.Call(ctx, method, args)
	}, "ip")

	key := "ratelimit-bridge-" + newAuditEventID(t).String()

	t.Run("allow enforces the shared token budget through the bridge", func(t *testing.T) {
		// The bucket refills (IPRateRefillPerSec/sec), so the exact allow/deny
		// boundary shifts with per-call latency; bursting well past capacity makes
		// at least one denial certain while the first call on a fresh bucket is
		// always allowed. That proves the bridge forwards Allow to the shared store.
		if !bridgeLimiter.Allow(key) {
			t.Fatalf("bridge Allow denied the first call on a fresh bucket")
		}
		allowed, denied := 1, 0
		for i := 0; i < 3*httpserver.IPRateCapacity; i++ {
			if bridgeLimiter.Allow(key) {
				allowed++
			} else {
				denied++
			}
		}
		if denied == 0 {
			t.Errorf("bridge Allow never denied over %d calls to one key (allowed %d)", 3*httpserver.IPRateCapacity+1, allowed)
		}
	})

	t.Run("storage kind matches a direct call", func(t *testing.T) {
		if bridgeLimiter.StorageKind() != ipLimiter.StorageKind() {
			t.Errorf("storage kind: bridge %q, direct %q", bridgeLimiter.StorageKind(), ipLimiter.StorageKind())
		}
		if bridgeLimiter.ActiveBuckets() < 1 {
			t.Errorf("bridge reported %d active buckets after draining one, want >= 1", bridgeLimiter.ActiveBuckets())
		}
	})
}
