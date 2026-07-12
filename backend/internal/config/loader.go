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

	// #nosec
	if _, err := os.Stat(envFile); err == nil {
		if err := cleanenv.ReadConfig(envFile, &cfg); err != nil {
			return nil, fmt.Errorf("failed to read config from %s: %w", envFile, err)
		}
	} else {
		if err := cleanenv.ReadEnv(&cfg); err != nil {
			return nil, fmt.Errorf("failed to read config from environment: %w", err)
		}
	}

	if err := applySecretFiles(&cfg); err != nil {
		return nil, fmt.Errorf("failed to read secret files: %w", err)
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
	if err := validateEnvironment(cfg); err != nil {
		return err
	}

	if err := validateAuth(cfg); err != nil {
		return err
	}

	if err := validateRedis(cfg); err != nil {
		return err
	}

	if err := validateYandex(cfg); err != nil {
		return err
	}

	if err := validateS3(cfg); err != nil {
		return err
	}

	if err := validateKafka(cfg); err != nil {
		return err
	}
	return nil
}

func validateEnvironment(cfg *Config) error {
	validEnvs := map[string]bool{
		"development": true,
		"staging":     true,
		"production":  true,
		"test":        true,
	}
	if !validEnvs[cfg.Environment] {
		return fmt.Errorf("invalid environment: %s (allowed: development, staging, production, test)", cfg.Environment)
	}
	if cfg.HTTP.Port == "" {
		return fmt.Errorf("HTTP_PORT cannot be empty")
	}
	if len(cfg.HTTP.WSAllowedOrigins) == 0 {
		return fmt.Errorf("WS_ALLOWED_ORIGINS cannot be empty")
	}
	if len(cfg.HTTP.CORSAllowedOrigins) == 0 {
		return fmt.Errorf("CORS_ALLOWED_ORIGINS cannot be empty")
	}
	if strings.TrimSpace(cfg.Database.URL) == "" {
		return fmt.Errorf("DATABASE_URL cannot be empty")
	}
	return nil
}

func validateAuth(cfg *Config) error {
	if strings.TrimSpace(cfg.Auth.JWTActiveKeyID) == "" {
		return fmt.Errorf("JWT_ACTIVE_KEY_ID cannot be empty")
	}
	if strings.TrimSpace(cfg.Auth.JWTPrivateKey) == "" {
		return fmt.Errorf("JWT_PRIVATE_KEY cannot be empty")
	}
	if strings.TrimSpace(cfg.Auth.JWTPublicKey) == "" && strings.TrimSpace(cfg.Auth.JWTPublicKeys) == "" {
		return fmt.Errorf("JWT_PUBLIC_KEY or JWT_PUBLIC_KEYS cannot be empty")
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
	if cfg.Auth.BcryptCost < 4 || cfg.Auth.BcryptCost > 31 {
		return fmt.Errorf("BCRYPT_COST must be between 4 and 31")
	}
	return nil
}

func validateRedis(cfg *Config) error {
	if cfg.Redis.MaxRetries < 0 {
		return fmt.Errorf("REDIS_MAX_RETRIES cannot be negative")
	}
	return nil
}

func validateYandex(cfg *Config) error {
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

func validateS3(cfg *Config) error {
	if err := validateS3RequiredFields(cfg); err != nil {
		return err
	}
	if err := validateS3Endpoints(cfg); err != nil {
		return err
	}
	return validateS3BucketUniqueness(
		cfg.S3.BucketUploads,
		cfg.S3.BucketModels,
		cfg.S3.BucketResults,
	)
}

func validateS3RequiredFields(cfg *Config) error {
	if strings.TrimSpace(cfg.S3.AccessKey) == "" || strings.TrimSpace(cfg.S3.SecretKey) == "" {
		return fmt.Errorf("s3 credentials are required")
	}
	if strings.TrimSpace(cfg.S3.Region) == "" {
		return fmt.Errorf("S3_REGION cannot be empty")
	}
	if strings.TrimSpace(cfg.S3.BucketUploads) == "" ||
		strings.TrimSpace(cfg.S3.BucketModels) == "" ||
		strings.TrimSpace(cfg.S3.BucketResults) == "" {
		return fmt.Errorf("s3 bucket names are required")
	}
	if cfg.S3.PresignedURLTTL <= 0 {
		return fmt.Errorf("S3_PRESIGNED_URL_TTL must be greater than 0")
	}
	return nil
}

func validateS3Endpoints(cfg *Config) error {
	if err := validateOptionalAbsoluteURL(cfg.S3.Endpoint, "S3_ENDPOINT"); err != nil {
		return err
	}
	return validateOptionalAbsoluteURL(cfg.S3.PublicEndpoint, "S3_PUBLIC_ENDPOINT")
}

func validateOptionalAbsoluteURL(value, name string) error {
	if strings.TrimSpace(value) == "" {
		return nil
	}

	u, err := url.Parse(value)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return fmt.Errorf("%s must be an absolute URL", name)
	}
	return nil
}

func validateS3BucketUniqueness(buckets ...string) error {
	seen := make(map[string]struct{}, len(buckets))
	for _, bucket := range buckets {
		name := strings.TrimSpace(bucket)
		if _, ok := seen[name]; ok {
			return fmt.Errorf("s3 bucket names must be unique")
		}
		seen[name] = struct{}{}
	}

	return nil
}

func applySecretFiles(cfg *Config) error {
	// #nosec G101 -- these are environment variable and Docker secret names, not credential values.
	secretTargets := map[string]secretFileTarget{
		"DATABASE_URL_FILE":         {target: &cfg.Database.URL, dockerSecretName: "database_url"},
		"REDIS_PASSWORD_FILE":       {target: &cfg.Redis.Password, dockerSecretName: "redis_password"},
		"JWT_PRIVATE_KEY_FILE":      {target: &cfg.Auth.JWTPrivateKey, dockerSecretName: "jwt_private_key"},
		"JWT_PUBLIC_KEY_FILE":       {target: &cfg.Auth.JWTPublicKey, dockerSecretName: "jwt_public_key"},
		"YANDEX_CLIENT_ID_FILE":     {target: &cfg.Auth.YandexClientID, dockerSecretName: "yandex_client_id"},
		"YANDEX_CLIENT_SECRET_FILE": {target: &cfg.Auth.YandexClientSecret, dockerSecretName: "yandex_client_secret"},
		"S3_ACCESS_KEY_FILE":        {target: &cfg.S3.AccessKey, dockerSecretName: "minio_root_user"},
		"S3_SECRET_KEY_FILE":        {target: &cfg.S3.SecretKey, dockerSecretName: "minio_root_password"},
		"KAFKA_SASL_USERNAME_FILE":  {target: &cfg.Kafka.SASLUsername, dockerSecretName: "kafka_sasl_username"},
		"KAFKA_SASL_PASSWORD_FILE":  {target: &cfg.Kafka.SASLPassword, dockerSecretName: "kafka_sasl_password"},
	}

	for envName, target := range secretTargets {
		if err := applySecretFile(envName, target); err != nil {
			return err
		}
	}

	return nil
}

type secretFileTarget struct {
	target           *string
	dockerSecretName string
}

func applySecretFile(envName string, secretTarget secretFileTarget) error {
	path := strings.TrimSpace(os.Getenv(envName))
	if path == "" {
		return nil
	}

	expectedPath := "/run/secrets/" + secretTarget.dockerSecretName
	if path != expectedPath {
		return fmt.Errorf("%s must point to %s", envName, expectedPath)
	}

	data, err := readDockerSecret(secretTarget.dockerSecretName)
	if err != nil {
		return fmt.Errorf("%s: %w", envName, err)
	}

	value := strings.TrimRight(string(data), "\r\n")
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("%s points to an empty secret file", envName)
	}

	*secretTarget.target = value

	return nil
}

func readDockerSecret(name string) ([]byte, error) {
	switch name {
	case "database_url":
		return os.ReadFile("/run/secrets/database_url")
	case "redis_password":
		return os.ReadFile("/run/secrets/redis_password")
	case "jwt_private_key":
		return os.ReadFile("/run/secrets/jwt_private_key")
	case "jwt_public_key":
		return os.ReadFile("/run/secrets/jwt_public_key")
	case "yandex_client_id":
		return os.ReadFile("/run/secrets/yandex_client_id")
	case "yandex_client_secret":
		return os.ReadFile("/run/secrets/yandex_client_secret")
	case "minio_root_user":
		return os.ReadFile("/run/secrets/minio_root_user")
	case "minio_root_password":
		return os.ReadFile("/run/secrets/minio_root_password")
	case "kafka_sasl_username":
		return os.ReadFile("/run/secrets/kafka_sasl_username")
	case "kafka_sasl_password":
		return os.ReadFile("/run/secrets/kafka_sasl_password")
	default:
		return nil, fmt.Errorf("unsupported docker secret %q", name)
	}
}

func validateKafka(cfg *Config) error {
	if len(cfg.Kafka.Brokers) == 0 {
		return fmt.Errorf("KAFKA_BROKERS cannot be empty")
	}

	if strings.TrimSpace(cfg.Kafka.TopicAnalysisRequest) == "" {
		return fmt.Errorf("KAFKA_TOPIC_ANALYSIS_REQUEST cannot be empty")
	}

	if strings.TrimSpace(cfg.Kafka.TopicAnalysisResult) == "" {
		return fmt.Errorf("KAFKA_TOPIC_ANALYSIS_RESULT cannot be empty")
	}

	if strings.TrimSpace(cfg.Kafka.ConsumerGroupID) == "" {
		return fmt.Errorf("KAFKA_CONSUMER_GROUP_ID cannot be empty")
	}

	if strings.TrimSpace(cfg.Kafka.TopicDLQ) == "" {
		return fmt.Errorf("KAFKA_TOPIC_DLQ cannot be empty")
	}

	if cfg.Kafka.MaxRetries < 1 {
		return fmt.Errorf("KAFKA_PRODUCER_MAX_RETRIES must be at least 1")
	}

	if cfg.Kafka.ConsumerMaxRetries < 0 {
		return fmt.Errorf("KAFKA_CONSUMER_MAX_RETRIES cannot be negative")
	}

	if cfg.Kafka.ConsumerRetryBackoffMin <= 0 || cfg.Kafka.ConsumerRetryBackoffMax <= 0 {
		return fmt.Errorf("KAFKA_CONSUMER_RETRY_BACKOFF_MIN/MAX must be greater than 0")
	}

	if cfg.Kafka.ConsumerRetryBackoffMin > cfg.Kafka.ConsumerRetryBackoffMax {
		return fmt.Errorf("KAFKA_CONSUMER_RETRY_BACKOFF_MIN cannot be greater than KAFKA_CONSUMER_RETRY_BACKOFF_MAX")
	}

	if cfg.Kafka.ConsumerFetchBackoffMin <= 0 || cfg.Kafka.ConsumerFetchBackoffMax <= 0 {
		return fmt.Errorf("KAFKA_CONSUMER_FETCH_BACKOFF_MIN/MAX must be greater than 0")
	}

	if cfg.Kafka.ConsumerFetchBackoffMin > cfg.Kafka.ConsumerFetchBackoffMax {
		return fmt.Errorf("KAFKA_CONSUMER_FETCH_BACKOFF_MIN cannot be greater than KAFKA_CONSUMER_FETCH_BACKOFF_MAX")
	}

	return nil
}
