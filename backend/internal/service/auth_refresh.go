package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/DedovInside/AutoInspect/backend/internal/domain"
	"github.com/DedovInside/AutoInspect/backend/internal/repository/postgres"
)

func (s *AuthService) Refresh(ctx context.Context, refreshToken string, userAgent, ipAddress *string) (*AuthResult, error) {
	if strings.TrimSpace(refreshToken) == "" {
		return nil, domain.ErrInvalidInput
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, domain.ErrInternal
	}

	defer func() {
		_ = tx.Rollback(ctx) // !
	}()

	sessionsTx := postgres.NewAuthSessionRepo(tx)
	usersTx := postgres.NewUserRepo(tx)

	tokenHash := HashToken(refreshToken)
	oldSession, err := sessionsTx.GetByTokenHash(ctx, tokenHash)

	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.ErrUnauthorized
		}
		return nil, err
	}

	if oldSession.RevokedAt != nil {
		_ = sessionsTx.RevokeFamily(ctx, oldSession.TokenFamilyID, "refresh token reuse detected")
		return nil, domain.ErrUnauthorized
	}

	if time.Now().UTC().After(oldSession.ExpiresAt) {
		_ = sessionsTx.Revoke(ctx, oldSession.ID, "refresh token expired", nil)
		return nil, domain.ErrUnauthorized
	}

	user, err := usersTx.GetByID(ctx, oldSession.UserID)
	if err != nil {
		return nil, err
	}

	if !user.IsActive {
		return nil, domain.ErrUnauthorized
	}

	ipAddr := stringToNetIPPtr(ipAddress)

	newResult, newSession, err := s.issueTokensWithRepos(ctx, user, userAgent, ipAddr, oldSession.TokenFamilyID, sessionsTx)

	if err != nil {
		return nil, err
	}

	if err := sessionsTx.Revoke(ctx, oldSession.ID, "refresh token rotated", &newSession.ID); err != nil {
		return nil, err
	}

	_ = sessionsTx.TouchLastUsed(ctx, oldSession.ID, time.Now().UTC())

	if err := tx.Commit(ctx); err != nil {
		return nil, domain.ErrInternal
	}

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
