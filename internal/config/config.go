package config

import (
	"flag"
	"os"
)

type Config struct {
	ServerAddress   string
	BaseURL         string
	LogLevel        string
	FileStoragePath string
}

/*
Cоздаёт конфигурацию со значениями по умолчанию.
Парсинг флагов не выполняется: для этого нужен явный вызов ParseFlags,
который должен вызываться только в точке входа (main), чтобы не конфликтовать
с тестами, где флаги не парсятся
*/
func New() *Config {
	return &Config{
		ServerAddress:   "localhost:8080",
		BaseURL:         "http://localhost:8080",
		LogLevel:        "info",
		FileStoragePath: "",
	}
}

// ParseFlags регистрирует флаги командной строки и выполняет их парсинг.
// Вызывать только из main().
func (c *Config) ParseFlags() {
	flag.StringVar(&c.ServerAddress, "a", c.ServerAddress, "Адрес запуска HTTP-сервера")
	flag.StringVar(&c.BaseURL, "b", c.BaseURL, "Базовый адрес результирующего сокращённого URL")
	flag.StringVar(&c.LogLevel, "l", "info", "log level")
	flag.StringVar(&c.FileStoragePath, "f", c.FileStoragePath, "Путь до файла с хранилищем URL")

	flag.Parse()
}

// ParseEnv парсит переменные окружения
func (c *Config) ParseEnv() {

	ServerAddress, exist := os.LookupEnv("SERVER_ADDRESS")
	if exist && ServerAddress != "" {
		c.ServerAddress = ServerAddress
	}

	BaseURL, exist := os.LookupEnv("BASE_URL")
	if exist && BaseURL != "" {
		c.BaseURL = BaseURL
	}

	LogLevel, exist := os.LookupEnv("LOG_LEVEL")
	if exist && LogLevel != "" {
		c.LogLevel = LogLevel
	}

	FileStoragePath, exist := os.LookupEnv("FILE_STORAGE_PATH")
	if exist && FileStoragePath != "" {
		c.FileStoragePath = FileStoragePath
	}
}
