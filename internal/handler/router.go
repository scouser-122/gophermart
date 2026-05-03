package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/scouser-122/gophermart/internal/config"
	"github.com/scouser-122/gophermart/internal/logger"
	"github.com/scouser-122/gophermart/internal/service"
)

// CreateChiRouter creates and returns chi router for processing http requests
func CreateChiRouter(handlers *[]Handler, serverConfig *config.ServerConfig, jwtService *service.JwtService) *chi.Mux {
	r := chi.NewRouter()
	middleware := func(h Handler) http.HandlerFunc {
		return GzipMiddleware(
			RequestLogger(
				AuthMiddleware(h.HandlerFn, jwtService),
				serverConfig,
			),
		)
	}
	for _, h := range *handlers {
		switch h.Method {
		case http.MethodGet:
			r.Get(h.URLPathPattern, middleware(h))
		case http.MethodPost:
			r.Post(h.URLPathPattern, middleware(h))
		default:
			logger.Sugar.Errorf("Provided unsupported request handler method: %q", h)
		}
	}
	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "text/plain")
		w.WriteHeader(405)
	})
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("content-type", "text/plain")
		w.WriteHeader(404)
	})
	return r
}
