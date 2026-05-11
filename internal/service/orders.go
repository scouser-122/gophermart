package service

import (
	"context"

	luhnmod10 "github.com/luhnmod10/go"

	"github.com/scouser-122/gophermart/internal/logger"
	"github.com/scouser-122/gophermart/internal/models"
	"github.com/scouser-122/gophermart/internal/repository"
	"github.com/scouser-122/gophermart/internal/repository/db"
	"github.com/scouser-122/gophermart/internal/utils"
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
	logger := logger.GetLoggerFromContext(ctx)
	isValid := luhnmod10.Valid(orderID)
	if !isValid {
		err := &models.CustomErr{Code: models.CustomErrOrderIDInvalidFormat}
		logger.Sugar().Error(err)
		return nil, err
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
		return []*models.Order{}, &models.CustomErr{Code: models.CustomErrUserOrdersListEmpty}
	}
	for i, o := range orders {
		accrualOrder, err := service.accrualService.GetOrderData(ctx, o.ID)
		if err != nil {
			return []*models.Order{}, err
		}
		if accrualOrder.Status != o.Status || !utils.EqualFloat32Ptr(accrualOrder.Accrual, o.Accrual, 1e-6) {
			updatedOrder, err := service.orderStorage.UpdateOrder(ctx, accrualOrder)
			if err != nil {
				return []*models.Order{}, err
			}
			orders[i] = updatedOrder
		}
	}
	return orders, nil
}

// GetWithdrawnForUser return total withdrawn points from all orders for specified user
func (service *OrdersService) GetWithdrawnForUser(ctx context.Context, login string) (float32, error) {
	return service.orderStorage.GetWithdrawnForUser(ctx, login)
}

// WithdrawBalanceForOrder withdraw user's loyalty points from balance for order with specified ID
func (service *OrdersService) WithdrawBalanceForOrder(ctx context.Context, request *models.WithdrawBalanceRequest, login string) error {
	logger := logger.GetLoggerFromContext(ctx)
	isValid := luhnmod10.Valid(request.Order)
	if !isValid {
		err := &models.CustomErr{Code: models.CustomErrOrderIDInvalidFormat}
		logger.Sugar().Error(err)
		return err
	}
	accrualOrder, err := service.accrualService.GetOrderData(ctx, request.Order)
	if err != nil {
		return err
	}
	updatedOrder, err := service.orderStorage.UpdateOrder(ctx, accrualOrder)
	if err != nil {
		return err
	}
	return service.orderStorage.WithdrawBalanceForOrder(ctx, updatedOrder.ID, login, request.Sum)
}

// WithdrawalsForUser returns slice of withdrawals data for specified user
func (service *OrdersService) WithdrawalsForUser(ctx context.Context, login string) ([]models.WithdrawalResponse, error) {
	logger := logger.GetLoggerFromContext(ctx)
	withdrawals, err := service.orderStorage.WithdrawalsForUser(ctx, login)
	if err != nil {
		return []models.WithdrawalResponse{}, err
	}
	if len(withdrawals) == 0 {
		err = &models.CustomErr{Code: models.CustomErrWithdrawalsListEmpty}
		logger.Sugar().Error(err)
		return withdrawals, err
	}
	return withdrawals, nil
}
