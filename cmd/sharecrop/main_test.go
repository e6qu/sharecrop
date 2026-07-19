package main

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"github.com/e6qu/sharecrop/internal/app"
)

func TestMigrateCommandDoesNotRequireHTTPOrAccessTokenConfiguration(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://example.test/sharecrop")
	t.Setenv("SHARECROP_MIGRATIONS_DIR", "migrations")
	t.Setenv("SHARECROP_HTTP_ADDR", "")
	t.Setenv("SHARECROP_ACCESS_TOKEN_SECRET", "")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if status := run(context.Background(), []string{"sharecrop", "migrate", "down"}, &stdout, &stderr); status != 2 {
		t.Fatalf("status = %d, want 2", status)
	}
	if stdout.String() != "usage: sharecrop migrate up\n" {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, migration loaded unrelated runtime configuration", stderr.String())
	}
}

func TestWASIGuestEnvironmentForwardsHTTPRuntimeConfiguration(t *testing.T) {
	t.Setenv("SHARECROP_INSECURE_COOKIES", "true")
	t.Setenv("SHARECROP_ACCOUNT_TOKEN_DELIVERY", "api")
	t.Setenv("SHARECROP_ADMIN_USER_IDS", "admin-1,admin-2")

	parsed := app.ParseConfig(app.EnvValues{
		HTTPAddress:       ":8080",
		DatabaseURL:       "postgres://example.test/sharecrop",
		MigrationsDir:     "migrations",
		AccessTokenSecret: "access-token-secret",
	})
	loaded, ok := parsed.(app.ConfigLoaded)
	if !ok {
		t.Fatalf("ParseConfig() = %T, want app.ConfigLoaded", parsed)
	}

	got := wasiGuestEnvironment(loaded.Value)
	want := map[string]string{
		"SHARECROP_ACCESS_TOKEN_SECRET":    "access-token-secret",
		"SHARECROP_INSECURE_COOKIES":       "true",
		"SHARECROP_ACCOUNT_TOKEN_DELIVERY": "api",
		"SHARECROP_ADMIN_USER_IDS":         "admin-1,admin-2",
	}
	if len(got) != len(want) {
		t.Fatalf("wasiGuestEnvironment() has %d values, want %d: %#v", len(got), len(want), got)
	}
	for key, value := range want {
		if got[key] != value {
			t.Errorf("wasiGuestEnvironment()[%q] = %q, want %q", key, got[key], value)
		}
	}
}

func TestApplicationShellRequiresShauthSessionWhenConfigured(t *testing.T) {
	staticFiles := fstest.MapFS{"index.html": &fstest.MapFile{Data: []byte("Sharecrop shell")}}
	handler := applicationShell(staticFiles, true)

	t.Run("redirects a new visitor to Shauth", func(t *testing.T) {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "https://sharecrop.example.test/", nil))
		if response.Code != http.StatusFound {
			t.Fatalf("status = %d, want %d", response.Code, http.StatusFound)
		}
		if location := response.Header().Get("Location"); location != "/api/auth/shauth" {
			t.Errorf("Location = %q, want %q", location, "/api/auth/shauth")
		}
	})

	t.Run("serves the application after the OIDC callback sets the refresh cookie", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodGet, "https://sharecrop.example.test/", nil)
		request.AddCookie(&http.Cookie{Name: "sharecrop_refresh_token", Value: "opaque-refresh-token"})
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
		}
		if body := response.Body.String(); body != "Sharecrop shell" {
			t.Errorf("body = %q, want application shell", body)
		}
	})
}

func TestShauthSSORoutesStayOnNativeHostBoundary(t *testing.T) {
	mux := http.NewServeMux()
	registerShauthHostBoundary(mux, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Sharecrop-Boundary", "native")
		w.WriteHeader(http.StatusNoContent)
	}))
	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Sharecrop-Boundary", "guest")
		w.WriteHeader(http.StatusNoContent)
	}))
	for _, route := range []struct{ method, path string }{
		{http.MethodGet, "/api/auth/shauth"},
		{http.MethodGet, "/api/auth/shauth/callback"},
		{http.MethodPost, "/api/auth/shauth/backchannel-logout"},
		{http.MethodPost, "/api/auth/logout"},
		{http.MethodGet, "/api/auth/signed-out"},
	} {
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, httptest.NewRequest(route.method, route.path, nil))
		if got := response.Header().Get("X-Sharecrop-Boundary"); got != "native" {
			t.Errorf("%s %s boundary = %q, want native", route.method, route.path, got)
		}
	}
}
