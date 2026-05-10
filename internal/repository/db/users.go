package db

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
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
		logger.Sugar().Error(zap.Error(err))
		return nil, err
	}
	if dbUser.Login != "" {
		return nil, &models.CustomErr{Code: models.CustomErrUserLoginBusy}
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
		return nil, &models.CustomErr{Code: models.CustomErrUserNotFound}
	}
	return &dbUser, nil
}
