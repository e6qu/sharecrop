package app

import "testing"

func TestLoadConfigRequiresHTTPAddress(t *testing.T) {
	result := ParseConfig(EnvValues{
		DatabaseURL:       "postgres://example",
		MigrationsDir:     "migrations",
		AccessTokenSecret: "01234567890123456789012345678901",
	})

	_, rejected := result.(ConfigRejected)
	if !rejected {
		t.Fatalf("result = %T, want ConfigRejected", result)
	}
}

func TestParseConfigLoadsExplicitValues(t *testing.T) {
	result := ParseConfig(EnvValues{
		HTTPAddress:       ":18080",
		DatabaseURL:       "postgres://example",
		MigrationsDir:     "migrations",
		AccessTokenSecret: "01234567890123456789012345678901",
	})

	loaded, matched := result.(ConfigLoaded)
	if !matched {
		t.Fatalf("result = %T, want ConfigLoaded", result)
	}

	if loaded.Value.HTTPAddress() != ":18080" {
		t.Fatalf("http address = %q, want :18080", loaded.Value.HTTPAddress())
	}

	if loaded.Value.AccessTokenSecret() != "01234567890123456789012345678901" {
		t.Fatalf("access token secret = %q, want explicit value", loaded.Value.AccessTokenSecret())
	}
}
