package main

import (
	"testing"

	"github.com/e6qu/sharecrop/internal/app"
)

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
