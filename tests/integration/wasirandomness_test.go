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

	"github.com/e6qu/sharecrop/internal/wasibridge/httpbridge"
	"github.com/e6qu/sharecrop/internal/wasibridge/rpc"
	"github.com/e6qu/sharecrop/internal/wasibridge/storehost"
)

// TestGuestPoolDrawsUniqueRandomnessPerInstance guards the production-default
// WASI hosting path against a shared-randomness regression. wazero's module
// config defaults to a DETERMINISTIC random source (seeded zeros) for
// reproducible sandboxing, so unless the host wires crypto/rand.Reader into
// every guest instance, all pooled instances draw the same crypto/rand stream.
// Every instance would then mint identical security-critical bytes - password
// salts, refresh/receipt tokens - and identical UUIDv7 ids whenever two
// instances align on the same millisecond, which surfaces as intermittent
// primary-key violations on insert.
//
// The check: register many accounts concurrently through a pool of fewer
// instances than there are registrations, so several instances each serve their
// FIRST register from stream position zero. A refresh token is 32 raw random
// bytes with no timestamp, so two instances at the same stream position emit
// byte-identical tokens deterministically. Requiring every registration to
// succeed with a distinct refresh cookie fails hard under the shared-stream bug
// and passes only when each instance has its own entropy source.
func TestGuestPoolDrawsUniqueRandomnessPerInstance(t *testing.T) {
	ctx := context.Background()
	pool := newPool(t)

	guestWASM, err := compileWASIGuest(t, "github.com/e6qu/sharecrop/cmd/sharecrop-wasi-app-guest")
	if err != nil {
		t.Fatalf("compile app guest: %v", err)
	}
	const poolSize = 4
	guestPool, err := rpc.NewPool(ctx, guestWASM, storehost.Dispatcher(pool), poolSize)
	if err != nil {
		t.Fatalf("new pool: %v", err)
	}
	guestPool.WithGuestEnv(map[string]string{"SHARECROP_ACCESS_TOKEN_SECRET": appRouteSecret})
	t.Cleanup(func() { _ = guestPool.Close(ctx) })
	guest := httpbridge.Handler(guestPool)

	const registrations = 16

	type outcome struct {
		status       int
		refreshToken string
		body         string
	}
	outcomes := make([]outcome, registrations)
	var wg sync.WaitGroup
	for i := 0; i < registrations; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			body := fmt.Sprintf(
				`{"email":"randomness-%d@example.com","password":"correct horse battery staple"}`,
				index,
			)
			req := httptest.NewRequest("POST", "/api/auth/register", strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			guest.ServeHTTP(rec, req)

			refresh := ""
			for _, cookie := range rec.Result().Cookies() {
				if cookie.Name == "sharecrop_refresh_token" {
					refresh = cookie.Value
				}
			}
			outcomes[index] = outcome{status: rec.Code, refreshToken: refresh, body: rec.Body.String()}
		}(i)
	}
	wg.Wait()

	seen := make(map[string]int, registrations)
	for index, result := range outcomes {
		if result.status != http.StatusCreated {
			t.Errorf("registration %d: status %d, want 201 (body %q)", index, result.status, result.body)
			continue
		}
		if result.refreshToken == "" {
			t.Errorf("registration %d: no refresh cookie was set", index)
			continue
		}
		if prior, clash := seen[result.refreshToken]; clash {
			t.Errorf(
				"registrations %d and %d minted an identical refresh token - guest instances are sharing a deterministic random stream",
				prior, index,
			)
			continue
		}
		seen[result.refreshToken] = index
	}
}
