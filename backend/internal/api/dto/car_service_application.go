package dto

import (
	"time"

	"github.com/DedovInside/AutoInspect/backend/internal/domain"
	"github.com/google/uuid"
)

type CreateCarServiceApplicationRequest struct {
	OrganizationName string  `json:"organization_name" binding:"required,min=1,max=255"`
	City             string  `json:"city" binding:"required,min=1,max=100"`
	Address          string  `json:"address" binding:"required,min=1,max=255"`
	Phone            *string `json:"phone" binding:"omitempty,max=50"`
	Email            *string `json:"email" binding:"omitempty,email,max=255"`
	ContactInfo      *string `json:"contact_info" binding:"omitempty,max=2000"`
	Description      string  `json:"description" binding:"required,min=1,max=2000"`
}

type ListCarServiceApplicationsQuery struct {
	Limit  int `form:"limit,default=20" binding:"omitempty,min=1,max=100"`
	Offset int `form:"offset,default=0" binding:"omitempty,min=0"`
}

type AdminListCarServiceApplicationsQuery struct {
	Status string `form:"status" binding:"omitempty,oneof=pending approved rejected"`
	Limit  int    `form:"limit,default=20" binding:"omitempty,min=1,max=100"`
	Offset int    `form:"offset,default=0" binding:"omitempty,min=0"`
}

type RejectCarServiceApplicationRequest struct {
	RejectionReason string `json:"rejection_reason" binding:"required,min=1,max=2000"`
}

type CarServiceApplicationResponse struct {
	ID               uuid.UUID                          `json:"id"`
	UserID           uuid.UUID                          `json:"user_id"`
	OrganizationName string                             `json:"organization_name"`
	City             string                             `json:"city"`
	Address          string                             `json:"address"`
	Phone            *string                            `json:"phone,omitempty"`
	Email            *string                            `json:"email,omitempty"`
	ContactInfo      *string                            `json:"contact_info,omitempty"`
	Description      string                             `json:"description"`
	Status           domain.CarServiceApplicationStatus `json:"status"`
	RejectionReason  *string                            `json:"rejection_reason,omitempty"`
	ReviewedBy       *uuid.UUID                         `json:"reviewed_by,omitempty"`
	ReviewedAt       *time.Time                         `json:"reviewed_at,omitempty"`
	CreatedProfileID *uuid.UUID                         `json:"created_profile_id,omitempty"`
	CreatedAt        time.Time                          `json:"created_at"`
	UpdatedAt        time.Time                          `json:"updated_at"`
}

type CarServiceProfileResponse struct {
	ID               uuid.UUID `json:"id"`
	UserID           uuid.UUID `json:"user_id"`
	OrganizationName string    `json:"organization_name"`
	City             string    `json:"city"`
	Address          string    `json:"address"`
	Phone            *string   `json:"phone,omitempty"`
	Email            *string   `json:"email,omitempty"`
	WebsiteURL       *string   `json:"website_url,omitempty"`
	ContactInfo      *string   `json:"contact_info,omitempty"`
	Description      *string   `json:"description,omitempty"`
	IsActive         bool      `json:"is_active"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type CreateCarServiceApplicationResponse struct {
	Application CarServiceApplicationResponse `json:"application"`
}

type GetCarServiceApplicationResponse struct {
	Application CarServiceApplicationResponse `json:"application"`
}

type ApproveCarServiceApplicationResponse struct {
	Application CarServiceApplicationResponse `json:"application"`
	Profile     CarServiceProfileResponse     `json:"profile"`
}

type RejectCarServiceApplicationResponse struct {
	Application CarServiceApplicationResponse `json:"application"`
}

type ListCarServiceApplicationsResponse struct {
	Items  []CarServiceApplicationResponse `json:"items"`
	Limit  int                             `json:"limit"`
	Offset int                             `json:"offset"`
	Meta   ListMeta                        `json:"meta"`
}

func ToCarServiceApplicationResponse(application *domain.CarServiceApplication) CarServiceApplicationResponse {
	if application == nil {
		return CarServiceApplicationResponse{}
	}

	return CarServiceApplicationResponse{
		ID:               application.ID,
		UserID:           application.UserID,
		OrganizationName: application.OrganizationName,
		City:             application.City,
		Address:          application.Address,
		Phone:            application.Phone,
		Email:            application.Email,
		ContactInfo:      application.ContactInfo,
		Description:      application.Description,
		Status:           application.Status,
		RejectionReason:  application.RejectionReason,
		ReviewedBy:       application.ReviewedBy,
		ReviewedAt:       application.ReviewedAt,
		CreatedProfileID: application.CreatedProfileID,
		CreatedAt:        application.CreatedAt,
		UpdatedAt:        application.UpdatedAt,
	}
}

func ToCarServiceProfileResponse(profile *domain.CarServiceProfile) CarServiceProfileResponse {
	if profile == nil {
		return CarServiceProfileResponse{}
	}

	return CarServiceProfileResponse{
		ID:               profile.ID,
		UserID:           profile.UserID,
		OrganizationName: profile.OrganizationName,
		City:             profile.City,
		Address:          profile.Address,
		Phone:            profile.Phone,
		Email:            profile.Email,
		WebsiteURL:       profile.WebsiteURL,
		ContactInfo:      profile.ContactInfo,
		Description:      profile.Description,
		IsActive:         profile.IsActive,
		CreatedAt:        profile.CreatedAt,
		UpdatedAt:        profile.UpdatedAt,
	}
}

func ToCarServiceApplicationResponseList(
	applications []*domain.CarServiceApplication,
) []CarServiceApplicationResponse {
	out := make([]CarServiceApplicationResponse, 0, len(applications))

	for _, application := range applications {
		if application == nil {
			continue
		}
		out = append(out, ToCarServiceApplicationResponse(application))
	}

	return out
}

func NewCarServiceApplicationListResponse(
	items []CarServiceApplicationResponse,
	limit, offset int,
) ListCarServiceApplicationsResponse {
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

	return ListCarServiceApplicationsResponse{
		Items:  items,
		Limit:  limit,
		Offset: offset,
		Meta:   meta,
	}
}
