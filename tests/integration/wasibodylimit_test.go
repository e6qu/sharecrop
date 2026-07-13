//go:build integration

package integration_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/e6qu/sharecrop/internal/wasibridge/httpbridge"
	"github.com/e6qu/sharecrop/internal/wasibridge/rpc"
	"github.com/e6qu/sharecrop/internal/wasibridge/storehost"
)

// TestGuestBridgeBoundsRequestBody covers the host's request-body handling for
// the production-default WASI hosting. The guest enforces a request-body limit,
// but the host reads the whole body first to frame it, so without a cap the host
// would buffer an arbitrarily large body before the guest ever sees it - an
// unauthenticated memory-pressure vector. The bridge must bound the read and
// reject an oversized body cleanly.
//
// The check sends a body far larger than the limit and requires a 413 rather
// than the 502 the bridge returns when it tries (and fails) to frame an
// over-limit payload - proving the host stopped reading at the cap instead of
// buffering the whole thing.
func TestGuestBridgeBoundsRequestBody(t *testing.T) {
	ctx := context.Background()
	pool := newPool(t)

	guestWASM, err := compileWASIGuest(t, "github.com/e6qu/sharecrop/cmd/sharecrop-wasi-app-guest")
	if err != nil {
		t.Fatalf("compile app guest: %v", err)
	}
	guestPool, err := rpc.NewPool(ctx, guestWASM, storehost.Dispatcher(pool), 2)
	if err != nil {
		t.Fatalf("new pool: %v", err)
	}
	guestPool.WithGuestEnv(map[string]string{"SHARECROP_ACCESS_TOKEN_SECRET": appRouteSecret})
	t.Cleanup(func() { _ = guestPool.Close(ctx) })
	guest := httpbridge.Handler(guestPool)

	// 14 MiB is both far above the 2 MiB body cap and large enough that, framed,
	// it would exceed the bridge's frame limit - so an unbounded host would 502
	// after buffering it all, while a bounded host rejects it at the cap.
	oversized := strings.Repeat("x", 14*1024*1024)
	req := httptest.NewRequest("POST", "/api/auth/register", strings.NewReader(oversized))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	guest.ServeHTTP(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("oversized request body: status %d, want 413 - the host is not bounding the body before framing it", rec.Code)
	}
}
