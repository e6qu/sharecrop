package main

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/e6qu/sharecrop/internal/app"
	"github.com/e6qu/sharecrop/internal/auth"
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

func TestHealthCheckCommandDoesNotRequireApplicationConfiguration(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet || request.URL.Path != "/healthz" {
			t.Fatalf("request = %s %s, want GET /healthz", request.Method, request.URL.Path)
		}
		response.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	t.Setenv("DATABASE_URL", "")
	t.Setenv("SHARECROP_ACCESS_TOKEN_SECRET", "")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if status := run(context.Background(), []string{"sharecrop", "healthcheck", server.URL + "/healthz"}, &stdout, &stderr); status != 0 {
		t.Fatalf("status = %d, want 0; stderr = %q", status, stderr.String())
	}
	if stdout.Len() != 0 || stderr.Len() != 0 {
		t.Fatalf("stdout = %q, stderr = %q", stdout.String(), stderr.String())
	}
}

func TestHealthCheckCommandRejectsAnUnhealthyServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		http.Error(response, "starting", http.StatusServiceUnavailable)
	}))
	t.Cleanup(server.Close)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if status := run(context.Background(), []string{"sharecrop", "healthcheck", server.URL}, &stdout, &stderr); status != 1 {
		t.Fatalf("status = %d, want 1", status)
	}
	if !strings.Contains(stderr.String(), "status=503") {
		t.Fatalf("stderr = %q, want status=503", stderr.String())
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
		"SHARECROP_ACCESS_TOKEN_SECRET":     "access-token-secret",
		"SHARECROP_INSECURE_COOKIES":        "true",
		"SHARECROP_ACCOUNT_TOKEN_DELIVERY":  "api",
		"SHARECROP_ADMIN_USER_IDS":          "admin-1,admin-2",
		"SHARECROP_REQUIRE_BROWSER_SESSION": "false",
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
	handler := applicationShell(staticFiles, activeBrowserSessionService{}, true)

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

type activeBrowserSessionService struct{}

func (activeBrowserSessionService) ValidateSession(context.Context, auth.RefreshTokenPlain) auth.ValidateRefreshTokenResult {
	return auth.RefreshTokenActive{}
}

type inactiveBrowserSessionService struct{}

func (inactiveBrowserSessionService) ValidateSession(context.Context, auth.RefreshTokenPlain) auth.ValidateRefreshTokenResult {
	return auth.RefreshTokenInactive{}
}

func TestApplicationShellRejectsARevokedCookie(t *testing.T) {
	staticFiles := fstest.MapFS{"index.html": &fstest.MapFile{Data: []byte("Sharecrop shell")}}
	handler := applicationShell(staticFiles, inactiveBrowserSessionService{}, true)
	request := httptest.NewRequest(http.MethodGet, "https://sharecrop.example.test/", nil)
	request.AddCookie(&http.Cookie{Name: "sharecrop_refresh_token", Value: "revoked-refresh-token"})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusFound || response.Header().Get("Location") != "/api/auth/shauth" {
		t.Fatalf("revoked shell request = %d location=%q", response.Code, response.Header().Get("Location"))
	}
	if strings.Contains(response.Body.String(), "Sharecrop shell") {
		t.Fatal("revoked session received the application shell")
	}
	if cookies := response.Result().Cookies(); len(cookies) != 1 || cookies[0].Name != "sharecrop_refresh_token" || cookies[0].MaxAge >= 0 {
		t.Fatalf("revoked cookie was not cleared: %#v", cookies)
	}
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
		{http.MethodGet, "/api/auth/shauth/frontchannel-logout"},
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
