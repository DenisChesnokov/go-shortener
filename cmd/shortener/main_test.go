package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DenisChesnokov/go-shortener.git/internal/auth"
	"github.com/DenisChesnokov/go-shortener.git/internal/handler"
	"github.com/DenisChesnokov/go-shortener.git/internal/repository"
	"github.com/DenisChesnokov/go-shortener.git/internal/service"

	"github.com/go-chi/chi/v5"

	"encoding/json"
	"strings"

	"github.com/DenisChesnokov/go-shortener.git/internal/model"
	"go.uber.org/zap"
)

// newTestRouter создаёт изолированный набор зависимостей (хранилище, сервис,
// хендлер, роутер) для конкретного подтеста, чтобы состояние не утекало
// между тестами.
func newTestRouter() (*chi.Mux, *repository.InMemory, error) {
	repo := repository.NewInMemory()
	svc := service.New(repo, "http://localhost:8080")
	log := zap.NewNop().Sugar()
	h := handler.New(svc, log, nil)
	jwtMgr := auth.NewJWTManager("test-secret", time.Hour)
	return handler.NewRouter(h, log, jwtMgr), repo, nil
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
			r, repo, err := newTestRouter()
			if err != nil {
				t.Fatalf("newTestRouter: %v", err)
			}

			// изолированное состояние хранилища для текущего кейса
			if tt.seedKey != "" {
				if _, err := repo.Save(context.Background(), tt.seedKey, tt.seedURL, ""); err != nil {
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

func TestAPIShorten(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		wantCode int
		wantType string
	}{
		{
			name:     "Успешный POST /api/shorten",
			body:     `{"url":"https://yandex.ru"}`,
			wantCode: http.StatusCreated,
			wantType: "application/json",
		},
		{
			name:     "Невалидный JSON",
			body:     `not a json`,
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "Пустое тело",
			body:     ``,
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "Пустой URL в JSON",
			body:     `{"url":""}`,
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "Отсутствует поле url",
			body:     `{"other":"value"}`,
			wantCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, _, err := newTestRouter()
			if err != nil {
				t.Fatalf("newTestRouter: %v", err)
			}

			req := httptest.NewRequest(http.MethodPost, "/api/shorten", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)

			res := rec.Result()
			defer res.Body.Close()

			if res.StatusCode != tt.wantCode {
				t.Errorf("status: получили %d, хотим %d", res.StatusCode, tt.wantCode)
				return
			}

			if tt.wantType != "" {
				if got := res.Header.Get("Content-Type"); got != tt.wantType {
					t.Errorf("Content-Type: получили %q, хотим %q", got, tt.wantType)
					return
				}
			}

			// Дополнительно: для успешного кейса проверяем непустой result
			if tt.wantCode == http.StatusCreated {
				var resp model.ShortenResponse
				if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
					t.Errorf("не удалось десериализовать ответ: %v", err)
					return
				}
				if resp.Result == "" {
					t.Error("ожидали непустой result, получили пустой")
					return
				}
			}
		})
	}
}

// TestGzipCompression проверяет двунаправленную gzip-обработку.
func TestGzipCompression(t *testing.T) {
	tests := []struct {
		name            string
		path            string
		acceptEncoding  string
		contentEncoding string
		body            string
		gzipBody        bool
		wantCode        int
		wantEncoding    string
		wantType        string
	}{
		{
			name:           "accepts_gzip_response_json",
			path:           "/api/shorten",
			acceptEncoding: "gzip",
			body:           `{"url":"https://yandex.ru"}`,
			wantCode:       http.StatusCreated,
			wantEncoding:   "gzip",
			wantType:       "application/json",
		},
		{
			name:         "sends_gzip_request",
			path:         "/api/shorten",
			body:         `{"url":"https://yandex.ru"}`,
			gzipBody:     true,
			wantCode:     http.StatusCreated,
			wantEncoding: "", // клиент не прислал Accept-Encoding, ответ несжатый
			wantType:     "application/json",
		},
		{
			name:           "text_plain_not_compressed",
			path:           "/",
			acceptEncoding: "gzip",
			body:           "https://yandex.ru",
			wantCode:       http.StatusCreated,
			wantEncoding:   "", // text/plain НЕ сжимаем по условию задачи
			wantType:       "text/plain",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, _, err := newTestRouter()
			if err != nil {
				t.Fatalf("newTestRouter: %v", err)
			}

			var bodyReader io.Reader
			if tt.gzipBody {
				// Готовим сжатое тело запроса
				buf := bytes.NewBuffer(nil)
				zb := gzip.NewWriter(buf)
				zb.Write([]byte(tt.body))
				zb.Close()
				bodyReader = buf
			} else if tt.body != "" {
				bodyReader = strings.NewReader(tt.body)
			}

			req := httptest.NewRequest(http.MethodPost, tt.path, bodyReader)
			if tt.acceptEncoding != "" {
				req.Header.Set("Accept-Encoding", tt.acceptEncoding)
			}
			if tt.gzipBody {
				req.Header.Set("Content-Encoding", "gzip")
			}

			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)

			res := rec.Result()
			defer res.Body.Close()

			if res.StatusCode != tt.wantCode {
				t.Errorf("status: получили %d, хотим %d", res.StatusCode, tt.wantCode)
				return
			}

			if got := res.Header.Get("Content-Encoding"); got != tt.wantEncoding {
				t.Errorf("Content-Encoding: получили %q, хотим %q", got, tt.wantEncoding)
				return
			}

			if got := res.Header.Get("Content-Type"); got != tt.wantType {
				t.Errorf("Content-Type: получили %q, хотим %q", got, tt.wantType)
				return
			}

			// Проверяем тело: если response сжат, надо сначала разжать
			var body []byte
			if res.Header.Get("Content-Encoding") == "gzip" {
				zr, err := gzip.NewReader(res.Body)
				if err != nil {
					t.Errorf("не удалось создать gzip.Reader: %v", err)
					return
				}
				body, err = io.ReadAll(zr)
				if err != nil {
					t.Errorf("не удалось прочитать сжатый ответ: %v", err)
					return
				}
			} else {
				body, _ = io.ReadAll(res.Body)
			}

			// Проверяем что тело не пустое
			if len(body) == 0 {
				t.Error("ожидали непустое тело ответа")
				return
			}

			// Для JSON дополнительно — десериализуем
			if tt.wantType == "application/json" {
				var resp model.ShortenResponse
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Errorf("не удалось десериализовать JSON: %v (тело: %q)", err, string(body))
					return
				}
				if resp.Result == "" {
					t.Error("ожидали непустой result")
					return
				}
			}
		})
	}
}

func TestPing_NoDatabase(t *testing.T) {
	r, _, err := newTestRouter()
	if err != nil {
		t.Fatalf("newTestRouter: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusInternalServerError {
		t.Errorf("status: получили %d, хотим %d", res.StatusCode, http.StatusInternalServerError)
	}
}

func TestBatchShorten(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		wantCode int
		wantLen  int // ожидаемая длина массива в ответе (0 = не проверять)
	}{
		{
			name:     "Успешный batch",
			body:     `[{"correlation_id":"1","original_url":"https://yandex.ru"},{"correlation_id":"2","original_url":"https://mail.ru"}]`,
			wantCode: http.StatusCreated,
			wantLen:  2,
		},
		{
			name:     "Пустой batch",
			body:     `[]`,
			wantCode: http.StatusBadRequest,
			wantLen:  0,
		},
		{
			name:     "Невалидный JSON",
			body:     `not json`,
			wantCode: http.StatusBadRequest,
			wantLen:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, _, err := newTestRouter()
			if err != nil {
				t.Fatalf("newTestRouter: %v", err)
			}

			req := httptest.NewRequest(http.MethodPost, "/api/shorten/batch", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)

			res := rec.Result()
			defer res.Body.Close()

			if res.StatusCode != tt.wantCode {
				t.Errorf("status: получили %d, хотим %d", res.StatusCode, tt.wantCode)
				return
			}

			if tt.wantLen > 0 && tt.wantCode == http.StatusCreated {
				var resp []model.BatchResponseItem
				if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
					t.Errorf("не удалось десериализовать ответ: %v", err)
					return
				}
				if len(resp) != tt.wantLen {
					t.Errorf("кол-во записей: получили %d, хотим %d", len(resp), tt.wantLen)
					return
				}
				// Проверяем correlation_id
				if resp[0].CorrelationID != "1" {
					t.Errorf("correlation_id[0]: получили %q, хотим %q", resp[0].CorrelationID, "1")
				}
				if resp[0].ShortURL == "" {
					t.Error("short_url[0] пустой")
				}
			}
		})
	}
}

// TestGetUserURLs_NoCookie проверяет автосоздание куки.
func TestGetUserURLs_NoCookie(t *testing.T) {
	r, _, err := newTestRouter()
	if err != nil {
		t.Fatalf("newTestRouter: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/user/urls", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	// Без URL должно быть 204
	if res.StatusCode != http.StatusNoContent {
		t.Errorf("status: получили %d, хотим %d", res.StatusCode, http.StatusNoContent)
	}

	// Должна быть установлена кука
	cookies := res.Cookies()
	var found bool
	for _, c := range cookies {
		if c.Name == "user_id" {
			found = true
			if c.Value == "" {
				t.Error("user_id cookie value пустой")
			}
		}
	}
	if !found {
		t.Error("cookie user_id не установлена")
	}
}

// TestGetUserURLs_WithData проверяет возврат URL пользователя.
func TestGetUserURLs_WithData(t *testing.T) {
	r, repo, err := newTestRouter()
	if err != nil {
		t.Fatalf("newTestRouter: %v", err)
	}

	// Создаём URL для пользователя
	userID := "test-user-abc"
	if _, err := repo.Save(context.Background(), "k1", "http://example1.com", userID); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if _, err := repo.Save(context.Background(), "k2", "http://example2.com", userID); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if _, err := repo.Save(context.Background(), "k3", "http://other.com", "other-user"); err != nil {
		t.Fatalf("Save other: %v", err)
	}

	// Генерируем валидный JWT для userID
	jwtMgr := auth.NewJWTManager("test-secret", time.Hour)
	token, err := jwtMgr.BuildJWTString(userID)
	if err != nil {
		t.Fatalf("BuildJWTString: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/user/urls", nil)
	req.AddCookie(&http.Cookie{Name: "user_id", Value: token})
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Errorf("status: получили %d, хотим %d", res.StatusCode, http.StatusOK)
	}

	var urls []model.UserURL
	if err := json.NewDecoder(res.Body).Decode(&urls); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(urls) != 2 {
		t.Errorf("кол-во URL: получили %d, хотим 2", len(urls))
	}

	// Проверяем, что short_url — полный URL с базовым адресом
	for _, u := range urls {
		if !strings.HasPrefix(u.ShortURL, "http://localhost:8080/") {
			t.Errorf("ShortURL должен быть полным URL, получили %q", u.ShortURL)
			return
		}
		if u.OriginalURL == "" {
			t.Errorf("OriginalURL пустой для записи %q", u.ShortURL)
			return
		}
	}
}

// TestGetUserURLs_InvalidCookie проверяет 401 при невалидной куке.
func TestGetUserURLs_InvalidCookie(t *testing.T) {
	r, _, err := newTestRouter()
	if err != nil {
		t.Fatalf("newTestRouter: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/user/urls", nil)
	req.AddCookie(&http.Cookie{Name: "user_id", Value: "invalid-token"})
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusUnauthorized {
		t.Errorf("status: получили %d, хотим %d", res.StatusCode, http.StatusUnauthorized)
	}
}

// TestDeleteUserURLs_Accepted проверяет, что DELETE возвращает 202 Accepted.
func TestDeleteUserURLs_Accepted(t *testing.T) {
	r, repo, err := newTestRouter()
	if err != nil {
		t.Fatalf("newTestRouter: %v", err)
	}

	userID := "test-user-delete"
	repo.Save(context.Background(), "delKey1", "http://example1.com", userID)
	repo.Save(context.Background(), "delKey2", "http://example2.com", userID)

	jwtMgr := auth.NewJWTManager("test-secret", time.Hour)
	token, _ := jwtMgr.BuildJWTString(userID)

	req := httptest.NewRequest(http.MethodDelete, "/api/user/urls",
		strings.NewReader(`["delKey1","delKey2"]`))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "user_id", Value: token})
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Errorf("status: got %d, want %d", rec.Code, http.StatusAccepted)
	}
}

// TestDeleteUserURLs_Unauthorized проверяет 401 без авторизации.
func TestDeleteUserURLs_Unauthorized(t *testing.T) {
	r, _, err := newTestRouter()
	if err != nil {
		t.Fatalf("newTestRouter: %v", err)
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/user/urls",
		strings.NewReader(`["delKey1"]`))
	req.AddCookie(&http.Cookie{Name: "user_id", Value: "invalid-token"})
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status: got %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

// TestDeleteUserURLs_EmptyBody проверяет 400 на пустом теле.
func TestDeleteUserURLs_EmptyBody(t *testing.T) {
	r, _, err := newTestRouter()
	if err != nil {
		t.Fatalf("newTestRouter: %v", err)
	}

	jwtMgr := auth.NewJWTManager("test-secret", time.Hour)
	token, _ := jwtMgr.BuildJWTString("any-user")

	req := httptest.NewRequest(http.MethodDelete, "/api/user/urls",
		strings.NewReader(``))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "user_id", Value: token})
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want %d", rec.Code, http.StatusBadRequest)
	}
}
