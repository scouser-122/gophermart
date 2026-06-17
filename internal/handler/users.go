package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/pkg/errors"

	"github.com/scouser-122/gophermart/internal/logger"
	"github.com/scouser-122/gophermart/internal/models"
	"github.com/scouser-122/gophermart/internal/service"
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
	logger := logger.GetSlogLoggerFromContext(req.Context())

	bodyBuf, err := io.ReadAll(req.Body)
	if err != nil {
		logger.Error("cannot read request body", "err", err)
		res.WriteHeader(http.StatusBadRequest)
		res.Write(models.NewErrorResponseBuffer(models.UnexpectedErrorMessage))
		return
	}

	var user models.User
	if err := json.Unmarshal(bodyBuf, &user); err != nil {
		logger.Error("cannot decode request json body", "err", err)
		res.WriteHeader(http.StatusBadRequest)
		res.Write(models.NewErrorResponseBuffer(models.UnexpectedErrorMessage))
		return
	}

	registeredUser, err := h.usersService.Register(req.Context(), &user)
	if err != nil {
		var customErr *models.CustomErr
		if errors.As(err, &customErr) {
			errMessage := processCustomErrorRegister(customErr, res)
			logger.Info(errMessage)
			res.Write(models.NewErrorResponseBuffer(errMessage))
			return
		} else {
			res.WriteHeader(http.StatusInternalServerError)
			logger.Info(models.UnexpectedErrorMessage)
			res.Write(models.NewErrorResponseBuffer(models.UnexpectedErrorMessage))
			return
		}
	}

	authToken, err := h.jwtService.GenerateJWT(registeredUser.Login)
	if err != nil {
		logger.Error("error generating JWT token", "err", err)
		res.WriteHeader(http.StatusInternalServerError)
		res.Write(models.NewErrorResponseBuffer(models.UnexpectedErrorMessage))
		return
	}

	res.Header().Add("Authorization", fmt.Sprintf("Bearer %s", authToken))

	successMessage := "user successfully registered"
	logger.Info(successMessage, slog.String("login", registeredUser.Login))
	res.WriteHeader(http.StatusOK)
	res.Write(models.NewSuccessResponseBuffer(successMessage))
}

func processCustomErrorRegister(customErr *models.CustomErr, res http.ResponseWriter) string {
	var errMessage string
	switch customErr.Code {
	case models.CustomErrUserLoginBusy:
		errMessage = "login already taken"
		res.WriteHeader(http.StatusConflict)
	case models.CustomErrUserLoginInvalidFormat:
		errMessage = "login invalid format"
		res.WriteHeader(http.StatusBadRequest)
	case models.CustomErrUserPasswordInvalidFormat:
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
	logger := logger.GetSlogLoggerFromContext(req.Context())

	bodyBuf, err := io.ReadAll(req.Body)
	if err != nil {
		logger.Error("cannot read request body", "err", err)
		res.WriteHeader(http.StatusBadRequest)
		res.Write(models.NewErrorResponseBuffer(models.UnexpectedErrorMessage))
		return
	}

	var user models.User
	if err := json.Unmarshal(bodyBuf, &user); err != nil {
		logger.Error("cannot decode request json body", "err", err)
		res.WriteHeader(http.StatusBadRequest)
		res.Write(models.NewErrorResponseBuffer(models.UnexpectedErrorMessage))
		return
	}

	err = h.usersService.Login(req.Context(), &user)
	if err != nil {
		var customErr *models.CustomErr
		if errors.As(err, &customErr) {
			errMessage := processCustomErrorLogin(customErr, res)
			logger.Error(customErr.Error())
			res.Write(models.NewErrorResponseBuffer(errMessage))
			return
		} else {
			res.WriteHeader(http.StatusInternalServerError)
			res.Write(models.NewErrorResponseBuffer(models.UnexpectedErrorMessage))
			return
		}
	}

	authToken, err := h.jwtService.GenerateJWT(user.Login)
	if err != nil {
		logger.Error("error generating JWT token", "err", err)
		res.WriteHeader(http.StatusInternalServerError)
		res.Write(models.NewErrorResponseBuffer(models.UnexpectedErrorMessage))
		return
	}

	res.Header().Add("Authorization", fmt.Sprintf("Bearer %s", authToken))

	successMessage := "successfully logged in"
	logger.Info(successMessage, slog.String("login", user.Login))
	res.WriteHeader(http.StatusOK)
	res.Write(models.NewSuccessResponseBuffer(successMessage))
}

func processCustomErrorLogin(customErr *models.CustomErr, res http.ResponseWriter) string {
	var errMessage string
	switch customErr.Code {
	case models.CustomErrUserLoginBusy:
		errMessage = "login already taken"
		res.WriteHeader(http.StatusConflict)
	case models.CustomErrUserLoginInvalidFormat:
		errMessage = "login invalid format"
		res.WriteHeader(http.StatusBadRequest)
	case models.CustomErrUserPasswordInvalidFormat:
		errMessage = "login invalid format"
		res.WriteHeader(http.StatusBadRequest)
	case models.CustomErrUserNotFound:
		errMessage = "user not found"
		res.WriteHeader(http.StatusUnauthorized)
	case models.CustomErrUserLoginPasswordNotMatch:
		errMessage = "login invalid format"
		res.WriteHeader(http.StatusUnauthorized)
	default:
		errMessage = models.UnexpectedErrorMessage
		res.WriteHeader(http.StatusInternalServerError)
	}
	return errMessage
}

// HandleUsersBalance processes get user balance request
func (h *UsersHandler) HandleUsersBalance(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("content-type", "application/json")
	logger := logger.GetSlogLoggerFromContext(req.Context())

	userLogin, ok := req.Context().Value(UserLoginContextKey).(string)
	if !ok {
		logger.Error("unauthorized request")
		res.WriteHeader(http.StatusUnauthorized)
		res.Write(models.NewErrorResponseBuffer("unauthorized request"))
		return
	}

	usersBalance, err := h.usersService.GetUserBalance(req.Context(), userLogin)
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		res.Write(models.NewErrorResponseBuffer(models.UnexpectedErrorMessage))
		return
	}

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	if err := enc.Encode(usersBalance); err != nil {
		logger.Error("cannot endcode request body", "err", err)
		res.WriteHeader(http.StatusInternalServerError)
		res.Write(models.NewErrorResponseBuffer(models.UnexpectedErrorMessage))
		return
	}
	logger.Info("users balance successfully obtained", slog.String("login", userLogin))
	res.WriteHeader(http.StatusOK)
	res.Write(buf.Bytes())
}
