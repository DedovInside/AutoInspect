package notify

import (
	"time"

	"github.com/google/uuid"
)

const (
	EventAnalysisCompleted = "analysis:completed"
	EventAnalysisFailed    = "analysis:failed"
)

type JobEvent struct {
	JobID     uuid.UUID `json:"job_id"`
	UserID    uuid.UUID `json:"user_id,omitempty"`
	Type      string    `json:"type"`
	Status    string    `json:"status,omitempty"`
	Payload   any       `json:"payload,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

type AnalysisCompletedPayload struct {
	ModelVersion string `json:"model_version"`
	DamageCount  int    `json:"damage_count"`
}

type AnalysisFailedPayload struct {
	Error string `json:"error"`
}
