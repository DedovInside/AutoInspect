package config

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/ilyakaznacheev/cleanenv"
)

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

	if err := validate(&cfg); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &cfg, nil
}

func MustLoad() *Config {
	cfg, err := Load()
	if err != nil {
		panic(fmt.Sprintf("Failed to load config: %v", err))
	}
	return cfg
}

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

	validLogLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if !validLogLevels[cfg.Logging.Level] {
		return fmt.Errorf("invalid log level: %s", cfg.Logging.Level)
	}

	if cfg.Environment == "production" && len(cfg.Auth.JWTSecret) < 32 {
		return fmt.Errorf("JWT_SECRET must be at least 32 characters in production (current length: %d)", len(cfg.Auth.JWTSecret))
	}
	if len(cfg.Auth.JWTSecret) < 16 {
		return fmt.Errorf("JWT_SECRET must be at least 16 characters")
	}

	if cfg.HTTP.Port == "" {
		return fmt.Errorf("HTTP_PORT cannot be empty")
	}

	if strings.TrimSpace(cfg.Database.URL) == "" {
		return fmt.Errorf("DATABASE_URL cannot be empty")
	}

	if strings.TrimSpace(cfg.GRPC.MLServiceAddr) == "" {
		return fmt.Errorf("GRPC_ML_SERVICE_ADDR cannot be empty")
	}

	if strings.TrimSpace(cfg.MinIO.Endpoint) == "" {
		return fmt.Errorf("MINIO_ENDPOINT cannot be empty")
	}

	if strings.TrimSpace(cfg.MinIO.ImagesBucket) == "" {
		return fmt.Errorf("MINIO_IMAGES_BUCKET cannot be empty")
	}

	if strings.TrimSpace(cfg.MinIO.ModelsBucket) == "" {
		return fmt.Errorf("MINIO_MODELS_BUCKET cannot be empty")
	}

	if cfg.Auth.BcryptCost < 4 || cfg.Auth.BcryptCost > 31 {
		return fmt.Errorf("BCRYPT_COST must be between 4 and 31")
	}

	if cfg.Worker.Concurrency <= 0 {
		return fmt.Errorf("WORKER_CONCURRENCY must be greater than 0")
	}

	if cfg.Redis.MaxRetries < 0 {
		return fmt.Errorf("REDIS_MAX_RETRIES cannot be negative")
	}

	if strings.TrimSpace(cfg.Auth.JWTIssuer) == "" {
		return fmt.Errorf("JWT_ISSUER cannot be empty")
	}

	if cfg.Auth.AccessTokenTTL <= 0 {
		return fmt.Errorf("ACCESS_TOKEN_TTL must be greater than 0")
	}

	if cfg.Auth.RefreshTokenTTL <= 0 {
		return fmt.Errorf("REFRESH_TOKEN_TTL must be greater than 0")
	}

	if cfg.Auth.OAuthStateTTL <= 0 {
		return fmt.Errorf("OAUTH_STATE_TTL must be greater than 0")
	}

	if strings.TrimSpace(cfg.Auth.YandexClientID) == "" {
		return fmt.Errorf("YANDEX_CLIENT_ID cannot be empty")
	}

	if strings.TrimSpace(cfg.Auth.YandexClientSecret) == "" {
		return fmt.Errorf("YANDEX_CLIENT_SECRET cannot be empty")
	}

	if strings.TrimSpace(cfg.Auth.YandexRedirectURL) == "" {
		return fmt.Errorf("YANDEX_REDIRECT_URL cannot be empty")
	}

	u, err := url.Parse(cfg.Auth.YandexRedirectURL)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return fmt.Errorf("YANDEX_REDIRECT_URL must be an absolute URL")
	}

	return nil
}
