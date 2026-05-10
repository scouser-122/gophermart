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
	database       *db.PostgresDatabase
	orderStorage   repository.OrderStorage
	accrualService *AccrualService
}

// NewOrdersService creates new OrdersService instance
func NewOrdersService(
	orderStorage repository.OrderStorage,
	accrualService *AccrualService,
) *OrdersService {
	service := OrdersService{}
	service.orderStorage = orderStorage
	service.accrualService = accrualService
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
	order, err := service.accrualService.GetOrderData(ctx, orderID)
	if err != nil {
		return nil, err
	}
	err = service.orderStorage.AddOrder(ctx, order, userLogin)
	if err != nil {
		return nil, err
	}
	return order, nil
}

// GetUserOrders returns slice of orders for user with specified login
func (service *OrdersService) GetUserOrders(ctx context.Context, userLogin string) ([]*models.Order, error) {
	orders, err := service.orderStorage.GetUserOrders(ctx, userLogin)
	if err != nil {
		return []*models.Order{}, err
	}
	if len(orders) == 0 {
		return []*models.Order{}, &models.CustomErr{
			Code: models.CustomErrUserOrdersListEmpty,
		}
	}
	return orders, nil
}
