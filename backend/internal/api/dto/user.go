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
	FirstName     *string     `json:"first_name,omitempty"`
	LastName      *string     `json:"last_name,omitempty"`
	DisplayName   *string     `json:"display_name,omitempty"`
	AvatarURL     *string     `json:"avatar_url,omitempty"`
	ContactName   *string     `json:"contact_name,omitempty"`
	ContactPhone  *string     `json:"contact_phone,omitempty"`
	ContactEmail  *string     `json:"contact_email,omitempty"`
	Role          domain.Role `json:"role"`
	EmailVerified bool        `json:"email_verified"`
	IsActive      bool        `json:"is_active"`
	CreatedAt     time.Time   `json:"created_at"`
	LastLogin     *time.Time  `json:"last_login,omitempty"`
}

type UpdateMeRequest struct {
	ContactName  *string `json:"contact_name" binding:"omitempty,max=255"`
	ContactPhone *string `json:"contact_phone" binding:"omitempty,max=50"`
	ContactEmail *string `json:"contact_email" binding:"omitempty,email,max=255"`
}

func ToUserResponse(u *domain.User) UserResponse {
	return UserResponse{
		ID:            u.ID,
		Username:      u.Username,
		Email:         u.Email,
		FirstName:     u.FirstName,
		LastName:      u.LastName,
		DisplayName:   u.DisplayName,
		AvatarURL:     u.AvatarURL,
		ContactName:   u.ContactName,
		ContactPhone:  u.ContactPhone,
		ContactEmail:  u.ContactEmail,
		Role:          u.Role,
		EmailVerified: u.EmailVerified,
		IsActive:      u.IsActive,
		CreatedAt:     u.CreatedAt,
		LastLogin:     u.LastLogin,
	}
}
