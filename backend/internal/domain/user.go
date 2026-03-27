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
	ID            uuid.UUID `json:"id" db:"id"`
	Username      string    `json:"username" db:"username"`
	Email         string    `json:"email" db:"email"`
	PasswordHash  string    `json:"-" db:"password_hash"`
	Role          Role      `json:"role" db:"role"`
	EmailVerified bool      `json:"email_verified" db:"email_verified"`
	IsActive      bool      `json:"is_active" db:"is_active"`

	TimestampFields
	LastLogin *time.Time `json:"last_login,omitempty" db:"last_login"`

	APICallsCount   int        `json:"api_calls_count" db:"api_calls_count"`
	APIQuotaResetAt *time.Time `json:"-" db:"api_quota_reset_at"`
}

type UserCreateRequest struct {
	Username string `json:"username" validate:"required,min=3,max=50"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
	Role     Role   `json:"role" validate:"required,oneof=user owner admin"`
}

type UserResponse struct {
	ID            uuid.UUID  `json:"id"`
	Username      string     `json:"username"`
	Email         string     `json:"email"`
	Role          Role       `json:"role"`
	EmailVerified bool       `json:"email_verified"`
	IsActive      bool       `json:"is_active"`
	CreatedAt     time.Time  `json:"created_at"`
	LastLogin     *time.Time `json:"last_login,omitempty"`
}

func (u *User) ToUserResponse() UserResponse {
	return UserResponse{
		ID:            u.ID,
		Username:      u.Username,
		Email:         u.Email,
		Role:          u.Role,
		EmailVerified: u.EmailVerified,
		IsActive:      u.IsActive,
		CreatedAt:     u.CreatedAt,
		LastLogin:     u.LastLogin,
	}
}
