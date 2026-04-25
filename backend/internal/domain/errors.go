package domain

import (
	"errors"
)

var (
	ErrNotFound           = errors.New("not found")
	ErrAlreadyExists      = errors.New("already exists")
	ErrInvalidInput       = errors.New("invalid input")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrInvalidToken       = errors.New("invalid token")
	ErrInternal           = errors.New("internal server error")
	ErrInvalidOAuthConfig = errors.New("invalid oauth configuration")
	ErrJobNotFound        = errors.New("analysis job not found")
	ErrInvalidModel       = errors.New("no suitable ML model found for this car")
	ErrJobNotReady        = errors.New("analysis job is not completed yet")
	ErrJobFailed          = errors.New("analysis job failed")
	ErrInvalidImage       = errors.New("invalid image data")
	ErrForbidden          = errors.New("forbidden")
)
