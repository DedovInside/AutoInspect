package domain

import (
	"time"

	"github.com/google/uuid"
)

// ModelStatus представляет жизненный цикл inference-модели.
type ModelStatus string

const (
	ModelStatusReady      ModelStatus = "ready"
	ModelStatusActive     ModelStatus = "active"
	ModelStatusDeprecated ModelStatus = "deprecated"
)

// IsValid проверяет, является ли статус модели допустимым.
func (ms ModelStatus) IsValid() bool {
	switch ms {
	case ModelStatusReady, ModelStatusActive, ModelStatusDeprecated:
		return true
	}
	return false
}

// MLModel представляет реестр inference-моделей для конкретной спецификации авто.
type MLModel struct {
	ID      uuid.UUID `json:"id" db:"id"`
	Version string    `json:"version" db:"version"`
	Name    string    `json:"name" db:"name"`

	// Спецификация автомобиля
	CarMake       string  `json:"car_make" db:"car_make"`
	CarModel      string  `json:"car_model" db:"car_model"`
	CarGeneration *string `json:"car_generation,omitempty" db:"car_generation"`
	YearFrom      *int    `json:"year_from,omitempty" db:"year_from"`
	YearTo        *int    `json:"year_to,omitempty" db:"year_to"`

	// Артефакт inference-модели
	WeightsPath string  `json:"weights_path" db:"weights_path"`
	ConfigPath  *string `json:"config_path,omitempty" db:"config_path"`

	Status    ModelStatus `json:"status" db:"status"`
	Active    bool        `json:"active" db:"active"`
	CreatedAt time.Time   `json:"created_at" db:"created_at"`
	UpdatedAt *time.Time  `json:"updated_at,omitempty" db:"updated_at"`
}

// ModelCreateRequest DTO для регистрации новой inference-модели.
type ModelCreateRequest struct {
	Version       string  `json:"version" validate:"required"`
	Name          string  `json:"name" validate:"required"`
	CarMake       string  `json:"car_make" validate:"required"`
	CarModel      string  `json:"car_model" validate:"required"`
	CarGeneration *string `json:"car_generation,omitempty"`
	YearFrom      *int    `json:"year_from,omitempty"`
	YearTo        *int    `json:"year_to,omitempty"`
	WeightsPath   string  `json:"weights_path" validate:"required"`
	ConfigPath    *string `json:"config_path,omitempty"`
}

// IsReady проверяет, готова ли модель к использованию.
func (m *MLModel) IsReady() bool {
	return m.Status == ModelStatusReady || m.Status == ModelStatusActive
}
