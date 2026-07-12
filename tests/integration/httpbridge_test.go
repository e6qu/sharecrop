//go:build integration

package integration_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	httpserver "github.com/e6qu/sharecrop/internal/http"
	"github.com/e6qu/sharecrop/internal/wasibridge/appmux"
	"github.com/e6qu/sharecrop/internal/wasibridge/httpbridge"
	"github.com/e6qu/sharecrop/internal/wasibridge/rpc"
	"github.com/e6qu/sharecrop/internal/wasibridge/storehost"
	"github.com/jackc/pgx/v5/pgxpool"
)

// serveRouteBothWays runs the same request through the native mux (over the db
// stores) and through the full-graph app guest (over the same db pool via the
// store host), returning both responses so a route test can assert they match.
// It centralizes the guest setup every route test shares.
func serveRouteBothWays(t *testing.T, ctx context.Context, pool *pgxpool.Pool, request func() *http.Request) (bridged, direct httpbridge.Response) {
	t.Helper()
	secret := requireAccessTokenSecret(t, appRouteSecret)
	direct = serveDirect(appmux.New(secret, appmuxStores(pool)), request())

	guestWASM, err := compileWASIGuest(t, "github.com/e6qu/sharecrop/cmd/sharecrop-wasi-app-guest")
	if err != nil {
		t.Fatalf("compile app guest: %v", err)
	}
	host, err := rpc.NewHost(ctx, guestWASM, storehost.Dispatcher(pool))
	if err != nil {
		t.Fatalf("new host: %v", err)
	}
	host.WithGuestEnv(map[string]string{"SHARECROP_ACCESS_TOKEN_SECRET": appRouteSecret})
	t.Cleanup(func() { _ = host.Close(ctx) })
	return serveThroughBridge(t, ctx, host, request()), direct
}

// assertBridgeMatchesNative checks the guest's response is 200 and byte-identical
// to the native mux's.
func assertBridgeMatchesNative(t *testing.T, bridged, direct httpbridge.Response) {
	t.Helper()
	if direct.Status != http.StatusOK {
		t.Fatalf("native status = %d, want 200 (body %q)", direct.Status, direct.Body)
	}
	if bridged.Status != direct.Status {
		t.Errorf("status: bridge %d, direct %d", bridged.Status, direct.Status)
	}
	if bridged.Header.Get("Content-Type") != direct.Header.Get("Content-Type") {
		t.Errorf("content-type: bridge %q, direct %q", bridged.Header.Get("Content-Type"), direct.Header.Get("Content-Type"))
	}
	if string(bridged.Body) != string(direct.Body) {
		t.Errorf("body: bridge %q, direct %q", bridged.Body, direct.Body)
	}
}

// TestHTTPBridgeByteIdentical is the Phase 4 checkpoint: a request handled by
// the real internal/http mux running inside a compiled wasip1 guest produces a
// byte-identical response to the same mux run in-process (what cmd/sharecrop
// serve uses). No database is involved - GET /healthz touches no store - so this
// isolates the request/response transport through the guest.
func TestHTTPBridgeByteIdentical(t *testing.T) {
	ctx := context.Background()

	guestWASM, err := compileWASIGuest(t, "github.com/e6qu/sharecrop/cmd/sharecrop-wasi-http-guest")
	if err != nil {
		t.Fatalf("compile http guest: %v", err)
	}
	host, err := rpc.NewHost(ctx, guestWASM, func(context.Context, string, []byte) ([]byte, error) {
		return nil, errors.New("no store bridged for this route")
	})
	if err != nil {
		t.Fatalf("new host: %v", err)
	}
	t.Cleanup(func() { _ = host.Close(ctx) })

	// The same real routing table cmd/sharecrop serve builds.
	directMux := httpserver.New(fstest.MapFS{}, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	cases := []struct {
		name   string
		method string
		target string
	}{
		{"healthz", "GET", "/healthz"},
		{"unknown api route 404s", "GET", "/api/does-not-exist"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			direct := serveDirect(directMux, httptest.NewRequest(testCase.method, testCase.target, nil))
			bridged := serveThroughBridge(t, ctx, host, httptest.NewRequest(testCase.method, testCase.target, nil))

			if bridged.Status != direct.Status {
				t.Errorf("status: bridge %d, direct %d", bridged.Status, direct.Status)
			}
			if bridged.Header.Get("Content-Type") != direct.Header.Get("Content-Type") {
				t.Errorf("content-type: bridge %q, direct %q",
					bridged.Header.Get("Content-Type"), direct.Header.Get("Content-Type"))
			}
			if string(bridged.Body) != string(direct.Body) {
				t.Errorf("body: bridge %q, direct %q", bridged.Body, direct.Body)
			}
		})
	}
}

func serveDirect(mux http.Handler, req *http.Request) httpbridge.Response {
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	result := rec.Result()
	body, _ := io.ReadAll(result.Body)
	return httpbridge.Response{Status: result.StatusCode, Header: result.Header, Body: body}
}

func serveThroughBridge(t *testing.T, ctx context.Context, host *rpc.Host, req *http.Request) httpbridge.Response {
	t.Helper()
	requestBytes, err := httpbridge.EncodeRequest(req)
	if err != nil {
		t.Fatalf("encode request: %v", err)
	}
	responseBytes, err := host.Call(ctx, "http.handle", requestBytes)
	if err != nil {
		t.Fatalf("bridge call: %v", err)
	}
	var response httpbridge.Response
	if err := json.Unmarshal(responseBytes, &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return response
}
