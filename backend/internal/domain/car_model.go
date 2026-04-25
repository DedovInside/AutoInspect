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

	// Путь к файлу .pth в MinIO (например: "models/volkswagen/polo/5/v1.3.0.pth")
	ModelS3Key   string `json:"model_s3_key" db:"model_s3_key"`
	ModelVersion string `json:"model_version" db:"model_version"`

	IsUniversal bool `json:"is_universal" db:"is_universal"` // Флаг "универсальная модель"
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
