package main

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWebhook(t *testing.T) {
	// Инициализируем карту начальными данными для теста GET
	linkBase["testKey"] = "https://example.com"

	// Структура для описания тестовых случаев
	type want struct {
		statusCode  int
		contentType string
		location    string
		bodyContent string
	}

	tests := []struct {
		name   string
		method string
		target string
		body   string
		want   want
	}{
		{
			name:   "Успешный POST запрос",
			method: http.MethodPost,
			target: "/",
			body:   "https://yandex.ru",
			want: want{
				statusCode:  http.StatusCreated,
				contentType: "text/plain",
			},
		},
		{
			name:   "POST с пустым телом",
			method: http.MethodPost,
			target: "/",
			body:   "",
			want: want{
				statusCode: http.StatusBadRequest,
			},
		},
		{
			name:   "Успешный GET запрос (редирект)",
			method: http.MethodGet,
			target: "/testKey",
			want: want{
				statusCode: http.StatusTemporaryRedirect,
				location:   "https://example.com",
			},
		},
		{
			name:   "GET с несуществующим ключом",
			method: http.MethodGet,
			target: "/nonExistentKey",
			want: want{
				statusCode: http.StatusBadRequest,
			},
		},
		{
			name:   "GET без ключа (на корень /)",
			method: http.MethodGet,
			target: "/",
			want: want{
				statusCode: http.StatusBadRequest,
			},
		},
		{
			name:   "Неподдерживаемый метод PUT",
			method: http.MethodPut,
			target: "/",
			want: want{
				statusCode: http.StatusBadRequest,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// объект запроса
			var bodyReader io.Reader
			if tt.body != "" {
				bodyReader = bytes.NewBufferString(tt.body)
			}
			req := httptest.NewRequest(tt.method, tt.target, bodyReader)

			// ResponseRecorder для записи ответа сервера
			w := httptest.NewRecorder()

			webhook(w, req)
			res := w.Result()
			defer res.Body.Close()

			// Проверка статус-кода
			if res.StatusCode != tt.want.statusCode {
				t.Errorf("Код ответа не совпадает: получили %d, хотим %d", res.StatusCode, tt.want.statusCode)
			}

			// Проверка заголовка Content-Type
			if tt.want.contentType != "" {
				if res.Header.Get("Content-Type") != tt.want.contentType {
					t.Errorf("Заголовок Content-Type не совпадает: получили %s, хотим %s", res.Header.Get("Content-Type"), tt.want.contentType)
				}
			}

			// Проверка заголовка Location для редиректа
			if tt.want.location != "" {
				if res.Header.Get("Location") != tt.want.location {
					t.Errorf("Заголовок Location не совпадает: получили %s, хотим %s", res.Header.Get("Location"), tt.want.location)
				}
			}
		})
	}
}
