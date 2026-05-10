package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/scouser-122/gophermart/internal/config"
	"github.com/scouser-122/gophermart/internal/handler"
	"github.com/scouser-122/gophermart/internal/logger"
	"github.com/scouser-122/gophermart/internal/repository/db"
	"github.com/scouser-122/gophermart/internal/service"
)

func main() {
	serverConfig := config.DefaultServerConfig()
	config.ParseFlags(&serverConfig)
	config.ParseEnvVariables(&serverConfig)
	if err := logger.Initialize(serverConfig.LogLevel, serverConfig.Environment); err != nil {
		panic(err)
	}

	database := db.NewPostgresDB(serverConfig)
	if err := database.Open(); err != nil {
		logger.Sugar.Errorf("cannot connect to database: %w", err)
		panic(err)
	}
	if err := database.Ping(context.Background()); err != nil {
		panic(fmt.Errorf("cannot connect to DB"))
	}
	defer database.Close()

	accrualService := service.NewAccrualService(&serverConfig)
	orderStorage := &db.PostgresOrderStorage{Database: &database}
	ordersService := service.NewOrdersService(orderStorage, accrualService)
	usersStorage := &db.PostgresUserStorage{Database: &database}
	userService := service.NewUsersService(usersStorage, ordersService)

	jwtService := service.NewJwtService(&serverConfig)

	handlers := handler.CreateHandlers(userService, ordersService, jwtService)
	router := handler.CreateChiRouter(&handlers, &serverConfig, jwtService)

	logger.Sugar.Infof("starting server on http://%s", serverConfig.RunAddr)
	logger.Sugar.Fatal(http.ListenAndServe(serverConfig.RunAddr, router))
}
