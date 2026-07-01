package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// NewRouter собирает и возвращает chi-роутер с привязанными обработчиками.
// Функция называется NewRouter (а не run), т.к. она конструирует объект-роутер
// и не запускает сервер
func NewRouter(h *Handler) *chi.Mux {
	r := chi.NewRouter()

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
	})
	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
	})

	r.Post("/", h.PostShorten)
	r.Get("/{shortLink}", h.GetRedirect)

	return r
}
