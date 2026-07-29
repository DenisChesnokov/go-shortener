package repository

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// PostgresStorage обёртка вокруг *sql.DB для PostgreSQL.
type PostgresStorage struct {
	db *sql.DB
}

// NewPostgres открывает соединение и пингует БД при старте.
func NewPostgres(dsn string, migrationsPath string) (*PostgresStorage, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}

	// golang-migrate определяет драйвер по схеме DSN.
	// Для pgx ожидается схема "pgx://", заменяем "postgres://".
	migrateDSN := strings.Replace(dsn, "postgres://", "pgx://", 1)
	m, err := migrate.New(migrationsPath, migrateDSN)
	if err != nil {
		db.Close()
		return nil, err
	}
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		db.Close()
		m.Close()
		return nil, err
	}
	m.Close()

	// Пинг при старте.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, err
	}

	return &PostgresStorage{db: db}, nil
}

// Save сохраняет длинный URL под коротким ключом.
// Если ключ уже существует — возвращает ErrAlreadyExists.
func (p *PostgresStorage) Save(ctx context.Context, key, longURL string) error {
	_, err := p.db.ExecContext(ctx,
		"INSERT INTO shortener (short_url, original_url) VALUES ($1, $2)",
		key, longURL)

	if err != nil {
		// Обработка UNIQUE constraint violation (код 23505 в PostgreSQL).
		// Приводим ошибку к строке: если в тексте есть "23505", считаем дублем.
		// Это универсально работает через любой драйвер (pgx v4, v5, lib/pq).
		if strings.Contains(err.Error(), "23505") {
			return ErrAlreadyExists
		}
		return err
	}
	return nil
}

// Get возвращает оригинальный URL по короткому ключу.
// Если ключ не найден — возвращает ErrNotFound.
func (p *PostgresStorage) Get(ctx context.Context, key string) (string, error) {
	var originalURL string
	err := p.db.QueryRowContext(ctx,
		"SELECT original_url FROM shortener WHERE short_url = $1",
		key).Scan(&originalURL)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrNotFound
		}
		return "", err
	}
	return originalURL, nil
}

// Ping проверяет соединение с БД.
func (p *PostgresStorage) Ping(ctx context.Context) error {
	return p.db.PingContext(ctx)
}

// Close закрывает соединение с БД.
func (p *PostgresStorage) Close() error {
	return p.db.Close()
}

// SaveBatch сохраняет множество записей в одной транзакции.
func (p *PostgresStorage) SaveBatch(ctx context.Context, items map[string]string) error {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	// defer Rollback безопасен: если Commit уже выполнен, Rollback проигнорируется
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx,
		"INSERT INTO shortener (short_url, original_url) VALUES ($1, $2)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for key, longURL := range items {
		_, err := stmt.ExecContext(ctx, key, longURL)
		if err != nil {
			// При ошибке (например, unique violation) транзакция откатится
			return err
		}
	}

	return tx.Commit()
}
