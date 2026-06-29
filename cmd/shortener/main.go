package main

import (
	"io"
	"math/rand"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Алфавит, из которого будет состоять короткий код
const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// храним ссылки в памяти
var linkBase = make(map[string]string)

// генерим случайную строку из символов charset
func generateShortKey() string {
	b := make([]byte, 8)
	for i := range b {
		// Выбираем случайный символ из алфавита
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

func makeShortLink(w http.ResponseWriter, r *http.Request) {
	// в любом случае закроем
	defer r.Body.Close()

	bodyBytes, err := io.ReadAll(r.Body)

	if err != nil || len(bodyBytes) == 0 {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	longURL := string(bodyBytes)

	var key string
	// Цикл будет работать, пока не сгенерируется уникальный ключ
	for {
		key = generateShortKey()
		if _, exists := linkBase[key]; !exists {
			break // Ключ уникален, выходим из цикла
		}
	}

	linkBase[key] = longURL

	// установим правильный заголовок для типа данных
	w.Header().Set("Content-Type", "text/plain")

	// установим правильный код ответа 201
	w.WriteHeader(http.StatusCreated)

	// возвращаем  итоговую ссылку
	shortURL := Cfg.BaseURL + "/" + key
	w.Write([]byte(shortURL))
}

func getFullLink(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "shortLink")

	// если нет короткой ссылки
	if key == "" {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// Возвращаем полную ссылку
	longURL, exists := linkBase[key]
	if !exists {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// установим правильные заголовоки
	w.Header().Set("Content-Type", "text/plain")
	w.Header().Add("Location", longURL)

	// установим правильный код ответа 307
	w.WriteHeader(http.StatusTemporaryRedirect)
}

// функция run будет полезна при инициализации зависимостей сервера перед запуском
func run() *chi.Mux {

	r := chi.NewRouter()

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Bad Request", http.StatusBadRequest)
	})
	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Bad Request", http.StatusBadRequest)
	})

	r.Get("/{shortLink}", getFullLink)
	r.Post("/", makeShortLink)

	return r
}

func main() {

	Init()

	r := run()
	if err := http.ListenAndServe(Cfg.ServerAddress, r); err != nil {
		panic(err)
	}
}
