package handlers

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/DedovInside/AutoInspect/backend/internal/api/middleware"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

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

func userIDOrAbort(c *gin.Context) (uuid.UUID, bool) {
	userID, ok := middleware.UserIDFromContext(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "unauthorized", "missing user context")
		return uuid.Nil, false
	}

	return userID, true
}

func bindQueryOrAbort[T any](c *gin.Context) (T, bool) {
	var query T
	if err := c.ShouldBindQuery(&query); err != nil {
		writeError(c, http.StatusBadRequest, "invalid_query", err.Error())
		return query, false
	}
	return query, true
}

func bindJSONOrAbort[T any](c *gin.Context) (T, bool) {
	var req T
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return req, false
	}

	return req, true
}

func userIDAndQueryOrAbort[T any](c *gin.Context) (uuid.UUID, T, bool) {
	userID, ok := userIDOrAbort(c)
	if !ok {
		var zero T
		return uuid.Nil, zero, false
	}

	query, ok := bindQueryOrAbort[T](c)

	return userID, query, ok
}

func handleUserQueryList[TQuery any, TResult any](
	c *gin.Context,
	load func(context.Context, uuid.UUID, TQuery) (TResult, error),
	write func(*gin.Context, TResult, TQuery),
	handleErr func(*gin.Context, error),
) {
	userID, query, ok := userIDAndQueryOrAbort[TQuery](c)
	if !ok {
		return
	}

	result, err := load(c.Request.Context(), userID, query)
	if err != nil {
		handleErr(c, err)
		return
	}

	write(c, result, query)
}

func handleQueryList[TQuery any, TResult any](
	c *gin.Context,
	load func(context.Context, TQuery) (TResult, error),
	write func(*gin.Context, TResult, TQuery),
	handleErr func(*gin.Context, error),
) {
	query, ok := bindQueryOrAbort[TQuery](c)
	if !ok {
		return
	}

	result, err := load(c.Request.Context(), query)
	if err != nil {
		handleErr(c, err)
		return
	}

	write(c, result, query)
}

func tokenMetaFromContext(c *gin.Context) (string, time.Time) {
	ctx := c.Request.Context()
	jti, _ := ctx.Value(middleware.AccessJTIContextKey).(string)
	exp, _ := ctx.Value(middleware.AccessExpContextKey).(time.Time)

	return jti, exp
}
