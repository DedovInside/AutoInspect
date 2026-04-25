package dto

import (
	"time"

	"github.com/DedovInside/AutoInspect/backend/internal/domain"
	"github.com/google/uuid"
)

type UserCreateRequest struct {
	Username string      `json:"username" binding:"required,min=3,max=50"`
	Email    string      `json:"email" binding:"required,email"`
	Password string      `json:"password" binding:"required,min=8"`
	Role     domain.Role `json:"role" binding:"required,oneof=user car_service admin"`
}

type UserResponse struct {
	ID            uuid.UUID   `json:"id"`
	Username      string      `json:"username"`
	Email         string      `json:"email"`
	Role          domain.Role `json:"role"`
	EmailVerified bool        `json:"email_verified"`
	IsActive      bool        `json:"is_active"`
	CreatedAt     time.Time   `json:"created_at"`
	LastLogin     *time.Time  `json:"last_login,omitempty"`
}

func ToUserResponse(u *domain.User) UserResponse {
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
