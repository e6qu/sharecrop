package main

import (
	"context"
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"golang.org/x/net/http2"

	"github.com/e6qu/sharecrop/internal/app"
)

func healthzTestMux() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok " + r.Proto))
	})
	return mux
}

// TestH2CHandlerServesCleartextHTTP2 verifies the SHARECROP_HTTP_PROTOCOL=h2c
// wrapping end to end: an HTTP/2 client with prior knowledge (AllowHTTP over
// a plain TCP dial, no TLS) gets a 200 from /healthz over HTTP/2.0.
func TestH2CHandlerServesCleartextHTTP2(t *testing.T) {
	server := httptest.NewServer(wrapHTTPProtocol(healthzTestMux(), app.HTTPProtocolH2C))
	defer server.Close()

	client := &http.Client{Transport: &http2.Transport{
		AllowHTTP: true,
		DialTLSContext: func(ctx context.Context, network string, address string, _ *tls.Config) (net.Conn, error) {
			var dialer net.Dialer
			return dialer.DialContext(ctx, network, address)
		},
	}}
	response, err := client.Get(server.URL + "/healthz")
	if err != nil {
		t.Fatalf("h2c request failed: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.StatusCode)
	}
	if response.ProtoMajor != 2 {
		t.Fatalf("proto = %s, want HTTP/2.0", response.Proto)
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if !strings.Contains(string(body), "HTTP/2.0") {
		t.Fatalf("handler saw %q, want an HTTP/2.0 request", string(body))
	}
}

// TestH2CHandlerKeepsServingHTTP1 verifies HTTP/1.1 clients still work
// against the h2c-wrapped handler (the wrapper upgrades, never replaces).
func TestH2CHandlerKeepsServingHTTP1(t *testing.T) {
	server := httptest.NewServer(wrapHTTPProtocol(healthzTestMux(), app.HTTPProtocolH2C))
	defer server.Close()

	response, err := http.Get(server.URL + "/healthz")
	if err != nil {
		t.Fatalf("h1 request failed: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.StatusCode)
	}
	if response.ProtoMajor != 1 {
		t.Fatalf("proto = %s, want HTTP/1.1", response.Proto)
	}
}

// TestH1ProtocolLeavesHandlerUnwrapped confirms the default protocol serves
// plain HTTP/1.1 and rejects nothing.
func TestH1ProtocolLeavesHandlerUnwrapped(t *testing.T) {
	server := httptest.NewServer(wrapHTTPProtocol(healthzTestMux(), app.HTTPProtocolH1))
	defer server.Close()

	response, err := http.Get(server.URL + "/healthz")
	if err != nil {
		t.Fatalf("h1 request failed: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.StatusCode)
	}
}
