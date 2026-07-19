//go:build integration

package integration_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/e6qu/sharecrop/internal/wasibridge/httpbridge"
	"github.com/e6qu/sharecrop/internal/wasibridge/rpc"
	"github.com/e6qu/sharecrop/internal/wasibridge/storehost"
)

// TestGuestPoolUsesRealWallClock guards the production-default WASI hosting path
// against a frozen-clock regression. wazero's module config defaults to a FAKE
// clock (a fixed 2022-01-01 walltime), so unless the host wires its real
// walltime into every guest instance, a timestamp the guest computes with
// time.Now() is written as that frozen epoch. Audit events are stamped guest
// side, so their created_at would silently read 2022-01-01 for every event -
// wrong timestamps and a broken time ordering in the audit log.
//
// The check: drive a real request through the app-guest pool that records an
// audit event (creating a privacy request emits privacy_request_created), then
// read that event's stored created_at back and require it to sit near real wall
// time rather than the fake epoch.
func TestGuestPoolUsesRealWallClock(t *testing.T) {
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

	// Register an account through the guest and keep its access token.
	registerBody := fmt.Sprintf(`{"email":%q,"password":"correct horse battery staple"}`, uniqueIntegrationEmail(t, "guest-clock"))
	registerReq := httptest.NewRequest("POST", "/api/auth/register", strings.NewReader(registerBody))
	registerReq.Header.Set("Content-Type", "application/json")
	registerRec := httptest.NewRecorder()
	guest.ServeHTTP(registerRec, registerReq)
	if registerRec.Code != http.StatusCreated {
		t.Fatalf("register: status %d, want 201 (body %q)", registerRec.Code, registerRec.Body.String())
	}
	var registered struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(registerRec.Body.Bytes(), &registered); err != nil {
		t.Fatalf("decode register response: %v", err)
	}

	// Creating a privacy request records a privacy_request_created audit event,
	// whose created_at the guest stamps with time.Now().
	before := time.Now().Add(-time.Hour)
	privacyReq := httptest.NewRequest("POST", "/api/privacy-requests", strings.NewReader(`{"kind":"data_export"}`))
	privacyReq.Header.Set("Content-Type", "application/json")
	privacyReq.Header.Set("Authorization", "Bearer "+registered.AccessToken)
	privacyRec := httptest.NewRecorder()
	guest.ServeHTTP(privacyRec, privacyReq)
	if privacyRec.Code != http.StatusCreated {
		t.Fatalf("create privacy request: status %d, want 201 (body %q)", privacyRec.Code, privacyRec.Body.String())
	}
	after := time.Now().Add(time.Hour)

	var createdAt time.Time
	if err := pool.QueryRow(ctx,
		"select created_at from audit_events where action = $1 order by created_at desc limit 1",
		"privacy_request_created",
	).Scan(&createdAt); err != nil {
		t.Fatalf("read audit event created_at: %v", err)
	}

	// The fake wazero walltime is fixed at 2022-01-01, decades before `before`.
	if createdAt.Before(before) || createdAt.After(after) {
		t.Errorf(
			"guest-stamped audit created_at = %s, want within [%s, %s] - the guest is using wazero's fake wall clock",
			createdAt.UTC().Format(time.RFC3339), before.UTC().Format(time.RFC3339), after.UTC().Format(time.RFC3339),
		)
	}
}
