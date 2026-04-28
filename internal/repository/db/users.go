package db

import (
	"context"
	"fmt"
	"time"

	"github.com/scouser-122/gophermart/internal/models"
)

type PostgresUserStorage struct {
	Database *PostgresDatabase
}

func (storage *PostgresUserStorage) AddUser(ctx context.Context, user *models.User) (*models.User, error) {
	var dbUser models.User
	err := storage.Database.Select(ctx, &dbUser, "SELECT * FROM users WHERE login = $1", user.Login)
	if err != nil {
		return nil, err
	}
	if dbUser.Login != "" {
		return nil, &models.CustomErr{Code: models.CustomErrLoginBusy}
	}
	user.CreatedAt = time.Now()
	_, err = storage.Database.Insert(ctx, user)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (storage *PostgresUserStorage) GetUser(ctx context.Context, login string) (*models.User, error) {
	var dbUser models.User
	err := storage.Database.Select(ctx, &dbUser, "SELECT * FROM users WHERE login = $1", login)
	if err != nil {
		return nil, err
	}
	if dbUser.Login == "" {
		return nil, &models.CustomErr{Code: models.CustomErrUserNotFound}
	}
	return nil, fmt.Errorf("user not found")
}
