package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"github.com/pkg/errors"

	"github.com/scouser-122/gophermart/internal/config"
	"github.com/scouser-122/gophermart/internal/logger"
	"github.com/scouser-122/gophermart/internal/models"
	"github.com/scouser-122/gophermart/internal/service"
	"go.uber.org/zap"
)

type UsersHandler struct {
	Service      *service.UserService
	ServerConfig *config.ServerConfig
}

func NewUsersHandler(service *service.UserService) *UsersHandler {
	return &UsersHandler{
		Service: service,
	}
}

func (h *UsersHandler) HandleRegister(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("content-type", "application/json")
	logger := logger.GetLogger(req.Context())

	bodyBuf, err := io.ReadAll(req.Body)
	if err != nil {
		logger.Error("cannot read request body", zap.Error(err))
		res.WriteHeader(http.StatusBadRequest)
		res.Write(models.NewErrorResponseBuffer(models.UnexpectedErrorMessage))
		return
	}

	var user models.User
	if err := json.Unmarshal(bodyBuf, &user); err != nil {
		logger.Error("cannot decode request json body", zap.Error(err))
		res.WriteHeader(http.StatusBadRequest)
		res.Write(models.NewErrorResponseBuffer(models.UnexpectedErrorMessage))
		return
	}

	_, err = h.Service.Register(req.Context(), &user)
	if err != nil {
		var customErr *models.CustomErr
		if errors.As(err, &customErr) {
			var errMessage string
			switch customErr.Code {
			case models.CustomErrLoginBusy:
				errMessage = "login already taken"
				res.WriteHeader(http.StatusConflict)
			case models.CustomErrLoginInvalidFormat:
				errMessage = "login invalid format"
				res.WriteHeader(http.StatusBadRequest)
			default:
				errMessage = models.UnexpectedErrorMessage
				res.WriteHeader(http.StatusInternalServerError)
			}
			logger.Error(errMessage, zap.Error(err))
			res.Write(models.NewErrorResponseBuffer(errMessage))
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
		Message: "user successfully registered",
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	if err := enc.Encode(response); err != nil {
		logger.Error("error encoding response", zap.Error(err))
		res.WriteHeader(http.StatusInternalServerError)
		res.Write(models.NewErrorResponseBuffer(models.UnexpectedErrorMessage))
		return
	}

	logger.Info("user successfully registered")
	res.WriteHeader(http.StatusOK)
	res.Write(buf.Bytes())
}
