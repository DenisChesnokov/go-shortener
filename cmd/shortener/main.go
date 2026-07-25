package main

import (
	"log"
	"net/http"

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
		pg, err = repository.NewPostgres(cfg.DatabaseDSN)
		if err != nil {
			log.Fatal(err)
		}
		defer pg.Close()

		repo = repository.NewInMemory()
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
	h := handler.New(svc, appLog, pg)
	r := handler.NewRouter(h, appLog)

	if err := http.ListenAndServe(cfg.ServerAddress, r); err != nil {
		log.Fatal(err)
	}
}
