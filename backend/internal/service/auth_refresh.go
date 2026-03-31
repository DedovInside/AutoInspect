package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/DedovInside/AutoInspect/backend/internal/domain"
)

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


