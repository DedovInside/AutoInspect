package domain

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// JobStatus представляет статус задачи обучения
type JobStatus string

const (
	JobStatusQueued    JobStatus = "queued"
	JobStatusRunning   JobStatus = "running"
	JobStatusCompleted JobStatus = "completed"
	JobStatusFailed    JobStatus = "failed"
	JobStatusCancelled JobStatus = "cancelled"
)

// IsValid проверяет, является ли статус задачи допустимым
func (js JobStatus) IsValid() bool {
	switch js {
	case JobStatusQueued, JobStatusRunning, JobStatusCompleted, JobStatusFailed, JobStatusCancelled:
		return true
	}
	return false
}

// TrainingParams представляет параметры обучения модели
type TrainingParams struct {
	Epochs    int     `json:"epochs"`
	BatchSize int     `json:"batch_size"`
	LR        float64 `json:"lr"`
	Optimizer string  `json:"optimizer"`
}

// Scan реализует интерфейс sql.Scanner для TrainingParams
func (tp *TrainingParams) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, tp)
}

// Value реализует интерфейс driver.Valuer для TrainingParams
func (tp TrainingParams) Value() (driver.Value, error) {
	return json.Marshal(tp)
}

// TrainingMetrics представляет метрики обучения по эпохам
type TrainingMetrics struct {
	Epochs []struct {
		Epoch    int     `json:"epoch"`
		Loss     float64 `json:"loss"`
		Accuracy float64 `json:"accuracy,omitempty"`
		ValLoss  float64 `json:"val_loss,omitempty"`
	} `json:"epochs"`
}

// Scan реализует интерфейс sql.Scanner для TrainingMetrics
func (tm *TrainingMetrics) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, tm)
}

// Value реализует интерфейс driver.Valuer для TrainingMetrics
func (tm TrainingMetrics) Value() (driver.Value, error) {
	return json.Marshal(tm)
}

// TrainingJob представляет задачу обучения модели
type TrainingJob struct {
	ID          uuid.UUID `json:"id" db:"id"`
	DatasetID   uuid.UUID `json:"dataset_id" db:"dataset_id"`
	BaseModelID uuid.UUID `json:"base_model_id" db:"base_model_id"`

	Status JobStatus `json:"status" db:"status"`

	// Параметры обучения
	Params TrainingParams `json:"params" db:"params_json"`

	// Результат
	ResultModelID *uuid.UUID `json:"result_model_id,omitempty" db:"result_model_id"`

	// Логи и метрики
	LogsPath *string          `json:"logs_path,omitempty" db:"logs_path"`
	Metrics  *TrainingMetrics `json:"metrics,omitempty" db:"metrics_json"`

	// Временные метки
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	StartedAt   *time.Time `json:"started_at,omitempty" db:"started_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty" db:"completed_at"`

	// Система
	WorkerID     *string `json:"worker_id,omitempty" db:"worker_id"`
	ErrorMessage *string `json:"error_message,omitempty" db:"error_message"`

	// Уведомления
	WebhookURL *string    `json:"webhook_url,omitempty" db:"webhook_url"`
	NotifiedAt *time.Time `json:"notified_at,omitempty" db:"notified_at"`
}

// TrainingJobCreateRequest DTO для создания новой задачи обучения
type TrainingJobCreateRequest struct {
	DatasetID   uuid.UUID      `json:"dataset_id" validate:"required"`
	BaseModelID uuid.UUID      `json:"base_model_id" validate:"required"`
	Params      TrainingParams `json:"params" validate:"required"`
	WebhookURL  *string        `json:"webhook_url,omitempty" validate:"omitempty,url"`
}

// IsCompleted проверяет, завершена ли задача обучения
func (tj *TrainingJob) IsCompleted() bool {
	return tj.Status == JobStatusCompleted
}

// IsFailed проверяет, провалилась ли задача обучения
func (tj *TrainingJob) IsFailed() bool {
	return tj.Status == JobStatusFailed
}

// IsRunning проверяет, выполняется ли задача обучения
func (tj *TrainingJob) IsRunning() bool {
	return tj.Status == JobStatusRunning
}

// Duration вычисляет длительность выполнения задачи
func (tj *TrainingJob) Duration() *time.Duration {
	if tj.StartedAt == nil {
		return nil
	}

	endTime := time.Now()
	if tj.CompletedAt != nil {
		endTime = *tj.CompletedAt
	}

	duration := endTime.Sub(*tj.StartedAt)
	return &duration
}
