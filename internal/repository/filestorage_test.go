package repository

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestFileStorage_SaveAndGet(t *testing.T) {
	// временный файл
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "storage.json")

	fs, err := NewFileStorage(path)
	if err != nil {
		t.Fatal(err)
	}

	// первый Save
	if err := fs.Save(context.Background(), "key1", "http://example.com"); err != nil {
		t.Errorf("Save: %v", err)
		return
	}

	// Get проверка
	got, err := fs.Get(context.Background(), "key1")
	if err != nil {
		t.Errorf("Get: %v", err)
		return
	}
	if got != "http://example.com" {
		t.Errorf("Get: получили %q, хотим %q", got, "http://example.com")
	}
}

func TestFileStorage_PersistsAcrossInstances(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "storage.json")

	// записываем
	fs1, err := NewFileStorage(path)
	if err != nil {
		t.Fatalf("NewFileStorage (fs1): %v", err)
	}

	fs1.Save(context.Background(), "key1", "http://example.com")
	fs1.Save(context.Background(), "key2", "http://yandex.ru")

	// открываем новое хранилище с тем же файлом (имитация рестарта)
	fs2, err := NewFileStorage(path)
	if err != nil {
		t.Fatal(err)
	}

	// проверяем, что данные восстановлены
	got, err := fs2.Get(context.Background(), "key1")
	if err != nil || got != "http://example.com" {
		t.Errorf("key1: получили %q, err %v", got, err)
	}
	got, err = fs2.Get(context.Background(), "key2")
	if err != nil || got != "http://yandex.ru" {
		t.Errorf("key2: получили %q, err %v", got, err)
	}

	// UUID продолжается — new запись должна получить uuid=3
	fs2.Save(context.Background(), "key3", "http://mail.ru")
	// читаем raw файл и проверяем, что uuids 1,2,3 присутствуют
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
		return
	}

	var records []FileRecord
	if err := json.Unmarshal(data, &records); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	uuids := make(map[string]bool, len(records))
	for _, r := range records {
		uuids[r.UUID] = true
	}
	for _, want := range []string{"1", "2", "3"} {
		if !uuids[want] {
			t.Errorf("в файле отсутствует запись с uuid=%q. Все uuid: %v", want, uuids)
			return
		}
	}
}

func TestFileStorage_FileFormat(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "storage.json")

	fs, _ := NewFileStorage(path)
	fs.Save(context.Background(), "key1", "http://example.com")

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
		return
	}
	// Десериализуем — проверка и структуры, и формата.
	var records []FileRecord
	if err := json.Unmarshal(data, &records); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if len(records) != 1 {
		t.Errorf("кол-во записей: %d, хотим 1", len(records))
		return
	}

	r := records[0]
	if r.UUID != "1" {
		t.Errorf("UUID: получили %q, хотим %q", r.UUID, "1")
		return
	}
	if r.ShortURL != "key1" {
		t.Errorf("ShortURL: получили %q, хотим %q", r.ShortURL, "key1")
		return
	}
	if r.OriginalURL != "http://example.com" {
		t.Errorf("OriginalURL: получили %q, хотим %q", r.OriginalURL, "http://example.com")
		return
	}
}
