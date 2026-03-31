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

	"github.com/DedovInside/AutoInspect/backend/internal/api"
	"github.com/DedovInside/AutoInspect/backend/internal/api/handlers"
	"github.com/DedovInside/AutoInspect/backend/internal/config"
	rediscache "github.com/DedovInside/AutoInspect/backend/internal/queue/redis"
	"github.com/DedovInside/AutoInspect/backend/internal/repository/postgres"
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
	ctx := context.Background()

	db, err := postgres.New(ctx, cfg.Database.URL, cfg.Database.MaxOpenConns, cfg.Database.ConnMaxLifetime)
	if err != nil {
		return fmt.Errorf("init postgres: %w", err)
	}
	defer db.Close()

	redisClient := rediscache.New(cfg.Redis)
	if err := redisClient.Ping(ctx); err != nil {
		return fmt.Errorf("init redis: %w", err)
	}
	defer func() {
		if closeErr := redisClient.Close(); closeErr != nil {
			log.Printf("warning: redis close failed: %v", closeErr)
		}
	}()

	sessionCache := redisClient

	userRepo := postgres.NewUserRepo(db)
	sessionRepo := postgres.NewAuthSessionRepo(db)
	oauthRepo := postgres.NewOAuthIdentityRepo(db)

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

	yandexClient := service.NewYandexOAuthClient(
		cfg.Auth.YandexClientID,
		cfg.Auth.YandexClientSecret,
		cfg.Auth.YandexRedirectURL,
	)
	if yandexClient == nil {
		return fmt.Errorf("init yandex oauth client: empty oauth config")
	}

	authService := service.NewAuthService(
		userRepo,
		sessionRepo,
		oauthRepo,
		tokenManager,
		sessionCache,
		yandexClient,
	)

	authHandler := handlers.NewAuthHandler(authService)
	router := api.NewGinRouter(authHandler, tokenManager, sessionCache)

	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%s", cfg.HTTP.Host, cfg.HTTP.Port),
		Handler:      router,
		ReadTimeout:  cfg.HTTP.ReadTimeout,
		WriteTimeout: cfg.HTTP.WriteTimeout,
	}

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
		log.Printf("shutdown signal: %s", sig.String())
	case serveErr := <-errCh:
		return fmt.Errorf("listen and serve: %w", serveErr)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.HTTP.ShutdownTimeout)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("graceful shutdown: %w", err)
	}

	time.Sleep(100 * time.Millisecond)
	return nil
}
