package handler

import (
	"errors"
	"io"
	"log"
	"net/http"

	"github.com/DenisChesnokov/go-shortener.git/internal/repository"
	"github.com/DenisChesnokov/go-shortener.git/internal/service"
	"github.com/go-chi/chi/v5"
)

type Handler struct {
	svc *service.Shortener
}

func New(svc *service.Shortener) *Handler {
	return &Handler{
		svc: svc,
	}
}

// PostShorten обрабатывает POST /: принимает в теле длинный URL как text/plain
// и возвращает сокращённый URL в теле ответа с кодом 201 Created.
func (h *Handler) PostShorten(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil || len(bodyBytes) == 0 {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	longURL := string(bodyBytes)

	shortURL, err := h.svc.Shorten(r.Context(), longURL)
	if err != nil {
		log.Printf("shorten failed: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(shortURL))
}

// GetRedirect обрабатывает GET /{shortLink}: ищет оригинальный URL по короткому
// ключу и возвращает 307 Temporary Redirect с заголовком Location.
func (h *Handler) GetRedirect(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "shortLink")
	if key == "" {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	longURL, err := h.svc.Resolve(r.Context(), key)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		log.Printf("resolve failed: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Location", longURL)
	w.WriteHeader(http.StatusTemporaryRedirect)
}
