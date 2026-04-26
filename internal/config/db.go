package config

type DBConnectionConfig struct {
	DSN         string
	RetryConfig RetryConfig
}
