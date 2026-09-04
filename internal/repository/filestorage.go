package repository

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"strconv"
	"sync"

	"github.com/DenisChesnokov/go-shortener.git/internal/model"
)

// FileRecord — формат одной записи в файле хранилища.
type FileRecord struct {
	UUID        string `json:"uuid"`
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
	UserID      string `json:"user_id"`
	IsDeleted   bool   `json:"is_deleted"`
}

// FileStorage — реализация репозитория через файл на диске.
// В памяти держит map[key]FileRecord для быстрых Get-операций,
type FileStorage struct {
	mu       sync.RWMutex
	data     map[string]FileRecord
	counter  int // счетчик для генерации UUID
	filePath string
	file     *os.File
}

// no-op для запроса Ping
func (fs *FileStorage) Ping(ctx context.Context) error { return nil }

// NewFileStorage создаёт хранилище и загружает существующие данные из файла.
// Если файл не существует — начинается с пустого хранилища.
func NewFileStorage(filePath string) (*FileStorage, error) {
	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return nil, err
	}

	fs := &FileStorage{
		data:     make(map[string]FileRecord),
		filePath: filePath,
		file:     file,
	}
	if err := fs.load(); err != nil {
		return nil, err
	}
	return fs, nil
}

// load читает файл и восстанавливает данные в память.
func (fs *FileStorage) load() error {
	file, err := os.OpenFile(fs.filePath, os.O_RDONLY|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	maxUUID := 0
	for {
		var r FileRecord
		if err := decoder.Decode(&r); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return err
		}
		fs.data[r.ShortURL] = r
		if u, err := strconv.Atoi(r.UUID); err == nil && u > maxUUID {
			maxUUID = u
		}
	}
	fs.counter = maxUUID
	return nil
}

// Save сохраняет длинный URL под коротким ключом.
func (fs *FileStorage) Save(ctx context.Context, key, longURL string, userID string) (string, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	if _, exists := fs.data[key]; exists {
		return "", ErrAlreadyExists
	}

	fs.counter++
	record := FileRecord{
		UUID:        strconv.Itoa(fs.counter),
		ShortURL:    key,
		OriginalURL: longURL,
		UserID:      userID,
	}
	fs.data[key] = record
	return key, fs.write(record)
}

// Get возвращает длинный URL по короткому ключу.
func (fs *FileStorage) Get(ctx context.Context, key string) (string, bool, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	record, exists := fs.data[key]
	if !exists {
		return "", false, ErrNotFound
	}
	return record.OriginalURL, record.IsDeleted, nil
}

// append одной записи
func (fs *FileStorage) write(record FileRecord) error {
	data, err := json.Marshal(record)
	if err != nil {
		return err
	}
	data = append(data, '\n') // <-- разделитель JSONL
	_, err = fs.file.Write(data)
	return err
}

func (fs *FileStorage) Close() error {
	return fs.file.Close()
}

// SaveBatch сохраняет множество записей атомарно.
func (fs *FileStorage) SaveBatch(ctx context.Context, items map[string]string, userID string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	// Проверяем коллизии
	for key := range items {
		if _, exists := fs.data[key]; exists {
			return ErrAlreadyExists
		}
	}

	// Добавляем в память + генерим UUID
	for key, longURL := range items {
		fs.counter++
		record := FileRecord{
			UUID:        strconv.Itoa(fs.counter),
			ShortURL:    key,
			OriginalURL: longURL,
			UserID:      userID,
		}
		fs.data[key] = record
		if err := fs.write(record); err != nil {
			return err
		}
	}

	return nil
}

func (fs *FileStorage) GetByUserID(ctx context.Context, userID string) ([]model.UserURL, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	var result []model.UserURL
	for _, record := range fs.data {
		if record.UserID == userID && !record.IsDeleted {
			result = append(result, model.UserURL{
				ShortURL:    record.ShortURL,
				OriginalURL: record.OriginalURL,
			})
		}
	}
	return result, nil
}

func (fs *FileStorage) MarkDeleted(ctx context.Context, items []struct{ ShortURL, UserID string }) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	for _, item := range items {
		record, exists := fs.data[item.ShortURL]
		if !exists || record.UserID != item.UserID {
			continue
		}
		record.IsDeleted = true
		fs.data[item.ShortURL] = record
		// Append tombstone в файл
		fs.write(record)
	}
	return nil
}
