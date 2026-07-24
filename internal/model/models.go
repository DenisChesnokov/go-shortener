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
