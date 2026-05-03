package repository

import (
	"context"

	"github.com/scouser-122/gophermart/internal/models"
)

// OrderStorage declares methods to store orders data
type OrderStorage interface {
	// AddOrder adds specified order,
	// returns error if order with specified ID already exists or add process failed
	AddOrder(ctx context.Context, orderID string, userLogin string) (*models.Order, error)
}
