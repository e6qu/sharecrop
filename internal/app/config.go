package app

import "os"

const (
	defaultHTTPAddress = ":18080"
	defaultDatabaseURL = "postgres://sharecrop:sharecrop@localhost:15432/sharecrop?sslmode=disable"
	defaultMigrations  = "migrations"
)

type Config struct {
	httpAddress   string
	databaseURL   string
	migrationsDir string
}

func LoadConfig() Config {
	return Config{
		httpAddress:   valueOrDefault(os.Getenv("SHARECROP_HTTP_ADDR"), defaultHTTPAddress),
		databaseURL:   valueOrDefault(os.Getenv("DATABASE_URL"), defaultDatabaseURL),
		migrationsDir: valueOrDefault(os.Getenv("SHARECROP_MIGRATIONS_DIR"), defaultMigrations),
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

func valueOrDefault(value string, fallback string) string {
	if value == "" {
		return fallback
	}

	return value
}
