package domain

import (
	"time"

	"github.com/google/uuid"
)

type CarServiceProfile struct {
	ID               uuid.UUID `json:"id" db:"id"`
	UserID           uuid.UUID `json:"user_id" db:"user_id"`
	OrganizationName string    `json:"organization_name" db:"organization_name"`
	City             string    `json:"city" db:"city"`
	Address          string    `json:"address" db:"address"`
	Phone            *string   `json:"phone,omitempty" db:"phone"`
	Email            *string   `json:"email,omitempty" db:"email"`
	WebsiteURL       *string   `json:"website_url,omitempty" db:"website_url"`
	ContactInfo      *string   `json:"contact_info,omitempty" db:"contact_info"`
	Description      *string   `json:"description,omitempty" db:"description"`
	IsActive         bool      `json:"is_active" db:"is_active"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time `json:"updated_at" db:"updated_at"`
}

type CreateCarServiceProfileInput struct {
	UserID           uuid.UUID
	OrganizationName string
	City             string
	Address          string
	Phone            *string
	Email            *string
	WebsiteURL       *string
	ContactInfo      *string
	Description      *string
}

type UpdateCarServiceProfileInput struct {
	UserID           uuid.UUID
	OrganizationName string
	City             string
	Address          string
	Phone            *string
	Email            *string
	WebsiteURL       *string
	ContactInfo      *string
	Description      *string
	IsActive         bool
}

type CarServiceMatchCriterion struct {
	DamageTypeCode   string
	PartCategoryCode string
}

type CarServiceMatch struct {
	Profile       *CarServiceProfile
	PrimaryImage  *CarServiceImage
	Images        []*CarServiceImage
	MatchCount    int
	RequiredCount int
	Score         float64
}

type CarServiceMatchWithImageURL struct {
	Match                 *CarServiceMatch
	PrimaryImageURL       string
	PrimaryImageExpiresAt *time.Time
	ImageURLs             []CarServiceImageURL
}

type CarServiceImageURL struct {
	Image     *CarServiceImage
	URL       string
	ExpiresAt time.Time
}
