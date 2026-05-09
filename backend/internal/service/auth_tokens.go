package service

import (
	"context"
	"net/netip"
	"time"

	"github.com/DedovInside/AutoInspect/backend/internal/domain"
	"github.com/DedovInside/AutoInspect/backend/internal/repository"
	"github.com/google/uuid"
)

func (s *AuthService) issueTokensWithRepos(ctx context.Context,
	user *domain.User, userAgent *string, ipAddress *netip.Addr, familyID uuid.UUID,
	sessionRepo repository.AuthSessionRepository) (*AuthResult, *domain.AuthSession, error) {
	accessToken, _, accessExp, err := s.tokens.GenerateAccessToken(user)
	if err != nil {
		return nil, nil, err
	}

	refreshToken, err := s.tokens.GenerateOpaqueToken(48)
	if err != nil {
		return nil, nil, err
	}

	if familyID == uuid.Nil {
		familyID = uuid.New()
	}

	now := time.Now().UTC()
	session := &domain.AuthSession{
		ID:            uuid.New(),
		UserID:        user.ID,
		TokenHash:     HashToken(refreshToken),
		TokenFamilyID: familyID,
		UserAgent:     userAgent,
		IPAddress:     ipAddress,
		ExpiresAt:     now.Add(s.tokens.RefreshTTL()),
		CreatedAt:     now,
	}

	if err := sessionRepo.Create(ctx, session); err != nil {
		return nil, nil, err
	}

	result := &AuthResult{
		Tokens: domain.AuthTokens{
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
			TokenType:    "Bearer",
			ExpiresAt:    accessExp,
		},
		User: *user,
	}

	return result, session, nil
}
