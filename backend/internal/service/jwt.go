package service

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"strings"
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
	activeKeyID   string
	privateKey    crypto.Signer
	publicKeys    map[string]crypto.PublicKey
	issuer        string
	accessTTL     time.Duration
	refreshTTL    time.Duration
	oauthStateTTL time.Duration
}

type TokenManagerConfig struct {
	ActiveKeyID   string
	PrivateKeyPEM string
	PublicKeyPEM  string
	PublicKeysPEM string
	Issuer        string
	AccessTTL     time.Duration
	RefreshTTL    time.Duration
	OAuthStateTTL time.Duration
}

func NewTokenManager(cfg *TokenManagerConfig) (*TokenManager, error) {
	if cfg == nil {
		return nil, fmt.Errorf("token manager config is required")
	}

	activeKeyID := strings.TrimSpace(cfg.ActiveKeyID)
	if activeKeyID == "" {
		return nil, fmt.Errorf("jwt active key id is required")
	}

	privateKey, err := parseRSAPrivateKeyPEM(cfg.PrivateKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("parse jwt private key: %w", err)
	}

	publicKeys, err := parsePublicKeysPEM(activeKeyID, cfg.PublicKeyPEM, cfg.PublicKeysPEM)
	if err != nil {
		return nil, err
	}

	if _, ok := publicKeys[activeKeyID]; !ok {
		return nil, fmt.Errorf("jwt public keys do not contain active key id %q", activeKeyID)
	}

	issuer := strings.TrimSpace(cfg.Issuer)
	if issuer == "" {
		issuer = "autoinspect-api"
	}

	if cfg.AccessTTL <= 0 || cfg.RefreshTTL <= 0 {
		return nil, fmt.Errorf("token ttl must be positive")
	}

	oauthStateTTL := cfg.OAuthStateTTL
	if oauthStateTTL <= 0 {
		oauthStateTTL = 10 * time.Minute
	}

	return &TokenManager{
		activeKeyID:   activeKeyID,
		privateKey:    privateKey,
		publicKeys:    publicKeys,
		issuer:        issuer,
		accessTTL:     cfg.AccessTTL,
		refreshTTL:    cfg.RefreshTTL,
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

func (m *TokenManager) GenerateAccessToken(user *domain.User) (tokenString, jti string, expiresAt time.Time, err error) {
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

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = m.activeKeyID
	tokenString, err = token.SignedString(m.privateKey)

	if err != nil {
		return "", "", time.Time{}, fmt.Errorf("sign access token: %w", err)
	}

	return tokenString, jti, expiresAt, nil
}

func (m *TokenManager) ParseAccessToken(tokenString string) (*AccessClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &AccessClaims{}, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodRS256 {
			return nil, domain.ErrInvalidToken
		}

		kid, ok := token.Header["kid"].(string)
		if !ok || strings.TrimSpace(kid) == "" {
			return nil, domain.ErrInvalidToken
		}

		publicKey, ok := m.publicKeys[kid]
		if !ok {
			return nil, domain.ErrInvalidToken
		}

		return publicKey, nil
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

func parsePublicKeysPEM(activeKeyID, publicKeyPEM, publicKeysPEM string) (map[string]crypto.PublicKey, error) {
	publicKeys := make(map[string]crypto.PublicKey)

	if strings.TrimSpace(publicKeyPEM) != "" {
		publicKey, err := parseRSAPublicKeyPEM(publicKeyPEM)
		if err != nil {
			return nil, fmt.Errorf("parse jwt public key: %w", err)
		}
		publicKeys[activeKeyID] = publicKey
	}

	if strings.TrimSpace(publicKeysPEM) != "" {
		decoded := make(map[string]string)
		if err := json.Unmarshal([]byte(publicKeysPEM), &decoded); err != nil {
			return nil, fmt.Errorf("parse JWT_PUBLIC_KEYS json: %w", err)
		}
		for kid, pemValue := range decoded {
			kid = strings.TrimSpace(kid)
			if kid == "" {
				return nil, fmt.Errorf("JWT_PUBLIC_KEYS contains empty key id")
			}

			publicKey, err := parseRSAPublicKeyPEM(pemValue)
			if err != nil {
				return nil, fmt.Errorf("parse jwt public key %q: %w", kid, err)
			}
			publicKeys[kid] = publicKey
		}
	}

	if len(publicKeys) == 0 {
		return nil, fmt.Errorf("at least one jwt public key is required")
	}

	return publicKeys, nil
}

func parseRSAPrivateKeyPEM(value string) (*rsa.PrivateKey, error) {
	block, err := decodePEMBlock(value)
	if err != nil {
		return nil, err
	}

	switch block.Type {
	case "RSA PRIVATE KEY":
		return x509.ParsePKCS1PrivateKey(block.Bytes)
	case "PRIVATE KEY":
		key, parseErr := x509.ParsePKCS8PrivateKey(block.Bytes)
		if parseErr != nil {
			return nil, parseErr
		}
		privateKey, ok := key.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("private key is not RSA")
		}
		return privateKey, nil
	default:
		return nil, fmt.Errorf("unsupported private key PEM type %q", block.Type)
	}
}

func parseRSAPublicKeyPEM(value string) (*rsa.PublicKey, error) {
	block, err := decodePEMBlock(value)
	if err != nil {
		return nil, err
	}

	switch block.Type {
	case "RSA PUBLIC KEY":
		return x509.ParsePKCS1PublicKey(block.Bytes)
	case "PUBLIC KEY":
		key, parseErr := x509.ParsePKIXPublicKey(block.Bytes)
		if parseErr != nil {
			return nil, parseErr
		}
		publicKey, ok := key.(*rsa.PublicKey)
		if !ok {
			return nil, fmt.Errorf("public key is not RSA")
		}
		return publicKey, nil
	default:
		return nil, fmt.Errorf("unsupported public key PEM type %q", block.Type)
	}
}

func decodePEMBlock(value string) (*pem.Block, error) {
	normalized := strings.ReplaceAll(strings.TrimSpace(value), `\n`, "\n")
	block, _ := pem.Decode([]byte(normalized))
	if block == nil {
		return nil, fmt.Errorf("invalid PEM")
	}
	return block, nil
}
