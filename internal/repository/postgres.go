package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/DenisChesnokov/go-shortener.git/internal/model"
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
func (p *PostgresStorage) Save(ctx context.Context, key, longURL string, userID string) (string, error) {
	_, err := p.pool.Exec(ctx,
		"INSERT INTO shortener (short_url, original_url, user_id) VALUES ($1, $2, $3)",
		key, longURL, userID)
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
func (p *PostgresStorage) Get(ctx context.Context, key string) (string, bool, error) {
	var originalURL string
	var isDeleted bool
	err := p.pool.QueryRow(ctx,
		"SELECT original_url, is_deleted FROM shortener WHERE short_url = $1",
		key).Scan(&originalURL, &isDeleted)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", false, ErrNotFound
		}
		return "", false, err
	}
	return originalURL, isDeleted, nil
}

func (p *PostgresStorage) Ping(ctx context.Context) error {
	return p.pool.Ping(ctx)
}

func (p *PostgresStorage) Close() error {
	p.pool.Close()
	return nil
}

// SaveBatch сохраняет множество записей в одной транзакции.
func (p *PostgresStorage) SaveBatch(ctx context.Context, items map[string]string, userID string) error {
	if len(items) == 0 {
		return nil
	}

	// Генерируем multi-row INSERT: INSERT INTO ... VALUES ($1,$2), ($3,$4), ...
	var queryBuilder strings.Builder
	queryBuilder.WriteString("INSERT INTO shortener (short_url, original_url, user_id) VALUES ")
	args := make([]interface{}, 0, len(items)*2)
	i := 1
	for key, url := range items {
		if i > 1 {
			queryBuilder.WriteString(", ")
		}
		queryBuilder.WriteString(fmt.Sprintf("($%d, $%d, $%d)", i, i+1, i+2))
		args = append(args, key, url, userID)
		i += 3
	}

	_, err := p.pool.Exec(ctx, queryBuilder.String(), args...)
	return err
}

func (p *PostgresStorage) GetByUserID(ctx context.Context, userID string) ([]model.UserURL, error) {
	rows, err := p.pool.Query(ctx,
		"SELECT short_url, original_url FROM shortener WHERE user_id = $1 AND is_deleted = FALSE", userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []model.UserURL
	for rows.Next() {
		var u model.UserURL
		if err := rows.Scan(&u.ShortURL, &u.OriginalURL); err != nil {
			return nil, err
		}
		result = append(result, u)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func (p *PostgresStorage) MarkDeleted(ctx context.Context, items []model.DeleteTask) error {
	if len(items) == 0 {
		return nil
	}

	// Генерируем: UPDATE shortener SET is_deleted = TRUE WHERE (short_url, user_id) IN (($1, $2), ($3, $4), ...)
	var queryBuilder strings.Builder
	queryBuilder.WriteString("UPDATE shortener SET is_deleted = TRUE WHERE (short_url, user_id) IN (")

	args := make([]interface{}, 0, len(items)*2)
	for i, item := range items {
		if i > 0 {
			queryBuilder.WriteString(", ")
		}
		queryBuilder.WriteString(fmt.Sprintf("($%d, $%d)", i*2+1, i*2+2))
		args = append(args, item.ShortURL, item.UserID)
	}
	queryBuilder.WriteString(")")

	_, err := p.pool.Exec(ctx, queryBuilder.String(), args...)
	return err
}
