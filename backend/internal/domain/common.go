package domain

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

// Создаём типы-обёртки для часто используемых типов с поддержкой NULL

// NullTime представляет время, которое может быть NULL в базе данных
type NullTime = sql.NullTime

// NullString представляет строку, которая может быть NULL в базе данных
type NullString = sql.NullString

// NullInt64 представляет целое число, которое может быть NULL в базе данных
type NullInt64 = sql.NullInt64

// TimestampFields - структура для общих полей временных меток
type TimestampFields struct {
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt NullTime  `json:"updated_at" db:"updated_at"`
}

// NewUUID генерирует новый UUID
func NewUUID() uuid.UUID {
	return uuid.New()
}

// ParseUUID парсит строку в UUID
func ParseUUID(s string) (uuid.UUID, error) {
	return uuid.Parse(s)
}
