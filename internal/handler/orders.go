package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"github.com/pkg/errors"

	"github.com/scouser-122/gophermart/internal/logger"
	"github.com/scouser-122/gophermart/internal/models"
	"github.com/scouser-122/gophermart/internal/service"
	"go.uber.org/zap"
)

// OrdersHandler specifies http request handler for requests to Orders service
type OrdersHandler struct {
	ordersService *service.OrdersService
	jwtService    *service.JwtService
}

// NewOrdersHandler creates and returns pointer to new OrdersHandler
func NewOrdersHandler(
	ordersService *service.OrdersService,
	jwtService *service.JwtService,
) *OrdersHandler {
	return &OrdersHandler{
		ordersService: ordersService,
		jwtService:    jwtService,
	}
}

// HandleUploadOrder processes order upload request
func (h *OrdersHandler) HandleUploadOrder(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("content-type", "application/json")
	logger := logger.GetLoggerFromContext(req.Context())

	reqContentType := req.Header.Get("Content-Type")
	if reqContentType != "text/plain" {
		logger.Sugar().Errorf("incorrect request content type", reqContentType)
		res.WriteHeader(http.StatusBadRequest)
		res.Write(models.NewErrorResponseBuffer("incorrect request content type"))
		return
	}

	bodyBuf, err := io.ReadAll(req.Body)
	if err != nil {
		logger.Error("cannot read request body", zap.Error(err))
		res.WriteHeader(http.StatusBadRequest)
		res.Write(models.NewErrorResponseBuffer("cannot read request body"))
		return
	}

	orderID := string(bodyBuf)

	userLogin, ok := req.Context().Value(UserLoginContextKey).(string)
	if !ok {
		logger.Error("unauthorized request")
		res.WriteHeader(http.StatusUnauthorized)
		res.Write(models.NewErrorResponseBuffer("unauthorized request"))
		return
	}

	_, err = h.ordersService.Upload(req.Context(), orderID, userLogin)
	if err != nil {
		var customErr *models.CustomErr
		if errors.As(err, &customErr) {
			status, message := processCustomErrorOrderUpload(customErr)
			res.WriteHeader(status)
			if status >= http.StatusBadRequest {
				logger.Error(message, zap.Error(err))
				res.Write(models.NewErrorResponseBuffer(message))
			} else {
				logger.Sugar().Info(message)
				res.Write(models.NewSuccessResponseBuffer(message))
			}
			return
		} else {
			res.WriteHeader(http.StatusInternalServerError)
			res.Write(models.NewErrorResponseBuffer(models.UnexpectedErrorMessage))
			return
		}
	}

	successMessage := "order successfully saved"
	logger.Info(successMessage)
	res.WriteHeader(http.StatusAccepted)
	res.Write(models.NewSuccessResponseBuffer(successMessage))
}

func processCustomErrorOrderUpload(customErr *models.CustomErr) (int, string) {
	var status int
	var errMessage string
	switch customErr.Code {
	case models.CustomErrOrderIDInvalidFormat:
		errMessage = "order id invalid format"
		status = http.StatusUnprocessableEntity
	case models.CustomErrOrderAlreadyUploaded:
		errMessage = "order already uploaded"
		status = http.StatusOK
	case models.CustomErrOrderAlreadyUploadedByAnotherUser:
		errMessage = "order already uploaded by another user"
		status = http.StatusConflict
	case models.CustomErrAccrualOrderNotRegistered:
	case models.CustomErrAccrualTooManyRequests:
	case models.CustomErrAccrualInternalServerError:
		errMessage = models.UnexpectedErrorMessage
		status = http.StatusInternalServerError
	default:
		errMessage = models.UnexpectedErrorMessage
		status = http.StatusInternalServerError
	}
	return status, errMessage
}

// HandleGetUserOrders processes user orders request
func (h *OrdersHandler) HandleGetUserOrders(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("content-type", "application/json")
	logger := logger.GetLoggerFromContext(req.Context())

	userLogin, ok := req.Context().Value(UserLoginContextKey).(string)
	if !ok {
		logger.Error("unauthorized request")
		res.WriteHeader(http.StatusUnauthorized)
		res.Write(models.NewErrorResponseBuffer("unauthorized request"))
		return
	}

	orders, err := h.ordersService.GetUserOrders(req.Context(), userLogin)
	if err != nil {
		var customErr *models.CustomErr
		if errors.As(err, &customErr) {
			status, message := processCustomErrorGetUserOrders(customErr)
			res.WriteHeader(status)
			if status >= http.StatusBadRequest {
				logger.Error(message, zap.Error(err))
				res.Write(models.NewErrorResponseBuffer(message))
			} else {
				logger.Sugar().Info(message)
				res.Write(models.NewSuccessResponseBuffer(message))
			}
			return
		} else {
			res.WriteHeader(http.StatusInternalServerError)
			res.Write(models.NewErrorResponseBuffer(models.UnexpectedErrorMessage))
			return
		}
	}

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	if err := enc.Encode(orders); err != nil {
		logger.Error("error encoding response ", zap.Error(err))
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	logger.Info("orders list successfully obtained")
	res.WriteHeader(http.StatusOK)
	res.Write(buf.Bytes())
}

func processCustomErrorGetUserOrders(customErr *models.CustomErr) (int, string) {
	var status int
	var errMessage string
	switch customErr.Code {
	case models.CustomErrUserOrdersListEmpty:
		errMessage = "orders absent for this user"
		status = http.StatusNoContent
	default:
		errMessage = models.UnexpectedErrorMessage
		status = http.StatusInternalServerError
	}
	return status, errMessage
}
