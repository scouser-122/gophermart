package logger

import (
	"context"
	"fmt"

	"go.uber.org/zap"
)

// Log default zap logger to use in code
var Log *zap.Logger = zap.NewNop()

// Sugar is a sugar wrapper for default zap logger
var Sugar = Log.Sugar()

// Initialize creates default zap logger with specified logging level and run environment
func Initialize(level string, environment string) error {
	Log = NewZapLogger(level, environment)
	Sugar = Log.Sugar()
	return nil
}

// NewZapLogger creates new zap logger with specified logging level and run environment
func NewZapLogger(level string, environment string) *zap.Logger {
	lvl, err := zap.ParseAtomicLevel(level)
	if err != nil {
		return nil
	}
	var cfg zap.Config
	switch environment {
	case "dev":
		cfg = zap.NewDevelopmentConfig()
	case "prod":
		cfg = zap.NewProductionConfig()
	default:
		panic(fmt.Errorf("unknown environment for logging config: %s", environment))
	}
	cfg.Level = lvl
	zl, err := cfg.Build(zap.AddStacktrace(zap.ErrorLevel))
	if err != nil {
		return nil
	}
	return zl
}

// LoggerKey key value to store logger in context
const LoggerKey string = "logger"

// GetLogger takes logger from context or returns global zap logger
func GetLogger(ctx context.Context) *zap.Logger {
	if logger, ok := ctx.Value(LoggerKey).(*zap.Logger); ok {
		return logger
	}
	return zap.L()
}
