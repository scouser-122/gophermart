package handler

import (
	"context"
	"net/http"
	"strings"

	"github.com/scouser-122/gophermart/internal/logger"
	"github.com/scouser-122/gophermart/internal/models"
	"github.com/scouser-122/gophermart/internal/service"
	"go.uber.org/zap"
)

const UserLoginContextKey string = "userLogin"

// AuthMiddleware is a middleware which retrieves auth data from incoming requests
func AuthMiddleware(h http.HandlerFunc, jwtService *service.JwtService) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			h.ServeHTTP(w, r)
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			logger.Sugar.Error("invalid authorization format")
			w.WriteHeader(http.StatusBadRequest)
			w.Write(models.NewErrorResponseBuffer("authorization failed"))
			return
		}

		tokenString := parts[1]

		userLogin, err := jwtService.GetUserLoginFromJWT(tokenString)
		if err != nil {
			logger.Sugar.Error("invalid or expired token", zap.Error(err))
			w.WriteHeader(http.StatusUnauthorized)
			w.Write(models.NewErrorResponseBuffer("authorization failed"))
			return
		}

		ctx := context.WithValue(r.Context(), UserLoginContextKey, userLogin)
		h.ServeHTTP(w, r.WithContext(ctx))
	})
}
