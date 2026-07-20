package logger

import (
	"net/http"
	"time"

	"go.uber.org/zap"
)

// По аналогии с теорией для скилов Алисы

// sugar — синглтон-логер. По умолчанию no-op, чтобы код не падал,
// если Initialize не вызвана (например, в тестах).
var sugar *zap.SugaredLogger = zap.NewNop().Sugar()

// Initialize инициализирует синглтон логера с заданным уровнем.
func Initialize(level string) error {
	lvl, err := zap.ParseAtomicLevel(level)
	if err != nil {
		return err
	}

	cfg := zap.NewProductionConfig()
	cfg.Level = lvl

	zl, err := cfg.Build()
	if err != nil {
		return err
	}

	sugar = zl.Sugar()
	return nil
}

// Sync очищает буфера логера. Должна вызываться в main через defer.
func Sync() error {
	return sugar.Sync()
}

type (
	// responseData хранит сведения об ответе
	responseData struct {
		status int
		size   int
	}

	// loggingResponseWriter перехватывает запись ответа
	loggingResponseWriter struct {
		http.ResponseWriter // встраиваем оригинальный http.ResponseWriter
		responseData        *responseData
	}
)

func (r *loggingResponseWriter) Write(b []byte) (int, error) {
	size, err := r.ResponseWriter.Write(b)
	r.responseData.size += size
	return size, err
}

func (r *loggingResponseWriter) WriteHeader(statusCode int) {
	r.ResponseWriter.WriteHeader(statusCode)
	r.responseData.status = statusCode
}

// RequestLogger — middleware-логер для входящих HTTP-запросов.
// Принимает http.HandlerFunc и возвращает http.HandlerFunc,
// что позволяет использовать её с chi-методами r.Post/r.Get/r.NotFound/r.MethodNotAllowed.
func RequestLogger(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		responseData := &responseData{
			status: 0,
			size:   0,
		}
		lw := loggingResponseWriter{
			ResponseWriter: w,
			responseData:   responseData,
		}
		h(&lw, r)

		duration := time.Since(start)

		sugar.Infoln(
			"uri", r.RequestURI,
			"method", r.Method,
			"status", responseData.status,
			"duration", duration,
			"size", responseData.size,
		)
	}
}
