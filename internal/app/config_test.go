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

func TestParseConfigHTTPProtocol(t *testing.T) {
	base := EnvValues{
		HTTPAddress:       ":18080",
		DatabaseURL:       "postgres://example",
		MigrationsDir:     "migrations",
		AccessTokenSecret: "01234567890123456789012345678901",
	}

	for name, values := range map[string]struct {
		raw  string
		want HTTPProtocol
	}{
		"unset": {raw: "", want: HTTPProtocolH1},
		"h1":    {raw: "h1", want: HTTPProtocolH1},
		"h2c":   {raw: "h2c", want: HTTPProtocolH2C},
	} {
		t.Run(name, func(t *testing.T) {
			env := base
			env.HTTPProtocol = values.raw
			loaded, matched := ParseConfig(env).(ConfigLoaded)
			if !matched {
				t.Fatalf("protocol %q rejected", values.raw)
			}
			if loaded.Value.HTTPProtocol() != values.want {
				t.Fatalf("protocol = %q, want %q", loaded.Value.HTTPProtocol().String(), values.want.String())
			}
		})
	}

	env := base
	env.HTTPProtocol = "h3"
	rejected, matched := ParseConfig(env).(ConfigRejected)
	if !matched {
		t.Fatalf("invalid protocol accepted")
	}
	if rejected.Reason != "SHARECROP_HTTP_PROTOCOL must be one of \"h1\" or \"h2c\" (or unset for h1)" {
		t.Fatalf("reason = %q", rejected.Reason)
	}
}

func TestParseMigrationConfigRequiresOnlyDatabaseAndMigrations(t *testing.T) {
	result := ParseMigrationConfig(MigrationEnvValues{
		DatabaseURL:   "postgres://example",
		MigrationsDir: "migrations",
	})
	loaded, matched := result.(MigrationConfigLoaded)
	if !matched {
		t.Fatalf("result = %T, want MigrationConfigLoaded", result)
	}
	if loaded.Value.DatabaseURL() != "postgres://example" || loaded.Value.MigrationsDir() != "migrations" {
		t.Fatalf("migration config = %#v", loaded.Value)
	}
}

// TestParseMCPConfigDoesNotRequireHTTPAddress pins the stdio transport's
// config surface: `sharecrop mcp` serves no HTTP, so it must start without
// SHARECROP_HTTP_ADDR.
func TestParseMCPConfigDoesNotRequireHTTPAddress(t *testing.T) {
	result := ParseMCPConfig(MCPEnvValues{
		DatabaseURL:       "postgres://example",
		MigrationsDir:     "migrations",
		AccessTokenSecret: "01234567890123456789012345678901",
	})
	loaded, matched := result.(MCPConfigLoaded)
	if !matched {
		t.Fatalf("result = %T, want MCPConfigLoaded", result)
	}
	if loaded.Value.DatabaseURL() != "postgres://example" || loaded.Value.MigrationsDir() != "migrations" || loaded.Value.AccessTokenSecret() != "01234567890123456789012345678901" {
		t.Fatalf("mcp config = %#v", loaded.Value)
	}
}

func TestParseMCPConfigRejectsMissingRequiredValues(t *testing.T) {
	for name, values := range map[string]MCPEnvValues{
		"database":   {MigrationsDir: "migrations", AccessTokenSecret: "secret"},
		"migrations": {DatabaseURL: "postgres://example", AccessTokenSecret: "secret"},
		"secret":     {DatabaseURL: "postgres://example", MigrationsDir: "migrations"},
	} {
		t.Run(name, func(t *testing.T) {
			if _, rejected := ParseMCPConfig(values).(MCPConfigRejected); !rejected {
				t.Fatalf("expected MCPConfigRejected for missing %s", name)
			}
		})
	}
}

func TestParseMigrationConfigRejectsMissingDatabaseOrMigrations(t *testing.T) {
	for name, values := range map[string]MigrationEnvValues{
		"database":   {MigrationsDir: "migrations"},
		"migrations": {DatabaseURL: "postgres://example"},
	} {
		t.Run(name, func(t *testing.T) {
			if result := ParseMigrationConfig(values); result == nil {
				t.Fatal("result is nil")
			} else if _, rejected := result.(MigrationConfigRejected); !rejected {
				t.Fatalf("result = %T, want MigrationConfigRejected", result)
			}
		})
	}
}
