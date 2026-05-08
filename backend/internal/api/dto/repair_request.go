package dto

import (
	"time"

	"github.com/DedovInside/AutoInspect/backend/internal/domain"
	"github.com/google/uuid"
)

type CreateRepairRequestRequest struct {
	AnalysisJobID       uuid.UUID `json:"analysis_job_id" binding:"required"`
	CarServiceProfileID uuid.UUID `json:"car_service_profile_id" binding:"required"`
	CustomerName        *string   `json:"customer_name" binding:"omitempty,max=255"`
	CustomerPhone       *string   `json:"customer_phone" binding:"omitempty,max=50"`
	CustomerEmail       *string   `json:"customer_email" binding:"omitempty,email,max=255"`
	CustomerComment     *string   `json:"customer_comment" binding:"omitempty,max=2000"`
}

type ListRepairRequestsQuery struct {
	Limit  int `form:"limit,default=20" binding:"omitempty,min=1,max=100"`
	Offset int `form:"offset,default=0" binding:"omitempty,min=0"`
}

type RepairRequestResponse struct {
	ID                  uuid.UUID                  `json:"id"`
	UserID              uuid.UUID                  `json:"user_id"`
	AnalysisJobID       uuid.UUID                  `json:"analysis_job_id"`
	CarServiceProfileID uuid.UUID                  `json:"car_service_profile_id"`
	Status              domain.RepairRequestStatus `json:"status"`
	RepairSummary       []RepairSummaryItem        `json:"repair_summary"`
	ServiceEstimate     []RepairEstimateItem       `json:"service_estimate,omitempty"`
	CustomerName        *string                    `json:"customer_name,omitempty"`
	CustomerPhone       *string                    `json:"customer_phone,omitempty"`
	CustomerEmail       *string                    `json:"customer_email,omitempty"`
	CustomerComment     *string                    `json:"customer_comment,omitempty"`
	ServiceComment      *string                    `json:"service_comment,omitempty"`
	EstimatedPriceMin   *float64                   `json:"estimated_price_min,omitempty"`
	EstimatedPriceMax   *float64                   `json:"estimated_price_max,omitempty"`
	CreatedAt           time.Time                  `json:"created_at"`
	UpdatedAt           time.Time                  `json:"updated_at"`
	RespondedAt         *time.Time                 `json:"responded_at,omitempty"`
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

type CreateRepairRequestResponse struct {
	Request RepairRequestResponse `json:"request"`
}

type GetRepairRequestResponse struct {
	Request RepairRequestResponse `json:"request"`
}

type ListRepairRequestsResponse struct {
	Items  []RepairRequestResponse `json:"items"`
	Limit  int                     `json:"limit"`
	Offset int                     `json:"offset"`
	Meta   ListMeta                `json:"meta"`
}

type CancelRepairRequestResponse struct {
	Status string `json:"status"`
}

func ToRepairRequestResponse(request *domain.RepairRequest) RepairRequestResponse {
	if request == nil {
		return RepairRequestResponse{}
	}

	return RepairRequestResponse{
		ID:                  request.ID,
		UserID:              request.UserID,
		AnalysisJobID:       request.AnalysisJobID,
		CarServiceProfileID: request.CarServiceProfileID,
		Status:              request.Status,
		RepairSummary:       ToRepairSummaryItemList(request.RepairSummary),
		ServiceEstimate:     ToRepairEstimateItemList(request.ServiceEstimate),
		CustomerName:        request.CustomerName,
		CustomerPhone:       request.CustomerPhone,
		CustomerEmail:       request.CustomerEmail,
		CustomerComment:     request.CustomerComment,
		ServiceComment:      request.ServiceComment,
		EstimatedPriceMin:   request.EstimatedPriceMin,
		EstimatedPriceMax:   request.EstimatedPriceMax,
		CreatedAt:           request.CreatedAt,
		UpdatedAt:           request.UpdatedAt,
		RespondedAt:         request.RespondedAt,
	}
}

func ToRepairRequestResponseList(requests []*domain.RepairRequest) []RepairRequestResponse {
	out := make([]RepairRequestResponse, 0, len(requests))
	for _, request := range requests {
		if request == nil {
			continue
		}
		out = append(out, ToRepairRequestResponse(request))
	}

	return out
}

func ToRepairSummaryItemList(items []domain.RepairSummaryItem) []RepairSummaryItem {
	out := make([]RepairSummaryItem, 0, len(items))
	for i := range items {
		out = append(out, ToRepairSummaryItem(&items[i]))
	}

	return out
}

func ToRepairSummaryItem(item *domain.RepairSummaryItem) RepairSummaryItem {
	if item == nil {
		return RepairSummaryItem{}
	}

	return RepairSummaryItem{
		PartName:     item.PartName,
		PartNameRU:   item.PartNameRU,
		ParentName:   item.ParentName,
		ParentNameRU: item.ParentNameRU,
		IsPair:       item.IsPair,
		Side:         item.Side,
		SideRU:       item.SideRU,
		DamageCount:  item.DamageCount,
		DamageTypes:  ToRepairDamageTypeSummaryList(item.DamageTypes),
	}
}

func ToRepairDamageTypeSummaryList(items []domain.RepairDamageTypeSummary) []RepairDamageTypeSummary {
	out := make([]RepairDamageTypeSummary, 0, len(items))
	for i := range items {
		out = append(out, RepairDamageTypeSummary{
			Code:   items[i].Code,
			NameRU: items[i].NameRU,
			Count:  items[i].Count,
		})
	}

	return out
}

func ToRepairEstimateItemList(items []domain.RepairEstimateItem) []RepairEstimateItem {
	out := make([]RepairEstimateItem, 0, len(items))
	for i := range items {
		out = append(out, ToRepairEstimateItem(&items[i]))
	}

	return out
}

func ToRepairEstimateItem(item *domain.RepairEstimateItem) RepairEstimateItem {
	if item == nil {
		return RepairEstimateItem{}
	}

	return RepairEstimateItem{
		PartName:     item.PartName,
		PartNameRU:   item.PartNameRU,
		ParentName:   item.ParentName,
		ParentNameRU: item.ParentNameRU,
		IsPair:       item.IsPair,
		Side:         item.Side,
		SideRU:       item.SideRU,
		DamageCode:   item.DamageCode,
		DamageNameRU: item.DamageNameRU,
		Quantity:     item.Quantity,
		PriceMin:     item.PriceMin,
		PriceMax:     item.PriceMax,
		Comment:      item.Comment,
	}
}

func NewRepairRequestListResponse(
	items []RepairRequestResponse,
	limit, offset int,
) ListRepairRequestsResponse {
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

	return ListRepairRequestsResponse{
		Items:  items,
		Limit:  limit,
		Offset: offset,
		Meta:   meta,
	}
}
