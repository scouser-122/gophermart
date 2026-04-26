package handler

import (
	"net/http"

	"github.com/scouser-122/gophermart/internal/service"
)

type Handler struct {
	Method         string
	URLPathPattern string
	HandlerFn      http.HandlerFunc
}

func CreateHandlers(
	userService *service.UserService,
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
