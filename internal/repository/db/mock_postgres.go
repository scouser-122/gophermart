package db

import (
	"github.com/pashagolub/pgxmock/v5"
	"github.com/scouser-122/gophermart/internal/config"
)

// MockPostgresDBTestData used to mock postgres DB calls
type MockPostgresDBTestData struct {
	PgxPoolIface pgxmock.PgxPoolIface
	MockDBCalls  func(tt MockPostgresDBTestData)
}

// NewPgxMockDB creates new postgres DB mock instance
func NewPgxMockDB(serverConfig config.ServerConfig, pgxMock pgxmock.PgxPoolIface) PostgresDatabase {
	database := PostgresDatabase{
		Config: config.DBConnectionConfig{
			DSN:         serverConfig.DBDataSourceName,
			RetryConfig: config.DefaultRetryConfig(),
		},
		pool: pgxMock,
	}
	return database
}
