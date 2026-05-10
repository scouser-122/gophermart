package repository

import (
	"context"

	"github.com/scouser-122/gophermart/internal/models"
)

// OrderStorage declares methods to store orders data
type OrderStorage interface {
	// AddOrder adds specified order, links it with user, and increments user's balance if accrual present,
	// returns error if order with specified ID already exists or add process failed
	AddOrder(ctx context.Context, order *models.Order, userLogin string) error

	// GetUserOrders returns slice of orders for user with specified login
	GetUserOrders(ctx context.Context, userLogin string) ([]*models.Order, error)

	// GetWithdrawnForUser return total withdrawn points from all orders for specified user
	GetWithdrawnForUser(ctx context.Context, login string) (float32, error)

	// WithdrawBalanceForOrder withdraw user's loyalty points from balance for order with specified ID
	WithdrawBalanceForOrder(ctx context.Context, orderID string, login string, sum float32) error

	// WithdrawalsForUser returns slice of withdrawals data for specified user
	WithdrawalsForUser(ctx context.Context, login string) ([]models.WithdrawalResponse, error)
}
