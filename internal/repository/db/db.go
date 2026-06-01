package db

import "github.com/scouser-122/gophermart/internal/config"

// NewPostgresDB creates new PostgresDatabase object to interact with Postgres DB
func NewPostgresDB(serverConfig config.ServerConfig) PostgresDatabase {
	database := PostgresDatabase{
		Config: config.DBConnectionConfig{
			DSN:         serverConfig.DBDataSourceName,
			RetryConfig: config.DefaultRetryConfig(),
		},
	}
	return database
}
