package observability

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type DependencyCheck func(ctx context.Context) error

type HealthChecker struct {
	serviceName string
	checks      map[string]DependencyCheck
	timeout     time.Duration
	startedAt   time.Time
}

type HealthSnapshot struct {
	Status    string                     `json:"status"`
	Service   string                     `json:"service"`
	StartedAt time.Time                  `json:"started_at"`
	CheckedAt time.Time                  `json:"checked_at"`
	Checks    map[string]DependencyState `json:"checks,omitempty"`
}

type DependencyState struct {
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

const (
	healthStatusOK       = "ok"
	healthStatusFailed   = "failed"
	healthStatusDegraded = "degraded"
)

func NewHealthChecker(serviceName string, checks map[string]DependencyCheck) *HealthChecker {
	return &HealthChecker{
		serviceName: serviceName,
		checks:      checks,
		timeout:     3 * time.Second,
		startedAt:   time.Now().UTC(),
	}
}

func (h *HealthChecker) LiveGin(c *gin.Context) {
	c.JSON(http.StatusOK, HealthSnapshot{
		Status:    healthStatusOK,
		Service:   h.serviceName,
		StartedAt: h.startedAt,
		CheckedAt: time.Now().UTC(),
	})
}

func (h *HealthChecker) ReadyGin(c *gin.Context) {
	snapshot, ready := h.Snapshot(c.Request.Context())
	if !ready {
		c.JSON(http.StatusServiceUnavailable, snapshot)
		return
	}
	c.JSON(http.StatusOK, snapshot)
}

func (h *HealthChecker) SummaryGin(c *gin.Context) {
	h.ReadyGin(c)
}

func (h *HealthChecker) LiveHTTP(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, HealthSnapshot{
		Status:    healthStatusOK,
		Service:   h.serviceName,
		StartedAt: h.startedAt,
		CheckedAt: time.Now().UTC(),
	})
}

func (h *HealthChecker) ReadyHTTP(w http.ResponseWriter, r *http.Request) {
	snapshot, ready := h.Snapshot(r.Context())
	if !ready {
		writeJSON(w, http.StatusServiceUnavailable, snapshot)
		return
	}
	writeJSON(w, http.StatusOK, snapshot)
}

func (h *HealthChecker) SummaryHTTP(w http.ResponseWriter, r *http.Request) {
	h.ReadyHTTP(w, r)
}

func (h *HealthChecker) Snapshot(ctx context.Context) (HealthSnapshot, bool) {
	snapshot := HealthSnapshot{
		Status:    healthStatusOK,
		Service:   h.serviceName,
		StartedAt: h.startedAt,
		CheckedAt: time.Now().UTC(),
		Checks:    make(map[string]DependencyState, len(h.checks)),
	}

	ready := true
	for name, check := range h.checks {
		state := h.check(ctx, check)
		snapshot.Checks[name] = state
		if state.Status != healthStatusOK {
			ready = false
		}
		statusValue := 1.0
		if state.Status != healthStatusOK {
			statusValue = 0
		}
		ReadinessDependencyStatus.WithLabelValues(h.serviceName, name).Set(statusValue)
	}

	if !ready {
		snapshot.Status = healthStatusDegraded
	}

	return snapshot, ready
}

func (h *HealthChecker) check(parent context.Context, check DependencyCheck) DependencyState {
	ctx, cancel := context.WithTimeout(parent, h.timeout)
	defer cancel()

	if err := check(ctx); err != nil {
		return DependencyState{Status: healthStatusFailed, Error: err.Error()}
	}
	return DependencyState{Status: healthStatusOK}
}
