package config

import (
	"flag"
	"log"

	"github.com/caarlos0/env/v6"
)

type ServerConfig struct {
	RunAddr          string `env:"RUN_ADDRESS"`
	LogLevel         string `env:"LOG_LEVEL"`
	Environment      string `env:"SERVICE_ENVIRONMENT"`
	DBDataSourceName string `env:"DATABASE_URI"`
}

func DefaultServerConfig() ServerConfig {
	return ServerConfig{
		RunAddr:          "localhost:8080",
		LogLevel:         "info",
		Environment:      "dev",
		DBDataSourceName: "postgres://postgres:password@localhost:5432/mydb?sslmode=disable",
	}
}

func ParseFlags(config *ServerConfig) {
	flag.StringVar(&config.RunAddr, "a", "localhost:8080", "address and port to run server")
	flag.StringVar(&config.LogLevel, "l", "info", "logging level")
	flag.StringVar(&config.Environment, "e", "dev", "environment")
	flag.StringVar(&config.DBDataSourceName, "d", "", "URI for database connection")
	flag.Parse()
}

func ParseEnvVariables(config *ServerConfig) {
	err := env.Parse(config)
	if err != nil {
		log.Fatal(err)
	}
}
