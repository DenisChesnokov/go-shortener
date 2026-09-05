package main

import (
	"log"
	"net/http"
	"time"

	"github.com/DenisChesnokov/go-shortener.git/internal/auth"
	"github.com/DenisChesnokov/go-shortener.git/internal/config"
	"github.com/DenisChesnokov/go-shortener.git/internal/handler"
	"github.com/DenisChesnokov/go-shortener.git/internal/logger"
	"github.com/DenisChesnokov/go-shortener.git/internal/repository"
	"github.com/DenisChesnokov/go-shortener.git/internal/service"
)

func main() {
	cfg := config.New()
	cfg.ParseFlags()
	cfg.ParseEnv()

	// Создаём логер и прокидываем во все компоненты.
	appLog, err := logger.New(cfg.LogLevel)
	if err != nil {
		log.Fatal(err)
	}
	defer appLog.Sync()

	var repo service.Repository
	var pg *repository.PostgresStorage

	if cfg.DatabaseDSN != "" {
		pg, err = repository.NewPostgres(cfg.DatabaseDSN, "file://migrations")
		if err != nil {
			log.Fatal(err)
		}
		defer pg.Close()

		repo = pg
	} else if cfg.FileStoragePath != "" {
		fs, err := repository.NewFileStorage(cfg.FileStoragePath)
		if err != nil {
			log.Fatal(err)
		}
		repo = fs
	} else {
		repo = repository.NewInMemory()
	}

	svc := service.New(repo, cfg.BaseURL)
	defer svc.Close()

	h := handler.New(svc, appLog, pg)
	jwtMgr := auth.NewJWTManager(cfg.JWTSecret, 24*time.Hour)
	r := handler.NewRouter(h, appLog, jwtMgr)

	if err := http.ListenAndServe(cfg.ServerAddress, r); err != nil {
		log.Fatal(err)
	}
}
