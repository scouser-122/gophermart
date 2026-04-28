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
) []Handler {
	handlers := []Handler{}

	usersHandler := UsersHandler{
		Service: userService,
	}
	handlers = append(handlers, Handler{
		Method:         http.MethodPost,
		URLPathPattern: "/api/user/register",
		HandlerFn:      usersHandler.HandleRegister,
	})

	return handlers
}
