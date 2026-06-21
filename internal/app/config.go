package app

import "os"

type Config struct {
	httpAddress       string
	databaseURL       string
	migrationsDir     string
	accessTokenSecret string
}

type EnvValues struct {
	HTTPAddress       string
	DatabaseURL       string
	MigrationsDir     string
	AccessTokenSecret string
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
	return ParseConfig(EnvValues{
		HTTPAddress:       os.Getenv("SHARECROP_HTTP_ADDR"),
		DatabaseURL:       os.Getenv("DATABASE_URL"),
		MigrationsDir:     os.Getenv("SHARECROP_MIGRATIONS_DIR"),
		AccessTokenSecret: os.Getenv("SHARECROP_ACCESS_TOKEN_SECRET"),
	})
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

	return ConfigLoaded{
		Value: Config{
			httpAddress:       values.HTTPAddress,
			databaseURL:       values.DatabaseURL,
			migrationsDir:     values.MigrationsDir,
			accessTokenSecret: values.AccessTokenSecret,
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

func (c Config) AccessTokenSecret() string {
	return c.accessTokenSecret
}
