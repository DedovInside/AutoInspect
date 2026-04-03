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
)
