package config

import (
	"time"
)

type Config struct {
	Environment string `env:"ENVIRONMENT" env-default:"development"`

	HTTP     HTTPConfig
	Database DatabaseConfig
	Redis    RedisConfig
	Auth     AuthConfig
	S3       S3Config
	Kafka    KafkaConfig
}

type HTTPConfig struct {
	Host               string        `env:"HTTP_HOST" env-default:"0.0.0.0"`
	Port               string        `env:"HTTP_PORT" env-default:"8080"`
	ReadTimeout        time.Duration `env:"HTTP_READ_TIMEOUT" env-default:"10s"`
	WriteTimeout       time.Duration `env:"HTTP_WRITE_TIMEOUT" env-default:"10s"`
	ShutdownTimeout    time.Duration `env:"HTTP_SHUTDOWN_TIMEOUT" env-default:"30s"`
	CORSAllowedOrigins []string      `env:"CORS_ALLOWED_ORIGINS" env-default:"http://localhost:5173,http://localhost:3000,http://localhost:8080"`
	WSAllowedOrigins   []string      `env:"WS_ALLOWED_ORIGINS" env-default:"http://localhost:5173,http://localhost:3000,http://localhost:8080"`
}

type DatabaseConfig struct {
	URL             string        `env:"DATABASE_URL"`
	MaxOpenConns    int           `env:"DB_MAX_OPEN_CONNS" env-default:"25"`
	MaxIdleConns    int           `env:"DB_MAX_IDLE_CONNS" env-default:"5"`
	ConnMaxLifetime time.Duration `env:"DB_CONN_MAX_LIFETIME" env-default:"5m"`
	ConnMaxIdleTime time.Duration `env:"DB_CONN_MAX_IDLE_TIME" env-default:"10m"`
}

type RedisConfig struct {
	Host     string `env:"REDIS_HOST" env-default:"localhost"`
	Port     string `env:"REDIS_PORT" env-default:"6379"`
	Password string `env:"REDIS_PASSWORD" env-default:""`
	DB       int    `env:"REDIS_DB" env-default:"0"`

	CacheName         string        `env:"REDIS_CACHE_NAME" env-default:"autoinspect:analysis:cache"`
	MaxRetries        int           `env:"REDIS_MAX_RETRIES" env-default:"3"`
	VisibilityTimeout time.Duration `env:"REDIS_VISIBILITY_TIMEOUT" env-default:"5m"`

	NotifyChannel string `env:"REDIS_NOTIFY_CHANNEL" env-default:"notify:analysis:job"`
}

type AuthConfig struct {
	JWTActiveKeyID  string        `env:"JWT_ACTIVE_KEY_ID" env-default:"local-dev"`
	JWTPrivateKey   string        `env:"JWT_PRIVATE_KEY"`
	JWTPublicKey    string        `env:"JWT_PUBLIC_KEY"`
	JWTPublicKeys   string        `env:"JWT_PUBLIC_KEYS"`
	JWTIssuer       string        `env:"JWT_ISSUER" env-default:"autoinspect-api"`
	AccessTokenTTL  time.Duration `env:"ACCESS_TOKEN_TTL" env-default:"15m"`
	RefreshTokenTTL time.Duration `env:"REFRESH_TOKEN_TTL" env-default:"168h"`
	OAuthStateTTL   time.Duration `env:"OAUTH_STATE_TTL" env-default:"10m"`

	BcryptCost int `env:"BCRYPT_COST" env-default:"10"`

	RateLimitPerMinute int `env:"RATE_LIMIT_PER_MINUTE" env-default:"60"`

	YandexClientID     string `env:"YANDEX_CLIENT_ID"`
	YandexClientSecret string `env:"YANDEX_CLIENT_SECRET"`
	YandexRedirectURL  string `env:"YANDEX_REDIRECT_URL" env-default:"http://localhost:8080/v1/auth/yandex/callback"`
}

type S3Config struct {
	Endpoint       string `env:"S3_ENDPOINT" env-default:"http://localhost:9000"`
	PublicEndpoint string `env:"S3_PUBLIC_ENDPOINT" env-default:""`
	AccessKey      string `env:"S3_ACCESS_KEY" env-default:"minioadmin"`
	SecretKey      string `env:"S3_SECRET_KEY" env-default:"minioadmin"`
	Region         string `env:"S3_REGION" env-default:"us-east-1"`
	UseSSL         bool   `env:"S3_USE_SSL" env-default:"false"`
	UsePathStyle   bool   `env:"S3_USE_PATH_STYLE" env-default:"true"`

	BucketUploads string `env:"S3_BUCKET_UPLOADS" env-default:"autoinspect-uploads"`
	BucketModels  string `env:"S3_BUCKET_MODELS" env-default:"autoinspect-models"`
	BucketResults string `env:"S3_BUCKET_RESULTS" env-default:"autoinspect-results"`

	PresignedURLTTL time.Duration `env:"S3_PRESIGNED_URL_TTL" env-default:"15m"`
}

type KafkaConfig struct {
	Brokers []string `env:"KAFKA_BROKERS" env-default:"localhost:9092"`

	TopicAnalysisRequest string `env:"KAFKA_TOPIC_ANALYSIS_REQUEST" env-default:"autoinspect.analysis.request"`
	TopicAnalysisResult  string `env:"KAFKA_TOPIC_ANALYSIS_RESULT" env-default:"autoinspect.analysis.result"`

	RequiredAcks string `env:"KAFKA_REQUIRED_ACKS" env-default:"all"`

	MaxRetries int `env:"KAFKA_PRODUCER_MAX_RETRIES" env-default:"3"`

	ConsumerGroupID         string        `env:"KAFKA_CONSUMER_GROUP_ID" env-default:"autoinspect-backend"`
	ConsumerMaxRetries      int           `env:"KAFKA_CONSUMER_MAX_RETRIES" env-default:"3"`
	ConsumerRetryBackoffMin time.Duration `env:"KAFKA_CONSUMER_RETRY_BACKOFF_MIN" env-default:"200ms"`
	ConsumerRetryBackoffMax time.Duration `env:"KAFKA_CONSUMER_RETRY_BACKOFF_MAX" env-default:"5s"`
	ConsumerFetchBackoffMin time.Duration `env:"KAFKA_CONSUMER_FETCH_BACKOFF_MIN" env-default:"200ms"`
	ConsumerFetchBackoffMax time.Duration `env:"KAFKA_CONSUMER_FETCH_BACKOFF_MAX" env-default:"5s"`
	TopicDLQ                string        `env:"KAFKA_TOPIC_DLQ" env-default:"autoinspect.analysis.dlq"`

	SecurityProtocol string `env:"KAFKA_SECURITY_PROTOCOL" env-default:"PLAINTEXT"`
	SASLUsername     string `env:"KAFKA_SASL_USERNAME" env-default:""`
	SASLPassword     string `env:"KAFKA_SASL_PASSWORD" env-default:""`
}
