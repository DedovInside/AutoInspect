package domain

import (
	"time"

	"github.com/google/uuid"
)

type DamageType struct {
	Code      string    `json:"code" db:"code"`
	NameRU    string    `json:"name_ru" db:"name_ru"`
	IsActive  bool      `json:"is_active" db:"is_active"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

type PartCategory struct {
	Code      string    `json:"code" db:"code"`
	NameRU    string    `json:"name_ru" db:"name_ru"`
	IsPair    bool      `json:"is_pair" db:"is_pair"`
	IsActive  bool      `json:"is_active" db:"is_active"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

type CarServiceSpecialization struct {
	ID               uuid.UUID `json:"id" db:"id"`
	ProfileID        uuid.UUID `json:"profile_id" db:"profile_id"`
	DamageTypeCode   string    `json:"damage_type_code" db:"damage_type_code"`
	PartCategoryCode string    `json:"part_category_code" db:"part_category_code"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
}

type CarServiceSpecializationInput struct {
	DamageTypeCode   string
	PartCategoryCode string
}

type SpecializationOptions struct {
	DamageTypes    []*DamageType
	PartCategories []*PartCategory
}
