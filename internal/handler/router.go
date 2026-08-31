package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/DenisChesnokov/go-shortener.git/internal/auth"
	"github.com/DenisChesnokov/go-shortener.git/internal/logger"
	"github.com/DenisChesnokov/go-shortener.git/internal/middleware"
)

// NewRouter собирает и возвращает chi-роутер с привязанными обработчиками.
// Функция называется NewRouter (а не run), т.к. она конструирует объект-роутер
// и не запускает сервер
func NewRouter(h *Handler, log *zap.SugaredLogger, jwtMgr *auth.JWTManager) *chi.Mux {
	r := chi.NewRouter()

	r.Use(logger.RequestLogger(log))
	r.Use(middleware.GzipMiddleware)
	r.Use(middleware.CookieAuth(jwtMgr))

	NotFound := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
	})
	MethodNotAllowed := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
	})

	r.NotFound(NotFound)
	r.MethodNotAllowed(MethodNotAllowed)

	r.Get("/ping", h.GetPing)
	r.Post("/", h.PostShorten)
	r.Post("/api/shorten", h.PostShortenJSON)
	r.Get("/{shortLink}", h.GetRedirect)
	r.Post("/api/shorten/batch", h.PostShortenBatch)
	r.Get("/api/user/urls", h.GetUserURLs)

	return r
}
