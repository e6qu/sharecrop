//go:build integration

package integration_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/e6qu/sharecrop/internal/wasibridge/httpbridge"
	"github.com/e6qu/sharecrop/internal/wasibridge/rpc"
	"github.com/e6qu/sharecrop/internal/wasibridge/storehost"
)

// TestGuestPoolServesMCPAcrossInstances covers the MCP transport over the
// production-default WASI guest pool. MCP is session-stateful: initialize mints
// an Mcp-Session-Id, and every later request carries it. But each request is
// served by whichever pooled guest instance is free, and a session created on
// one instance is absent from another instance's in-memory session map. It only
// works because the guest wires its MCP session persistence through the store
// bridge (NewGuestMCPSessionPersistence), so a missing local session rehydrates
// from the shared Postgres session store.
//
// This exercises exactly that: initialize once, then fire many tools/list calls
// concurrently through a small pool so instances that never saw the initialize
// must rehydrate the session. The http_e2e MCP suite only runs against the
// native mux, so without this the WASI MCP path had no automated coverage.
func TestGuestPoolServesMCPAcrossInstances(t *testing.T) {
	ctx := context.Background()
	pool := newPool(t)

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

	do := func(method, path, authorization, sessionID, body string) *http.Response {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		if authorization != "" {
			req.Header.Set("Authorization", "Bearer "+authorization)
		}
		if strings.HasSuffix(path, "/mcp") {
			req.Header.Set("Accept", "application/json, text/event-stream")
		}
		if sessionID != "" {
			req.Header.Set("Mcp-Session-Id", sessionID)
		}
		rec := httptest.NewRecorder()
		guest.ServeHTTP(rec, req)
		return rec.Result()
	}

	// Register an account and issue an agent credential MCP authenticates with.
	registerResp := do("POST", "/api/auth/register", "", "",
		fmt.Sprintf(`{"email":%q,"password":"correct horse battery staple"}`, uniqueIntegrationEmail(t, "mcp-pool")))
	if registerResp.StatusCode != http.StatusCreated {
		t.Fatalf("register: status %d, want 201", registerResp.StatusCode)
	}
	var registered struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(registerResp.Body).Decode(&registered); err != nil {
		t.Fatalf("decode register response: %v", err)
	}

	credResp := do("POST", "/api/agent-credentials", registered.AccessToken, "",
		`{"label":"mcp pool test","scopes":["tasks_read"]}`)
	if credResp.StatusCode != http.StatusCreated {
		t.Fatalf("create agent credential: status %d, want 201", credResp.StatusCode)
	}
	var credential struct {
		Secret string `json:"secret"`
	}
	if err := json.NewDecoder(credResp.Body).Decode(&credential); err != nil {
		t.Fatalf("decode agent credential response: %v", err)
	}

	// Initialize the MCP session on one instance.
	initResp := do("POST", "/mcp", credential.Secret, "",
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`)
	if initResp.StatusCode != http.StatusOK {
		t.Fatalf("mcp initialize: status %d, want 200", initResp.StatusCode)
	}
	sessionID := initResp.Header.Get("Mcp-Session-Id")
	if sessionID == "" {
		t.Fatal("mcp initialize did not return an Mcp-Session-Id")
	}

	// Fire many tools/list calls concurrently so pooled instances that never saw
	// the initialize must rehydrate the session from the shared store. Any that
	// cannot would return an error instead of the tool list.
	const calls = 12
	type result struct {
		status int
		body   string
	}
	results := make([]result, calls)
	var wg sync.WaitGroup
	for i := 0; i < calls; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			resp := do("POST", "/mcp", credential.Secret, sessionID,
				fmt.Sprintf(`{"jsonrpc":"2.0","id":%d,"method":"tools/list","params":{}}`, index+2))
			body, _ := io.ReadAll(resp.Body)
			results[index] = result{status: resp.StatusCode, body: string(body)}
		}(i)
	}
	wg.Wait()

	for index, res := range results {
		if res.status != http.StatusOK {
			t.Errorf("tools/list call %d: status %d, want 200 (body %q)", index, res.status, res.body)
			continue
		}
		if !strings.Contains(res.body, `"tools"`) {
			t.Errorf("tools/list call %d: response missing tool list - session did not rehydrate on this instance (body %q)", index, res.body)
		}
	}
}
