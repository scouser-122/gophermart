package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

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

	var jsonWriter io.Writer
	if serverConfig.Environment == "prod" {
		file, err := os.OpenFile("app.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			panic(err)
		}
		defer file.Close()
		autoFlushWriter := logger.NewAutoFlushWriter(file, 1024, 100*time.Millisecond)
		defer autoFlushWriter.Close()
		jsonWriter = autoFlushWriter
	}
	logger.Initialize(serverConfig.LogLevel, os.Stdout, jsonWriter)

	database := db.NewPostgresDB(serverConfig)
	if err := database.Open(); err != nil {
		slog.Error("cannot connect to database", "err", err)
		panic(err)
	}
	if err := database.Ping(context.Background()); err != nil {
		panic(fmt.Errorf("cannot connect to DB"))
	}
	defer database.Close()

	repositoryUtils := db.NewPostgresRepositoryUtils(&database)
	accrualService := service.NewAccrualService(&serverConfig)
	ordersStorage := db.NewPostgresOrderStorage(&database)
	usersStorage := db.NewPostgresUserStorage(&database)
	ordersService := service.NewOrdersService(ordersStorage, usersStorage, repositoryUtils, accrualService)
	userService := service.NewUsersService(usersStorage, ordersStorage)

	jwtService := service.NewJwtService(&serverConfig)

	handlers := handler.CreateHandlers(userService, ordersService, jwtService)
	router := handler.CreateChiRouter(&handlers, &serverConfig, jwtService)

	slog.Info("starting server", "address", serverConfig.RunAddr)
	err := http.ListenAndServe(serverConfig.RunAddr, router)
	if err != nil {
		slog.Error(err.Error())
		panic(err)
	}
}
