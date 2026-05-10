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
}
