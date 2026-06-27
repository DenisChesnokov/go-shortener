// пакеты исполняемых приложений должны называться main
package main

import (
	"io"
	"math/rand"
	"net/http"
)

// Алфавит, из которого будет состоять короткий код (буквы и цифры)
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

func main() {

	// Тестовые данные для отладки
	linkBase["EwHXdJfB"] = "https://practicum.yandex.ru/"

	if err := run(); err != nil {
		panic(err)
	}
}

// функция run будет полезна при инициализации зависимостей сервера перед запуском
func run() error {
	return http.ListenAndServe(`:8080`, http.HandlerFunc(webhook))
}

// функция webhook — обработчик HTTP-запроса
func webhook(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {

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
		shortURL := "http://localhost:8080/" + key
		w.Write([]byte(shortURL))

		return
	}

	if r.Method == http.MethodGet {

		key := r.URL.Path[1:]

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

		return

	}
	// Все остальные случаи 400
	http.Error(w, "Bad Request", http.StatusBadRequest)
}
