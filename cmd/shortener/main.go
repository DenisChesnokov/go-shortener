package main

import (
	"log"
	"net/http"

	"github.com/DenisChesnokov/go-shortener.git/internal/config"
	"github.com/DenisChesnokov/go-shortener.git/internal/handler"
	"github.com/DenisChesnokov/go-shortener.git/internal/repository"
	"github.com/DenisChesnokov/go-shortener.git/internal/service"
)

// инъекция зависимостей и запуск HTTP-сервера
func main() {
	cfg := config.New()
	cfg.ParseFlags()

	repo := repository.NewInMemory()
	svc := service.New(repo, cfg.BaseURL)
	h := handler.New(svc)
	r := handler.NewRouter(h)

	if err := http.ListenAndServe(cfg.ServerAddress, r); err != nil {
		log.Fatal(err)
	}
}
