package app

import "testing"

func TestLoadConfigRequiresHTTPAddress(t *testing.T) {
	t.Setenv("SHARECROP_HTTP_ADDR", "")
	t.Setenv("DATABASE_URL", "postgres://example")
	t.Setenv("SHARECROP_MIGRATIONS_DIR", "migrations")

	result := LoadConfig()

	_, rejected := result.(ConfigRejected)
	if !rejected {
		t.Fatalf("result = %T, want ConfigRejected", result)
	}
}

func TestLoadConfigLoadsExplicitValues(t *testing.T) {
	t.Setenv("SHARECROP_HTTP_ADDR", ":18080")
	t.Setenv("DATABASE_URL", "postgres://example")
	t.Setenv("SHARECROP_MIGRATIONS_DIR", "migrations")

	result := LoadConfig()

	loaded, matched := result.(ConfigLoaded)
	if !matched {
		t.Fatalf("result = %T, want ConfigLoaded", result)
	}

	if loaded.Value.HTTPAddress() != ":18080" {
		t.Fatalf("http address = %q, want :18080", loaded.Value.HTTPAddress())
	}
}
