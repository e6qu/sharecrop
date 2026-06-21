//go:build http_e2e

package http_e2e_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	httpserver "github.com/e6qu/sharecrop/internal/http"
	"github.com/e6qu/sharecrop/web"
)

func TestHealthEndpoint(t *testing.T) {
	staticFiles, err := web.StaticFiles()
	if err != nil {
		t.Fatalf("static files: %v", err)
	}

	server := httptest.NewServer(httpserver.New(staticFiles))
	defer server.Close()

	response, err := http.Get(server.URL + "/healthz")
	if err != nil {
		t.Fatalf("get health: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusOK)
	}
}
