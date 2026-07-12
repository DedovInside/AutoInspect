package middleware

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DedovInside/AutoInspect/backend/internal/domain"
	"github.com/DedovInside/AutoInspect/backend/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type fakeDenylistChecker struct {
	denylisted bool
	err        error
}

func (f fakeDenylistChecker) IsDenylistedJTI(context.Context, string) (bool, error) {
	return f.denylisted, f.err
}

func TestAuthMiddlewareAcceptsValidBearerToken(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	tokenManager := testTokenManager(t)
	user := &domain.User{ID: uuid.New(), Role: domain.RoleAdmin}
	token, _, _, err := tokenManager.GenerateAccessToken(user)
	require.NoError(t, err)

	router := gin.New()
	router.GET("/protected", Auth(tokenManager, nil), RequireRole(domain.RoleAdmin), func(c *gin.Context) {
		userID, ok := UserIDFromContext(c)
		require.True(t, ok)
		require.Equal(t, user.ID, userID)

		role, ok := UserRoleFromContext(c)
		require.True(t, ok)
		require.Equal(t, domain.RoleAdmin, role)

		c.Status(http.StatusNoContent)
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/protected", http.NoBody)
	request.Header.Set("Authorization", "Bearer "+token)

	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusNoContent, recorder.Code)
}

func TestAuthMiddlewareRejectsMissingInvalidAndRevokedTokens(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	tokenManager := testTokenManager(t)
	revokedToken, _, _, err := tokenManager.GenerateAccessToken(&domain.User{ID: uuid.New(), Role: domain.RoleUser})
	require.NoError(t, err)

	tests := []struct {
		name          string
		header        string
		cache         denylistChecker
		wantStatus    int
		wantErrorCode string
	}{
		{
			name:          "missing bearer",
			wantStatus:    http.StatusUnauthorized,
			wantErrorCode: "unauthorized",
		},
		{
			name:          "invalid token",
			header:        "Bearer nope",
			wantStatus:    http.StatusUnauthorized,
			wantErrorCode: "unauthorized",
		},
		{
			name:          "revoked token",
			header:        "Bearer " + revokedToken,
			cache:         fakeDenylistChecker{denylisted: true},
			wantStatus:    http.StatusUnauthorized,
			wantErrorCode: "unauthorized",
		},
		{
			name:          "cache error",
			header:        "Bearer " + revokedToken,
			cache:         fakeDenylistChecker{err: errors.New("redis unavailable")},
			wantStatus:    http.StatusUnauthorized,
			wantErrorCode: "unauthorized",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			router := gin.New()
			router.GET("/protected", Auth(tokenManager, tt.cache), func(c *gin.Context) {
				c.Status(http.StatusNoContent)
			})

			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/protected", http.NoBody)
			if tt.header != "" {
				request.Header.Set("Authorization", tt.header)
			}

			router.ServeHTTP(recorder, request)

			require.Equal(t, tt.wantStatus, recorder.Code)
			require.Contains(t, recorder.Body.String(), tt.wantErrorCode)
		})
	}
}

func TestRequireRoleRejectsInsufficientRole(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	tokenManager := testTokenManager(t)
	token, _, _, err := tokenManager.GenerateAccessToken(&domain.User{ID: uuid.New(), Role: domain.RoleUser})
	require.NoError(t, err)

	router := gin.New()
	router.GET("/admin", Auth(tokenManager, nil), RequireRole(domain.RoleAdmin), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/admin", http.NoBody)
	request.Header.Set("Authorization", "Bearer "+token)

	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusForbidden, recorder.Code)
	require.Contains(t, recorder.Body.String(), "forbidden")
}

func testTokenManager(t *testing.T) *service.TokenManager {
	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	require.NoError(t, err)
	publicKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyBytes,
	})

	tokenManager, err := service.NewTokenManager(&service.TokenManagerConfig{
		ActiveKeyID:   "test-key",
		PrivateKeyPEM: string(privateKeyPEM),
		PublicKeyPEM:  string(publicKeyPEM),
		Issuer:        "autoinspect-api",
		AccessTTL:     time.Hour,
		RefreshTTL:    24 * time.Hour,
		OAuthStateTTL: 10 * time.Minute,
	})
	require.NoError(t, err)

	return tokenManager
}
