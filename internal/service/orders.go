package service

import (
	"context"
	"errors"

	luhnmod10 "github.com/luhnmod10/go"

	"github.com/scouser-122/gophermart/internal/logger"
	"github.com/scouser-122/gophermart/internal/models"
	"github.com/scouser-122/gophermart/internal/repository"
	"github.com/scouser-122/gophermart/internal/repository/db"
	"github.com/scouser-122/gophermart/internal/utils"
)

// OrdersService service to work with orders
type OrdersService struct {
	database        *db.PostgresDatabase
	ordersStorage   repository.OrdersStorage
	usersStorage    repository.UsersStorage
	repositoryUtils repository.RepositoryUtils
	accrualService  *AccrualService
}

// NewOrdersService creates new OrdersService instance
func NewOrdersService(
	ordersStorage repository.OrdersStorage,
	usersStorage repository.UsersStorage,
	repositoryUtils repository.RepositoryUtils,
	accrualService *AccrualService,
) *OrdersService {
	service := OrdersService{}
	service.ordersStorage = ordersStorage
	service.usersStorage = usersStorage
	service.repositoryUtils = repositoryUtils
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
	status := models.NewOrder
	var accrual *float32
	order := service.accrualService.GetOrderData(ctx, orderID)
	if order != nil {
		status = order.Status
		accrual = order.Accrual

	}
	order, err := service.ordersStorage.Get(ctx, orderID)
	if order != nil {
		if order.UserLogin == userLogin {
			err = &models.CustomErr{Code: models.CustomErrOrderAlreadyUploaded}
			logger.Sugar().Error(err)
			return nil, err
		} else {
			err = &models.CustomErr{Code: models.CustomErrOrderAlreadyUploadedByAnotherUser}
			logger.Sugar().Error(err)
			return nil, err
		}
	} else if err != nil {
		var customErr *models.CustomErr
		if errors.As(err, &customErr) && customErr.Code == models.CustomErrOrderNotFound {
			tx, err := service.repositoryUtils.CreateTransaction(ctx)
			if err != nil {
				return nil, err
			}
			defer tx.Rollback(ctx)
			ctx = context.WithValue(ctx, models.DbTransactionKey, tx)
			order, err = service.ordersStorage.Create(ctx, orderID, status, accrual, userLogin)
			if err != nil {
				return nil, err
			}
			if accrual != nil {
				err = service.usersStorage.AddBalance(ctx, userLogin, accrual)
				if err != nil {
					return nil, err
				}
			}
			err = service.repositoryUtils.CommitTransaction(ctx, tx)
			if err != nil {
				return nil, err
			}
			return order, nil
		} else {
			return nil, err
		}
	}
	return order, nil
}

// GetUserOrders returns slice of orders for user with specified login
func (service *OrdersService) GetUserOrders(ctx context.Context, userLogin string) ([]*models.Order, error) {
	logger := logger.GetLoggerFromContext(ctx)
	orders, err := service.ordersStorage.GetAllForUser(ctx, userLogin)
	if err != nil {
		return []*models.Order{}, err
	}
	if len(orders) == 0 {
		err = &models.CustomErr{Code: models.CustomErrUserOrdersListEmpty}
		logger.Sugar().Error(err)
		return []*models.Order{}, err
	}
	for i, o := range orders {
		accrualOrder := service.accrualService.GetOrderData(ctx, o.ID)
		if accrualOrder != nil {
			if accrualOrder.Status != o.Status || !utils.EqualFloat32Ptr(accrualOrder.Accrual, o.Accrual, 1e-6) {
				orders[i], err = service.updateChangedOrder(ctx, accrualOrder, userLogin)
				if err != nil {
					return []*models.Order{}, err
				}
			}
		}
	}
	return orders, nil
}

func (service *OrdersService) updateChangedOrder(
	ctx context.Context,
	accrualOrder *models.Order,
	userLogin string,
) (*models.Order, error) {
	tx, err := service.repositoryUtils.CreateTransaction(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	ctx = context.WithValue(ctx, models.DbTransactionKey, tx)
	err = service.ordersStorage.Update(ctx, accrualOrder, userLogin)
	if err != nil {
		return nil, err
	}
	err = service.usersStorage.AddBalance(ctx, userLogin, accrualOrder.Accrual)
	if err != nil {
		return nil, err
	}
	err = service.repositoryUtils.CommitTransaction(ctx, tx)
	if err != nil {
		return nil, err
	}
	return service.ordersStorage.Get(ctx, accrualOrder.ID)
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
	order, err := service.ordersStorage.Get(ctx, request.Order)
	if err != nil {
		var customErr *models.CustomErr
		if errors.As(err, &customErr) && customErr.Code == models.CustomErrOrderNotFound {
			order = nil
		} else {
			return err
		}
	}
	if order == nil {
		order, err = service.Upload(ctx, request.Order, login)
		if err != nil {
			return err
		}
	} else {
		accrualOrder := service.accrualService.GetOrderData(ctx, request.Order)
		if accrualOrder != nil {
			if accrualOrder.Status != order.Status || !utils.EqualFloat32Ptr(accrualOrder.Accrual, order.Accrual, 1e-6) {
				order, err = service.updateChangedOrder(ctx, accrualOrder, login)
				if err != nil {
					return err
				}
			}
		}
	}
	tx, err := service.repositoryUtils.CreateTransaction(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	ctx = context.WithValue(ctx, models.DbTransactionKey, tx)
	err = service.usersStorage.WithdrawBalance(ctx, login, request.Sum)
	if err != nil {
		return err
	}
	err = service.ordersStorage.WithdrawSum(ctx, request.Order, request.Sum, login)
	if err != nil {
		return err
	}
	err = service.repositoryUtils.CommitTransaction(ctx, tx)
	if err != nil {
		return err
	}
	return nil
}

// WithdrawalsForUser returns slice of withdrawals data for specified user
func (service *OrdersService) WithdrawalsForUser(ctx context.Context, login string) ([]models.WithdrawalResponse, error) {
	logger := logger.GetLoggerFromContext(ctx)
	withdrawals, err := service.ordersStorage.WithdrawalsForUser(ctx, login)
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
