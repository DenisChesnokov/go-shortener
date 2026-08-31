package service

import (
	"context"
	"errors"
	"math/rand"
	"net/url"

	"github.com/DenisChesnokov/go-shortener.git/internal/model"
	"github.com/DenisChesnokov/go-shortener.git/internal/repository"
)

// максимальное число попыток сгенерировать уникальный ключ
const maxAttempts = 10

// charset — алфавит, из которого состоит короткий код
const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

var ErrKeyGeneration = errors.New("failed to generate unique short key")

// AlreadyExistsError возвращается, когда URL уже был сокращён ранее.
// Содержит готовый короткий URL для ответа клиенту (409 Conflict).
type AlreadyExistsError struct {
	ShortURL string
}

func (e *AlreadyExistsError) Error() string {
	return "url already exists"
}

type Repository interface {
	Save(ctx context.Context, key, longURL, userID string) (string, error)
	Get(ctx context.Context, key string) (string, error)
	SaveBatch(ctx context.Context, items map[string]string, userID string) error
	GetByUserID(ctx context.Context, userID string) ([]model.UserURL, error)
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
func (s *Shortener) Shorten(ctx context.Context, longURL string, userID string) (string, error) {
	for range maxAttempts {
		key := generateShortKey()

		// Save возвращает (existingKey, error)
		existingKey, err := s.repo.Save(ctx, key, longURL, userID)
		if err == nil {
			shortURL, err := url.JoinPath(s.baseURL, key)
			if err != nil {
				return "", err
			}
			return shortURL, nil
		}

		// Если ошибка не связана с дубликатом — пробрасываем наверх
		if !errors.Is(err, repository.ErrAlreadyExists) {
			return "", err
		}

		// Если existingKey не пустой — это дубликат оригинального URL
		if existingKey != "" {
			shortURL, err := url.JoinPath(s.baseURL, existingKey)
			if err != nil {
				return "", err
			}
			// Возвращаем доменную ошибку с готовой короткой ссылкой
			return "", &AlreadyExistsError{ShortURL: shortURL}
		}
		// Если existingKey пустой — это коллизия сгенерированного ключа, пробуем снова
	}

	return "", ErrKeyGeneration
}

// Resolve возвращает оригинальный URL по короткому ключу.
func (s *Shortener) Resolve(ctx context.Context, key string) (string, error) {
	return s.repo.Get(ctx, key)
}

// ShortenBatch сокращает множество URL за один вызов.
// Принимает slice BatchRequestItem, возвращает slice BatchResponseItem.
// Ключи генерирует сервис, репозиторий сохраняет их атомарно.
func (s *Shortener) ShortenBatch(ctx context.Context, items []model.BatchRequestItem, userID string) ([]model.BatchResponseItem, error) {
	// map для передачи в репозиторий: key → original_url
	batch := make(map[string]string, len(items))
	// Ответ собираем в том же порядке, что и запрос
	result := make([]model.BatchResponseItem, 0, len(items))

	for _, item := range items {
		key := generateShortKey()
		batch[key] = item.OriginalURL

		shortURL, err := url.JoinPath(s.baseURL, key)
		if err != nil {
			return nil, err
		}

		result = append(result, model.BatchResponseItem{
			CorrelationID: item.CorrelationID,
			ShortURL:      shortURL,
		})
	}

	// Атомарное сохранение всего батча
	if err := s.repo.SaveBatch(ctx, batch, userID); err != nil {
		return nil, err
	}

	return result, nil
}

// generateShortKey возвращает случайную строку из символов charset.
func generateShortKey() string {
	b := make([]byte, 8)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

// GetUserURLs возвращает все URL, сокращённые пользователем.
func (s *Shortener) GetUserURLs(ctx context.Context, userID string) ([]model.UserURL, error) {
	records, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	result := make([]model.UserURL, 0, len(records))
	for _, r := range records {
		shortURL, err := url.JoinPath(s.baseURL, r.ShortURL)
		if err != nil {
			return nil, err
		}
		result = append(result, model.UserURL{
			ShortURL:    shortURL,
			OriginalURL: r.OriginalURL,
		})
	}
	return result, nil
}
