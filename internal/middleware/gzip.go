package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
)

// compressWriter реализует http.ResponseWriter и прозрачно для хендлера
// сжимает передаваемые данные, если Content-Type попадает в список
// поддерживаемых (application/json или text/html).
type compressWriter struct {
	w  http.ResponseWriter
	zw *gzip.Writer
}

func newCompressWriter(w http.ResponseWriter) *compressWriter {
	return &compressWriter{
		w:  w,
		zw: gzip.NewWriter(w),
	}
}

func (c *compressWriter) Header() http.Header {
	return c.w.Header()
}

// WriteHeader выставляет Content-Encoding: gzip только для успешных ответов
// (статус < 300) с поддерживаемым Content-Type.
func (c *compressWriter) WriteHeader(statusCode int) {
	if statusCode < 300 && isCompressible(c.w.Header().Get("Content-Type")) {
		c.w.Header().Set("Content-Encoding", "gzip")
	}
	c.w.WriteHeader(statusCode)
}

// Write направляет данные через gzip.Writer только если было выставлено
// Content-Encoding: gzip. Иначе пишет напрямую в оригинальный writer —
func (c *compressWriter) Write(p []byte) (int, error) {
	if c.w.Header().Get("Content-Encoding") == "gzip" {
		return c.zw.Write(p)
	}
	return c.w.Write(p)
}

// Close флашит gzip.Writer только если он реально использовался.
// Если ответ не сжимался (text/plain или 4xx/5xx), Close ничего не делает —
// это защищает от записи пустого gzip-стрима в незапланированный ответ.
func (c *compressWriter) Close() error {
	if c.w.Header().Get("Content-Encoding") == "gzip" {
		return c.zw.Close()
	}
	return nil
}

// compressReader реализует io.ReadCloser и прозрачно для хендлера
// декомпрессирует gzip-тело запроса.
type compressReader struct {
	r  io.ReadCloser
	zr *gzip.Reader
}

func newCompressReader(r io.ReadCloser) (*compressReader, error) {
	zr, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}
	return &compressReader{
		r:  r,
		zr: zr,
	}, nil
}

func (c *compressReader) Read(p []byte) (int, error) {
	return c.zr.Read(p)
}

func (c *compressReader) Close() error {
	if err := c.r.Close(); err != nil {
		return err
	}
	return c.zr.Close()
}

// isCompressible возвращает true, если content-type поддерживает сжатие.
// Используем strings.Contains, чтобы корректно обрабатывать значения
// вида "application/json; charset=utf-8".
func isCompressible(contentType string) bool {
	return strings.Contains(contentType, "application/json") ||
		strings.Contains(contentType, "text/html")
}

// GzipMiddleware — HTTP middleware для поддержки gzip.
// Оборачивает хендлер двунаправленной обработкой:
//   - если клиент прислал Content-Encoding: gzip — декомпрессирует тело запроса;
//   - если клиент ждёт Accept-Encoding: gzip — сжимает тело ответа
//     (только для поддерживаемых Content-Type).
func GzipMiddleware(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// По умолчанию передаём оригинальный ResponseWriter.
		ow := w

		// Сжатие ответа — если клиент поддерживает gzip.
		if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			cw := newCompressWriter(w)
			ow = cw
			defer cw.Close()
		}

		// Декомпрессия запроса — если клиент прислал сжатое тело.
		if strings.Contains(r.Header.Get("Content-Encoding"), "gzip") {
			cr, err := newCompressReader(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			r.Body = cr
			defer cr.Close()
		}

		h.ServeHTTP(ow, r)
	}
}
