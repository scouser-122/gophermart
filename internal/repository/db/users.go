package db

import (
	"context"
	"fmt"

	"github.com/scouser-122/gophermart/internal/models"
)

type PostgresUserStorage struct {
	Database *PostgresDatabase
}

func (storage *PostgresUserStorage) AddUser(ctx context.Context, user *models.User) (*models.User, error) {
	// row, err := storage.Database.QueryRow(
	// 	ctx,
	// 	"SELECT login FROM metrics WHERE id = $1 AND type = $2",
	// 	name, models.Counter,
	// )
	return nil, fmt.Errorf("cant add user")
}

func (storage *PostgresUserStorage) GetUser(ctx context.Context, login string) (*models.User, error) {
	return nil, fmt.Errorf("user not found")
}
