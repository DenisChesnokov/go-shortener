package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/DenisChesnokov/go-shortener.git/internal/logger"
	"github.com/DenisChesnokov/go-shortener.git/internal/middleware"
)

// NewRouter собирает и возвращает chi-роутер с привязанными обработчиками.
// Функция называется NewRouter (а не run), т.к. она конструирует объект-роутер
// и не запускает сервер
func NewRouter(h *Handler, log *zap.SugaredLogger) *chi.Mux {
	r := chi.NewRouter()

	r.Get("/ping", logger.RequestLogger(middleware.GzipMiddleware(h.GetPing), log))
	r.NotFound(logger.RequestLogger(middleware.GzipMiddleware(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
	}), log))
	r.MethodNotAllowed(logger.RequestLogger(middleware.GzipMiddleware(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
	}), log))

	r.Post("/", logger.RequestLogger(middleware.GzipMiddleware(h.PostShorten), log))
	r.Post("/api/shorten", logger.RequestLogger(middleware.GzipMiddleware(h.PostShortenJSON), log))
	r.Get("/{shortLink}", logger.RequestLogger(middleware.GzipMiddleware(h.GetRedirect), log))
	r.Post("/api/shorten/batch", logger.RequestLogger(middleware.GzipMiddleware(h.PostShortenBatch), log))

	return r
}
