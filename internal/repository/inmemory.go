package repository

import (
	"context"
	"errors"
	"fmt"
	"sync"
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
type InMemory struct {
	mu   sync.RWMutex
	data map[string]string
}

// NewInMemory создаёт пустое хранилище в памяти
func NewInMemory() *InMemory {
	return &InMemory{
		data: make(map[string]string),
	}
}

// no-op для запроса Ping
func (r *InMemory) Ping(ctx context.Context) error { return nil }

// Save сохраняет соответствие key -> longURL
// Если ключ уже занят, возвращает ErrAlreadyExists, не перезаписывая значение
func (r *InMemory) Save(ctx context.Context, key, longURL string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.data[key]; exists {
		return "", fmt.Errorf("%w: key %q", ErrAlreadyExists, key)
	}

	r.data[key] = longURL
	return key, nil
}

// Get возвращает оригинальный URL по короткому ключу
// Если ключ не найден, возвращает ErrNotFound
func (r *InMemory) Get(ctx context.Context, key string) (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	longURL, exists := r.data[key]
	if !exists {
		return "", fmt.Errorf("%w: %q", ErrNotFound, key)
	}

	return longURL, nil
}

// SaveBatch сохраняет множество записей атомарно (под одной блокировкой).
func (r *InMemory) SaveBatch(ctx context.Context, items map[string]string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Сначала проверяем все ключи на коллизии
	for key := range items {
		if _, exists := r.data[key]; exists {
			return ErrAlreadyExists
		}
	}

	// Затем сохраняем все
	for key, longURL := range items {
		r.data[key] = longURL
	}
	return nil
}
