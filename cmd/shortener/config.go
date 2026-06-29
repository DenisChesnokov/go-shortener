package main

import (
	"flag"
	"os"
	"strings"
)

// Config хранит настройки для запуска HTTP-сервера и сокращателя ссылок.
type Config struct {
	ServerAddress string
	BaseURL       string
}

var Cfg *Config // Глобальная переменная видна везде в пакете main

func Init() {
	Cfg = &Config{}

	flag.StringVar(&Cfg.ServerAddress, "a", "localhost:8080", "Адрес запуска HTTP-сервера")
	flag.StringVar(&Cfg.BaseURL, "b", "http://localhost:8080", "Базовый адрес результирующего сокращённого URL")

	// Если мы запускаем тесты (через go test), флаги парсить не нужно, чтобы не было конфликтов
	if !strings.HasSuffix(os.Args[0], ".test") && !strings.Contains(os.Args[0], "/_test/") {
		flag.Parse()
	}
}
