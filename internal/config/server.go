package config

import (
	"flag"
	"log"

	"github.com/caarlos0/env/v6"
)

// ServerConfig for gophermart service
type ServerConfig struct {
	// RunAddr specifies address and port to run server app
	RunAddr string `env:"RUN_ADDRESS"`

	// LogLevel specifies logging level
	LogLevel string `env:"LOG_LEVEL"`

	// Environment specifies run environment, dev or prod
	Environment string `env:"SERVICE_ENVIRONMENT"`

	// DBDataSourceName specifies database URI
	DBDataSourceName string `env:"DATABASE_URI"`

	// JwtSecretKey specifies secret key for JWT token generation
	JwtSecretKey string `env:"JWT_SECRET_KEY"`

	// JwtTokenExp specifies time in hours for JWT token generation
	JwtTokenExp int `env:"JWT_TOKEN_EXP"`

	// AccrualSystemAddress adddress of accrual system for orders
	AccrualSystemAddress string `env:"ACCRUAL_SYSTEM_ADDRESS"`
}

// DefaultServerConfig specified default config for server app
// which may be redefined by command line args or environrment variables
func DefaultServerConfig() ServerConfig {
	return ServerConfig{
		RunAddr:              "localhost:8080",
		LogLevel:             "debug",
		Environment:          "dev",
		DBDataSourceName:     "postgres://postgres:password@localhost:5432/mydb?sslmode=disable",
		JwtSecretKey:         "defaultSecretKey",
		JwtTokenExp:          3,
		AccrualSystemAddress: "http://localhost:8081",
	}
}

// ParseFlags parses command line args for server configuration
func ParseFlags(config *ServerConfig) {
	flag.StringVar(&config.RunAddr, "a", "localhost:8080", "address and port to run server")
	flag.StringVar(&config.LogLevel, "l", "debug", "logging level")
	flag.StringVar(&config.Environment, "e", "dev", "environment")
	flag.StringVar(&config.DBDataSourceName, "d", "", "URI for database connection")
	flag.StringVar(&config.AccrualSystemAddress, "r", "http://localhost:8081", "accrual system address")
	flag.Parse()
}

// ParseEnvVariables parses environment variables for server configuration
func ParseEnvVariables(config *ServerConfig) {
	err := env.Parse(config)
	if err != nil {
		log.Fatal(err)
	}
}
