package logger

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"
)

// TestRequestLogger_PassThrough проверяет, что middleware не искажает
// ответ хендлера: статус и тело доходят до клиента без изменений.
func TestRequestLogger_PassThrough(t *testing.T) {
	body := "http://localhost:8080/abc"
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(body))
	})

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("https://yandex.ru"))
	rec := httptest.NewRecorder()

	RequestLogger(handler).ServeHTTP(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusCreated {
		t.Errorf("status: получили %d, хотим %d", res.StatusCode, http.StatusCreated)
		return
	}

	got, err := io.ReadAll(res.Body)
	if err != nil {
		t.Errorf("не удалось прочитать тело: %v", err)
		return
	}
	if string(got) != body {
		t.Errorf("тело: получили %q, хотим %q", string(got), body)
		return
	}
}

// TestLoggingResponseWriter_CapturesStatusAndSize проверяет, что
// loggingResponseWriter перехватывает код статуса и размер ответа,
// и при этом данные проходят в исходный writer.
func TestLoggingResponseWriter_CapturesStatusAndSize(t *testing.T) {
	rec := httptest.NewRecorder()
	rd := &responseData{}

	lw := loggingResponseWriter{
		ResponseWriter: rec,
		responseData:   rd,
	}

	lw.WriteHeader(http.StatusTeapot) // 418
	n, err := lw.Write([]byte("hello world"))
	if err != nil {
		t.Fatalf("не удалось записать тело: %v", err)
		return
	}

	if rd.status != http.StatusTeapot {
		t.Errorf("captured status: получили %d, хотим %d", rd.status, http.StatusTeapot)
		return
	}
	if rd.size != 11 {
		t.Errorf("captured size: получили %d, хотим 11", rd.size)
		return
	}
	if n != 11 {
		t.Errorf("written size: получили %d, хотим 11", n)
		return
	}
	if rec.Code != http.StatusTeapot {
		t.Errorf("passed status: получили %d, хотим %d", rec.Code, http.StatusTeapot)
		return
	}
	if rec.Body.String() != "hello world" {
		t.Errorf("passed body: получили %q, хотим %q", rec.Body.String(), "hello world")
		return
	}
}

// TestInitialize_ValidLevel проверяет, что Initialize с корректным уровнем
// возвращает nil и не падает.
func TestInitialize_ValidLevel(t *testing.T) {
	t.Cleanup(func() { sugar = zap.NewNop().Sugar() })

	if err := Initialize("info"); err != nil {
		t.Errorf("ожидали nil, получили %v", err)
	}
}

// TestInitialize_InvalidLevel проверяет, что Initialize с некорректным
// уровнем возвращает ошибку.
func TestInitialize_InvalidLevel(t *testing.T) {
	t.Cleanup(func() { sugar = zap.NewNop().Sugar() })

	if err := Initialize("invalid"); err == nil {
		t.Error("ожидали ошибку, получили nil")
	}
}