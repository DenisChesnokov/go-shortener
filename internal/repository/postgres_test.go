package repository

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestPostgres_Ping(t *testing.T) {
	dsn := os.Getenv("DATABASE_DSN")
	if dsn == "" {
		t.Skip("DATABASE_DSN not set, skipping integration test")
	}

	pg, err := NewPostgres(dsn)
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
