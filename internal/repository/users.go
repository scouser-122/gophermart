package repository

import (
	"context"

	"github.com/scouser-122/gophermart/internal/models"
)

type UserStorage interface {
	AddUser(ctx context.Context, user *models.User) (*models.User, error)
	GetUser(ctx context.Context, login string) (*models.User, error)
}
