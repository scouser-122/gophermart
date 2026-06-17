package handler

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/scouser-122/gophermart/internal/config"
	"github.com/scouser-122/gophermart/internal/logger"
	"github.com/scouser-122/gophermart/internal/models"
)

// RequestLogger is a middleware which logs incoming requests
func RequestLogger(h http.HandlerFunc, serverConfig *config.ServerConfig) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		requestID := uuid.New().String()
		reqLogger := slog.Default().With(
			slog.String("request_id", requestID),
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
		)
		reqLogger.Info("request received")
		ctx := context.WithValue(r.Context(), logger.LoggerKey, reqLogger)

		responseData := &models.ResponseData{}
		lw := models.LoggingResponseWriter{
			ResponseWriter: w,
			ResponseData:   responseData,
		}

		if reqLogger.Enabled(nil, slog.LevelDebug) {
			bodyBytes, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "can't read body", http.StatusBadRequest)
				return
			}
			if len(bodyBytes) > 0 {
				reqLogger.With(slog.String("request_body", string(bodyBytes))).Debug("")
			}
			r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		}

		h(&lw, r.WithContext(ctx))

		duration := time.Since(start)

		// reqLogger.With(
		// 	zap.Int("status", responseData.Status),
		// 	zap.Duration("duration", duration),
		// ).Info("request processed")
		reqLogger.With(
			slog.Int("status", responseData.Status),
			slog.Duration("duration", duration),
		).Info("request processed")
	})
}
