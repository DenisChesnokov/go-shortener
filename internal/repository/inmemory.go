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

// Save сохраняет соответствие key -> longURL
// Если ключ уже занят, возвращает ErrAlreadyExists, не перезаписывая значение
func (r *InMemory) Save(ctx context.Context, key, longURL string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.data[key]; exists {
		return ErrAlreadyExists
	}

	r.data[key] = longURL
	return nil
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
