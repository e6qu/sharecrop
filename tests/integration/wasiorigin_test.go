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

	"github.com/e6qu/sharecrop/internal/wasibridge/httpbridge"
	"github.com/e6qu/sharecrop/internal/wasibridge/rpc"
	"github.com/e6qu/sharecrop/internal/wasibridge/storehost"
)

// TestGuestMCPOriginCheckUsesRealHost covers the MCP origin (CSRF) check over
// the production-default WASI guest. The check accepts a request only when the
// Origin header's host equals r.Host. The guest rebuilds the request with
// httptest.NewRequest, which hardcodes r.Host to "example.com", and Go does not
// keep Host in the header map - so unless the bridge forwards it, every
// browser MCP request (browsers always send Origin) is rejected as
// cross-origin, while a request with no Origin slips through. The bridge must
// carry the real Host so same-origin requests are accepted and the CSRF
// protection still rejects a genuinely foreign origin.
func TestGuestMCPOriginCheckUsesRealHost(t *testing.T) {
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

	const host = "sharecrop.test"

	do := func(method, path, authorization, host, origin, body string) *http.Response {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		if authorization != "" {
			req.Header.Set("Authorization", "Bearer "+authorization)
		}
		if host != "" {
			req.Host = host
		}
		if origin != "" {
			req.Header.Set("Origin", origin)
		}
		req.Header.Set("Accept", "application/json, text/event-stream")
		rec := httptest.NewRecorder()
		guest.ServeHTTP(rec, req)
		return rec.Result()
	}

	registerResp := do("POST", "/api/auth/register", "", host, "",
		fmt.Sprintf(`{"email":%q,"password":"correct horse battery staple"}`, uniqueIntegrationEmail(t, "mcp-origin")))
	if registerResp.StatusCode != http.StatusCreated {
		t.Fatalf("register: status %d, want 201", registerResp.StatusCode)
	}
	var registered struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(registerResp.Body).Decode(&registered); err != nil {
		t.Fatalf("decode register response: %v", err)
	}
	credResp := do("POST", "/api/agent-credentials", registered.AccessToken, host, "",
		`{"label":"mcp origin test","scopes":["tasks_read"]}`)
	if credResp.StatusCode != http.StatusCreated {
		t.Fatalf("create agent credential: status %d, want 201", credResp.StatusCode)
	}
	var credential struct {
		Secret string `json:"secret"`
	}
	if err := json.NewDecoder(credResp.Body).Decode(&credential); err != nil {
		t.Fatalf("decode agent credential response: %v", err)
	}

	initialize := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`

	t.Run("same-origin request is accepted", func(t *testing.T) {
		resp := do("POST", "/mcp", credential.Secret, host, "https://"+host, initialize)
		if resp.StatusCode != http.StatusOK {
			t.Errorf("same-origin MCP initialize: status %d, want 200 - the guest is not seeing the real Host", resp.StatusCode)
		}
	})

	t.Run("foreign origin is still rejected", func(t *testing.T) {
		resp := do("POST", "/mcp", credential.Secret, host, "https://evil.example.com", initialize)
		if resp.StatusCode != http.StatusForbidden {
			t.Errorf("cross-origin MCP initialize: status %d, want 403", resp.StatusCode)
		}
	})
}
