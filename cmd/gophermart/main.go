package main

import (
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
	defer database.Close()

	userService, err := service.NewUsersService(&database)
	if err != nil {
		panic(err)
	}
	jwtService := service.NewJwtService(&serverConfig)

	handlers := handler.CreateHandlers(userService, jwtService)
	router := handler.CreateChiRouter(&handlers, &serverConfig)

	logger.Sugar.Infof("starting server on http://%s", serverConfig.RunAddr)
	logger.Sugar.Fatal(http.ListenAndServe(serverConfig.RunAddr, router))
}
