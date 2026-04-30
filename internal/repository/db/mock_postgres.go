package db

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/scouser-122/gophermart/internal/config"
	"github.com/scouser-122/gophermart/internal/models"
	"github.com/stretchr/testify/mock"
)

type MockPostgresDBTestData struct {
	MockPool  *MockPostgresPool
	MockRow   pgx.Row
	MockTag   pgconn.CommandTag
	MockError error
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

// Мок для pgx.Rows
type MockPostgresRows struct {
	mock.Mock
	users []models.User
	index int
}

func (m *MockPostgresRows) Next() bool {
	return m.index < len(m.users)
}

func (m *MockPostgresRows) Scan(dest ...interface{}) error {
	if m.index >= len(m.users) {
		return errors.New("no more rows")
	}

	metric := m.users[m.index]
	*dest[0].(*string) = metric.Login
	*dest[1].(*string) = metric.Password
	*dest[2].(*float64) = metric.Balance
	*dest[3].(*time.Time) = metric.CreatedAt
	m.index++
	return nil
}

func (m *MockPostgresRows) Close()     {}
func (m *MockPostgresRows) Err() error { return nil }

func (m *MockPostgresRows) CommandTag() pgconn.CommandTag {
	return pgconn.NewCommandTag("SELECT 3")
}

// Мок для pgx.Row
type MockPostgresRow struct {
	User *models.User
	Err  error
}

func (m *MockPostgresRow) Scan(dest ...interface{}) error {
	if m.Err != nil {
		return m.Err
	}
	if m.User == nil {
		return pgx.ErrNoRows
	}

	*dest[0].(*string) = m.User.Login
	*dest[1].(*string) = m.User.Password
	if len(dest) == 3 {
		*dest[2].(*float64) = m.User.Balance
	} else if len(dest) == 4 {
		*dest[2].(*float64) = m.User.Balance
		*dest[2].(*time.Time) = m.User.CreatedAt
	}
	return nil
}
