package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/DedovInside/AutoInspect/backend/internal/domain"
	"github.com/DedovInside/AutoInspect/backend/internal/repository"
	"github.com/google/uuid"
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

func (s *AuthService) StartYandexOAuth(ctx context.Context) (string, error) {
	if s.yandex == nil {
		return "", domain.ErrUnauthorized
	}
	if s.cache == nil {
		return "", errors.New("yandex oauth requires cache for state management")
	}

	state, err := s.tokens.GenerateOpaqueToken(24)
	if err != nil {
		return "", err
	}

	if err := s.cache.StoreOAuthState(ctx, state, s.tokens.OAuthStateTTL()); err != nil {
		return "", err
	}

	return s.yandex.AuthCodeURL(state), nil
}

func (s *AuthService) ExchangeYandexCode(ctx context.Context, req domain.OAuthYandexExchangeRequest, userAgent, ipAddress *string) (*AuthResult, error) {
	if s.yandex == nil {
		return nil, domain.ErrUnauthorized
	}

	if strings.TrimSpace(req.Code) == "" {
		return nil, domain.ErrInvalidInput
	}

	if s.cache == nil {
		return nil, errors.New("yandex oauth requires cache for state management")
	}

	ok, err := s.cache.ConsumeOAuthState(ctx, req.State)

	if err != nil {
		return nil, err
	}

	if !ok {
		return nil, domain.ErrUnauthorized
	}

	profile, err := s.yandex.ExchangeCodeAndFetchProfile(ctx, req.Code)

	if err != nil {
		return nil, err
	}

	identity, err := s.identities.GetByProviderSubject(ctx, domain.OAuthProviderYandex, profile.ID)

	if err == nil {
		user, getErr := s.users.GetByID(ctx, identity.UserID)

		if getErr != nil {
			return nil, getErr
		}

		return s.issueTokens(ctx, user, userAgent, ipAddress, uuid.Nil)
	} else if !errors.Is(err, domain.ErrNotFound) {
		return nil, err
	}

	if profile.DefaultEmail == "" {
		return nil, domain.ErrInvalidInput
	}

	user, err := s.users.GetByEmail(ctx, strings.ToLower(profile.DefaultEmail))
	if err != nil {
		if !errors.Is(err, domain.ErrNotFound) {
			return nil, err
		}

		username := profile.Login
		if username == "" {
			username = strings.Split(profile.DefaultEmail, "@")[0]
		}

		username = fmt.Sprintf("%s_%s", username, uuid.NewString()[:6])

		user = &domain.User{
			ID:            uuid.New(),
			Username:      username,
			Email:         strings.ToLower(profile.DefaultEmail),
			PasswordHash:  "",
			Role:          domain.RoleUser,
			EmailVerified: true,
			IsActive:      true,
		}

		if err := s.users.Create(ctx, user); err != nil {
			return nil, err
		}
	}

	identity = &domain.OAuthIdentity{
		ID:             uuid.New(),
		UserID:         user.ID,
		Provider:       domain.OAuthProviderYandex,
		ProviderUserID: profile.ID,
		Email:          &profile.DefaultEmail,
	}

	if err := s.identities.Create(ctx, identity); err != nil {
		return nil, err
	}

	return s.issueTokens(ctx, user, userAgent, ipAddress, uuid.Nil)
}

func (s *AuthService) Refresh(ctx context.Context, refreshToken string, userAgent, ipAddress *string) (*AuthResult, error) {
	if strings.TrimSpace(refreshToken) == "" {
		return nil, domain.ErrInvalidInput
	}

	tokenHash := HashToken(refreshToken)
	session, err := s.sessions.GetByTokenHash(ctx, tokenHash)

	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.ErrUnauthorized
		}
		return nil, err
	}

	now := time.Now().UTC()

	if session.RevokedAt != nil {
		_ = s.sessions.RevokeFamily(ctx, session.TokenFamilyID, "refresh token reuse detected")
		return nil, domain.ErrUnauthorized
	}

	if now.After(session.ExpiresAt) {
		_ = s.sessions.Revoke(ctx, session.ID, "refresh token expired", nil)
		return nil, domain.ErrUnauthorized
	}

	user, err := s.users.GetByID(ctx, session.UserID)

	if err != nil {
		return nil, err
	}

	if !user.IsActive {
		return nil, domain.ErrUnauthorized
	}

	newResult, err := s.issueTokens(ctx, user, userAgent, ipAddress, session.TokenFamilyID)

	if err != nil {
		return nil, err
	}

	newTokenHash := HashToken(newResult.Tokens.RefreshToken)
	newSession, err := s.sessions.GetByTokenHash(ctx, newTokenHash)

	if err != nil {
		return nil, err
	}

	if err := s.sessions.Revoke(ctx, session.ID, "refresh token rotated", &newSession.ID); err != nil {
		return nil, err
	}

	_ = s.sessions.TouchLastUsed(ctx, session.ID, now)

	return newResult, nil
}

func (s *AuthService) Logout(ctx context.Context, refreshToken, accessJTI string, accessExpiry time.Time) error {
	if strings.TrimSpace(refreshToken) != "" {
		tokenHash := HashToken(refreshToken)
		session, err := s.sessions.GetByTokenHash(ctx, tokenHash)

		if err == nil {
			_ = s.sessions.Revoke(ctx, session.ID, "user logout", nil)
		}
	}

	if strings.TrimSpace(accessJTI) != "" {
		ttl := time.Until(accessExpiry)
		if ttl > 0 {
			if err := s.cache.SetDenylistJTI(ctx, accessJTI, ttl); err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *AuthService) GetMe(ctx context.Context, userID uuid.UUID) (*domain.UserResponse, error) {
	user, err := s.users.GetByID(ctx, userID)

	if err != nil {
		return nil, err
	}

	resp := user.ToUserResponse()
	return &resp, nil
}

func (s *AuthService) issueTokens(ctx context.Context, user *domain.User, userAgent, ipAddress *string, familyID uuid.UUID) (*AuthResult, error) {
	accessToken, _, accessExp, err := s.tokens.GenerateAccessToken(user)

	if err != nil {
		return nil, err
	}

	refreshToken, err := s.tokens.GenerateOpaqueToken(48)

	if err != nil {
		return nil, err
	}

	if familyID == uuid.Nil {
		familyID = uuid.New()
	}

	session := &domain.AuthRefreshSession{
		ID:            uuid.New(),
		UserID:        user.ID,
		TokenHash:     HashToken(refreshToken),
		TokenFamilyID: familyID,
		UserAgent:     userAgent,
		IPAddress:     ipAddress,
		ExpiresAt:     time.Now().UTC().Add(s.tokens.RefreshTTL()),
	}

	if err := s.sessions.Create(ctx, session); err != nil {
		return nil, err
	}

	return &AuthResult{
		Tokens: domain.AuthTokens{
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
			TokenType:    "Bearer",
			ExpiresAt:    accessExp,
		},
		User: user.ToUserResponse(),
	}, nil
}
