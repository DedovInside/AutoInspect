package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/DedovInside/AutoInspect/backend/internal/api"
	"github.com/DedovInside/AutoInspect/backend/internal/api/handlers"
	"github.com/DedovInside/AutoInspect/backend/internal/broadcast"
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
		log.Printf("api failed: %v", err)
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

	redisCacheClient := rediscache.New(&cfg.Redis)
	defer func() {
		if closeErr := redisCacheClient.Close(); closeErr != nil {
			log.Printf("warning: redis close failed: %v", closeErr)
		}
	}()

	if err := redisCacheClient.Ping(initCtx); err != nil {
		return fmt.Errorf("init redis: %w", err)
	}

	userRepo := postgres.NewUserRepo(db)
	sessionRepo := postgres.NewAuthSessionRepo(db)
	oauthRepo := postgres.NewOAuthIdentityRepo(db)
	jobRepo := postgres.NewAnalysisJobRepo(db)
	modelRepo := postgres.NewCarModelRepo(db)
	modelTrainingRequestRepo := postgres.NewModelTrainingRequestRepo(db)
	carServiceApplicationRepo := postgres.NewCarServiceApplicationRepo(db)
	carServiceProfileRepo := postgres.NewCarServiceProfileRepo(db)
	carServiceImageRepo := postgres.NewCarServiceImageRepo(db)
	carServiceMatchRepo := postgres.NewCarServiceMatchRepo(db)
	repairRequestRepo := postgres.NewRepairRequestRepo(db)
	damageTypeRepo := postgres.NewDamageTypeRepo(db)
	partCategoryRepo := postgres.NewPartCategoryRepo(db)
	carServiceSpecializationRepo := postgres.NewCarServiceSpecializationRepo(db)

	tokenManager, err := service.NewTokenManager(
		cfg.Auth.JWTSecret,
		cfg.Auth.JWTIssuer,
		cfg.Auth.AccessTokenTTL,
		cfg.Auth.RefreshTokenTTL,
		cfg.Auth.OAuthStateTTL,
	)

	if err != nil {
		return fmt.Errorf("init token manager: %w", err)
	}

	yandexClient, err := service.NewYandexOAuthClient(
		cfg.Auth.YandexClientID,
		cfg.Auth.YandexClientSecret,
		cfg.Auth.YandexRedirectURL,
		10*time.Second,
	)

	if err != nil {
		return fmt.Errorf("init yandex oauth client: %w", err)
	}

	authService := service.NewAuthService(db, userRepo, sessionRepo, oauthRepo, tokenManager, redisCacheClient, yandexClient)

	s3Client, err := s3.New(context.Background(), &cfg.S3)

	if err != nil {
		return fmt.Errorf("init s3 client: %w", err)
	}

	kafkaProducer, err := kafka.NewProducer(&cfg.Kafka)

	if err != nil {
		return fmt.Errorf("init kafka producer: %w", err)
	}

	defer func() {
		if closeErr := kafkaProducer.Close(); closeErr != nil {
			log.Printf("warning: kafka producer close failed: %v", closeErr)
		}
	}()

	broadcastMgr := broadcast.NewManager()
	defer broadcastMgr.Close()

	rawRedisClient := redisCacheClient.Raw()
	redisNotifier, err := notify.NewRedisNotifier(rawRedisClient, cfg.Redis.NotifyChannel)
	if err != nil {
		return fmt.Errorf("init redis notifier: %w", err)
	}

	analysisService := service.NewAnalysisService(
		s3Client,
		jobRepo,
		modelRepo,
		damageTypeRepo,
		kafkaProducer,
		redisNotifier,
		&cfg.S3,
		&cfg.Kafka,
	)
	carServiceMatchingService := service.NewCarServiceMatchingService(
		jobRepo,
		carServiceMatchRepo,
		carServiceImageRepo,
		s3Client,
		&cfg.S3,
	)

	modelService := service.NewModelService(modelRepo, s3Client, &cfg.S3)

	modelTrainingRequestService := service.NewModelTrainingRequestService(modelTrainingRequestRepo, modelRepo)

	carServiceApplicationService := service.NewCarServiceApplicationService(
		db,
		carServiceApplicationRepo,
		carServiceProfileRepo,
		userRepo,
	)

	carServiceProfileService := service.NewCarServiceProfileService(
		db,
		carServiceProfileRepo,
		carServiceImageRepo,
		damageTypeRepo,
		partCategoryRepo,
		carServiceSpecializationRepo,
		s3Client,
		&cfg.S3,
	)

	repairRequestService := service.NewRepairRequestService(
		repairRequestRepo,
		jobRepo,
		carServiceProfileRepo,
	)

	authHandler := handlers.NewAuthHandler(authService)
	modelHandler := handlers.NewModelHandler(modelService)
	modelTrainingRequestHandler := handlers.NewModelTrainingRequestHandler(modelTrainingRequestService)
	carServiceApplicationHandler := handlers.NewCarServiceApplicationHandler(carServiceApplicationService)
	carServiceProfileHandler := handlers.NewCarServiceProfileHandler(carServiceProfileService)
	repairRequestHandler := handlers.NewRepairRequestHandler(repairRequestService)

	analysisHandler := handlers.NewAnalysisHandler(
		analysisService,
		carServiceMatchingService,
		broadcastMgr,
		10,
		100,
		[]string{"image/jpeg", "image/png", "image/webp"},
		cfg.HTTP.WSAllowedOrigins,
	)

	stopSubscriber := startRedisSubscriber(redisNotifier, broadcastMgr)
	defer stopSubscriber()

	router := api.NewGinRouter(
		authHandler,
		analysisHandler,
		modelHandler,
		modelTrainingRequestHandler,
		carServiceApplicationHandler,
		carServiceProfileHandler,
		repairRequestHandler,
		tokenManager,
		redisCacheClient,
	)
	server := newHTTPServer(cfg, router)
	defer func() {
		if closeErr := server.Close(); closeErr != nil && !errors.Is(closeErr, http.ErrServerClosed) {
			log.Printf("warning: http server close failed: %v", closeErr)
		}
	}()

	return serveHTTP(server, cfg.HTTP.ShutdownTimeout)
}

func startRedisSubscriber(redisNotifier *notify.RedisNotifier, broadcastMgr *broadcast.Manager) func() {
	subscribeCtx, stopSubscriber := context.WithCancel(context.Background())
	var subscriberWG sync.WaitGroup
	subscriberWG.Add(1)

	go func() {
		defer subscriberWG.Done()
		handler := func(ctx context.Context, event notify.JobEvent) {
			_ = broadcastMgr.NotifyJobEvent(ctx, &event)
		}
		if err := redisNotifier.Subscribe(subscribeCtx, handler); err != nil {
			log.Printf("Redis notifier subscriber stopped: %v", err)
		}
	}()

	return func() {
		stopSubscriber()
		subscriberWG.Wait()
	}
}

func newHTTPServer(cfg *config.Config, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:           fmt.Sprintf("%s:%s", cfg.HTTP.Host, cfg.HTTP.Port),
		Handler:        handler,
		ReadTimeout:    cfg.HTTP.ReadTimeout,
		WriteTimeout:   cfg.HTTP.WriteTimeout,
		MaxHeaderBytes: 1 << 20,
	}
}

func serveHTTP(server *http.Server, shutdownTimeout time.Duration) error {
	errCh := make(chan error, 1)
	go func() {
		log.Printf("api listening on %s", server.Addr)
		if serveErr := server.ListenAndServe(); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			errCh <- serveErr
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigCh:
		log.Printf("shutdown signal received: %s", sig.String())
	case serveErr := <-errCh:
		return fmt.Errorf("listen and serve: %w", serveErr)
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("graceful shutdown: %w", err)
	}

	return nil
}
