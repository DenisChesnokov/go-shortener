package repository

import (
	"context"
	"database/sql"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// PostgresStorage обёртка вокруг *sql.DB для PostgreSQL.
type PostgresStorage struct {
	db *sql.DB
}

// NewPostgres открывает соединение и пингует БД при старте.
func NewPostgres(dsn string) (*PostgresStorage, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, err
	}

	return &PostgresStorage{db: db}, nil
}

// Ping проверяет соединение с БД.
func (p *PostgresStorage) Ping(ctx context.Context) error {
	return p.db.PingContext(ctx)
}

// Close закрывает соединение с БД.
func (p *PostgresStorage) Close() error {
	return p.db.Close()
}
