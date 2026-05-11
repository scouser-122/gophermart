package service

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-resty/resty/v2"
	"github.com/scouser-122/gophermart/internal/config"
	"github.com/scouser-122/gophermart/internal/logger"
	"github.com/scouser-122/gophermart/internal/models"
	"go.uber.org/zap"
)

// AccrualService service to work with loyalty points calculation system
type AccrualService struct {
	serverConfig *config.ServerConfig
}

// NewAccrualService creates new AccrualService
func NewAccrualService(serverConfig *config.ServerConfig) *AccrualService {
	return &AccrualService{
		serverConfig: serverConfig,
	}
}

// GetOrderData obtains order data by id from loyalty points calculation system
func (service *AccrualService) GetOrderData(ctx context.Context, orderID string) (*models.Order, error) {
	logger := logger.GetLoggerFromContext(ctx)
	client := resty.New()
	url := fmt.Sprintf(
		"%s/api/orders/%s",
		service.serverConfig.AccrualSystemAddress,
		orderID,
	)
	var order models.Order
	resp, err := client.R().
		SetResult(&order).
		Get(url)
	if err != nil {
		if resp.StatusCode() == http.StatusNoContent {
			err = &models.CustomErr{Code: models.CustomErrAccrualOrderNotRegistered}
			logger.Sugar().Error(err)
			return nil, err
		}
		logger.Sugar().Error(zap.Error(err))
		return nil, err
	}
	if resp.StatusCode() == http.StatusOK {
		if order.Status == models.RegisteredOrder {
			order.Status = models.NewOrder
		}
		return &order, nil
	}
	if resp.StatusCode() == http.StatusNoContent {
		err = &models.CustomErr{Code: models.CustomErrAccrualOrderNotRegistered}
		logger.Sugar().Error(err)
		return nil, err
	}
	if resp.StatusCode() == http.StatusTooManyRequests {
		err = &models.CustomErr{Code: models.CustomErrAccrualTooManyRequests}
		logger.Sugar().Error(err)
		return nil, err
	}
	err = &models.CustomErr{Code: models.CustomErrAccrualInternalServerError}
	logger.Sugar().Error(err)
	return nil, err
}
