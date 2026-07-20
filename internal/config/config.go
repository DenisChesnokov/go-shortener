package config

import (
	"flag"
	"os"
)

type Config struct {
	ServerAddress string
	BaseURL       string
}

/*
Cоздаёт конфигурацию со значениями по умолчанию.
Парсинг флагов не выполняется: для этого нужен явный вызов ParseFlags,
который должен вызываться только в точке входа (main), чтобы не конфликтовать
с тестами, где флаги не парсятся
*/
func New() *Config {
	return &Config{
		ServerAddress: "localhost:8080",
		BaseURL:       "http://localhost:8080",
	}
}

// ParseFlags регистрирует флаги командной строки и выполняет их парсинг.
// Вызывать только из main().
func (c *Config) ParseFlags() {
	flag.StringVar(&c.ServerAddress, "a", c.ServerAddress, "Адрес запуска HTTP-сервера")
	flag.StringVar(&c.BaseURL, "b", c.BaseURL, "Базовый адрес результирующего сокращённого URL")

	flag.Parse()
}

// ParseEnv парсит переменные окружения
func (c *Config) ParseEnv() {

	ServerAddress, exist := os.LookupEnv("SERVER_ADDRESS")
	if exist {
		c.ServerAddress = ServerAddress
	}

	BaseURL, exist := os.LookupEnv("BASE_URL")
	if exist {
		c.BaseURL = BaseURL
	}

}
