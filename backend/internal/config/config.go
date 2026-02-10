package config

import (
	"time"
)

type Config struct {
	Environment string `env:"ENVIRONMENT" env-default:"development"`

	HTTP     HTTPConfig
	Database DatabaseConfig
	Redis    RedisConfig
	MinIO    MinIOConfig
	GRPC     GRPCConfig
	Auth     AuthConfig
	Worker   WorkerConfig
	Logging  LoggingConfig
}

type HTTPConfig struct {
	Host            string        `env:"HTTP_HOST" env-default:"0.0.0.0"`
	Port            string        `env:"HTTP_PORT" env-default:"8080"`
	ReadTimeout     time.Duration `env:"HTTP_READ_TIMEOUT" env-default:"10s"`
	WriteTimeout    time.Duration `env:"HTTP_WRITE_TIMEOUT" env-default:"10s"`
	ShutdownTimeout time.Duration `env:"HTTP_SHUTDOWN_TIMEOUT" env-default:"30s"`
	CORS            CORSConfig
}

// CORSConfig настройки CORS
type CORSConfig struct {
	AllowedOrigins []string `env:"CORS_ALLOWED_ORIGINS" env-default:"http://localhost:3000" env-separator:","`
	AllowedMethods []string `env:"CORS_ALLOWED_METHODS" env-default:"GET,POST,PUT,DELETE,PATCH,OPTIONS" env-separator:","`
	AllowedHeaders []string `env:"CORS_ALLOWED_HEADERS" env-default:"Content-Type,Authorization" env-separator:","`
}

// DatabaseConfig настройки PostgreSQL
type DatabaseConfig struct {
	URL             string        `env:"DATABASE_URL" env-required:"true"`
	MaxOpenConns    int           `env:"DB_MAX_OPEN_CONNS" env-default:"25"`
	MaxIdleConns    int           `env:"DB_MAX_IDLE_CONNS" env-default:"5"`
	ConnMaxLifetime time.Duration `env:"DB_CONN_MAX_LIFETIME" env-default:"5m"`
	ConnMaxIdleTime time.Duration `env:"DB_CONN_MAX_IDLE_TIME" env-default:"10m"`
}

// RedisConfig настройки Redis
type RedisConfig struct {
	Host     string `env:"REDIS_HOST" env-default:"localhost"`
	Port     string `env:"REDIS_PORT" env-default:"6379"`
	Password string `env:"REDIS_PASSWORD" env-default:""`
	DB       int    `env:"REDIS_DB" env-default:"0"`

	// Для задач обработки
	QueueName         string        `env:"REDIS_QUEUE_NAME" env-default:"autoinspect:analysis:queue"`
	MaxRetries        int           `env:"REDIS_MAX_RETRIES" env-default:"3"`
	VisibilityTimeout time.Duration `env:"REDIS_VISIBILITY_TIMEOUT" env-default:"5m"`
}

// MinIOConfig настройки MinIO/S3
type MinIOConfig struct {
	Endpoint  string `env:"MINIO_ENDPOINT" env-default:"localhost:9000"`
	AccessKey string `env:"MINIO_ACCESS_KEY" env-default:"minioadmin"`
	SecretKey string `env:"MINIO_SECRET_KEY" env-default:"minioadmin"`
	UseSSL    bool   `env:"MINIO_USE_SSL" env-default:"false"`

	// Buckets
	ImagesBucket   string `env:"MINIO_IMAGES_BUCKET" env-default:"images"`
	ModelsBucket   string `env:"MINIO_MODELS_BUCKET" env-default:"models"`
	DatasetsBucket string `env:"MINIO_DATASETS_BUCKET" env-default:"datasets"`
	LogsBucket     string `env:"MINIO_LOGS_BUCKET" env-default:"logs"`
}

// GRPCConfig настройки gRPC клиента для ML сервиса
type GRPCConfig struct {
	MLServiceAddr string        `env:"GRPC_ML_SERVICE_ADDR" env-default:"localhost:50051"`
	Timeout       time.Duration `env:"GRPC_TIMEOUT" env-default:"30s"`
	MaxRetries    int           `env:"GRPC_MAX_RETRIES" env-default:"3"`
}

// Для настройки аутентификации
type AuthConfig struct {
	JWTSecret     string        `env:"JWT_SECRET" env-required:"true"`
	JWTExpiration time.Duration `env:"JWT_EXPIRATION" env-default:"24h"`

	// Для хеширования паролей
	BcryptCost int `env:"BCRYPT_COST" env-default:"10"`

	// Для ограничения количества запросов к API
	RateLimitPerMinute int `env:"RATE_LIMIT_PER_MINUTE" env-default:"60"`
}

// Для настройки воркера
type WorkerConfig struct {
	Concurrency       int           `env:"WORKER_CONCURRENCY" env-default:"5"`
	PollInterval      time.Duration `env:"WORKER_POLL_INTERVAL" env-default:"1s"`
	MaxJobDuration    time.Duration `env:"WORKER_MAX_JOB_DURATION" env-default:"10m"`
	HeartbeatInterval time.Duration `env:"WORKER_HEARTBEAT_INTERVAL" env-default:"30s"`
}

// Для настройки логирования
type LoggingConfig struct {
	Level  string `env:"LOG_LEVEL" env-default:"info"`  // debug, info, warn, error
	Format string `env:"LOG_FORMAT" env-default:"json"` // json, text
}
