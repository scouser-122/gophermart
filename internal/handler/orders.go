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
	logger := logger.GetLogger(req.Context())

	bodyBuf, err := io.ReadAll(req.Body)
	if err != nil {
		logger.Error("cannot read request body", zap.Error(err))
		res.WriteHeader(http.StatusBadRequest)
		res.Write(models.NewErrorResponseBuffer(models.UnexpectedErrorMessage))
		return
	}

	orderID := string(bodyBuf)

	userLogin, ok := req.Context().Value(UserLoginContextKey).(string)
	if !ok {
		logger.Error("unauthorized request", zap.Error(err))
		res.WriteHeader(http.StatusUnauthorized)
		res.Write(models.NewErrorResponseBuffer(models.UnexpectedErrorMessage))
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
			logger.Error(models.UnexpectedErrorMessage, zap.Error(err))
			res.WriteHeader(http.StatusInternalServerError)
			res.Write(models.NewErrorResponseBuffer(models.UnexpectedErrorMessage))
			return
		}
	}

	response := models.CommonResponse{
		Status:  "ok",
		Message: "order successfully saved",
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	if err := enc.Encode(response); err != nil {
		logger.Error("error encoding response", zap.Error(err))
		res.WriteHeader(http.StatusInternalServerError)
		res.Write(models.NewErrorResponseBuffer(models.UnexpectedErrorMessage))
		return
	}

	logger.Info(response.Message)
	res.WriteHeader(http.StatusAccepted)
	res.Write(buf.Bytes())
}

func processCustomErrorOrderUpload(customErr *models.CustomErr) (int, string) {
	var status int
	var errMessage string
	switch customErr.Code {
	case models.CustomErrOrderIDInvalidFormat:
		errMessage = "order id invalid format"
		status = http.StatusBadRequest
	case models.CustomErrOrderAlreadyUploaded:
		errMessage = "order already uploaded"
		status = http.StatusOK
	default:
		errMessage = models.UnexpectedErrorMessage
		status = http.StatusInternalServerError
	}
	return status, errMessage
}
