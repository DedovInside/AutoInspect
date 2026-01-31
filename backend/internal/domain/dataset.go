package domain

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// DatasetType представляет тип датасета в системе.
type DatasetType string

const (
	DatasetTypeUserUpload DatasetType = "user_upload"
	DatasetTypePublic     DatasetType = "public"
	DatasetTypeGenerated  DatasetType = "generated"
)

// IsValid проверяет, является ли тип датасета допустимым.
func (dt DatasetType) IsValid() bool {
	switch dt {
	case DatasetTypeUserUpload, DatasetTypePublic, DatasetTypeGenerated:
		return true
	}
	return false
}

// DatasetStatus представляет статус обработки датасета
type DatasetStatus string

const (
	DatasetStatusPending    DatasetStatus = "pending"
	DatasetStatusProcessing DatasetStatus = "processing"
	DatasetStatusReady      DatasetStatus = "ready"
	DatasetStatusFailed     DatasetStatus = "failed"
)

// IsValid проверяет, является ли статус датасета допустимым.
func (ds DatasetStatus) IsValid() bool {
	switch ds {
	case DatasetStatusPending, DatasetStatusProcessing, DatasetStatusReady, DatasetStatusFailed:
		return true
	}
	return false
}

// DatasetClasses представляет список классов дефектов в датасете
type DatasetClasses []string

// Scan реализует интерфейс sql.Scanner для DatasetClasses
func (dc *DatasetClasses) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, dc)
}

// Value реализует интерфейс driver.Valuer для DatasetClasses
func (dc DatasetClasses) Value() (driver.Value, error) {
	return json.Marshal(dc)
}

// Dataset представляет датасет для обучения моделей
type Dataset struct {
	ID      uuid.UUID `json:"id" db:"id"`
	OwnerID uuid.UUID `json:"owner_id" db:"owner_id"`

	Name        string  `json:"name" db:"name"`
	Description *string `json:"description,omitempty" db:"description"`

	// Тип и статус
	DatasetType DatasetType   `json:"dataset_type" db:"dataset_type"`
	Status      DatasetStatus `json:"status" db:"status"`

	// Хранилище
	FileKey        *string `json:"file_key,omitempty" db:"file_key"`
	TotalSizeBytes *int64  `json:"total_size_bytes,omitempty" db:"total_size_bytes"`

	// Метаданные разметки
	AnnotationFormat *string         `json:"annotation_format,omitempty" db:"annotation_format"` // COCO, YOLO, custom
	ImagesCount      int             `json:"images_count" db:"images_count"`
	AnnotationsCount int             `json:"annotations_count" db:"annotations_count"`
	Classes          *DatasetClasses `json:"classes,omitempty" db:"classes_json"`

	// Для какой марки
	CarMake  *string `json:"car_make,omitempty" db:"car_make"`
	CarModel *string `json:"car_model,omitempty" db:"car_model"`

	// Временные метки
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   *time.Time `json:"updated_at,omitempty" db:"updated_at"`
	ValidatedAt *time.Time `json:"validated_at,omitempty" db:"validated_at"`
}

// DatasetCreateRequest DTO для создания нового датасета
type DatasetCreateRequest struct {
	Name             string      `json:"name" validate:"required,min=3,max=255"`
	Description      *string     `json:"description,omitempty"`
	DatasetType      DatasetType `json:"dataset_type" validate:"required"`
	AnnotationFormat *string     `json:"annotation_format,omitempty"`
	CarMake          *string     `json:"car_make,omitempty"`
	CarModel         *string     `json:"car_model,omitempty"`
}

// IsReady проверяет, готов ли датасет к использованию
func (d *Dataset) IsReady() bool {
	return d.Status == DatasetStatusReady
}

// IsFailed проверяет, провалилась ли обработка датасета
func (d *Dataset) IsFailed() bool {
	return d.Status == DatasetStatusFailed
}
