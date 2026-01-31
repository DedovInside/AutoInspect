package domain

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// AnalysisStatus представляет статус анализа изображения
type AnalysisStatus string

const (
	AnalysisStatusQueued     AnalysisStatus = "queued"
	AnalysisStatusProcessing AnalysisStatus = "processing"
	AnalysisStatusCompleted  AnalysisStatus = "completed"
	AnalysisStatusFailed     AnalysisStatus = "failed"
	AnalysisStatusCancelled  AnalysisStatus = "cancelled"
)

// IsValid проверяет, является ли статус анализа допустимым
func (as AnalysisStatus) IsValid() bool {
	switch as {
	case AnalysisStatusQueued, AnalysisStatusProcessing, AnalysisStatusCompleted,
		AnalysisStatusFailed, AnalysisStatusCancelled:
		return true
	}
	return false
}

// DefectType представляет тип дефекта
type DefectType string

const (
	DefectTypeScratch     DefectType = "scratch"
	DefectTypeDent        DefectType = "dent"
	DefectTypeCrack       DefectType = "crack"
	DefectTypeBrokenGlass DefectType = "broken_glass"
)

// DefectSeverity представляет серьёзность повреждения
type DefectSeverity string

const (
	DefectSeverityMinor DefectSeverity = "minor"
	DefectSeverityMajor DefectSeverity = "major"
)

// BoundingBox представляет координаты ограничивающего прямоугольника
type BoundingBox struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

// Defect представляет обнаруженный дефект
type Defect struct {
	ID                string         `json:"id"`
	PartName          string         `json:"part_name"`
	PartID            string         `json:"part_id"`
	DefectType        DefectType     `json:"defect_type"`
	Severity          DefectSeverity `json:"severity"`
	BBox              BoundingBox    `json:"bbox"`
	Mask              *string        `json:"mask,omitempty"`
	Confidence        float64        `json:"confidence"`
	RecommendedAction *string        `json:"recommended_action,omitempty"`
}

// AnalysisResult представляет результат анализа изображения
type AnalysisResult struct {
	ViewAngle string   `json:"view_angle,omitempty"` // front, rear, side_left, side_right
	Defects   []Defect `json:"defects"`
	Summary   struct {
		TotalDefects  int      `json:"total_defects"`
		CriticalCount int      `json:"critical_count"`
		EstimatedCost *float64 `json:"estimated_cost,omitempty"`
	} `json:"summary"`
}

// Scan реализует интерфейс sql.Scanner для AnalysisResult
func (ar *AnalysisResult) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, ar)
}

// Value реализует интерфейс driver.Valuer для AnalysisResult
func (ar AnalysisResult) Value() (driver.Value, error) {
	return json.Marshal(ar)
}

// ImageMetadata представляет метаданные изображения
type ImageMetadata struct {
	Size       int64  `json:"size"`
	Format     string `json:"format"`
	Dimensions struct {
		Width  int `json:"width"`
		Height int `json:"height"`
	} `json:"dimensions"`
}

// Scan реализует интерфейс sql.Scanner для ImageMetadata
func (im *ImageMetadata) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, im)
}

// Value реализует интерфейс driver.Valuer для ImageMetadata
func (im ImageMetadata) Value() (driver.Value, error) {
	return json.Marshal(im)
}

// Analysis представляет задачу анализа изображения
type Analysis struct {
	ID     uuid.UUID      `json:"id" db:"id"`
	UserID uuid.UUID      `json:"user_id" db:"user_id"`
	Status AnalysisStatus `json:"status" db:"status"`

	// Изображение
	ImageKey      string         `json:"image_key" db:"image_key"`
	ImageMetadata *ImageMetadata `json:"image_metadata,omitempty" db:"image_metadata"`

	// ML модель
	ModelVersion string     `json:"model_version" db:"model_version"`
	ModelID      *uuid.UUID `json:"model_id,omitempty" db:"model_id"`

	// Результаты
	Result *AnalysisResult `json:"result,omitempty" db:"result_json"`

	// Ошибки
	ErrorMessage *string `json:"error_message,omitempty" db:"error_message"`
	ErrorCode    *string `json:"error_code,omitempty" db:"error_code"`
	RetryCount   int     `json:"retry_count" db:"retry_count"`

	// Временные метки
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   *time.Time `json:"updated_at,omitempty" db:"updated_at"`
	ProcessedAt *time.Time `json:"processed_at,omitempty" db:"processed_at"`
}

// AnalysisCreateRequest DTO для создания нового анализа
type AnalysisCreateRequest struct {
	ImageKey     string  `json:"image_key" validate:"required"`
	ModelVersion *string `json:"model_version,omitempty"`
}

// IsCompleted проверяет, завершён ли анализ
func (a *Analysis) IsCompleted() bool {
	return a.Status == AnalysisStatusCompleted
}

// IsFailed проверяет, завершился ли анализ с ошибкой
func (a *Analysis) IsFailed() bool {
	return a.Status == AnalysisStatusFailed
}

// CanRetry проверяет, можно ли повторить анализ
func (a *Analysis) CanRetry() bool {
	return a.Status == AnalysisStatusFailed && a.RetryCount < 3
}
