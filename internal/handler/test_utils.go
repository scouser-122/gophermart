package handler

import (
	"github.com/go-chi/chi/v5"
	"github.com/scouser-122/gophermart/internal/config"
	"github.com/scouser-122/gophermart/internal/repository/db"
	"github.com/scouser-122/gophermart/internal/service"
)

type want struct {
	code               int
	contentType        string
	body               string
	headersToBePresent []string
}

type request struct {
	method      string
	contentType string
	path        string
	body        string
	headers     map[string]string
}

func createTestRouter(mockDB *db.MockPostgresDBTestData) *chi.Mux {
	serverConfig := config.DefaultServerConfig()
	mockDB.MockPool.MockMethods(*mockDB)
	database := db.NewMockPostgresDB(serverConfig, mockDB.MockPool)

	userService := service.NewUsersService(&database)
	ordersService := service.NewOrdersService(&database)
	jwtService := service.NewJwtService(&serverConfig)

	handlers := CreateHandlers(userService, ordersService, jwtService)

	return CreateChiRouter(&handlers, &serverConfig, jwtService)
}
