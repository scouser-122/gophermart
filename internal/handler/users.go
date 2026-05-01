package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/pkg/errors"

	"github.com/scouser-122/gophermart/internal/logger"
	"github.com/scouser-122/gophermart/internal/models"
	"github.com/scouser-122/gophermart/internal/service"
	"go.uber.org/zap"
)

// UsersHandler specifies http request handler for requests to Users service
type UsersHandler struct {
	usersService *service.UsersService
	jwtService   *service.JwtService
}

// NewUsersHandler creates and returns pointer to new UsersHandler
func NewUsersHandler(
	usersService *service.UsersService,
	jwtService *service.JwtService,
) *UsersHandler {
	return &UsersHandler{
		usersService: usersService,
		jwtService:   jwtService,
	}
}

// HandleRegister processes user registration request
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

	registeredUser, err := h.usersService.Register(req.Context(), &user)
	if err != nil {
		var customErr *models.CustomErr
		if errors.As(err, &customErr) {
			errMessage := processCustomErrorRegister(customErr, res)
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

	authToken, err := h.jwtService.GenerateJWT(registeredUser.Login)
	if err != nil {
		logger.Error("error generating JWT token", zap.Error(err))
		res.WriteHeader(http.StatusInternalServerError)
		res.Write(models.NewErrorResponseBuffer(models.UnexpectedErrorMessage))
		return
	}

	res.Header().Add("Authorization", fmt.Sprintf("Bearer %s", authToken))

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

	logger.Info(response.Message)
	res.WriteHeader(http.StatusOK)
	res.Write(buf.Bytes())
}

func processCustomErrorRegister(customErr *models.CustomErr, res http.ResponseWriter) string {
	var errMessage string
	switch customErr.Code {
	case models.CustomErrLoginBusy:
		errMessage = "login already taken"
		res.WriteHeader(http.StatusConflict)
	case models.CustomErrLoginInvalidFormat:
		errMessage = "login invalid format"
		res.WriteHeader(http.StatusBadRequest)
	case models.CustomErrPasswordInvalidFormat:
		errMessage = "login invalid format"
		res.WriteHeader(http.StatusBadRequest)
	default:
		errMessage = models.UnexpectedErrorMessage
		res.WriteHeader(http.StatusInternalServerError)
	}
	return errMessage
}

// HandleLogin processes user login request
func (h *UsersHandler) HandleLogin(res http.ResponseWriter, req *http.Request) {
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

	err = h.usersService.Login(req.Context(), &user)
	if err != nil {
		var customErr *models.CustomErr
		if errors.As(err, &customErr) {
			errMessage := processCustomErrorLogin(customErr, res)
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

	authToken, err := h.jwtService.GenerateJWT(user.Login)
	if err != nil {
		logger.Error("error generating JWT token", zap.Error(err))
		res.WriteHeader(http.StatusInternalServerError)
		res.Write(models.NewErrorResponseBuffer(models.UnexpectedErrorMessage))
		return
	}

	res.Header().Add("Authorization", fmt.Sprintf("Bearer %s", authToken))

	response := models.CommonResponse{
		Status:  "ok",
		Message: "successfully logged in",
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
	res.WriteHeader(http.StatusOK)
	res.Write(buf.Bytes())
}

func processCustomErrorLogin(customErr *models.CustomErr, res http.ResponseWriter) string {
	var errMessage string
	switch customErr.Code {
	case models.CustomErrLoginBusy:
		errMessage = "login already taken"
		res.WriteHeader(http.StatusConflict)
	case models.CustomErrLoginInvalidFormat:
		errMessage = "login invalid format"
		res.WriteHeader(http.StatusBadRequest)
	case models.CustomErrPasswordInvalidFormat:
		errMessage = "login invalid format"
		res.WriteHeader(http.StatusBadRequest)
	case models.CustomErrLoginFailed:
		errMessage = "login invalid format"
		res.WriteHeader(http.StatusUnauthorized)
	default:
		errMessage = models.UnexpectedErrorMessage
		res.WriteHeader(http.StatusInternalServerError)
	}
	return errMessage
}
