package service

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"testing"
	"time"

	"github.com/DedovInside/AutoInspect/backend/internal/domain"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type testRSAKeyPair struct {
	privatePEM string
	publicPEM  string
}

func TestTokenManagerSignsAccessTokenWithRS256AndKid(t *testing.T) {
	t.Parallel()

	keyPair := generateTestRSAKeyPair(t)
	tokenManager, err := NewTokenManager(&TokenManagerConfig{
		ActiveKeyID:   "test-key",
		PrivateKeyPEM: keyPair.privatePEM,
		PublicKeyPEM:  keyPair.publicPEM,
		Issuer:        "autoinspect-api",
		AccessTTL:     time.Hour,
		RefreshTTL:    24 * time.Hour,
		OAuthStateTTL: 10 * time.Minute,
	})
	require.NoError(t, err)

	user := &domain.User{ID: uuid.New(), Role: domain.RoleUser}
	tokenString, _, _, err := tokenManager.GenerateAccessToken(user)
	require.NoError(t, err)

	token, _, err := jwt.NewParser().ParseUnverified(tokenString, jwt.MapClaims{})
	require.NoError(t, err)
	require.Equal(t, jwt.SigningMethodRS256.Alg(), token.Method.Alg())
	require.Equal(t, "test-key", token.Header["kid"])

	claims, err := tokenManager.ParseAccessToken(tokenString)
	require.NoError(t, err)
	require.Equal(t, user.ID.String(), claims.UserID)
	require.Equal(t, domain.RoleUser, claims.Role)
}

func TestTokenManagerAcceptsPreviousPublicKeyDuringRotation(t *testing.T) {
	t.Parallel()

	oldKeyPair := generateTestRSAKeyPair(t)
	newKeyPair := generateTestRSAKeyPair(t)
	publicKeysJSON := marshalPublicKeys(t, map[string]string{
		"2026-07-old": oldKeyPair.publicPEM,
		"2026-08-new": newKeyPair.publicPEM,
	})

	oldTokenManager, err := NewTokenManager(&TokenManagerConfig{
		ActiveKeyID:   "2026-07-old",
		PrivateKeyPEM: oldKeyPair.privatePEM,
		PublicKeysPEM: publicKeysJSON,
		Issuer:        "autoinspect-api",
		AccessTTL:     time.Hour,
		RefreshTTL:    24 * time.Hour,
		OAuthStateTTL: 10 * time.Minute,
	})
	require.NoError(t, err)

	newTokenManager, err := NewTokenManager(&TokenManagerConfig{
		ActiveKeyID:   "2026-08-new",
		PrivateKeyPEM: newKeyPair.privatePEM,
		PublicKeysPEM: publicKeysJSON,
		Issuer:        "autoinspect-api",
		AccessTTL:     time.Hour,
		RefreshTTL:    24 * time.Hour,
		OAuthStateTTL: 10 * time.Minute,
	})
	require.NoError(t, err)

	oldToken, _, _, err := oldTokenManager.GenerateAccessToken(&domain.User{ID: uuid.New(), Role: domain.RoleAdmin})
	require.NoError(t, err)

	claims, err := newTokenManager.ParseAccessToken(oldToken)
	require.NoError(t, err)
	require.Equal(t, domain.RoleAdmin, claims.Role)
}

func generateTestRSAKeyPair(t *testing.T) testRSAKeyPair {
	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	privatePEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	require.NoError(t, err)
	publicPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyBytes,
	})

	return testRSAKeyPair{
		privatePEM: string(privatePEM),
		publicPEM:  string(publicPEM),
	}
}

func marshalPublicKeys(t *testing.T, keys map[string]string) string {
	t.Helper()

	data, err := json.Marshal(keys)
	require.NoError(t, err)
	return string(data)
}
