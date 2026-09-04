package repository

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/DenisChesnokov/go-shortener.git/internal/model"
)

var ErrAlreadyExists = errors.New("short key already exists")

var ErrNotFound = errors.New("short key not found")

/*
InMemory — хранилище сокращённых ссылок в памяти процесса.
Безопасно для конкурентного использования: все операции
защищены RWMutex. Запись проверяет уникальность ключа и выполняет
вставку под одной блокировкой писателя, что исключает состояние гонки
между проверкой и записью
*/

// UserRecord хранит URL вместе с.userID.
type UserRecord struct {
	LongURL   string
	UserID    string
	IsDeleted bool
}

type InMemory struct {
	mu       sync.RWMutex
	dataUser map[string]UserRecord // key - {LongURL, UserID}
}

// NewInMemory создаёт пустое хранилище в памяти
func NewInMemory() *InMemory {
	return &InMemory{
		dataUser: make(map[string]UserRecord),
	}
}

// no-op для запроса Ping
func (r *InMemory) Ping(ctx context.Context) error { return nil }

// Save сохраняет соответствие key -> longURL
// Если ключ уже занят, возвращает ErrAlreadyExists, не перезаписывая значение
func (r *InMemory) Save(ctx context.Context, key, longURL string, userID string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.dataUser[key]; exists {
		return "", fmt.Errorf("%w: key %q", ErrAlreadyExists, key)
	}

	r.dataUser[key] = UserRecord{LongURL: longURL, UserID: userID}
	return key, nil
}

// Get возвращает оригинальный URL по короткому ключу
// Если ключ не найден, возвращает ErrNotFound
func (r *InMemory) Get(ctx context.Context, key string) (string, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	record, exists := r.dataUser[key]
	if !exists {
		return "", false, fmt.Errorf("%w: %q", ErrNotFound, key)
	}

	return record.LongURL, record.IsDeleted, nil
}

// SaveBatch сохраняет множество записей атомарно (под одной блокировкой).
func (r *InMemory) SaveBatch(ctx context.Context, items map[string]string, userID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Сначала проверяем все ключи на коллизии
	for key := range items {
		if _, exists := r.dataUser[key]; exists {
			return ErrAlreadyExists
		}
	}

	// Затем сохраняем все
	for key, longURL := range items {
		r.dataUser[key] = UserRecord{LongURL: longURL, UserID: userID}
	}
	return nil
}

func (r *InMemory) GetByUserID(ctx context.Context, userID string) ([]model.UserURL, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []model.UserURL
	for key, record := range r.dataUser {
		if record.UserID == userID && !record.IsDeleted {
			result = append(result, model.UserURL{
				ShortURL:    key,
				OriginalURL: record.LongURL,
			})
		}
	}
	return result, nil
}

func (r *InMemory) MarkDeleted(ctx context.Context, items []struct{ ShortURL, UserID string }) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, item := range items {
		record, exists := r.dataUser[item.ShortURL]
		if !exists {
			continue // пропускаем несуществующие
		}
		if record.UserID != item.UserID {
			continue // чужой URL — пропускаем
		}
		record.IsDeleted = true
		r.dataUser[item.ShortURL] = record
	}
	return nil
}
