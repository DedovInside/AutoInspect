package observability

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func MetricsHandlerHTTP() http.Handler {
	return promhttp.Handler()
}

func MetricsHandler() gin.HandlerFunc {
	handler := MetricsHandlerHTTP()
	return func(c *gin.Context) {
		handler.ServeHTTP(c.Writer, c.Request)
	}
}

func HTTPMetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.FullPath() == "/metrics" {
			c.Next()
			return
		}

		HTTPRequestsInFlight.Inc()
		startedAt := time.Now()
		c.Next()
		HTTPRequestsInFlight.Dec()

		ObserveHTTPRequest(c.Request.Method, normalizedRoutePath(c), c.Writer.Status(), startedAt)
	}
}

func normalizedRoutePath(c *gin.Context) string {
	if path := strings.TrimSpace(c.FullPath()); path != "" {
		return path
	}
	if c.Writer.Status() == http.StatusNotFound {
		return "not_found"
	}
	return "unknown"
}
