package domain

import (
	"time"

	"github.com/google/uuid"
)

type ModelTrainingRequestStatus string

const (
	ModelTrainingRequestStatusPending    ModelTrainingRequestStatus = "pending"
	ModelTrainingRequestStatusApproved   ModelTrainingRequestStatus = "approved"
	ModelTrainingRequestStatusRejected   ModelTrainingRequestStatus = "rejected"
	ModelTrainingRequestStatusInProgress ModelTrainingRequestStatus = "in_progress"
	ModelTrainingRequestStatusCompleted  ModelTrainingRequestStatus = "completed"
)

func (s ModelTrainingRequestStatus) IsValid() bool {
	switch s {
	case ModelTrainingRequestStatusPending,
		ModelTrainingRequestStatusApproved,
		ModelTrainingRequestStatusRejected,
		ModelTrainingRequestStatusInProgress,
		ModelTrainingRequestStatusCompleted:
		return true
	default:
		return false
	}
}

func (s ModelTrainingRequestStatus) IsActive() bool {
	switch s {
	case ModelTrainingRequestStatusPending,
		ModelTrainingRequestStatusApproved,
		ModelTrainingRequestStatusInProgress:
		return true
	default:
		return false
	}
}

func (s ModelTrainingRequestStatus) IsTerminal() bool {
	switch s {
	case ModelTrainingRequestStatusRejected, ModelTrainingRequestStatusCompleted:
		return true
	default:
		return false
	}
}

type ModelTrainingRequest struct {
	ID              uuid.UUID                  `json:"id" db:"id"`
	InitiatorUserID uuid.UUID                  `json:"initiator_user_id" db:"initiator_user_id"`
	InitiatorRole   Role                       `json:"initiator_role" db:"initiator_role"`
	Make            string                     `json:"make" db:"make"`
	Model           string                     `json:"model" db:"model"`
	Generation      string                     `json:"generation" db:"generation"`
	YearFrom        int                        `json:"year_from" db:"year_from"`
	YearTo          int                        `json:"year_to" db:"year_to"`
	Description     string                     `json:"description" db:"description"`
	Status          ModelTrainingRequestStatus `json:"status" db:"status"`
	AdminComment    string                     `json:"admin_comment,omitempty" db:"admin_comment"`
	ReviewedBy      *uuid.UUID                 `json:"reviewed_by,omitempty" db:"reviewed_by"`
	ReviewedAt      *time.Time                 `json:"reviewed_at,omitempty" db:"reviewed_at"`
	CreatedModelID  *uuid.UUID                 `json:"created_model_id,omitempty" db:"created_model_id"`
	IdempotencyKey  *string                    `json:"idempotency_key,omitempty" db:"idempotency_key"`
	CreatedAt       time.Time                  `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time                  `json:"updated_at" db:"updated_at"`
}

type CreateModelTrainingRequestInput struct {
	InitiatorUserID uuid.UUID
	InitiatorRole   Role
	Make            string
	Model           string
	Generation      string
	YearFrom        int
	YearTo          int
	Description     string
	IdempotencyKey  *string
}

type UpdateModelTrainingRequestStatusInput struct {
	ID             uuid.UUID
	Status         ModelTrainingRequestStatus
	AdminComment   string
	ReviewedBy     uuid.UUID
	CreatedModelID *uuid.UUID
}
