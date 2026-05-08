package domain

import (
	"time"

	"github.com/google/uuid"
)

type RepairRequestStatus string

const (
	RepairRequestStatusPending  RepairRequestStatus = "pending"
	RepairRequestStatusAccepted RepairRequestStatus = "accepted"
	RepairRequestStatusRejected RepairRequestStatus = "rejected"
	RepairRequestStatusCanceled RepairRequestStatus = "canceled"
)

func (s RepairRequestStatus) IsValid() bool {
	switch s {
	case RepairRequestStatusPending,
		RepairRequestStatusAccepted,
		RepairRequestStatusRejected,
		RepairRequestStatusCanceled:
		return true
	default:
		return false
	}
}

func (s RepairRequestStatus) IsTerminal() bool {
	switch s {
	case RepairRequestStatusAccepted,
		RepairRequestStatusRejected,
		RepairRequestStatusCanceled:
		return true
	default:
		return false
	}
}

type RepairRequest struct {
	ID                  uuid.UUID           `json:"id" db:"id"`
	UserID              uuid.UUID           `json:"user_id" db:"user_id"`
	AnalysisJobID       uuid.UUID           `json:"analysis_job_id" db:"analysis_job_id"`
	CarServiceProfileID uuid.UUID           `json:"car_service_profile_id" db:"car_service_profile_id"`
	Status              RepairRequestStatus `json:"status" db:"status"`

	RepairSummary   []RepairSummaryItem  `json:"repair_summary" db:"repair_summary"`
	ServiceEstimate []RepairEstimateItem `json:"service_estimate,omitempty" db:"service_estimate"`

	CustomerName    *string `json:"customer_name,omitempty" db:"customer_name"`
	CustomerPhone   *string `json:"customer_phone,omitempty" db:"customer_phone"`
	CustomerEmail   *string `json:"customer_email,omitempty" db:"customer_email"`
	CustomerComment *string `json:"customer_comment,omitempty" db:"customer_comment"`

	ServiceComment    *string  `json:"service_comment,omitempty" db:"service_comment"`
	EstimatedPriceMin *float64 `json:"estimated_price_min,omitempty" db:"estimated_price_min"`
	EstimatedPriceMax *float64 `json:"estimated_price_max,omitempty" db:"estimated_price_max"`

	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
	RespondedAt *time.Time `json:"responded_at,omitempty" db:"responded_at"`
}

type RepairRequestDetails struct {
	Request  *RepairRequest           `json:"request"`
	Analysis *AnalysisJob             `json:"analysis"`
	Images   []RepairRequestImageLink `json:"images"`
}

type RepairRequestImageLink struct {
	Index     int       `json:"index"`
	URL       string    `json:"url"`
	ExpiresAt time.Time `json:"expires_at"`
}

type RepairSummaryItem struct {
	PartName     string                    `json:"part_name"`
	PartNameRU   string                    `json:"part_name_ru,omitempty"`
	ParentName   string                    `json:"parent_name,omitempty"`
	ParentNameRU string                    `json:"parent_name_ru,omitempty"`
	IsPair       bool                      `json:"is_pair"`
	Side         string                    `json:"side,omitempty"`
	SideRU       string                    `json:"side_ru,omitempty"`
	DamageCount  int                       `json:"damage_count"`
	DamageTypes  []RepairDamageTypeSummary `json:"damage_types"`
}

type RepairDamageTypeSummary struct {
	Code   string `json:"code"`
	NameRU string `json:"name_ru,omitempty"`
	Count  int    `json:"count"`
}

type RepairEstimateItem struct {
	PartName     string   `json:"part_name"`
	PartNameRU   string   `json:"part_name_ru,omitempty"`
	ParentName   string   `json:"parent_name,omitempty"`
	ParentNameRU string   `json:"parent_name_ru,omitempty"`
	IsPair       bool     `json:"is_pair"`
	Side         string   `json:"side,omitempty"`
	SideRU       string   `json:"side_ru,omitempty"`
	DamageCode   string   `json:"damage_code"`
	DamageNameRU string   `json:"damage_name_ru,omitempty"`
	Quantity     int      `json:"quantity"`
	PriceMin     *float64 `json:"price_min,omitempty"`
	PriceMax     *float64 `json:"price_max,omitempty"`
	Comment      *string  `json:"comment,omitempty"`
}

type CreateRepairRequestInput struct {
	UserID              uuid.UUID
	AnalysisJobID       uuid.UUID
	CarServiceProfileID uuid.UUID
	CustomerName        *string
	CustomerPhone       *string
	CustomerEmail       *string
	CustomerComment     *string
}

type RespondRepairRequestInput struct {
	ID                uuid.UUID
	CarServiceUserID  uuid.UUID
	Status            RepairRequestStatus
	ServiceComment    *string
	ServiceEstimate   []RepairEstimateItem
	EstimatedPriceMin *float64
	EstimatedPriceMax *float64
}

type AcceptRepairRequestInput struct {
	ID                uuid.UUID
	CarServiceUserID  uuid.UUID
	ServiceComment    *string
	ServiceEstimate   []RepairEstimateItem
	EstimatedPriceMin *float64
	EstimatedPriceMax *float64
}

type RejectRepairRequestInput struct {
	ID               uuid.UUID
	CarServiceUserID uuid.UUID
	ServiceComment   string
}
