package main

import (
	"net/http"

	"github.com/scouser-122/gophermart/internal/config"
	"github.com/scouser-122/gophermart/internal/handler"
	"github.com/scouser-122/gophermart/internal/logger"
	"github.com/scouser-122/gophermart/internal/service"
)

func main() {
	serverConfig := config.DefaultServerConfig()
	config.ParseFlags(&serverConfig)
	config.ParseEnvVariables(&serverConfig)
	if err := logger.Initialize(serverConfig.LogLevel, serverConfig.Environment); err != nil {
		panic(err)
	}

	userService := service.NewUserService()

	handlers := handler.CreateHandlers(userService)
	router := handler.CreateChiRouter(&handlers, &serverConfig)

	logger.Sugar.Infof("starting server on http://%s", serverConfig.RunAddr)
	logger.Sugar.Fatal(http.ListenAndServe(serverConfig.RunAddr, router))
}
