package config

import (
	"fmt"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
)

// Данный метод загружает конфигурацию из .env файла и переменных окружения
func Load() (*Config, error) {
	var cfg Config

	envFile := os.Getenv("CONFIG_FILE")
	if envFile == "" {
		envFile = ".env"
	}

	if _, err := os.Stat(envFile); err == nil {
		if err := cleanenv.ReadConfig(envFile, &cfg); err != nil {
			return nil, fmt.Errorf("failed to read config from %s: %w", envFile, err)
		}
	} else {
		if err := cleanenv.ReadEnv(&cfg); err != nil {
			return nil, fmt.Errorf("failed to read config from environment: %w", err)
		}
	}

	// Валидация критических параметров
	if err := validate(&cfg); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &cfg, nil
}

// Данный метод загружает конфигурацию и паникует при ошибке
func MustLoad() *Config {
	cfg, err := Load()
	if err != nil {
		panic(fmt.Sprintf("Failed to load config: %v", err))
	}
	return cfg
}

// Данный метод проверяет корректность конфигурации
func validate(cfg *Config) error {
	validEnvs := map[string]bool{
		"development": true,
		"staging":     true,
		"production":  true,
		"test":        true,
	}
	if !validEnvs[cfg.Environment] {
		return fmt.Errorf("invalid environment: %s (allowed: development, staging, production, test)", cfg.Environment)
	}

	// Проверка уровня логирования
	validLogLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if !validLogLevels[cfg.Logging.Level] {
		return fmt.Errorf("invalid log level: %s", cfg.Logging.Level)
	}

	// Проверка JWT секрета в production
	if cfg.Environment == "production" && len(cfg.Auth.JWTSecret) < 32 {
		return fmt.Errorf("JWT_SECRET must be at least 32 characters in production (current length: %d)", len(cfg.Auth.JWTSecret))
	}

	// Проверка порта
	if cfg.HTTP.Port == "" {
		return fmt.Errorf("HTTP_PORT cannot be empty")
	}

	return nil
}
