package handlers

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/DedovInside/AutoInspect/backend/internal/api/dto"
	"github.com/DedovInside/AutoInspect/backend/internal/api/middleware"
	"github.com/DedovInside/AutoInspect/backend/internal/domain"
	"github.com/DedovInside/AutoInspect/backend/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type AuthHandler struct {
	auth *service.AuthService
}

func NewAuthHandler(auth *service.AuthService) *AuthHandler {
	return &AuthHandler{auth: auth}
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	var req dto.RefreshRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	result, err := h.auth.Refresh(c.Request.Context(), req.RefreshToken, optionalHeader(c.Request, "User-Agent"), clientIPFromGin(c))

	if err != nil {
		handleAuthError(c, err)
		return
	}

	resp := dto.AuthResponse{
		Tokens: dto.AuthTokensResponse{
			AccessToken:  result.Tokens.AccessToken,
			RefreshToken: result.Tokens.RefreshToken,
			TokenType:    result.Tokens.TokenType,
			ExpiresAt:    result.Tokens.ExpiresAt,
		},
		User: dto.ToUserResponse(&result.User),
	}
	writeJSON(c, http.StatusOK, resp)
}

func (h *AuthHandler) Logout(c *gin.Context) {
	var req dto.RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	accessJTI, accessExp := tokenMetaFromContext(c)

	if err := h.auth.Logout(c.Request.Context(), req.RefreshToken, accessJTI, accessExp); err != nil {
		handleAuthError(c, err)
		return
	}

	writeJSON(c, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *AuthHandler) Me(c *gin.Context) {
	userID, ok := userIDFromContext(c)

	if !ok {
		writeError(c, http.StatusUnauthorized, "unauthorized", "missing user context")
		return
	}

	user, err := h.auth.GetMe(c.Request.Context(), userID)

	if err != nil {
		handleAuthError(c, err)
		return
	}

	resp := dto.ToUserResponse(user)
	writeJSON(c, http.StatusOK, resp)
}

func (h *AuthHandler) YandexStart(c *gin.Context) {
	url, err := h.auth.StartYandexOAuth(c.Request.Context())

	if err != nil {
		handleAuthError(c, err)
		return
	}

	writeJSON(c, http.StatusOK, map[string]string{"auth_url": url})
}

func (h *AuthHandler) YandexCallback(c *gin.Context) {
	code := strings.TrimSpace(c.Query("code"))
	state := strings.TrimSpace(c.Query("state"))

	result, err := h.auth.ExchangeYandexCode(c.Request.Context(), code, state, optionalHeader(c.Request, "User-Agent"), clientIPFromGin(c))

	if err != nil {
		handleAuthError(c, err)
		return
	}

	resp := dto.AuthResponse{
		Tokens: dto.AuthTokensResponse{
			AccessToken:  result.Tokens.AccessToken,
			RefreshToken: result.Tokens.RefreshToken,
			TokenType:    result.Tokens.TokenType,
			ExpiresAt:    result.Tokens.ExpiresAt,
		},
		User: dto.ToUserResponse(&result.User),
	}
	writeJSON(c, http.StatusOK, resp)
}

func (h *AuthHandler) YandexExchange(c *gin.Context) {
	var req dto.OAuthYandexExchangeRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	result, err := h.auth.ExchangeYandexCode(c.Request.Context(), req.Code, req.State, optionalHeader(c.Request, "User-Agent"), clientIPFromGin(c))

	if err != nil {
		handleAuthError(c, err)
		return
	}

	resp := dto.AuthResponse{
		Tokens: dto.AuthTokensResponse{
			AccessToken:  result.Tokens.AccessToken,
			RefreshToken: result.Tokens.RefreshToken,
			TokenType:    result.Tokens.TokenType,
			ExpiresAt:    result.Tokens.ExpiresAt,
		},
		User: dto.ToUserResponse(&result.User),
	}
	writeJSON(c, http.StatusOK, resp)
}

func handleAuthError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidInput):
		writeError(c, http.StatusBadRequest, "invalid_input", err.Error())
	case errors.Is(err, domain.ErrAlreadyExists):
		writeError(c, http.StatusConflict, "already_exists", err.Error())
	case errors.Is(err, domain.ErrInvalidCredentials):
		writeError(c, http.StatusUnauthorized, "invalid_credentials", err.Error())
	case errors.Is(err, domain.ErrUnauthorized):
		writeError(c, http.StatusUnauthorized, "unauthorized", err.Error())
	case errors.Is(err, domain.ErrNotFound):
		writeError(c, http.StatusNotFound, "not_found", err.Error())
	default:
		writeError(c, http.StatusInternalServerError, "internal_error", "internal server error")
	}
}

type errorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func writeJSON(c *gin.Context, statusCode int, payload any) {
	c.PureJSON(statusCode, payload)
}

func writeError(c *gin.Context, statusCode int, code, message string) {
	writeJSON(c, statusCode, errorResponse{Code: code, Message: message})
}

func optionalHeader(r *http.Request, key string) *string {
	value := strings.TrimSpace(r.Header.Get(key))

	if value == "" {
		return nil
	}

	return &value
}

func clientIPFromGin(c *gin.Context) *string {
	ip := strings.TrimSpace(c.ClientIP())
	if ip == "" {
		return nil
	}

	return &ip
}

func userIDFromContext(c *gin.Context) (uuid.UUID, bool) {
	ctx := c.Request.Context()
	uid, ok := ctx.Value(middleware.UserIDContextKey).(uuid.UUID)
	if !ok {
		return uuid.Nil, false
	}
	return uid, true
}

func tokenMetaFromContext(c *gin.Context) (string, time.Time) {
	ctx := c.Request.Context()
	jti, _ := ctx.Value(middleware.AccessJTIContextKey).(string)
	exp, _ := ctx.Value(middleware.AccessExpContextKey).(time.Time)
	return jti, exp
}
