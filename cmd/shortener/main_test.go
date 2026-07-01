package main

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DenisChesnokov/go-shortener.git/internal/handler"
	"github.com/DenisChesnokov/go-shortener.git/internal/repository"
	"github.com/DenisChesnokov/go-shortener.git/internal/service"

	"github.com/go-chi/chi/v5"
)

// newTestRouter создаёт изолированный набор зависимостей (хранилище, сервис,
// хендлер, роутер) для конкретного подтеста, чтобы состояние не утекало
// между тестами.
func newTestRouter() (*chi.Mux, *repository.InMemory) {
	repo := repository.NewInMemory()
	svc := service.New(repo, "http://localhost:8080")
	h := handler.New(svc)
	return handler.NewRouter(h), repo
}

func TestWebhook(t *testing.T) {
	tests := []struct {
		name     string
		method   string
		target   string
		body     string
		seedKey  string // seedKey позволяет заранее положить запись в хранилище для GET-тестов
		seedURL  string
		wantCode int
		wantType string
		wantLoc  string
		wantBody string
	}{
		{
			name:     "Успешный POST запрос",
			method:   http.MethodPost,
			target:   "/",
			body:     "https://yandex.ru",
			wantCode: http.StatusCreated,
			wantType: "text/plain",
		},
		{
			name:     "POST с пустым телом",
			method:   http.MethodPost,
			target:   "/",
			body:     "",
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "Успешный GET запрос",
			method:   http.MethodGet,
			target:   "/testKey",
			seedKey:  "testKey",
			seedURL:  "https://example.com",
			wantCode: http.StatusTemporaryRedirect,
			wantLoc:  "https://example.com",
		},
		{
			name:     "GET с несуществующим ключом",
			method:   http.MethodGet,
			target:   "/nonExistentKey",
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "GET без ключа",
			method:   http.MethodGet,
			target:   "/",
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "Неподдерживаемый метод PUT",
			method:   http.MethodPut,
			target:   "/",
			wantCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, repo := newTestRouter()

			// изолированное состояние хранилища для текущего кейса
			if tt.seedKey != "" {
				if err := repo.Save(context.Background(), tt.seedKey, tt.seedURL); err != nil {
					t.Fatalf("не удалось подготовить хранилище: %v", err)
					return
				}
			}

			var bodyReader io.Reader
			if tt.body != "" {
				bodyReader = bytes.NewBufferString(tt.body)
			}
			req := httptest.NewRequest(tt.method, tt.target, bodyReader)

			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			res := w.Result()
			defer res.Body.Close()

			if res.StatusCode != tt.wantCode {
				t.Errorf("Код ответа не совпадает: получили %d, хотим %d", res.StatusCode, tt.wantCode)
				return
			}

			if tt.wantType != "" {
				if got := res.Header.Get("Content-Type"); got != tt.wantType {
					t.Errorf("Заголовок Content-Type не совпадает: получили %q, хотим %q", got, tt.wantType)
					return
				}
			}

			if tt.wantLoc != "" {
				if got := res.Header.Get("Location"); got != tt.wantLoc {
					t.Errorf("Заголовок Location не совпадает: получили %q, хотим %q", got, tt.wantLoc)
					return
				}
			}

			if tt.wantBody != "" {
				got, err := io.ReadAll(res.Body)
				if err != nil {
					t.Errorf("не удалось прочитать тело ответа: %v", err)
					return
				}
				if string(got) != tt.wantBody {
					t.Errorf("Тело ответа не совпадает: получили %q, хотим %q", string(got), tt.wantBody)
					return
				}
			}
		})
	}
}
