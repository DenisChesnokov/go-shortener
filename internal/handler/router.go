package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/DenisChesnokov/go-shortener.git/internal/logger"
	"github.com/DenisChesnokov/go-shortener.git/internal/middleware"
)

// NewRouter собирает и возвращает chi-роутер с привязанными обработчиками.
// Функция называется NewRouter (а не run), т.к. она конструирует объект-роутер
// и не запускает сервер
func NewRouter(h *Handler) *chi.Mux {
	r := chi.NewRouter()

	r.NotFound(logger.RequestLogger(middleware.GzipMiddleware(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
	})))
	r.MethodNotAllowed(logger.RequestLogger(middleware.GzipMiddleware(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
	})))

	r.Post("/", logger.RequestLogger(middleware.GzipMiddleware(h.PostShorten)))
	r.Post("/api/shorten", logger.RequestLogger(middleware.GzipMiddleware(h.PostShortenJSON)))
	r.Get("/{shortLink}", logger.RequestLogger(middleware.GzipMiddleware(h.GetRedirect)))

	return r
}
