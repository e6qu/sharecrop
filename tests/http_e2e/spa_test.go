//go:build http_e2e

package http_e2e_test

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestDeepLinksServeAppShell(t *testing.T) {
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	// Every in-app route loads the same shell so deep links and refreshes work.
	for _, path := range []string{"/", "/tasks", "/tasks/new", "/tasks/some-id", "/discovery", "/funding", "/agents", "/collectibles", "/organizations"} {
		response, err := http.Get(server.URL + path)
		if err != nil {
			t.Fatalf("get %q: %v", path, err)
		}
		body, err := io.ReadAll(response.Body)
		response.Body.Close()
		if err != nil {
			t.Fatalf("read %q body: %v", path, err)
		}
		if response.StatusCode != http.StatusOK {
			t.Fatalf("get %q status = %d, want 200", path, response.StatusCode)
		}
		if !strings.Contains(string(body), `id="app"`) {
			t.Fatalf("get %q did not return the app shell", path)
		}
	}

	// Unknown API paths still return 404 rather than the shell.
	apiResponse, err := http.Get(server.URL + "/api/does-not-exist")
	if err != nil {
		t.Fatalf("get unknown api path: %v", err)
	}
	defer apiResponse.Body.Close()
	if apiResponse.StatusCode != http.StatusNotFound {
		t.Fatalf("unknown api path status = %d, want 404", apiResponse.StatusCode)
	}
}
