package domain

import (
	"time"

	"github.com/google/uuid"
)

type Role string

const (
	RoleUser  Role = "user"
	RoleOwner Role = "owner"
	RoleAdmin Role = "admin"
)

func (r Role) IsValid() bool {
	switch r {
	case RoleUser, RoleOwner, RoleAdmin:
		return true
	}
	return false
}

type User struct {
	ID            uuid.UUID  `json:"id" db:"id"`
	Username      string     `json:"username" db:"username"`
	Email         string     `json:"email" db:"email"`
	PasswordHash  string     `json:"-" db:"password_hash"`
	Role          Role       `json:"role" db:"role"`
	EmailVerified bool       `json:"email_verified" db:"email_verified"`
	IsActive      bool       `json:"is_active" db:"is_active"`
	CreatedAt     time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at" db:"updated_at"`
	LastLogin     *time.Time `json:"last_login,omitempty" db:"last_login"`
}
