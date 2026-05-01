package db

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/scouser-122/gophermart/internal/config"
	"github.com/stretchr/testify/mock"
)

type MockPostgresDBTestData struct {
	MockPool *MockPostgresPool
}

func NewMockPostgresDB(serverConfig config.ServerConfig, mockPool *MockPostgresPool) PostgresDatabase {
	database := PostgresDatabase{
		config: config.DBConnectionConfig{
			DSN:         serverConfig.DBDataSourceName,
			RetryConfig: config.DefaultRetryConfig(),
		},
		pool: mockPool,
	}
	return database
}

// Мок для pgxpool.Pool
type MockPostgresPool struct {
	mock.Mock
	MockMethods func(tt MockPostgresDBTestData)
}

func (m *MockPostgresPool) Ping(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockPostgresPool) Close() {
	m.Called()
}

func (m *MockPostgresPool) Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error) {
	args := m.Called(ctx, sql, arguments)
	return args.Get(0).(pgconn.CommandTag), args.Error(1)
}

func (m *MockPostgresPool) QueryRow(ctx context.Context, sql string, arguments ...interface{}) pgx.Row {
	args := m.Called(ctx, sql, arguments)
	return args.Get(0).(pgx.Row)
}

func (m *MockPostgresPool) Query(ctx context.Context, sql string, arguments ...interface{}) (pgx.Rows, error) {
	args := m.Called(ctx, sql, arguments)
	return args.Get(0).(pgx.Rows), args.Error(1)
}

func (m *MockPostgresPool) Begin(ctx context.Context) (pgx.Tx, error) {
	args := m.Called(ctx)
	return args.Get(0).(pgx.Tx), args.Error(1)
}
