package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/DedovInside/AutoInspect/backend/internal/domain"
	"github.com/google/uuid"
)

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

