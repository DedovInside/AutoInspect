package api

import (
	"net/http"

	"github.com/DedovInside/AutoInspect/backend/internal/api/handlers"
	"github.com/DedovInside/AutoInspect/backend/internal/api/middleware"
	"github.com/DedovInside/AutoInspect/backend/internal/service"
)

func NewRouter(authHandler *handlers.AuthHandler, tokenManager *service.TokenManager, cache service.SessionCache) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	mux.HandleFunc("POST /v1/auth/refresh", authHandler.Refresh)
	mux.HandleFunc("GET /v1/auth/yandex/start", authHandler.YandexStart)
	mux.HandleFunc("GET /v1/auth/yandex/callback", authHandler.YandexCallback)
	mux.HandleFunc("POST /v1/auth/oauth/yandex", authHandler.YandexExchange)

	authMW := middleware.Auth(tokenManager, cache)
	mux.Handle("GET /v1/auth/me", authMW(http.HandlerFunc(authHandler.Me)))
	mux.Handle("POST /v1/auth/logout", authMW(http.HandlerFunc(authHandler.Logout)))

	return mux
}
