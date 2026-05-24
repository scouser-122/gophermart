package db

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/scouser-122/gophermart/internal/config"
	"github.com/scouser-122/gophermart/internal/logger"
	"github.com/scouser-122/gophermart/internal/models"
	"go.uber.org/zap"
)

// PostgresUserStorage implements UserStorage interface to store users data in Postgres DB
type PostgresUserStorage struct {
	Database *PostgresDatabase
}

// AddUser adds specified user,
// returns error if user with specified login already exists or save process failed
func (storage *PostgresUserStorage) AddUser(ctx context.Context, user *models.User) (*models.User, error) {
	logger := logger.GetLoggerFromContext(ctx)
	var dbUser models.User
	err := storage.Database.Select(ctx, &dbUser, "SELECT * FROM users WHERE login = $1", user.Login)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		logger.Sugar().Error(err)
		return nil, err
	}
	if dbUser.Login != "" {
		err = &models.CustomErr{Code: models.CustomErrUserLoginBusy}
		logger.Sugar().Error(err)
		return nil, err
	}
	user.CreatedAt = time.Now()
	_, err = storage.Database.Insert(ctx, user)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// GetUser takes user by login,
// returns error if retrive process failed or user not found
func (storage *PostgresUserStorage) GetUser(ctx context.Context, login string) (*models.User, error) {
	logger := logger.GetLoggerFromContext(ctx)
	var dbUser models.User
	err := storage.Database.Select(ctx, &dbUser, "SELECT * FROM users WHERE login = $1", login)
	if err != nil {
		logger.Sugar().Error(zap.Error(err))
		return nil, err
	}
	if dbUser.Login == "" {
		err = &models.CustomErr{Code: models.CustomErrUserNotFound}
		logger.Sugar().Error(err)
		return nil, err
	}
	return &dbUser, nil
}

// GetUserBalance return user's loyalty points balance and total withdrawn points
func (storage *PostgresUserStorage) GetUserBalance(ctx context.Context, login string) (float32, error) {
	logger := logger.GetLoggerFromContext(ctx)
	var result float32
	row := storage.Database.QueryRow(ctx, "SELECT balance FROM users WHERE login = $1", login)
	err := config.DataBaseRequestRetry(
		ctx,
		storage.Database.Config.RetryConfig,
		func() error {
			return row.Scan(&result)
		},
	)
	if err != nil {
		logger.Sugar().Error(err)
		return 0.0, err
	}
	return result, nil
}
