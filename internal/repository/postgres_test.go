package repository

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestPostgres_SaveAndGet проверяет основные операции с БД: Save, Get, обработку дублей и отсутствующих ключей.
func TestPostgres_SaveAndGet(t *testing.T) {
	dsn := os.Getenv("DATABASE_DSN")
	if dsn == "" {
		t.Skip("DATABASE_DSN not set, skipping integration test")
	}

	// Тест выполняется из директории internal/repository,
	// поэтому поднимаемся на две директории вверх до корня проекта.
	rootDir, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("failed to get root dir: %v", err)
	}
	migrationsPath := "file://" + filepath.Join(rootDir, "migrations")

	pg, err := NewPostgres(dsn, migrationsPath)
	if err != nil {
		t.Fatalf("NewPostgres: %v", err)
	}
	defer pg.Close()

	// Устанавливаем таймаут на операции с БД.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Очищаем таблицу перед тестом (чтобы тест был идемпотентным).
	_, err = pg.db.ExecContext(ctx, "DELETE FROM shortener")
	if err != nil {
		t.Fatalf("failed to clean table: %v", err)
	}

	// 1. Успешный Save.
	if _, err := pg.Save(ctx, "key1", "http://example.com"); err != nil {
		t.Errorf("Save: %v", err)
		return
	}

	// 2. Успешный Get.
	got, err := pg.Get(ctx, "key1")
	if err != nil {
		t.Errorf("Get: %v", err)
		return
	}
	if got != "http://example.com" {
		t.Errorf("Get: получили %q, хотим %q", got, "http://example.com")
	}

	// 3. Попытка сохранить дубликат ключа.
	_, err = pg.Save(ctx, "key1", "http://other.com")
	if err != ErrAlreadyExists {
		t.Errorf("ожидали ErrAlreadyExists, получили %v", err)
	}

	// 4. Запрос несуществующего ключа.
	_, err = pg.Get(ctx, "nonexistent")
	if err != ErrNotFound {
		t.Errorf("ожидали ErrNotFound, получили %v", err)
	}
}

// TestPostgres_Ping проверяет, что Ping работает.
func TestPostgres_Ping(t *testing.T) {
	dsn := os.Getenv("DATABASE_DSN")
	if dsn == "" {
		t.Skip("DATABASE_DSN not set, skipping integration test")
	}

	rootDir, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("failed to get root dir: %v", err)
	}
	migrationsPath := "file://" + filepath.Join(rootDir, "migrations")

	pg, err := NewPostgres(dsn, migrationsPath)
	if err != nil {
		t.Fatalf("NewPostgres: %v", err)
	}
	defer pg.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	if err := pg.Ping(ctx); err != nil {
		t.Errorf("Ping: %v", err)
	}
}

func TestPostgres_SaveBatch(t *testing.T) {
	dsn := os.Getenv("DATABASE_DSN")
	if dsn == "" {
		t.Skip("DATABASE_DSN not set, skipping integration test")
	}

	rootDir, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("failed to get root dir: %v", err)
	}
	migrationsPath := "file://" + filepath.Join(rootDir, "migrations")

	pg, err := NewPostgres(dsn, migrationsPath)
	if err != nil {
		t.Fatalf("NewPostgres: %v", err)
	}
	defer pg.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Очистка
	_, err = pg.db.ExecContext(ctx, "DELETE FROM shortener")
	if err != nil {
		t.Fatalf("failed to clean table: %v", err)
	}

	items := map[string]string{
		"batchKey1": "http://example.com",
		"batchKey2": "http://yandex.ru",
	}

	if err := pg.SaveBatch(ctx, items); err != nil {
		t.Errorf("SaveBatch: %v", err)
		return
	}

	// Проверяем, что обе записи сохранились
	got1, err := pg.Get(ctx, "batchKey1")
	if err != nil || got1 != "http://example.com" {
		t.Errorf("Get batchKey1: получили %q, err %v", got1, err)
	}
	got2, err := pg.Get(ctx, "batchKey2")
	if err != nil || got2 != "http://yandex.ru" {
		t.Errorf("Get batchKey2: получили %q, err %v", got2, err)
	}
}
