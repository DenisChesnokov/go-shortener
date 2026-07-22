package repository

import (
	"context"
	"encoding/json"
	"os"
	"strconv"
	"sync"
)

// FileRecord — формат одной записи в файле хранилища.
type FileRecord struct {
	UUID        string `json:"uuid"`
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

// FileStorage — реализация репозитория через файл на диске.
// В памяти держит map[key]FileRecord для быстрых Get-операций,
type FileStorage struct {
	mu       sync.RWMutex
	data     map[string]FileRecord
	counter  int // счетчик для генерации UUID
	filePath string
}

// NewFileStorage создаёт хранилище и загружает существующие данные из файла.
// Если файл не существует — начинается с пустого хранилища.
func NewFileStorage(filePath string) (*FileStorage, error) {
	fs := &FileStorage{
		data:     make(map[string]FileRecord),
		filePath: filePath,
	}
	if err := fs.load(); err != nil {
		return nil, err
	}
	return fs, nil
}

// load читает файл и восстанавливает данные в память.
func (fs *FileStorage) load() error {
	data, err := os.ReadFile(fs.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // файла нет, начнём с пустого хранилища
		}
		return err
	}
	if len(data) == 0 {
		return nil // пустой файл, тоже норм
	}

	var records []FileRecord
	if err := json.Unmarshal(data, &records); err != nil {
		return err
	}

	maxUUID := 0
	for _, r := range records {
		fs.data[r.ShortURL] = r
		if u, err := strconv.Atoi(r.UUID); err == nil && u > maxUUID {
			maxUUID = u
		}
	}
	fs.counter = maxUUID // продолжаем нумерацию с максимума
	return nil
}

// Save сохраняет длинный URL под коротким ключом.
// Обновляет map в памяти и атомарно перезаписывает файл.
func (fs *FileStorage) Save(ctx context.Context, key, longURL string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	if _, exists := fs.data[key]; exists {
		return ErrAlreadyExists
	}

	fs.counter++
	fs.data[key] = FileRecord{
		UUID:        strconv.Itoa(fs.counter),
		ShortURL:    key,
		OriginalURL: longURL,
	}

	return fs.persist()
}

// Get возвращает длинный URL по короткому ключу.
func (fs *FileStorage) Get(ctx context.Context, key string) (string, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	record, exists := fs.data[key]
	if !exists {
		return "", ErrNotFound
	}
	return record.OriginalURL, nil
}

// persist перезаписывает файл со всеми записями.
// пишем во временный файл, затем os.Rename.
func (fs *FileStorage) persist() error {
	records := make([]FileRecord, 0, len(fs.data))
	for _, r := range fs.data {
		records = append(records, r)
	}

	data, err := json.MarshalIndent(records, "", " ")
	if err != nil {
		return err
	}

	tmpPath := fs.filePath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0666); err != nil {
		return err
	}
	return os.Rename(tmpPath, fs.filePath)
}
