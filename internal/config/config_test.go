package config

import "testing"

func TestNew(t *testing.T) {
	cfg := New()

	if cfg.ServerAddress != "localhost:8080" {
		t.Errorf("ServerAddress: получили %q, хотим %q", cfg.ServerAddress, "localhost:8080")
	}
	if cfg.BaseURL != "http://localhost:8080" {
		t.Errorf("BaseURL: получили %q, хотим %q", cfg.BaseURL, "http://localhost:8080")
	}
}

func TestParseEnv(t *testing.T) {
	t.Setenv("SERVER_ADDRESS", "0.0.0.0:9090")
	t.Setenv("BASE_URL", "http://example.com")

	cfg := New()
	cfg.ParseEnv()

	if cfg.ServerAddress != "0.0.0.0:9090" {
		t.Errorf("ServerAddress: получили %q, хотим %q", cfg.ServerAddress, "0.0.0.0:9090")
	}
	if cfg.BaseURL != "http://example.com" {
		t.Errorf("BaseURL: получили %q, хотим %q", cfg.BaseURL, "http://example.com")
	}
}

func TestParseEnvNoVars(t *testing.T) {
	cfg := New()
	cfg.ParseEnv()

	if cfg.ServerAddress != "localhost:8080" {
		t.Errorf("ServerAddress: получили %q, хотим %q", cfg.ServerAddress, "localhost:8080")
	}
	if cfg.BaseURL != "http://localhost:8080" {
		t.Errorf("BaseURL: получили %q, хотим %q", cfg.BaseURL, "http://localhost:8080")
	}
}

func TestPriority(t *testing.T) {
	t.Setenv("SERVER_ADDRESS", "env-host:1111")
	t.Setenv("BASE_URL", "http://env.example.com")

	cfg := New()

	// Имитируем результат ParseFlags (как будто флаги установили другие значения)
	cfg.ServerAddress = "flag-host:2222"
	cfg.BaseURL = "http://flag.example.com"

	// ParseEnv должен перезаписать флаги значениями из env
	cfg.ParseEnv()

	if cfg.ServerAddress != "env-host:1111" {
		t.Errorf("ServerAddress: получили %q, хотим %q", cfg.ServerAddress, "env-host:1111")
	}
	if cfg.BaseURL != "http://env.example.com" {
		t.Errorf("BaseURL: получили %q, хотим %q", cfg.BaseURL, "http://env.example.com")
	}
}
