package domain

import (
	"time"

	"github.com/google/uuid"
)

type JobStatus string

const (
	StatusPending    JobStatus = "pending"
	StatusProcessing JobStatus = "processing"
	StatusCompleted  JobStatus = "completed"
	StatusFailed     JobStatus = "failed"
)

type AnalysisJob struct {
	ID     uuid.UUID `json:"id" db:"id"`
	UserID uuid.UUID `json:"user_id" db:"user_id"`

	IdempotencyKey *string `json:"idempotency_key,omitempty" db:"idempotency_key"`

	CarMake       string `json:"car_make" db:"car_make"`
	CarModel      string `json:"car_model" db:"car_model"`
	CarGeneration string `json:"car_generation" db:"car_generation"`
	CarYear       int    `json:"car_year" db:"car_year"`

	ImageKeys []string `json:"image_keys" db:"image_keys"` // Храним как JSONB или отдельной таблицей, в домене слайс

	CorrelationID uuid.UUID `json:"correlation_id" db:"correlation_id"`

	Status       JobStatus `json:"status" db:"status"`
	ErrorMessage *string   `json:"error_message,omitempty" db:"error_message"`

	Result *AnalysisResult `json:"result,omitempty" db:"result"`

	UsedModelVersion string     `json:"used_model_version,omitempty" db:"used_model_version"`
	RequestedAt      time.Time  `json:"requested_at" db:"requested_at"`
	StartedAt        *time.Time `json:"started_at,omitempty" db:"started_at"`
	CompletedAt      *time.Time `json:"completed_at,omitempty" db:"completed_at"`
}
