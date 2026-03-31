package service

import (
	"context"
	"time"

	"github.com/DedovInside/AutoInspect/backend/internal/domain"
	"github.com/google/uuid"
)

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

