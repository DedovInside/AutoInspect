package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/DedovInside/AutoInspect/backend/internal/broker"
	"github.com/DedovInside/AutoInspect/backend/internal/broker/kafka"
	rediscache "github.com/DedovInside/AutoInspect/backend/internal/cache/redis"
	"github.com/DedovInside/AutoInspect/backend/internal/config"
	"github.com/DedovInside/AutoInspect/backend/internal/notify"
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

	var s3Client *s3.Client

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
		nil,
		redisNotifier,
		&cfg.S3,
		&cfg.Kafka,
	)

	handler := func(ctx context.Context, msg broker.Message) error {
		return analysisService.HandleAnalysisResult(ctx, msg)
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
