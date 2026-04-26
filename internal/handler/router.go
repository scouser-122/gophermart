package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/scouser-122/gophermart/internal/config"
	"github.com/scouser-122/gophermart/internal/logger"
)

func CreateChiRouter(handlers *[]Handler, serverConfig *config.ServerConfig) *chi.Mux {
	r := chi.NewRouter()
	for _, h := range *handlers {
		switch h.Method {
		case http.MethodGet:
			r.Get(h.URLPathPattern, GzipMiddleware(RequestLogger(h.HandlerFn, serverConfig)))
		case http.MethodPost:
			r.Post(h.URLPathPattern, GzipMiddleware(RequestLogger(h.HandlerFn, serverConfig)))
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
