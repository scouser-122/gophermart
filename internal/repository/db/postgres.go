package db

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	"github.com/golang-migrate/migrate/v4"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/scouser-122/gophermart/internal/config"
	"github.com/scouser-122/gophermart/internal/logger"
)

type DBInterface interface {
	Ping(ctx context.Context) error
	Close()
	Exec(ctx context.Context, sql string, arguments ...interface{}) (pgconn.CommandTag, error)
	QueryRow(ctx context.Context, sql string, arguments ...interface{}) pgx.Row
	Query(ctx context.Context, sql string, arguments ...interface{}) (pgx.Rows, error)
	Begin(ctx context.Context) (pgx.Tx, error)
}

type PostgresDatabase struct {
	Config config.DBConnectionConfig
	Pool   DBInterface
}

func (db *PostgresDatabase) Open() error {
	if db.Config.DSN == "" {
		return fmt.Errorf("connection string is empty")
	}

	var err error
	config, err := pgxpool.ParseConfig(db.Config.DSN)
	if err != nil {
		return fmt.Errorf("failed to parse connection string: %q, err: %w", db.Config.DSN, err)
	}

	db.Pool, err = pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return fmt.Errorf("failed to create connection pool: %w", err)
	}

	if err := db.Pool.Ping(context.Background()); err != nil {
		db.Pool.Close()
		db.Pool = nil
		return fmt.Errorf("failed to ping database: %w", err)
	}

	logger.Sugar.Info("successfully connected to DB")

	err = db.runMigrations()
	if err != nil {
		db.Pool.Close()
		db.Pool = nil
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

func (db *PostgresDatabase) runMigrations() error {
	path, err := getMigrationsPath()
	if err != nil {
		return err
	}
	m, err := migrate.New(
		path,
		db.Config.DSN,
	)
	if err != nil {
		return err
	}
	defer m.Close()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return err
	}

	logger.Sugar.Info("DB migrations completed successfully")
	return nil
}

func getMigrationsPath() (string, error) {
	execPath, err := os.Executable()
	if err != nil {
		return "", err
	}

	execDir := filepath.Dir(execPath)

	migrationsPath := filepath.Join(execDir, "../../migrations")

	return "file://" + migrationsPath, nil
}

func (db *PostgresDatabase) Close() {
	if !isNil(db.Pool) {
		db.Pool.Close()
	}
}

func (db *PostgresDatabase) Ping(ctx context.Context) error {
	if isNil(db.Pool) {
		return fmt.Errorf("database connection was not opened")
	}
	return config.DataBaseRequestRetry(
		ctx,
		db.Config.RetryConfig,
		func() error {
			return db.Pool.Ping(ctx)
		},
	)
}

func (db *PostgresDatabase) Exec(ctx context.Context, query string, args ...any) (pgconn.CommandTag, error) {
	if isNil(db.Pool) {
		return pgconn.CommandTag{}, fmt.Errorf("database connection was not opened")
	}
	var commandTag pgconn.CommandTag
	err := config.DataBaseRequestRetry(
		ctx,
		db.Config.RetryConfig,
		func() error {
			var err error
			commandTag, err = db.Pool.Exec(ctx, query, args...)
			return err
		},
	)
	return commandTag, err
}

func (db *PostgresDatabase) Query(ctx context.Context, query string, args ...any) (pgx.Rows, error) {
	if isNil(db.Pool) {
		return nil, fmt.Errorf("database connection was not opened")
	}
	var rows pgx.Rows
	err := config.DataBaseRequestRetry(
		ctx,
		db.Config.RetryConfig,
		func() error {
			var err error
			rows, err = db.Pool.Query(ctx, query, args...)
			return err
		},
	)
	return rows, err
}

func (db *PostgresDatabase) QueryRow(ctx context.Context, query string, args ...any) (pgx.Row, error) {
	if isNil(db.Pool) {
		return nil, fmt.Errorf("database connection was not opened")
	}
	return db.Pool.QueryRow(ctx, query, args...), nil
}

func (db *PostgresDatabase) Begin(ctx context.Context) (pgx.Tx, error) {
	if isNil(db.Pool) {
		return nil, fmt.Errorf("database connection was not opened")
	}
	var tx pgx.Tx
	err := config.DataBaseRequestRetry(
		ctx,
		db.Config.RetryConfig,
		func() error {
			var err error
			tx, err = db.Pool.Begin(ctx)
			return err
		},
	)
	return tx, err
}

func isNil(i interface{}) bool {
	if i == nil {
		return true
	}
	v := reflect.ValueOf(i)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Map, reflect.Pointer, reflect.UnsafePointer, reflect.Interface, reflect.Slice:
		return v.IsNil()
	}
	return false
}
