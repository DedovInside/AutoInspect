package domain

import (
	"database/sql"
	"errors"
	"time"
)

var (
	ErrNotFound           = errors.New("not found")
	ErrAlreadyExists      = errors.New("already exists")
	ErrInvalidInput       = errors.New("invalid input")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrInvalidToken       = errors.New("invalid token")
)

type NullTime = sql.NullTime

type NullString = sql.NullString

type NullInt64 = sql.NullInt64

type TimestampFields struct {
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}
