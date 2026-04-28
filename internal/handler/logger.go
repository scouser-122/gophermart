package handler

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/scouser-122/gophermart/internal/config"
	"github.com/scouser-122/gophermart/internal/logger"
	"github.com/scouser-122/gophermart/internal/models"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// RequestLogger is a middleware which logs incoming requests
func RequestLogger(h http.HandlerFunc, serverConfig *config.ServerConfig) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		requestID := uuid.New().String()
		reqLogger := logger.NewZapLogger(serverConfig.LogLevel, serverConfig.Environment).With(
			zap.String("request_id", requestID),
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
		)
		reqLogger.Info("request received")
		ctx := context.WithValue(r.Context(), logger.LoggerKey, reqLogger)

		responseData := &models.ResponseData{}
		lw := models.LoggingResponseWriter{
			ResponseWriter: w,
			ResponseData:   responseData,
		}

		if logger.Log.Level() == zapcore.DebugLevel {
			bodyBytes, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "can't read body", http.StatusBadRequest)
				return
			}
			if len(bodyBytes) > 0 {
				reqLogger.With(zap.String("request_body", string(bodyBytes))).Debug("")
			}
			r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		}

		h(&lw, r.WithContext(ctx))

		duration := time.Since(start)

		reqLogger.With(
			zap.Int("status", responseData.Status),
			zap.Duration("duration", duration),
		).Info("request processed")
	})
}
