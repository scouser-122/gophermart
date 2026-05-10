package handler

import (
	"net/http"

	"github.com/scouser-122/gophermart/internal/service"
)

// Handler specifies parameters for http request handler
type Handler struct {
	// Method specifies http method
	Method string

	// URLPathPattern specifies request path
	URLPathPattern string

	// HandlerFn is a function which will be called to process request
	HandlerFn http.HandlerFunc
}

// CreateHandlers creates and returns http request handlers for this service
func CreateHandlers(
	userService *service.UsersService,
	ordersService *service.OrdersService,
	jwtService *service.JwtService,
) []Handler {
	handlers := []Handler{}

	usersHandler := NewUsersHandler(userService, jwtService)
	handlers = append(handlers, Handler{
		Method:         http.MethodPost,
		URLPathPattern: "/api/user/register",
		HandlerFn:      usersHandler.HandleRegister,
	})
	handlers = append(handlers, Handler{
		Method:         http.MethodPost,
		URLPathPattern: "/api/user/login",
		HandlerFn:      usersHandler.HandleLogin,
	})
	handlers = append(handlers, Handler{
		Method:         http.MethodGet,
		URLPathPattern: "/api/user/balance",
		HandlerFn:      usersHandler.HandleUsersBalance,
	})

	ordersHandler := NewOrdersHandler(ordersService, jwtService)
	handlers = append(handlers, Handler{
		Method:         http.MethodPost,
		URLPathPattern: "/api/user/orders",
		HandlerFn:      ordersHandler.HandleUploadOrder,
	})
	handlers = append(handlers, Handler{
		Method:         http.MethodGet,
		URLPathPattern: "/api/user/orders",
		HandlerFn:      ordersHandler.HandleGetUserOrders,
	})
	handlers = append(handlers, Handler{
		Method:         http.MethodPost,
		URLPathPattern: "/api/user/balance/withdraw",
		HandlerFn:      ordersHandler.HandleWithdrawBalance,
	})

	return handlers
}
