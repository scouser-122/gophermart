package repository

import (
	"context"

	"github.com/scouser-122/gophermart/internal/models"
)

// UserGenStorage declares methods to store users data
type UserGenStorage interface {
	// Create creates new user,
	// returns error if user with specified login already exists or save process failed
	Create(ctx context.Context, login string, password string) (*models.User, error)

	// Get obtains user from storage
	Get(ctx context.Context, login string) (*models.User, error)

	// AddBalance adds specified accrual to user's balance
	AddBalance(ctx context.Context, login string, accrual *float32) error

	// WithdrawBalance withdraws specified sum from user's balance
	WithdrawBalance(ctx context.Context, login string, sum float32) error
}
