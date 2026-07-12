package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/DedovInside/AutoInspect/backend/internal/broker"
	"github.com/DedovInside/AutoInspect/backend/internal/broker/kafka"
	rediscache "github.com/DedovInside/AutoInspect/backend/internal/cache/redis"
	"github.com/DedovInside/AutoInspect/backend/internal/config"
	"github.com/DedovInside/AutoInspect/backend/internal/notify"
	"github.com/DedovInside/AutoInspect/backend/internal/observability"
	"github.com/DedovInside/AutoInspect/backend/internal/repository/postgres"
	"github.com/DedovInside/AutoInspect/backend/internal/repository/s3"
	"github.com/DedovInside/AutoInspect/backend/internal/service"
)

func main() {
	if err := run(); err != nil {
		log.Printf("worker failed: %v", err)
		os.Exit(1)
	}
}

func run() error {
	cfg := config.MustLoad()
	initCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db, err := postgres.New(initCtx, cfg.Database.URL, cfg.Database.MaxOpenConns, cfg.Database.ConnMaxLifetime)

	if err != nil {
		return fmt.Errorf("init postgres: %w", err)
	}

	defer db.Close()

	jobRepo := postgres.NewAnalysisJobRepo(db)
	modelRepo := postgres.NewCarModelRepo(db)
	damageTypeRepo := postgres.NewDamageTypeRepo(db)

	s3Client, err := s3.New(context.Background(), &cfg.S3)
	if err != nil {
		return fmt.Errorf("init s3 client: %w", err)
	}

	redisCacheClient := rediscache.New(&cfg.Redis)
	defer func() {
		if closeErr := redisCacheClient.Close(); closeErr != nil {
			log.Printf("warning: redis close failed: %v", closeErr)
		}
	}()
	if err := redisCacheClient.Ping(initCtx); err != nil {
		return fmt.Errorf("init redis: %w", err)
	}

	rawRedisClient := redisCacheClient.Raw()
	redisNotifier, err := notify.NewRedisNotifier(rawRedisClient, cfg.Redis.NotifyChannel)
	if err != nil {
		return fmt.Errorf("init redis notifier: %w", err)
	}

	kafkaConsumer, err := kafka.NewConsumer(&cfg.Kafka)
	if err != nil {
		return fmt.Errorf("init kafka consumer: %w", err)
	}

	defer func() {
		if closeErr := kafkaConsumer.Close(); closeErr != nil {
			log.Printf("warning: kafka consumer close failed: %v", closeErr)
		}
	}()

	analysisService := service.NewAnalysisService(
		s3Client,
		jobRepo,
		modelRepo,
		damageTypeRepo,
		nil,
		redisNotifier,
		&cfg.S3,
		&cfg.Kafka,
	)

	stopObservability := startWorkerObservabilityServer(cfg, db, redisCacheClient, s3Client)
	defer stopObservability()

	handler := func(ctx context.Context, msg broker.Message) error {
		startedAt := time.Now()
		err := analysisService.HandleAnalysisResult(ctx, msg)
		observability.ObserveKafkaConsumed(msg.Topic, err, startedAt)
		return err
	}

	log.Printf("worker started, listening on topic: %s", cfg.Kafka.TopicAnalysisResult)

	consumerCtx, stopConsumer := context.WithCancel(context.Background())
	defer stopConsumer()

	errCh := make(chan error, 1)
	go func() {
		if subErr := kafkaConsumer.Subscribe(consumerCtx, cfg.Kafka.TopicAnalysisResult, handler); subErr != nil {
			errCh <- fmt.Errorf("kafka consumer error: %w", subErr)
			return
		}
		errCh <- nil
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigCh:
		log.Printf("shutdown signal received: %s", sig.String())
	case consumerErr := <-errCh:
		if consumerErr == nil {
			return nil
		}
		return consumerErr
	}

	stopConsumer()
	select {
	case consumerErr := <-errCh:
		if consumerErr != nil {
			return consumerErr
		}
	case <-time.After(cfg.HTTP.ShutdownTimeout):
		return fmt.Errorf("worker shutdown timed out")
	}

	log.Println("worker shutting down...")
	return nil
}

func startWorkerObservabilityServer(
	cfg *config.Config,
	db *postgres.DB,
	redisClient *rediscache.Client,
	s3Client *s3.Client,
) func() {
	healthChecker := observability.NewHealthChecker(cfg.Observe.WorkerServiceName, map[string]observability.DependencyCheck{
		"postgres": db.Ping,
		"redis":    redisClient.Ping,
		"s3":       s3Client.HealthCheck,
		"kafka": func(ctx context.Context) error {
			return observability.CheckKafkaBrokers(ctx, cfg.Kafka.Brokers)
		},
	})

	mux := http.NewServeMux()
	mux.Handle("/metrics", observability.MetricsHandlerHTTP())
	mux.HandleFunc("/health", healthChecker.SummaryHTTP)
	mux.HandleFunc("/health/live", healthChecker.LiveHTTP)
	mux.HandleFunc("/health/ready", healthChecker.ReadyHTTP)

	server := &http.Server{
		Addr:              fmt.Sprintf("%s:%s", cfg.Observe.WorkerHTTPHost, cfg.Observe.WorkerHTTPPort),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("worker observability listening on %s", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("worker observability server failed: %v", err)
		}
	}()

	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), cfg.HTTP.ShutdownTimeout)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			log.Printf("warning: worker observability shutdown failed: %v", err)
		}
	}
}
