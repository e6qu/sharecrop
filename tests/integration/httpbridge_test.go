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
	"github.com/e6qu/sharecrop/internal/wasibridge/httpbridge"
	"github.com/e6qu/sharecrop/internal/wasibridge/rpc"
)

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
			direct := serveDirect(directMux, testCase.method, testCase.target)
			bridged := serveThroughBridge(t, ctx, host, testCase.method, testCase.target)

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

func serveDirect(mux http.Handler, method, target string) httpbridge.Response {
	req := httptest.NewRequest(method, target, nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	result := rec.Result()
	body, _ := io.ReadAll(result.Body)
	return httpbridge.Response{Status: result.StatusCode, Header: result.Header, Body: body}
}

func serveThroughBridge(t *testing.T, ctx context.Context, host *rpc.Host, method, target string) httpbridge.Response {
	t.Helper()
	requestBytes, err := httpbridge.EncodeRequest(httptest.NewRequest(method, target, nil))
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
