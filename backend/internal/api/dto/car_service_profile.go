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

type DamageTypeResponse struct {
	Code   string `json:"code"`
	NameRU string `json:"name_ru"`
}

type PartCategoryResponse struct {
	Code   string `json:"code"`
	NameRU string `json:"name_ru"`
	IsPair bool   `json:"is_pair"`
}

type SpecializationOptionsResponse struct {
	DamageTypes    []DamageTypeResponse   `json:"damage_types"`
	PartCategories []PartCategoryResponse `json:"part_categories"`
}

type CarServiceSpecializationRequest struct {
	DamageTypeCode   string `json:"damage_type_code" binding:"required,min=1,max=100"`
	PartCategoryCode string `json:"part_category_code" binding:"required,min=1,max=100"`
}

type ReplaceCarServiceSpecializationsRequest struct {
	Items []CarServiceSpecializationRequest `json:"items" binding:"required,dive"`
}

type CarServiceSpecializationResponse struct {
	ID               uuid.UUID `json:"id"`
	DamageTypeCode   string    `json:"damage_type_code"`
	PartCategoryCode string    `json:"part_category_code"`
	CreatedAt        time.Time `json:"created_at"`
}

type ListCarServiceSpecializationsResponse struct {
	Items []CarServiceSpecializationResponse `json:"items"`
	Meta  ListMeta                           `json:"meta"`
}

type ReplaceCarServiceSpecializationsResponse struct {
	Items []CarServiceSpecializationResponse `json:"items"`
	Meta  ListMeta                           `json:"meta"`
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

func ToSpecializationOptionsResponse(options *domain.SpecializationOptions) SpecializationOptionsResponse {
	if options == nil {
		return SpecializationOptionsResponse{}
	}

	return SpecializationOptionsResponse{
		DamageTypes:    ToDamageTypeResponseList(options.DamageTypes),
		PartCategories: ToPartCategoryResponseList(options.PartCategories),
	}
}

func ToDamageTypeResponseList(damageTypes []*domain.DamageType) []DamageTypeResponse {
	out := make([]DamageTypeResponse, 0, len(damageTypes))
	for _, damageType := range damageTypes {
		if damageType == nil {
			continue
		}
		out = append(out, DamageTypeResponse{
			Code:   damageType.Code,
			NameRU: damageType.NameRU,
		})
	}
	return out
}

func ToPartCategoryResponseList(partCategories []*domain.PartCategory) []PartCategoryResponse {
	out := make([]PartCategoryResponse, 0, len(partCategories))
	for _, partCategory := range partCategories {
		if partCategory == nil {
			continue
		}
		out = append(out, PartCategoryResponse{
			Code:   partCategory.Code,
			NameRU: partCategory.NameRU,
			IsPair: partCategory.IsPair,
		})
	}
	return out
}

func ToCarServiceSpecializationInputList(
	requests []CarServiceSpecializationRequest,
) []domain.CarServiceSpecializationInput {
	out := make([]domain.CarServiceSpecializationInput, 0, len(requests))
	for _, request := range requests {
		out = append(out, domain.CarServiceSpecializationInput{
			DamageTypeCode:   request.DamageTypeCode,
			PartCategoryCode: request.PartCategoryCode,
		})
	}
	return out
}

func ToCarServiceSpecializationResponseList(
	specializations []*domain.CarServiceSpecialization,
) []CarServiceSpecializationResponse {
	out := make([]CarServiceSpecializationResponse, 0, len(specializations))
	for _, specialization := range specializations {
		if specialization == nil {
			continue
		}
		out = append(out, CarServiceSpecializationResponse{
			ID:               specialization.ID,
			DamageTypeCode:   specialization.DamageTypeCode,
			PartCategoryCode: specialization.PartCategoryCode,
			CreatedAt:        specialization.CreatedAt,
		})
	}
	return out
}

func NewCarServiceSpecializationsResponse(
	items []CarServiceSpecializationResponse,
) ListCarServiceSpecializationsResponse {
	return ListCarServiceSpecializationsResponse{
		Items: items,
		Meta: ListMeta{
			Count:   len(items),
			HasNext: false,
		},
	}
}

func NewReplaceCarServiceSpecializationsResponse(
	items []CarServiceSpecializationResponse,
) ReplaceCarServiceSpecializationsResponse {
	return ReplaceCarServiceSpecializationsResponse{
		Items: items,
		Meta: ListMeta{
			Count:   len(items),
			HasNext: false,
		},
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
