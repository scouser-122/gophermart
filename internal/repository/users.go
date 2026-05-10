package repository

import (
	"context"

	"github.com/scouser-122/gophermart/internal/models"
)

// UserStorage declares methods to store users data
type UserStorage interface {
	// AddUser adds specified user,
	// returns error if user with specified login already exists or save process failed
	AddUser(ctx context.Context, user *models.User) (*models.User, error)

	// GetUser takes user by login,
	// returns error if retrive process failed or user not found
	GetUser(ctx context.Context, login string) (*models.User, error)

	// GetUserBalance return user's loyalty points balance
	GetUserBalance(ctx context.Context, login string) (float32, error)
}
