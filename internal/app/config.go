package app

import "os"

// HTTPProtocol is the sealed set of listener protocols runServe can speak.
// H1 is plain HTTP/1.1 (the default); H2C adds cleartext HTTP/2 by wrapping
// the handler in an h2c upgrader (HTTP/1.1 keeps working alongside it).
type HTTPProtocol struct {
	value string
}

var (
	HTTPProtocolH1  = HTTPProtocol{value: "h1"}
	HTTPProtocolH2C = HTTPProtocol{value: "h2c"}
)

func (protocol HTTPProtocol) String() string {
	return protocol.value
}

type Config struct {
	httpAddress       string
	databaseURL       string
	migrationsDir     string
	accessTokenSecret string
	httpProtocol      HTTPProtocol
}

type EnvValues struct {
	HTTPAddress       string
	DatabaseURL       string
	MigrationsDir     string
	AccessTokenSecret string
	HTTPProtocol      string
}

type MigrationConfig struct {
	databaseURL   string
	migrationsDir string
}

type MigrationEnvValues struct {
	DatabaseURL   string
	MigrationsDir string
}

// MCPConfig is what the stdio MCP transport needs: database access and the
// access-token secret. It deliberately has no HTTP address — `sharecrop mcp`
// serves stdio, not HTTP, so SHARECROP_HTTP_ADDR must not be required there.
type MCPConfig struct {
	databaseURL       string
	migrationsDir     string
	accessTokenSecret string
}

type MCPEnvValues struct {
	DatabaseURL       string
	MigrationsDir     string
	AccessTokenSecret string
}

type MCPConfigResult interface {
	mcpConfigResult()
}

type MCPConfigLoaded struct {
	Value MCPConfig
}

type MCPConfigRejected struct {
	Reason string
}

func (MCPConfigLoaded) mcpConfigResult() {}

func (MCPConfigRejected) mcpConfigResult() {}

type MigrationConfigResult interface {
	migrationConfigResult()
}

type MigrationConfigLoaded struct {
	Value MigrationConfig
}

type MigrationConfigRejected struct {
	Reason string
}

func (MigrationConfigLoaded) migrationConfigResult()   {}
func (MigrationConfigRejected) migrationConfigResult() {}

type ConfigResult interface {
	configResult()
}

type ConfigLoaded struct {
	Value Config
}

type ConfigRejected struct {
	Reason string
}

func (ConfigLoaded) configResult() {}

func (ConfigRejected) configResult() {}

func LoadConfig() ConfigResult {
	return ParseConfig(EnvValues{
		HTTPAddress:       os.Getenv("SHARECROP_HTTP_ADDR"),
		DatabaseURL:       os.Getenv("DATABASE_URL"),
		MigrationsDir:     os.Getenv("SHARECROP_MIGRATIONS_DIR"),
		AccessTokenSecret: os.Getenv("SHARECROP_ACCESS_TOKEN_SECRET"),
		HTTPProtocol:      os.Getenv("SHARECROP_HTTP_PROTOCOL"),
	})
}

func LoadMigrationConfig() MigrationConfigResult {
	return ParseMigrationConfig(MigrationEnvValues{
		DatabaseURL:   os.Getenv("DATABASE_URL"),
		MigrationsDir: os.Getenv("SHARECROP_MIGRATIONS_DIR"),
	})
}

func LoadMCPConfig() MCPConfigResult {
	return ParseMCPConfig(MCPEnvValues{
		DatabaseURL:       os.Getenv("DATABASE_URL"),
		MigrationsDir:     os.Getenv("SHARECROP_MIGRATIONS_DIR"),
		AccessTokenSecret: os.Getenv("SHARECROP_ACCESS_TOKEN_SECRET"),
	})
}

func ParseMCPConfig(values MCPEnvValues) MCPConfigResult {
	if values.DatabaseURL == "" {
		return MCPConfigRejected{Reason: "DATABASE_URL is required"}
	}
	if values.MigrationsDir == "" {
		return MCPConfigRejected{Reason: "SHARECROP_MIGRATIONS_DIR is required"}
	}
	if values.AccessTokenSecret == "" {
		return MCPConfigRejected{Reason: "SHARECROP_ACCESS_TOKEN_SECRET is required"}
	}
	return MCPConfigLoaded{Value: MCPConfig{
		databaseURL:       values.DatabaseURL,
		migrationsDir:     values.MigrationsDir,
		accessTokenSecret: values.AccessTokenSecret,
	}}
}

func ParseMigrationConfig(values MigrationEnvValues) MigrationConfigResult {
	if values.DatabaseURL == "" {
		return MigrationConfigRejected{Reason: "DATABASE_URL is required"}
	}
	if values.MigrationsDir == "" {
		return MigrationConfigRejected{Reason: "SHARECROP_MIGRATIONS_DIR is required"}
	}
	return MigrationConfigLoaded{Value: MigrationConfig{databaseURL: values.DatabaseURL, migrationsDir: values.MigrationsDir}}
}

func ParseConfig(values EnvValues) ConfigResult {
	if values.HTTPAddress == "" {
		return ConfigRejected{Reason: "SHARECROP_HTTP_ADDR is required"}
	}

	if values.DatabaseURL == "" {
		return ConfigRejected{Reason: "DATABASE_URL is required"}
	}

	if values.MigrationsDir == "" {
		return ConfigRejected{Reason: "SHARECROP_MIGRATIONS_DIR is required"}
	}

	if values.AccessTokenSecret == "" {
		return ConfigRejected{Reason: "SHARECROP_ACCESS_TOKEN_SECRET is required"}
	}

	protocolResult := parseHTTPProtocol(values.HTTPProtocol)
	protocol, protocolMatched := protocolResult.(httpProtocolParsed)
	if !protocolMatched {
		return ConfigRejected{Reason: protocolResult.(httpProtocolRejected).reason}
	}

	return ConfigLoaded{
		Value: Config{
			httpAddress:       values.HTTPAddress,
			databaseURL:       values.DatabaseURL,
			migrationsDir:     values.MigrationsDir,
			accessTokenSecret: values.AccessTokenSecret,
			httpProtocol:      protocol.value,
		},
	}
}

type httpProtocolResult interface {
	httpProtocolResult()
}

type httpProtocolParsed struct {
	value HTTPProtocol
}

type httpProtocolRejected struct {
	reason string
}

func (httpProtocolParsed) httpProtocolResult() {}

func (httpProtocolRejected) httpProtocolResult() {}

// parseHTTPProtocol maps SHARECROP_HTTP_PROTOCOL onto the sealed enum. The
// empty string (unset) means HTTP/1.1.
func parseHTTPProtocol(raw string) httpProtocolResult {
	switch raw {
	case "", HTTPProtocolH1.value:
		return httpProtocolParsed{value: HTTPProtocolH1}
	case HTTPProtocolH2C.value:
		return httpProtocolParsed{value: HTTPProtocolH2C}
	default:
		return httpProtocolRejected{reason: "SHARECROP_HTTP_PROTOCOL must be one of \"h1\" or \"h2c\" (or unset for h1)"}
	}
}

func (c Config) HTTPAddress() string {
	return c.httpAddress
}

func (c Config) DatabaseURL() string {
	return c.databaseURL
}

func (c Config) MigrationsDir() string {
	return c.migrationsDir
}

func (c Config) AccessTokenSecret() string {
	return c.accessTokenSecret
}

func (c Config) HTTPProtocol() HTTPProtocol {
	return c.httpProtocol
}

func (c MigrationConfig) DatabaseURL() string {
	return c.databaseURL
}

func (c MigrationConfig) MigrationsDir() string {
	return c.migrationsDir
}

func (c MCPConfig) DatabaseURL() string {
	return c.databaseURL
}

func (c MCPConfig) MigrationsDir() string {
	return c.migrationsDir
}

func (c MCPConfig) AccessTokenSecret() string {
	return c.accessTokenSecret
}
