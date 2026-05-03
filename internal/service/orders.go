package service

import (
	"context"

	luhnmod10 "github.com/luhnmod10/go"

	"github.com/scouser-122/gophermart/internal/models"
	"github.com/scouser-122/gophermart/internal/repository"
	"github.com/scouser-122/gophermart/internal/repository/db"
)

// OrdersService service to work with orders
type OrdersService struct {
	orderStorage repository.OrderStorage
}

// NewOrdersService creates new OrdersService instance
func NewOrdersService(database *db.PostgresDatabase) *OrdersService {
	service := OrdersService{}
	dbStorage := db.PostgresOrderStorage{
		Database: database,
	}
	service.orderStorage = &dbStorage
	return &service
}

// Upload runs upload process for specified order
func (service *OrdersService) Upload(ctx context.Context, orderID string, userLogin string) (*models.Order, error) {
	if orderID == "" {
		return nil, &models.CustomErr{
			Code: models.CustomErrOrderIDInvalidFormat,
		}
	}
	isValid := luhnmod10.Valid(orderID)
	if !isValid {
		return nil, &models.CustomErr{
			Code: models.CustomErrOrderIDInvalidFormat,
		}
	}
	order, err := service.orderStorage.AddOrder(ctx, orderID, userLogin)
	if err != nil {
		return nil, err
	}
	return order, nil
}
