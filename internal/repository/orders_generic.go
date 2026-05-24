package repository

import (
	"context"

	"github.com/scouser-122/gophermart/internal/models"
)

// OrdersGenStorage declares methods to store orders data
type OrdersGenStorage interface {
	// Create creates new order,
	// returns error if order with specified ID already exists or process failed
	Create(
		ctx context.Context,
		ID string,
		status models.OrderStatus,
		accrual *float32,
		userLogin string,
	) (*models.Order, error)

	// Get obtains order from storage
	Get(ctx context.Context, ID string) (*models.Order, error)

	// Update updates order data
	Update(ctx context.Context, order *models.Order, userLogin string) error

	// GetAllForUser returns all orders for specified user
	GetAllForUser(ctx context.Context, userLogin string) ([]*models.Order, error)

	// GetWithdrawnForUser return total withdrawn points from all orders for specified user
	GetWithdrawnForUser(ctx context.Context, login string) (float32, error)

	// WithdrawSum set's withdrawn amount for specified order
	WithdrawSum(ctx context.Context, ID string, sum float32, login string) error

	// WithdrawalsForUser returns slice of withdrawals data for specified user
	WithdrawalsForUser(ctx context.Context, userLogin string) ([]models.WithdrawalResponse, error)
}
