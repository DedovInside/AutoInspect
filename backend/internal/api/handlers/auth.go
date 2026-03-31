package handlers

import (
	"errors"
	"net/http"
	"strings"
	"time"

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
	var req domain.RefreshRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	result, err := h.auth.Refresh(c.Request.Context(), req.RefreshToken, optionalHeader(c.Request, "User-Agent"), clientIPFromGin(c))

	if err != nil {
		handleAuthError(c, err)
		return
	}

	writeJSON(c, http.StatusOK, result)
}

func (h *AuthHandler) Logout(c *gin.Context) {
	var req domain.RefreshRequest
	_ = c.ShouldBindJSON(&req)

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

	resp, err := h.auth.GetMe(c.Request.Context(), userID)

	if err != nil {
		handleAuthError(c, err)
		return
	}

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
	req := domain.OAuthYandexExchangeRequest{
		Code:  strings.TrimSpace(c.Query("code")),
		State: strings.TrimSpace(c.Query("state")),
	}

	result, err := h.auth.ExchangeYandexCode(c.Request.Context(), req, optionalHeader(c.Request, "User-Agent"), clientIPFromGin(c))

	if err != nil {
		handleAuthError(c, err)
		return
	}

	writeJSON(c, http.StatusOK, result)
}

func (h *AuthHandler) YandexExchange(c *gin.Context) {
	var req domain.OAuthYandexExchangeRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	result, err := h.auth.ExchangeYandexCode(c.Request.Context(), req, optionalHeader(c.Request, "User-Agent"), clientIPFromGin(c))

	if err != nil {
		handleAuthError(c, err)
		return
	}

	writeJSON(c, http.StatusOK, result)
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
	value, ok := c.Get(string(middleware.UserIDContextKey))
	if !ok {
		return uuid.Nil, false
	}
	id, ok := value.(uuid.UUID)
	return id, ok
}

func tokenMetaFromContext(c *gin.Context) (string, time.Time) {
	jti, _ := c.Get(string(middleware.AccessJTIContextKey))
	exp, _ := c.Get(string(middleware.AccessExpContextKey))
	jtiValue, _ := jti.(string)
	expValue, _ := exp.(time.Time)

	return jtiValue, expValue
}
