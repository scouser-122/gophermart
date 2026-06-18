package service

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/scouser-122/gophermart/internal/config"
	"github.com/scouser-122/gophermart/internal/logger"
	"github.com/scouser-122/gophermart/internal/models"
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

// GetOrderData obtains order data by id from loyalty points calculation system,
// return pointer to order data or nil if order is absent in system or request failed
func (service *AccrualService) GetOrderData(ctx context.Context, orderID string) *models.Order {
	logger := logger.GetSlogLoggerFromContext(ctx)
	client := resty.New()
	client.SetRetryCount(3).
		SetRetryWaitTime(1 * time.Second).
		SetRetryMaxWaitTime(2 * time.Second)
	client.AddRetryCondition(
		func(r *resty.Response, err error) bool {
			return err != nil ||
				r.StatusCode() == http.StatusRequestTimeout ||
				r.StatusCode() == http.StatusTooManyRequests ||
				r.StatusCode() == http.StatusInternalServerError
		},
	)
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
		logger.Error(err.Error())
		return nil
	}
	logger.Info("accrual service response", "status", resp.StatusCode())
	if resp.StatusCode() == http.StatusOK {
		if order.Status == models.RegisteredOrder {
			order.Status = models.NewOrder
		}
		return &order
	}
	return nil
}
