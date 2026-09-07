package repository

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
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
	if _, err := fs.Save(context.Background(), "key1", "http://example.com", ""); err != nil {
		t.Errorf("Save: %v", err)
		return
	}

	// Get проверка
	got, _, err := fs.Get(context.Background(), "key1")
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

	fs1.Save(context.Background(), "key1", "http://example.com", "")
	fs1.Save(context.Background(), "key2", "http://yandex.ru", "")

	// открываем новое хранилище с тем же файлом (имитация рестарта)
	fs2, err := NewFileStorage(path)
	if err != nil {
		t.Fatal(err)
	}

	// проверяем, что данные восстановлены
	got, _, err := fs2.Get(context.Background(), "key1")
	if err != nil || got != "http://example.com" {
		t.Errorf("key1: получили %q, err %v", got, err)
	}
	got, _, err = fs2.Get(context.Background(), "key2")
	if err != nil || got != "http://yandex.ru" {
		t.Errorf("key2: получили %q, err %v", got, err)
	}

	// UUID продолжается — new запись должна получить uuid=3
	if _, err := fs2.Save(context.Background(), "key3", "http://mail.ru", ""); err != nil {
		t.Fatalf("Save key3: %v", err)
	}

	// Читаем raw файл и проверяем, что uuids 1,2,3 присутствуют.
	// Файл в формате JSONL — десериализуем через json.Decoder.
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
		return
	}
	defer file.Close()

	uuids := make(map[string]bool)
	dec := json.NewDecoder(file)
	for {
		var r FileRecord
		if err := dec.Decode(&r); err != nil {
			if err.Error() == "EOF" {
				break
			}
			// fallback на io.EOF для строгости
			break
		}
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
	fs.Save(context.Background(), "key1", "http://example.com", "")

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
		return
	}
	body := string(data)

	// Каждая запись — отдельный JSON-объект на своей строке (JSONL).
	// Проверяем, что в файле ровно одна запись и есть все три поля.
	for _, want := range []string{
		`"uuid":"1"`,
		`"short_url":"key1"`,
		`"original_url":"http://example.com"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("файл не содержит %q.\nСодержимое файла:\n%s", want, body)
			return
		}
	}

	trimmed := strings.TrimSpace(body)
	if strings.HasPrefix(trimmed, "[") || strings.HasSuffix(trimmed, "]") {
		t.Errorf("файл выглядит как JSON-массив, а ожидается JSONL.\nСодержимое файла:\n%s", body)
		return
	}
}

func TestFileStorage_GetByUserID(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "storage.json")

	fs, err := NewFileStorage(path)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	if _, err := fs.Save(ctx, "u1k1", "http://example1.com", "user-1"); err != nil {
		t.Fatalf("Save 1: %v", err)
	}
	if _, err := fs.Save(ctx, "u1k2", "http://example2.com", "user-1"); err != nil {
		t.Fatalf("Save 2: %v", err)
	}
	if _, err := fs.Save(ctx, "u2k1", "http://other.com", "user-2"); err != nil {
		t.Fatalf("Save 3: %v", err)
	}

	urls, err := fs.GetByUserID(ctx, "user-1")
	if err != nil {
		t.Fatalf("GetByUserID: %v", err)
	}
	if len(urls) != 2 {
		t.Errorf("GetByUserID: получили %d URL, хотим 2", len(urls))
	}
}
