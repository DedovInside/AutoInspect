package dto

import (
	"time"

	"github.com/DedovInside/AutoInspect/backend/internal/domain"
	"github.com/google/uuid"
)

type CreateCarServiceReviewRequest struct {
	Rating     int     `json:"rating" binding:"required,min=1,max=5"`
	AuthorName *string `json:"author_name" binding:"omitempty,max=255"`
	Comment    *string `json:"comment" binding:"omitempty,max=2000"`
}

type UpdateCarServiceReviewRequest struct {
	Rating     int     `json:"rating" binding:"required,min=1,max=5"`
	AuthorName *string `json:"author_name" binding:"omitempty,max=255"`
	Comment    *string `json:"comment" binding:"omitempty,max=2000"`
}

type ListCarServiceReviewsQuery struct {
	Limit  int `form:"limit,default=20" binding:"omitempty,min=1,max=100"`
	Offset int `form:"offset,default=0" binding:"omitempty,min=0"`
}

type CarServiceReviewResponse struct {
	ID                  uuid.UUID `json:"id"`
	RepairRequestID     uuid.UUID `json:"repair_request_id"`
	CarServiceProfileID uuid.UUID `json:"car_service_profile_id"`
	UserID              uuid.UUID `json:"user_id"`
	Rating              int       `json:"rating"`
	AuthorName          *string   `json:"author_name,omitempty"`
	Comment             *string   `json:"comment,omitempty"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

type CreateCarServiceReviewResponse struct {
	Review CarServiceReviewResponse `json:"review"`
}

type GetCarServiceReviewResponse struct {
	Review CarServiceReviewResponse `json:"review"`
}

type UpdateCarServiceReviewResponse struct {
	Review CarServiceReviewResponse `json:"review"`
}

type ListCarServiceReviewsResponse struct {
	Items  []CarServiceReviewResponse `json:"items"`
	Limit  int                        `json:"limit"`
	Offset int                        `json:"offset"`
	Meta   ListMeta                   `json:"meta"`
}

func ToCarServiceReviewResponse(review *domain.CarServiceReview) CarServiceReviewResponse {
	if review == nil {
		return CarServiceReviewResponse{}
	}

	return CarServiceReviewResponse{
		ID:                  review.ID,
		RepairRequestID:     review.RepairRequestID,
		CarServiceProfileID: review.CarServiceProfileID,
		UserID:              review.UserID,
		Rating:              review.Rating,
		AuthorName:          review.AuthorName,
		Comment:             review.Comment,
		CreatedAt:           review.CreatedAt,
		UpdatedAt:           review.UpdatedAt,
	}
}

func ToCarServiceReviewResponseList(reviews []*domain.CarServiceReview) []CarServiceReviewResponse {
	out := make([]CarServiceReviewResponse, 0, len(reviews))
	for _, review := range reviews {
		if review == nil {
			continue
		}
		out = append(out, ToCarServiceReviewResponse(review))
	}

	return out
}

func NewCarServiceReviewListResponse(
	items []CarServiceReviewResponse,
	limit, offset int,
) ListCarServiceReviewsResponse {
	count := len(items)
	hasNext := limit > 0 && count == limit
	meta := ListMeta{
		Count:   count,
		HasNext: hasNext,
	}

	if hasNext {
		next := offset + count
		meta.NextOffset = &next
	}

	return ListCarServiceReviewsResponse{
		Items:  items,
		Limit:  limit,
		Offset: offset,
		Meta:   meta,
	}
}
