package service

import (
	"context"
	"errors"
	"math/rand"
	"net/url"

	"github.com/DenisChesnokov/go-shortener.git/internal/repository"
)

// максимальное число попыток сгенерировать уникальный ключ
const maxAttempts = 10

// charset — алфавит, из которого состоит короткий код
const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

var ErrKeyGeneration = errors.New("failed to generate unique short key")

type Repository interface {
	Save(ctx context.Context, key, longURL string) error
	Get(ctx context.Context, key string) (string, error)
}

// Shortener содержит бизнес-логику сокращения и разрешения ссылок
type Shortener struct {
	repo    Repository
	baseURL string
}

// New создаёт Shortener с заданным репозиторием и базовым адресом коротких ссылок
func New(repo Repository, baseURL string) *Shortener {
	return &Shortener{
		repo:    repo,
		baseURL: baseURL,
	}
}

// Shorten сохраняет длинный URL и возвращает короткий.
// Генерирует кандидаты короткого ключа и пытается сохранить их в репозитории,
// если ключ уже занят, повторяет попытку не более maxAttempts раз
func (s *Shortener) Shorten(ctx context.Context, longURL string) (string, error) {
	for range maxAttempts {
		key := generateShortKey()

		err := s.repo.Save(ctx, key, longURL)
		if err == nil {
			shortURL, err := url.JoinPath(s.baseURL, key)
			if err != nil {
				return "", err
			}
			return shortURL, nil
		}
		if !errors.Is(err, repository.ErrAlreadyExists) {
			// Неожиданная ошибка репозитория — прокидываем наверх.
			return "", err
		}
		// Ключ уже занят — пробуем сгенерировать новый.
	}

	return "", ErrKeyGeneration
}

// Resolve возвращает оригинальный URL по короткому ключу.
func (s *Shortener) Resolve(ctx context.Context, key string) (string, error) {
	return s.repo.Get(ctx, key)
}

// generateShortKey возвращает случайную строку из символов charset.
func generateShortKey() string {
	b := make([]byte, 8)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}
