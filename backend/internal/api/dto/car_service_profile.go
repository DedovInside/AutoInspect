package dto

import (
	"time"

	"github.com/DedovInside/AutoInspect/backend/internal/domain"
	"github.com/google/uuid"
)

type UpdateCarServiceProfileRequest struct {
	OrganizationName string  `json:"organization_name" binding:"required,min=1,max=255"`
	City             string  `json:"city" binding:"required,min=1,max=100"`
	Address          string  `json:"address" binding:"required,min=1,max=255"`
	Phone            *string `json:"phone" binding:"omitempty,max=50"`
	Email            *string `json:"email" binding:"omitempty,max=255"`
	WebsiteURL       *string `json:"website_url" binding:"omitempty,max=500"`
	ContactInfo      *string `json:"contact_info" binding:"omitempty,max=2000"`
	Description      *string `json:"description" binding:"omitempty,max=2000"`
	IsActive         *bool   `json:"is_active" binding:"required"`
}

type SetCarServiceProfileActiveRequest struct {
	IsActive *bool `json:"is_active" binding:"required"`
}

type GetCarServiceProfileResponse struct {
	Profile CarServiceProfileResponse `json:"profile"`
}

type UpdateCarServiceProfileResponse struct {
	Profile CarServiceProfileResponse `json:"profile"`
}

type SetCarServiceProfileActiveResponse struct {
	Profile CarServiceProfileResponse `json:"profile"`
}

type CarServiceImageResponse struct {
	ID               uuid.UUID `json:"id"`
	IsPrimary        bool      `json:"is_primary"`
	SortOrder        int       `json:"sort_order"`
	OriginalFilename string    `json:"original_filename"`
	ContentType      string    `json:"content_type"`
	SizeBytes        int64     `json:"size_bytes"`
	URL              string    `json:"url"`
	ExpiresAt        time.Time `json:"expires_at"`
	CreatedAt        time.Time `json:"created_at"`
}

type UploadCarServiceImageResponse struct {
	Image CarServiceImageResponse `json:"image"`
}

type ListCarServiceImagesResponse struct {
	Items []CarServiceImageResponse `json:"items"`
	Meta  ListMeta                  `json:"meta"`
}

type SetPrimaryCarServiceImageResponse struct {
	Image CarServiceImageResponse `json:"image"`
}

func ToCarServiceImageResponse(
	image *domain.CarServiceImage,
	url string,
	expiresAt time.Time,
) CarServiceImageResponse {
	if image == nil {
		return CarServiceImageResponse{}
	}

	return CarServiceImageResponse{
		ID:               image.ID,
		IsPrimary:        image.IsPrimary,
		SortOrder:        image.SortOrder,
		OriginalFilename: image.OriginalFilename,
		ContentType:      image.ContentType,
		SizeBytes:        image.SizeBytes,
		URL:              url,
		ExpiresAt:        expiresAt,
		CreatedAt:        image.CreatedAt,
	}
}

func NewCarServiceImagesResponse(items []CarServiceImageResponse) ListCarServiceImagesResponse {
	return ListCarServiceImagesResponse{
		Items: items,
		Meta: ListMeta{
			Count:   len(items),
			HasNext: false,
		},
	}
}
