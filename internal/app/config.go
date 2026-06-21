package app

import "os"

type Config struct {
	httpAddress   string
	databaseURL   string
	migrationsDir string
}

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
	httpAddress := os.Getenv("SHARECROP_HTTP_ADDR")
	if httpAddress == "" {
		return ConfigRejected{Reason: "SHARECROP_HTTP_ADDR is required"}
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return ConfigRejected{Reason: "DATABASE_URL is required"}
	}

	migrationsDir := os.Getenv("SHARECROP_MIGRATIONS_DIR")
	if migrationsDir == "" {
		return ConfigRejected{Reason: "SHARECROP_MIGRATIONS_DIR is required"}
	}

	return ConfigLoaded{
		Value: Config{
			httpAddress:   httpAddress,
			databaseURL:   databaseURL,
			migrationsDir: migrationsDir,
		},
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
