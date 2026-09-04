package repository

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// setupTestPostgres поднимает PostgreSQL в Docker-контейнере для тестов.
func setupTestPostgres(t *testing.T) (*PostgresStorage, func()) {
	t.Helper()
	ctx := context.Background()

	// Подаём образ postgres:16-alpine
	pgContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("test"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second)),
	)
	if err != nil {
		t.Fatalf("failed to start postgres container: %v", err)
	}

	// Получаем DSN
	dsn, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		pgContainer.Terminate(ctx)
		t.Fatalf("failed to get connection string: %v", err)
	}

	// Путь к миграциям
	rootDir, err := filepath.Abs("../..")
	if err != nil {
		pgContainer.Terminate(ctx)
		t.Fatalf("failed to get root dir: %v", err)
	}
	migrationsPath := "file://" + filepath.Join(rootDir, "migrations")

	// Создаём хранилище (применяет миграции)
	pg, err := NewPostgres(dsn, migrationsPath)
	if err != nil {
		pgContainer.Terminate(ctx)
		t.Fatalf("NewPostgres: %v", err)
	}

	// Cleanup: закрываем storage и удаляем контейнер
	cleanup := func() {
		pg.Close()
		if err := pgContainer.Terminate(ctx); err != nil {
			t.Logf("failed to terminate container: %v", err)
		}
	}

	return pg, cleanup
}

// TestPostgres_SaveAndGet проверяет основные операции с БД: Save, Get, обработку дублей и отсутствующих ключей.
func TestPostgres_SaveAndGet(t *testing.T) {
	pg, cleanup := setupTestPostgres(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if _, err := pg.Save(ctx, "key1", "http://example.com", ""); err != nil {
		t.Errorf("Save: %v", err)
		return
	}

	got, _, err := pg.Get(ctx, "key1")
	if err != nil {
		t.Errorf("Get: %v", err)
		return
	}
	if got != "http://example.com" {
		t.Errorf("Get: получили %q, хотим %q", got, "http://example.com")
	}

	// Дубликат
	_, err = pg.Save(ctx, "key1", "http://other.com", "")
	if err != ErrAlreadyExists {
		t.Errorf("ожидали ErrAlreadyExists, получили %v", err)
	}

	// Несуществующий
	_, _, err = pg.Get(ctx, "nonexistent")
	if err != ErrNotFound {
		t.Errorf("ожидали ErrNotFound, получили %v", err)
	}
}

// TestPostgres_Ping проверяет, что Ping работает.
func TestPostgres_Ping(t *testing.T) {
	pg, cleanup := setupTestPostgres(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	if err := pg.Ping(ctx); err != nil {
		t.Errorf("Ping: %v", err)
	}
}

func TestPostgres_SaveBatch(t *testing.T) {
	pg, cleanup := setupTestPostgres(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	items := map[string]string{
		"batchKey1": "http://example.com",
		"batchKey2": "http://yandex.ru",
	}

	if err := pg.SaveBatch(ctx, items, ""); err != nil {
		t.Errorf("SaveBatch: %v", err)
		return
	}

	got1, _, err := pg.Get(ctx, "batchKey1")
	if err != nil || got1 != "http://example.com" {
		t.Errorf("Get batchKey1: получили %q, err %v", got1, err)
	}
	got2, _, err := pg.Get(ctx, "batchKey2")
	if err != nil || got2 != "http://yandex.ru" {
		t.Errorf("Get batchKey2: получили %q, err %v", got2, err)
	}
}

func TestPostgres_GetByUserID(t *testing.T) {
	pg, cleanup := setupTestPostgres(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	userID := "test-user-123"
	if _, err := pg.Save(ctx, "u1key1", "http://example1.com", userID); err != nil {
		t.Fatalf("Save 1: %v", err)
	}
	if _, err := pg.Save(ctx, "u1key2", "http://example2.com", userID); err != nil {
		t.Fatalf("Save 2: %v", err)
	}
	// чужой URL
	if _, err := pg.Save(ctx, "u2key1", "http://other.com", "other-user"); err != nil {
		t.Fatalf("Save 3: %v", err)
	}

	urls, err := pg.GetByUserID(ctx, userID)
	if err != nil {
		t.Fatalf("GetByUserID: %v", err)
	}
	if len(urls) != 2 {
		t.Errorf("GetByUserID: получили %d URL, хотим 2", len(urls))
	}
}
