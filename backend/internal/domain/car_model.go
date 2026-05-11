package domain

import (
	"time"

	"github.com/google/uuid"
)

type CarModel struct {
	ID         uuid.UUID `json:"id" db:"id"`
	Make       string    `json:"make" db:"make"`
	Model      string    `json:"model" db:"model"`
	Generation string    `json:"generation" db:"generation"`
	YearFrom   int       `json:"year_from" db:"year_from"`
	YearTo     int       `json:"year_to" db:"year_to"`

	PartsModelS3Key   string `json:"parts_model_s3_key" db:"parts_model_s3_key"`
	PartsConfigS3Key  string `json:"parts_config_s3_key" db:"parts_config_s3_key"`
	PartsCatalogS3Key string `json:"parts_catalog_s3_key" db:"parts_catalog_s3_key"`
	ModelVersion      string `json:"model_version" db:"model_version"`

	IsUniversal bool `json:"is_universal" db:"is_universal"`
	IsActive    bool `json:"is_active" db:"is_active"`

	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

type CarInfo struct {
	Make       string `json:"make"`
	Model      string `json:"model"`
	Generation string `json:"generation"`
	Year       int    `json:"year"`
}

func (cm *CarModel) MatchesQuery(carMake, model, generation string, year int) bool {
	if cm.Make != carMake || cm.Model != model {
		return false
	}

	if cm.Generation != generation && cm.Generation != "" {
		return false
	}

	if year < cm.YearFrom {
		return false
	}

	if cm.YearTo > 0 && year > cm.YearTo {
		return false
	}

	return true
}
