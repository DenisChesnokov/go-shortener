package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// PostgresStorage обёртка вокруг *sql.DB для PostgreSQL.
type PostgresStorage struct {
	pool *pgxpool.Pool
}

func NewPostgres(dsn string, migrationsPath string) (*PostgresStorage, error) {
	// 1. Миграции (golang-migrate использует DSN напрямую)
	migrateDSN := strings.Replace(dsn, "postgres://", "pgx://", 1)
	m, err := migrate.New(migrationsPath, migrateDSN)
	if err != nil {
		return nil, err
	}
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		m.Close()
		return nil, err
	}
	m.Close()

	// 2. Pool
	ctx, cancel := context.WithTimeout(context.TODO(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, err
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}

	return &PostgresStorage{pool: pool}, nil
}

// Save сохраняет длинный URL под коротким ключом.
// Возвращает (key, nil) при успехе.
// Возвращает ("", ErrAlreadyExists) при коллизии сгенерированного ключа.
// Возвращает (existingKey, ErrAlreadyExists) при дубликате оригинального URL.
func (p *PostgresStorage) Save(ctx context.Context, key, longURL string) (string, error) {
	_, err := p.pool.Exec(ctx,
		"INSERT INTO shortener (short_url, original_url) VALUES ($1, $2)",
		key, longURL)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			if strings.Contains(pgErr.Message, "idx_original_url") {
				var existingKey string
				err = p.pool.QueryRow(ctx,
					"SELECT short_url FROM shortener WHERE original_url = $1",
					longURL).Scan(&existingKey)
				if err != nil {
					return "", err
				}
				return existingKey, ErrAlreadyExists
			}
			return "", ErrAlreadyExists
		}
		return "", err
	}
	return key, nil
}

// Get возвращает оригинальный URL по короткому ключу.
// Если ключ не найден — возвращает ErrNotFound.
func (p *PostgresStorage) Get(ctx context.Context, key string) (string, error) {
	var originalURL string
	err := p.pool.QueryRow(ctx,
		"SELECT original_url FROM shortener WHERE short_url = $1",
		key).Scan(&originalURL)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrNotFound
		}
		return "", err
	}
	return originalURL, nil
}

func (p *PostgresStorage) Ping(ctx context.Context) error {
	return p.pool.Ping(ctx)
}

func (p *PostgresStorage) Close() error {
	p.pool.Close()
	return nil
}

// SaveBatch сохраняет множество записей в одной транзакции.
func (p *PostgresStorage) SaveBatch(ctx context.Context, items map[string]string) error {
	if len(items) == 0 {
		return nil
	}

	// Генерируем multi-row INSERT: INSERT INTO ... VALUES ($1,$2), ($3,$4), ...
	var queryBuilder strings.Builder
	queryBuilder.WriteString("INSERT INTO shortener (short_url, original_url) VALUES ")
	args := make([]interface{}, 0, len(items)*2)
	i := 1
	for key, url := range items {
		if i > 1 {
			queryBuilder.WriteString(", ")
		}
		queryBuilder.WriteString(fmt.Sprintf("($%d, $%d)", i, i+1))
		args = append(args, key, url)
		i += 2
	}

	_, err := p.pool.Exec(ctx, queryBuilder.String(), args...)
	return err
}
