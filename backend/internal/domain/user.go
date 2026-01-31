package domain

import (
	"time"

	"github.com/google/uuid"
)

// Role представляет роль пользователя в системе
type Role string

const (
	RoleUser  Role = "user"  // Обычный пользователь или клиент
	RoleOwner Role = "owner" // Владелец сервиса
	RoleAdmin Role = "admin" // Администратор системы
)

// IsValid проверяет, является ли роль допустимой
func (r Role) IsValid() bool {
	switch r {
	case RoleUser, RoleOwner, RoleAdmin:
		return true
	}
	return false
}

// User представляет пользователя в системе
type User struct {
	ID           uuid.UUID `json:"id" db:"id"`
	Username     string    `json:"username" db:"username"`
	Email        string    `json:"email" db:"email"`
	PasswordHash string    `json:"-" db:"password_hash"` // Храним хэш пароля, но не возвращаем его в JSON
	Role         Role      `json:"role" db:"role"`

	// Дополнительные поля
	EmailVerified bool `json:"email_verified" db:"email_verified"`
	IsActive      bool `json:"is_active" db:"is_active"`

	// Метаданные
	TimestampFields
	LastLogin *time.Time `json:"last_login,omitempty" db:"last_login"`

	// Опциональный rate limiting
	APICallsCount   int        `json:"api_calls_count" db:"api_calls_count"`
	APIQuotaResetAt *time.Time `json:"-" db:"api_quota_reset_at"`
}

// DTO UserCreateRequest для создания нового пользователя
type UserCreateRequest struct {
	Username string `json:"username" validate:"required,min=3,max=50"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
	Role     Role   `json:"role" validate:"required,oneof=user owner admin"`
}

// DTO UserResponse представляет данные пользователя, возвращаемые API
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

// ToUserResponse преобразует User в UserResponse для API
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

// IsAdmin проверяет, является ли пользователь администратором
func (u *User) IsAdmin() bool {
	return u.Role == RoleAdmin
}

// CanManageModels проверяет, может ли пользователь управлять моделями
func (u *User) CanManageModels() bool {
	return u.Role == RoleOwner || u.Role == RoleAdmin
}
