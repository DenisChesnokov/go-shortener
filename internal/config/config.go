package config

import (
	"crypto/rand"
	"encoding/hex"
	"flag"
	"os"
)

type Config struct {
	ServerAddress   string
	BaseURL         string
	LogLevel        string
	FileStoragePath string
	DatabaseDSN     string
	JWTSecret       string
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
		DatabaseDSN:     "",
	}
}

// ParseFlags регистрирует флаги командной строки и выполняет их парсинг.
// Вызывать только из main().
func (c *Config) ParseFlags() {
	flag.StringVar(&c.ServerAddress, "a", c.ServerAddress, "Адрес запуска HTTP-сервера")
	flag.StringVar(&c.BaseURL, "b", c.BaseURL, "Базовый адрес результирующего сокращённого URL")
	flag.StringVar(&c.LogLevel, "l", c.LogLevel, "log level")
	flag.StringVar(&c.FileStoragePath, "f", c.FileStoragePath, "Путь до файла с хранилищем URL")
	flag.StringVar(&c.DatabaseDSN, "d", c.DatabaseDSN, "Адрес подключения к БД")
	flag.StringVar(&c.JWTSecret, "jwt-key", "", "JWT secret key")

	flag.Parse()
}

// ParseEnv парсит переменные окружения
func (c *Config) ParseEnv() {

	serverAddress, exist := os.LookupEnv("SERVER_ADDRESS")
	if exist && serverAddress != "" {
		c.ServerAddress = serverAddress
	}

	baseURL, exist := os.LookupEnv("BASE_URL")
	if exist && baseURL != "" {
		c.BaseURL = baseURL
	}

	logLevel, exist := os.LookupEnv("LOG_LEVEL")
	if exist && logLevel != "" {
		c.LogLevel = logLevel
	}

	fileStoragePath, exist := os.LookupEnv("FILE_STORAGE_PATH")
	if exist && fileStoragePath != "" {
		c.FileStoragePath = fileStoragePath
	}

	dataBaseDSN, exist := os.LookupEnv("DATABASE_DSN")
	if exist && dataBaseDSN != "" {
		c.DatabaseDSN = dataBaseDSN
	}

	jwtSecret, exist := os.LookupEnv("JWT_SECRET")
	if exist && jwtSecret != "" {
		c.JWTSecret = jwtSecret
	}

	// Если секрет не задан — генерируем случайный
	if c.JWTSecret == "" {
		bytes := make([]byte, 32)
		if _, err := rand.Read(bytes); err != nil {
			c.JWTSecret = "default-secret-change-me"
			return
		}
		c.JWTSecret = hex.EncodeToString(bytes)
	}
}
