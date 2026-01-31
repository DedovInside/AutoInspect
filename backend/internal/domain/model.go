package domain

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// ModelStatus представляет статус модели в системе.
type ModelStatus string

const (
	// ModelStatusTraining указывает, что модель находится в процессе обучения.
	ModelStatusTraining ModelStatus = "training"
	// ModelStatusReady указывает, что модель готова к использованию.
	ModelStatusReady ModelStatus = "ready"
	// ModelStatusActive указывает, что модель активна и используется в продакшене.
	ModelStatusActive ModelStatus = "active"
	// ModelStatusDeprecated указывает, что модель устарела и не должна использоваться.
	ModelStatusDeprecated ModelStatus = "deprecated"
)

// IsValid проверяет, является ли статус модели допустимым.
func (ms ModelStatus) IsValid() bool {
	switch ms {
	case ModelStatusTraining, ModelStatusReady, ModelStatusActive, ModelStatusDeprecated:
		return true
	}
	return false
}

// ModelMetrics представляет метрики качества модели.
// Хранится в БД в формате JSON.
type ModelMetrics struct {
	Accuracy float64 `json:"accuracy,omitempty"`
	MAP      float64 `json:"map,omitempty"`
	Loss     float64 `json:"loss,omitempty"`
}

// Scan реализует интерфейс sql.Scanner для ModelMetrics для чтения из базы данных.
func (mm *ModelMetrics) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, mm)
}

// Value реализует интерфейс driver.Valuer для ModelMetrics для записи в базу данных.
func (mm ModelMetrics) Value() (driver.Value, error) {
	return json.Marshal(mm)
}

// MLModel представляет модель машинного обучения в системе.

type MLModel struct {
	ID      uuid.UUID `json:"id" db:"id"`
	Version string    `json:"version" db:"version"`
	Name    string    `json:"name" db:"name"`

	// Хранилище
	WeightsPath string  `json:"weights_path" db:"weights_path"`
	ConfigPath  *string `json:"config_path,omitempty" db:"config_path"`

	// Метаданные автомобиля
	CarMake  *string `json:"car_make" db:"car_make"`
	CarModel *string `json:"car_model" db:"car_model"`

	// Статус модели
	Status ModelStatus `json:"status" db:"status"`
	Active bool        `json:"active" db:"active"`

	// Метрики качества (из обучения)
	Metrics *ModelMetrics `json:"metrics,omitempty" db:"metrics_json"`

	// История доменной адаптации
	ParentModelID *uuid.UUID `json:"parent_model_id,omitempty" db:"parent_model_id"`
	TrainedAt     *time.Time `json:"trained_at,omitempty" db:"trained_at"`
	Description   *string    `json:"description,omitempty" db:"description"`

	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	CreatedBy *uuid.UUID `json:"created_by,omitempty" db:"created_by"`
}

// DTO ModelCreateRequest для создания новой модели
type ModelCreateRequest struct {
	Version     string        `json:"version" validate:"required"`
	Name        string        `json:"name" validate:"required"`
	WeightsPath string        `json:"weights_path" validate:"required"`
	CarMake     *string       `json:"car_make,omitempty"`
	CarModel    *string       `json:"car_model,omitempty"`
	Description *string       `json:"description,omitempty"`
	Metrics     *ModelMetrics `json:"metrics,omitempty"`
}

// IsReady проверяет, готова ли модель к использованию
func (m *MLModel) IsReady() bool {
	return m.Status == ModelStatusReady || m.Status == ModelStatusActive
}
