package handler

import (
	"github.com/go-chi/chi/v5"
	"github.com/pashagolub/pgxmock/v5"
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

type accrualResponse struct {
	status int
	body   string
	err    error
}

func createTestRouter(mockDB *db.MockPostgresDBTestData, accrualServiceURL string) *chi.Mux {
	serverConfig := config.DefaultServerConfig()
	serverConfig.AccrualSystemAddress = accrualServiceURL
	mock, err := pgxmock.NewPool()
	if err != nil {
		panic(err)
	}
	defer mock.Close()
	mockDB.PgxPoolIface = mock
	mockDB.MockDBCalls(*mockDB)
	database := db.NewPgxMockDB(serverConfig, mock)

	accrualService := service.NewAccrualService(&serverConfig)
	orderStorage := &db.PostgresOrderStorage{Database: &database}
	ordersService := service.NewOrdersService(orderStorage, accrualService)
	usersStorage := &db.PostgresUserStorage{Database: &database}
	userService := service.NewUsersService(usersStorage, ordersService)

	jwtService := service.NewJwtService(&serverConfig)

	handlers := CreateHandlers(userService, ordersService, jwtService)

	return CreateChiRouter(&handlers, &serverConfig, jwtService)
}

func Ptr[T any](v T) *T {
	return &v
}
