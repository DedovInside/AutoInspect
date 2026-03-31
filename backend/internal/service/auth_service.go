package service

import (
	"context"
	"time"

	"github.com/DedovInside/AutoInspect/backend/internal/domain"
	"github.com/DedovInside/AutoInspect/backend/internal/repository"
)

type SessionCache interface {
	SetDenylistJTI(ctx context.Context, jti string, ttl time.Duration) error
	IsDenylistedJTI(ctx context.Context, jti string) (bool, error)
	StoreOAuthState(ctx context.Context, state string, ttl time.Duration) error
	ConsumeOAuthState(ctx context.Context, state string) (bool, error)
}

type AuthService struct {
	users      repository.UserRepository
	sessions   repository.AuthSessionRepository
	identities repository.OAuthIdentityRepository
	tokens     *TokenManager
	cache      SessionCache
	yandex     *YandexOAuthClient
}

type AuthResult struct {
	Tokens domain.AuthTokens   `json:"tokens"`
	User   domain.UserResponse `json:"user"`
}

func NewAuthService(
	users repository.UserRepository,
	sessions repository.AuthSessionRepository,
	identities repository.OAuthIdentityRepository,
	tokens *TokenManager,
	cache SessionCache,
	yandex *YandexOAuthClient,
) *AuthService {
	return &AuthService{
		users:      users,
		sessions:   sessions,
		identities: identities,
		tokens:     tokens,
		cache:      cache,
		yandex:     yandex,
	}
}

