package service

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/DedovInside/AutoInspect/backend/internal/domain"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type AccessClaims struct {
	UserID string      `json:"uid"`
	Role   domain.Role `json:"role"`
	jwt.RegisteredClaims
}

type TokenManager struct {
	secret        []byte
	issuer        string
	accessTTL     time.Duration
	refreshTTL    time.Duration
	oauthStateTTL time.Duration
}

func NewTokenManager(secret, issuer string, accessTTL, refreshTTL, oauthStateTTL time.Duration) (*TokenManager, error) {

	if len(secret) < 16 {
		return nil, fmt.Errorf("jwt secret is too short")
	}

	if issuer == "" {
		issuer = "autoinspect-api"
	}

	if accessTTL <= 0 || refreshTTL <= 0 {
		return nil, fmt.Errorf("token ttl must be positive")
	}

	if oauthStateTTL <= 0 {
		oauthStateTTL = 10 * time.Minute
	}

	return &TokenManager{
		secret:        []byte(secret),
		issuer:        issuer,
		accessTTL:     accessTTL,
		refreshTTL:    refreshTTL,
		oauthStateTTL: oauthStateTTL,
	}, nil
}

func (m *TokenManager) AccessTTL() time.Duration {
	return m.accessTTL
}

func (m *TokenManager) RefreshTTL() time.Duration {
	return m.refreshTTL
}

func (m *TokenManager) OAuthStateTTL() time.Duration {
	return m.oauthStateTTL
}

func (m *TokenManager) GenerateAccessToken(user *domain.User) (tokenString string, jti string, expiresAt time.Time, err error) {
	now := time.Now().UTC()
	expiresAt = now.Add(m.accessTTL)
	jti = uuid.NewString()

	claims := AccessClaims{
		UserID: user.ID.String(),
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.issuer,
			Subject:   user.ID.String(),
			Audience:  []string{"autoinspect-web"},
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ID:        jti,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err = token.SignedString(m.secret)

	if err != nil {
		return "", "", time.Time{}, fmt.Errorf("sign access token: %w", err)
	}

	return tokenString, jti, expiresAt, nil
}

func (m *TokenManager) ParseAccessToken(tokenString string) (*AccessClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &AccessClaims{}, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, domain.ErrInvalidToken
		}
		return m.secret, nil
	})

	if err != nil {
		return nil, fmt.Errorf("parse access token: %w", domain.ErrInvalidToken)
	}

	claims, ok := token.Claims.(*AccessClaims)

	if !ok || !token.Valid {
		return nil, domain.ErrInvalidToken
	}

	return claims, nil
}

func (m *TokenManager) GenerateOpaqueToken(size int) (string, error) {
	if size <= 0 {
		size = 32
	}

	buf := make([]byte, size)

	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate random token: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func HashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}
