package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/DedovInside/AutoInspect/backend/internal/domain"
	"github.com/DedovInside/AutoInspect/backend/internal/repository/postgres"
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

func (s *AuthService) ExchangeYandexCode(ctx context.Context, code, state string, userAgent, ipAddress *string) (*AuthResult, error) {
	if s.yandex == nil {
		return nil, domain.ErrUnauthorized
	}

	if strings.TrimSpace(code) == "" {
		return nil, domain.ErrInvalidInput
	}

	if s.cache == nil {
		return nil, errors.New("yandex oauth requires cache for state management")
	}

	ok, err := s.cache.ConsumeOAuthState(ctx, state)

	if err != nil {
		return nil, err
	}

	if !ok {
		return nil, domain.ErrUnauthorized
	}

	profile, err := s.yandex.ExchangeCodeAndFetchProfile(ctx, code)

	if err != nil {
		return nil, err
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, domain.ErrInternal
	}
	defer func() {
		_ = tx.Rollback(ctx) // !
	}()

	usersTx := postgres.NewUserRepo(tx)
	identitiesTx := postgres.NewOAuthIdentityRepo(tx)
	sessionsTx := postgres.NewAuthSessionRepo(tx)

	identity, err := identitiesTx.GetByProviderSubject(ctx, domain.OAuthProviderYandex, profile.ID)

	if err == nil {
		user, err := usersTx.GetByID(ctx, identity.UserID)

		if err != nil {
			return nil, err
		}
		if !user.IsActive {
			return nil, domain.ErrUnauthorized
		}

		ipAddr := stringToNetIPPtr(ipAddress)

		result, _, err := s.issueTokensWithRepos(ctx, user, userAgent, ipAddr, uuid.Nil, sessionsTx)

		if err != nil {
			return nil, err
		}

		if err := tx.Commit(ctx); err != nil {
			return nil, domain.ErrInternal
		}
		return result, nil

	} else if !errors.Is(err, domain.ErrNotFound) {
		return nil, err
	}

	if profile.DefaultEmail == "" {
		return nil, domain.ErrInvalidInput
	}

	user, err := usersTx.GetByEmail(ctx, strings.ToLower(profile.DefaultEmail))

	if err != nil {
		if !errors.Is(err, domain.ErrNotFound) {
			return nil, err
		}

		now := time.Now().UTC()
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
			CreatedAt:     now,
			UpdatedAt:     now,
		}

		if err := usersTx.Create(ctx, user); err != nil {
			return nil, err
		}
	} else if !user.IsActive {
		return nil, domain.ErrUnauthorized
	}

	identity = &domain.OAuthIdentity{
		ID:             uuid.New(),
		UserID:         user.ID,
		Provider:       domain.OAuthProviderYandex,
		ProviderUserID: profile.ID,
		Email:          &profile.DefaultEmail,
		CreatedAt:      time.Now().UTC(),
	}

	if err := identitiesTx.Create(ctx, identity); err != nil {
		return nil, err
	}

	ipAddr := stringToNetIPPtr(ipAddress)
	result, _, err := s.issueTokensWithRepos(ctx, user, userAgent, ipAddr, uuid.Nil, sessionsTx)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, domain.ErrInternal
	}
	return result, nil
}
