package domain

import (
	"time"

	"github.com/google/uuid"
)

type CarServiceReview struct {
	ID                  uuid.UUID `json:"id" db:"id"`
	RepairRequestID     uuid.UUID `json:"repair_request_id" db:"repair_request_id"`
	CarServiceProfileID uuid.UUID `json:"car_service_profile_id" db:"car_service_profile_id"`
	UserID              uuid.UUID `json:"user_id" db:"user_id"`
	Rating              int       `json:"rating" db:"rating"`
	AuthorName          *string   `json:"author_name,omitempty" db:"author_name"`
	Comment             *string   `json:"comment,omitempty" db:"comment"`
	CreatedAt           time.Time `json:"created_at" db:"created_at"`
	UpdatedAt           time.Time `json:"updated_at" db:"updated_at"`
}

type CreateCarServiceReviewInput struct {
	UserID          uuid.UUID
	RepairRequestID uuid.UUID
	Rating          int
	AuthorName      *string
	Comment         *string
}

type UpdateCarServiceReviewInput struct {
	UserID          uuid.UUID
	RepairRequestID uuid.UUID
	Rating          int
	AuthorName      *string
	Comment         *string
}

type ListCarServiceReviewsInput struct {
	CarServiceProfileID uuid.UUID
	Limit               int32
	Offset              int32
}

type ListUserCarServiceReviewsInput struct {
	UserID uuid.UUID
	Limit  int32
	Offset int32
}

func IsValidReviewRating(rating int) bool {
	return rating >= 1 && rating <= 5
}
