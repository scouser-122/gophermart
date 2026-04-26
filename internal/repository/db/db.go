package db

import "github.com/scouser-122/gophermart/internal/config"

func NewPostgresDB(serverConfig config.ServerConfig) PostgresDatabase {
	database := PostgresDatabase{
		Config: config.DBConnectionConfig{
			DSN:         serverConfig.DBDataSourceName,
			RetryConfig: config.DefaultRetryConfig(),
		},
	}
	return database
}
