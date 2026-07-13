package middleware

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestRequestLoggerAddsRequestHeadersAndLogsRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var logs bytes.Buffer
	previousLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&logs, nil)))
	t.Cleanup(func() {
		slog.SetDefault(previousLogger)
	})

	userID := uuid.New()
	router := gin.New()
	router.Use(RequestLogger())
	router.GET("/items/:id", func(c *gin.Context) {
		ctx := context.WithValue(c.Request.Context(), UserIDContextKey, userID)
		c.Request = c.Request.WithContext(ctx)
		c.Status(http.StatusCreated)
	})

	request := httptest.NewRequest(http.MethodGet, "/items/42", http.NoBody)
	request.Header.Set(headerRequestID, "request-123")
	request.Header.Set(headerCorrelationID, "correlation-456")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusCreated, recorder.Code)
	require.Equal(t, "request-123", recorder.Header().Get(headerRequestID))
	require.Equal(t, "correlation-456", recorder.Header().Get(headerCorrelationID))

	output := logs.String()
	require.Contains(t, output, `"msg":"http request completed"`)
	require.Contains(t, output, `"method":"GET"`)
	require.Contains(t, output, `"path":"/items/:id"`)
	require.Contains(t, output, `"status":201`)
	require.Contains(t, output, `"request_id":"request-123"`)
	require.Contains(t, output, `"correlation_id":"correlation-456"`)
	require.Contains(t, output, `"user_id":"`+userID.String()+`"`)
}

func TestRequestLoggerSkipsProbePaths(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var logs bytes.Buffer
	previousLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&logs, nil)))
	t.Cleanup(func() {
		slog.SetDefault(previousLogger)
	})

	router := gin.New()
	router.Use(RequestLogger())
	router.GET("/health/ready", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	router.GET("/metrics", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	for _, path := range []string{"/health/ready", "/metrics"} {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, path, http.NoBody)
		router.ServeHTTP(recorder, request)
		require.Equal(t, http.StatusOK, recorder.Code)
	}

	require.Empty(t, logs.String())
}

func TestRequestLoggerGeneratesIDs(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var logs bytes.Buffer
	previousLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&logs, nil)))
	t.Cleanup(func() {
		slog.SetDefault(previousLogger)
	})

	router := gin.New()
	router.Use(RequestLogger())
	router.GET("/ping", func(c *gin.Context) {
		requestID := c.Request.Context().Value(RequestIDContextKey)
		correlationID := c.Request.Context().Value(CorrelationIDContextKey)
		require.NotEmpty(t, requestID)
		require.Equal(t, requestID, correlationID)
		time.Sleep(time.Millisecond)
		c.Status(http.StatusOK)
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/ping", http.NoBody)
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.NotEmpty(t, recorder.Header().Get(headerRequestID))
	require.Equal(t, recorder.Header().Get(headerRequestID), recorder.Header().Get(headerCorrelationID))
	require.Contains(t, logs.String(), `"duration_ms":`)
}
