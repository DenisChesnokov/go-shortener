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

// инъекция зависимостей и запуск HTTP-сервера
func main() {
	// Сначала создаем конфиг с дефолтными значениями
	cfg := config.New()
	// Перезаписываем значения если заданы флаги
	cfg.ParseFlags()
	// Перезаписываем значения если есть переменные окружения
	cfg.ParseEnv()

	if err := logger.Initialize("info"); err != nil {
		log.Fatal(err)
	}
	defer logger.Sync()

	var repo service.Repository
	// Если задан путь к файлу — используем файл
	// Иначе — in-memory
	if cfg.FileStoragePath != "" {
		fs, err := repository.NewFileStorage(cfg.FileStoragePath)
		if err != nil {
			log.Fatal(err)
		}
		repo = fs
	} else {
		repo = repository.NewInMemory()
	}

	svc := service.New(repo, cfg.BaseURL)
	h := handler.New(svc)
	r := handler.NewRouter(h)

	if err := http.ListenAndServe(cfg.ServerAddress, r); err != nil {
		log.Fatal(err)
	}
}
