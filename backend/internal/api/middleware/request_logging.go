package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type requestContextKey string

const (
	RequestIDContextKey     requestContextKey = "request_id"
	CorrelationIDContextKey requestContextKey = "correlation_id"
)

const (
	headerRequestID     = "X-Request-ID"
	headerCorrelationID = "X-Correlation-ID"
)

func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := requestIDFromHeaders(c)
		correlationID := correlationIDFromHeaders(c, requestID)

		ctx := context.WithValue(c.Request.Context(), RequestIDContextKey, requestID)
		ctx = context.WithValue(ctx, CorrelationIDContextKey, correlationID)
		c.Request = c.Request.WithContext(ctx)

		c.Header(headerRequestID, requestID)
		c.Header(headerCorrelationID, correlationID)

		startedAt := time.Now()
		c.Next()

		path := routePath(c)
		if !shouldLogRequestPath(path) {
			return
		}

		attrs := []slog.Attr{
			slog.String("method", c.Request.Method),
			slog.String("path", path),
			slog.Int("status", c.Writer.Status()),
			slog.Int64("duration_ms", time.Since(startedAt).Milliseconds()),
			slog.String("client_ip", c.ClientIP()),
			slog.String("request_id", requestID),
			slog.String("correlation_id", correlationID),
		}
		if userID, ok := c.Request.Context().Value(UserIDContextKey).(uuid.UUID); ok && userID != uuid.Nil {
			attrs = append(attrs, slog.String("user_id", userID.String()))
		}

		level := slog.LevelInfo
		if c.Writer.Status() >= http.StatusInternalServerError {
			level = slog.LevelError
		} else if c.Writer.Status() >= http.StatusBadRequest {
			level = slog.LevelWarn
		}

		slog.LogAttrs(c.Request.Context(), level, "http request completed", attrs...)
	}
}

func requestIDFromHeaders(c *gin.Context) string {
	if value := firstNonEmptyHeader(c, headerRequestID, "X-Amzn-Trace-Id"); value != "" {
		return value
	}
	return uuid.NewString()
}

func correlationIDFromHeaders(c *gin.Context, fallback string) string {
	if value := firstNonEmptyHeader(c, headerCorrelationID, "X-Correlation-Id"); value != "" {
		return value
	}
	return fallback
}

func firstNonEmptyHeader(c *gin.Context, names ...string) string {
	for _, name := range names {
		if value := strings.TrimSpace(c.GetHeader(name)); value != "" {
			return value
		}
	}
	return ""
}

func routePath(c *gin.Context) string {
	if fullPath := c.FullPath(); fullPath != "" {
		return fullPath
	}
	return c.Request.URL.Path
}

func shouldLogRequestPath(path string) bool {
	return path != "/metrics" && !strings.HasPrefix(path, "/health")
}
