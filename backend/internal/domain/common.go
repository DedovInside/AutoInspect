package domain

import (
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrNotFound      = errors.New("not found")
	ErrAlreadyExists = errors.New("already exists")
	ErrInvalidInput  = errors.New("invalid input")
)

type NullTime = sql.NullTime

type NullString = sql.NullString

type NullInt64 = sql.NullInt64

type TimestampFields struct {
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt NullTime  `json:"updated_at" db:"updated_at"`
}

func NewUUID() uuid.UUID {
	return uuid.New()
}

func ParseUUID(s string) (uuid.UUID, error) {
	return uuid.Parse(s)
}
