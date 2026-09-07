package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"

	"github.com/DenisChesnokov/go-shortener.git/internal/middleware"
	"github.com/DenisChesnokov/go-shortener.git/internal/model"
	"github.com/DenisChesnokov/go-shortener.git/internal/repository"
	"github.com/DenisChesnokov/go-shortener.git/internal/service"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

// Pinger — интерфейс для проверки соединения с хранилищем.
type Pinger interface {
	Ping(ctx context.Context) error
}

type Handler struct {
	svc *service.Shortener
	log *zap.SugaredLogger
	pg  Pinger
}

func New(svc *service.Shortener, log *zap.SugaredLogger, pg Pinger) *Handler {
	return &Handler{
		svc: svc,
		log: log,
		pg:  pg,
	}
}

// GetPing обрабатывает GET /ping: проверяет соединение с БД.
// Возвращает 200 OK при успехе, 500 Internal Server Error при неуспехе.
func (h *Handler) GetPing(w http.ResponseWriter, r *http.Request) {
	if h.pg == nil {
		h.log.Errorf("database not configured")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if err := h.pg.Ping(r.Context()); err != nil {
		h.log.Errorf("ping failed: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// PostShortenJSON обрабатывает POST /api/shorten: принимает JSON {"url":"..."},
// возвращает JSON {"result":"<short_url>"} с кодом 201 Created.
func (h *Handler) PostShortenJSON(w http.ResponseWriter, r *http.Request) {
	var req model.ShortenRequest
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&req); err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	// проверка пустого URL
	if req.URL == "" {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	userID, _ := middleware.GetUserID(r.Context())
	shortURL, err := h.svc.Shorten(r.Context(), req.URL, userID)
	if err != nil {
		var alreadyExistsErr *service.AlreadyExistsError
		if errors.As(err, &alreadyExistsErr) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict) // 409

			resp := model.ShortenResponse{Result: alreadyExistsErr.ShortURL}
			enc := json.NewEncoder(w)
			enc.Encode(resp)
			return
		}
		h.log.Errorf("shorten failed: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	resp := model.ShortenResponse{Result: shortURL}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	enc := json.NewEncoder(w)
	if err := enc.Encode(resp); err != nil {
		h.log.Errorf("encode response failed: %v", err)
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
	userID, _ := middleware.GetUserID(r.Context())

	shortURL, err := h.svc.Shorten(r.Context(), longURL, userID)
	if err != nil {
		var alreadyExistsErr *service.AlreadyExistsError
		if errors.As(err, &alreadyExistsErr) {
			w.WriteHeader(http.StatusConflict)
			w.Write([]byte(alreadyExistsErr.ShortURL))
			return
		}
		h.log.Errorf("shorten failed: %v", err)
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
		if errors.Is(err, service.ErrDeleted) {
			w.WriteHeader(http.StatusGone) // 410
			return
		}

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

// PostShortenBatch обрабатывает POST /api/shorten/batch: принимает JSON-массив
// [{"correlation_id":"...","original_url":"..."}, ...] и возвращает
// [{"correlation_id":"...","short_url":"..."}, ...] с кодом 201 Created.
func (h *Handler) PostShortenBatch(w http.ResponseWriter, r *http.Request) {
	var req []model.BatchRequestItem
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&req); err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	// Пустой батч — ошибка
	if len(req) == 0 {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	userID, _ := middleware.GetUserID(r.Context())
	resp, err := h.svc.ShortenBatch(r.Context(), req, userID)
	if err != nil {
		h.log.Errorf("batch shorten failed: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	enc := json.NewEncoder(w)
	if err := enc.Encode(resp); err != nil {
		h.log.Errorf("encode batch response failed: %v", err)
	}
}

func (h *Handler) GetUserURLs(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok || userID == "" {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	urls, err := h.svc.GetUserURLs(r.Context(), userID)
	if err != nil {
		h.log.Errorf("get user urls failed: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if len(urls) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.Encode(urls)
}

func (h *Handler) DeleteUserURLs(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok || userID == "" {
		http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		return
	}

	var req model.DeleteRequest
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&req); err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	if len(req) == 0 {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	h.svc.DeleteURLs(req, userID)
	w.WriteHeader(http.StatusAccepted) // 202
}
