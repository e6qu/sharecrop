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
	"testing"
	"time"

	"github.com/e6qu/sharecrop/internal/wasibridge/httpbridge"
	"github.com/e6qu/sharecrop/internal/wasibridge/rpc"
	"github.com/e6qu/sharecrop/internal/wasibridge/storehost"
)

// TestGuestPoolSSEDoesNotExhaustPool guards the production-default WASI hosting
// path against an MCP-stream denial of service. The RPC bridge runs one unit of
// work per request and returns its buffered response, so it cannot hold a
// streaming connection open. The MCP SSE endpoint (GET /mcp) would otherwise
// block forever waiting for live events, pinning its guest instance until the
// pool is exhausted and the whole server stops answering - a handful of MCP
// clients (which routinely open the stream) or one malicious caller could wedge
// it. The guest must instead return the replayed events and let the client
// reconnect.
//
// The check: with a small pool, open more concurrent SSE streams than there are
// instances AND a plain request, and require every one to finish promptly. If a
// stream pinned its instance, the plain request could never get one and this
// would block until the deadline.
func TestGuestPoolSSEDoesNotExhaustPool(t *testing.T) {
	ctx := context.Background()
	pool := newPool(t)

	guestWASM, err := compileWASIGuest(t, "github.com/e6qu/sharecrop/cmd/sharecrop-wasi-app-guest")
	if err != nil {
		t.Fatalf("compile app guest: %v", err)
	}
	const poolSize = 2
	guestPool, err := rpc.NewPool(ctx, guestWASM, storehost.Dispatcher(pool), poolSize)
	if err != nil {
		t.Fatalf("new pool: %v", err)
	}
	guestPool.WithGuestEnv(map[string]string{"SHARECROP_ACCESS_TOKEN_SECRET": appRouteSecret})
	t.Cleanup(func() { _ = guestPool.Close(ctx) })
	guest := httpbridge.Handler(guestPool)

	do := func(method, path, authorization, sessionID, accept, body string) *http.Response {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		if authorization != "" {
			req.Header.Set("Authorization", "Bearer "+authorization)
		}
		if accept != "" {
			req.Header.Set("Accept", accept)
		}
		if sessionID != "" {
			req.Header.Set("Mcp-Session-Id", sessionID)
		}
		rec := httptest.NewRecorder()
		guest.ServeHTTP(rec, req)
		return rec.Result()
	}

	registerResp := do("POST", "/api/auth/register", "", "", "",
		fmt.Sprintf(`{"email":%q,"password":"correct horse battery staple"}`, uniqueIntegrationEmail(t, "mcp-sse")))
	if registerResp.StatusCode != http.StatusCreated {
		t.Fatalf("register: status %d, want 201", registerResp.StatusCode)
	}
	var registered struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(registerResp.Body).Decode(&registered); err != nil {
		t.Fatalf("decode register response: %v", err)
	}
	credResp := do("POST", "/api/agent-credentials", registered.AccessToken, "", "",
		`{"label":"mcp sse test","scopes":["tasks_read"]}`)
	if credResp.StatusCode != http.StatusCreated {
		t.Fatalf("create agent credential: status %d, want 201", credResp.StatusCode)
	}
	var credential struct {
		Secret string `json:"secret"`
	}
	if err := json.NewDecoder(credResp.Body).Decode(&credential); err != nil {
		t.Fatalf("decode agent credential response: %v", err)
	}
	initResp := do("POST", "/mcp", credential.Secret, "", "application/json, text/event-stream",
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`)
	if initResp.StatusCode != http.StatusOK {
		t.Fatalf("mcp initialize: status %d, want 200", initResp.StatusCode)
	}
	sessionID := initResp.Header.Get("Mcp-Session-Id")
	if sessionID == "" {
		t.Fatal("mcp initialize did not return an Mcp-Session-Id")
	}

	// Concurrent work: several SSE streams (more than the pool has instances)
	// plus a plain tools/list, all through the same pool.
	type outcome struct {
		label  string
		status int
		ok     bool // for tools/list: response carried the tool list
	}
	done := make(chan outcome, 8)

	const sseStreams = 4 // 2x the pool
	for i := 0; i < sseStreams; i++ {
		go func(index int) {
			resp := do("GET", "/mcp", credential.Secret, sessionID, "text/event-stream", "")
			_, _ = io.Copy(io.Discard, resp.Body)
			done <- outcome{label: fmt.Sprintf("sse-%d", index), status: resp.StatusCode, ok: true}
		}(i)
	}
	go func() {
		resp := do("POST", "/mcp", credential.Secret, sessionID, "application/json, text/event-stream",
			`{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}`)
		body, _ := io.ReadAll(resp.Body)
		done <- outcome{label: "tools/list", status: resp.StatusCode, ok: strings.Contains(string(body), `"tools"`)}
	}()

	deadline := time.After(20 * time.Second)
	for completed := 0; completed < sseStreams+1; completed++ {
		select {
		case result := <-done:
			if result.status != http.StatusOK {
				t.Errorf("%s: status %d, want 200", result.label, result.status)
			}
			if !result.ok {
				t.Errorf("%s: response was not the expected shape", result.label)
			}
		case <-deadline:
			t.Fatalf("only %d of %d requests completed - an SSE stream is pinning a guest instance and starving the pool", completed, sseStreams+1)
		}
	}
}
