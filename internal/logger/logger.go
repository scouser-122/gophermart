package logger

import (
	"context"
	"fmt"

	"go.uber.org/zap"
)

var Log *zap.Logger = zap.NewNop()
var Sugar = Log.Sugar()

func Initialize(level string, environment string) error {
	Log = NewZapLogger(level, environment)
	Sugar = Log.Sugar()
	return nil
}

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
	zl, err := cfg.Build()
	if err != nil {
		return nil
	}
	return zl
}

const LoggerKey string = "logger"

func GetLogger(ctx context.Context) *zap.Logger {
	if logger, ok := ctx.Value(LoggerKey).(*zap.Logger); ok {
		return logger
	}
	return zap.L()
}
