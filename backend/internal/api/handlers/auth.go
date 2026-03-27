package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/DedovInside/AutoInspect/backend/internal/api/middleware"
	"github.com/DedovInside/AutoInspect/backend/internal/domain"
	"github.com/DedovInside/AutoInspect/backend/internal/service"
	"github.com/google/uuid"
)

type AuthHandler struct {
	auth *service.AuthService
}

func NewAuthHandler(auth *service.AuthService) *AuthHandler {
	return &AuthHandler{auth: auth}
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req domain.RefreshRequest

	if err := decodeJSONBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	result, err := h.auth.Refresh(r.Context(), req.RefreshToken, optionalHeader(r, "User-Agent"), clientIP(r))

	if err != nil {
		handleAuthError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	var req domain.RefreshRequest
	_ = decodeJSONBody(r, &req)

	accessJTI, accessExp := tokenMetaFromContext(r.Context())

	if err := h.auth.Logout(r.Context(), req.RefreshToken, accessJTI, accessExp); err != nil {
		handleAuthError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(r.Context())

	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "missing user context")
		return
	}

	resp, err := h.auth.GetMe(r.Context(), userID)

	if err != nil {
		handleAuthError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *AuthHandler) YandexStart(w http.ResponseWriter, r *http.Request) {
	url, err := h.auth.StartYandexOAuth(r.Context())

	if err != nil {
		handleAuthError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"auth_url": url})
}

func (h *AuthHandler) YandexCallback(w http.ResponseWriter, r *http.Request) {
	req := domain.OAuthYandexExchangeRequest{
		Code:  strings.TrimSpace(r.URL.Query().Get("code")),
		State: strings.TrimSpace(r.URL.Query().Get("state")),
	}

	result, err := h.auth.ExchangeYandexCode(r.Context(), req, optionalHeader(r, "User-Agent"), clientIP(r))

	if err != nil {
		handleAuthError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (h *AuthHandler) YandexExchange(w http.ResponseWriter, r *http.Request) {
	var req domain.OAuthYandexExchangeRequest

	if err := decodeJSONBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	result, err := h.auth.ExchangeYandexCode(r.Context(), req, optionalHeader(r, "User-Agent"), clientIP(r))

	if err != nil {
		handleAuthError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func handleAuthError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidInput):
		writeError(w, http.StatusBadRequest, "invalid_input", err.Error())
	case errors.Is(err, domain.ErrAlreadyExists):
		writeError(w, http.StatusConflict, "already_exists", err.Error())
	case errors.Is(err, domain.ErrInvalidCredentials):
		writeError(w, http.StatusUnauthorized, "invalid_credentials", err.Error())
	case errors.Is(err, domain.ErrUnauthorized):
		writeError(w, http.StatusUnauthorized, "unauthorized", err.Error())
	case errors.Is(err, domain.ErrNotFound):
		writeError(w, http.StatusNotFound, "not_found", err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
	}
}

type errorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func decodeJSONBody(r *http.Request, out any) error {
	if r.Body == nil {
		return errors.New("request body is empty")
	}

	defer func() { _ = r.Body.Close() }() // !

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(out); err != nil {

		if errors.Is(err, io.EOF) {
			return errors.New("request body is empty")
		}
		return err
	}

	return nil
}

func writeJSON(w http.ResponseWriter, statusCode int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	_ = encoder.Encode(payload)
}

func writeError(w http.ResponseWriter, statusCode int, code, message string) {
	writeJSON(w, statusCode, errorResponse{Code: code, Message: message})
}

func optionalHeader(r *http.Request, key string) *string {
	value := strings.TrimSpace(r.Header.Get(key))

	if value == "" {
		return nil
	}

	return &value
}

func clientIP(r *http.Request) *string {
	for _, header := range []string{"X-Forwarded-For", "X-Real-IP"} {
		value := strings.TrimSpace(r.Header.Get(header))

		if value != "" {
			parts := strings.Split(value, ",")
			v := strings.TrimSpace(parts[0])
			return &v
		}
	}
	return nil
}

func userIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	value := ctx.Value(middleware.UserIDContextKey)
	id, ok := value.(uuid.UUID)
	return id, ok
}

func tokenMetaFromContext(ctx context.Context) (string, time.Time) {
	jti, _ := ctx.Value(middleware.AccessJTIContextKey).(string)
	exp, _ := ctx.Value(middleware.AccessExpContextKey).(time.Time)
	return jti, exp
}
