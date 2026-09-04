package model

// По аналогии с Алисой

// ShortenRequest — тело запроса POST /api/shorten.
type ShortenRequest struct {
	URL string `json:"url"`
}

// ShortenResponse — тело ответа POST /api/shorten.
type ShortenResponse struct {
	Result string `json:"result"`
}

// BatchRequestItem — элемент запроса POST /api/shorten/batch.
type BatchRequestItem struct {
	CorrelationID string `json:"correlation_id"`
	OriginalURL   string `json:"original_url"`
}

// BatchResponseItem — элемент ответа POST /api/shorten/batch.
type BatchResponseItem struct {
	CorrelationID string `json:"correlation_id"`
	ShortURL      string `json:"short_url"`
}

// UserURL — элемент ответа GET /api/user/urls.
type UserURL struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

// URLRecord — запись URL с информацией об удалении.
type URLRecord struct {
	ShortURL    string
	OriginalURL string
	IsDeleted   bool
}

// DeleteRequest — тело запроса DELETE /api/user/urls.
type DeleteRequest []string // ["key1", "key2", ...]
