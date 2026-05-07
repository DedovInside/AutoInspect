package domain

import (
	"time"

	"github.com/google/uuid"
)

type CarServiceApplicationStatus string

const (
	CarServiceApplicationStatusPending  CarServiceApplicationStatus = "pending"
	CarServiceApplicationStatusApproved CarServiceApplicationStatus = "approved"
	CarServiceApplicationStatusRejected CarServiceApplicationStatus = "rejected"
)

func (s CarServiceApplicationStatus) IsValid() bool {
	switch s {
	case CarServiceApplicationStatusPending,
		CarServiceApplicationStatusApproved,
		CarServiceApplicationStatusRejected:
		return true
	default:
		return false
	}
}

func (s CarServiceApplicationStatus) IsTerminal() bool {
	switch s {
	case CarServiceApplicationStatusApproved, CarServiceApplicationStatusRejected:
		return true
	default:
		return false
	}
}

type CarServiceApplication struct {
	ID               uuid.UUID                   `json:"id" db:"id"`
	UserID           uuid.UUID                   `json:"user_id" db:"user_id"`
	OrganizationName string                      `json:"organization_name" db:"organization_name"`
	City             string                      `json:"city" db:"city"`
	Address          string                      `json:"address" db:"address"`
	Phone            *string                     `json:"phone,omitempty" db:"phone"`
	Email            *string                     `json:"email,omitempty" db:"email"`
	ContactInfo      *string                     `json:"contact_info,omitempty" db:"contact_info"`
	Description      string                      `json:"description" db:"description"`
	Status           CarServiceApplicationStatus `json:"status" db:"status"`
	RejectionReason  *string                     `json:"rejection_reason,omitempty" db:"rejection_reason"`
	ReviewedBy       *uuid.UUID                  `json:"reviewed_by,omitempty" db:"reviewed_by"`
	ReviewedAt       *time.Time                  `json:"reviewed_at,omitempty" db:"reviewed_at"`
	CreatedProfileID *uuid.UUID                  `json:"created_profile_id,omitempty" db:"created_profile_id"`
	CreatedAt        time.Time                   `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time                   `json:"updated_at" db:"updated_at"`
}

type CreateCarServiceApplicationInput struct {
	UserID           uuid.UUID
	UserRole         Role
	OrganizationName string
	City             string
	Address          string
	Phone            *string
	Email            *string
	ContactInfo      *string
	Description      string
}

type ApproveCarServiceApplicationInput struct {
	ID         uuid.UUID
	ReviewedBy uuid.UUID
}

type RejectCarServiceApplicationInput struct {
	ID              uuid.UUID
	ReviewedBy      uuid.UUID
	RejectionReason string
}
