package dto

import (
	"time"

	"github.com/DedovInside/AutoInspect/backend/internal/domain"
	"github.com/google/uuid"
)

type CreateModelTrainingRequestRequest struct {
	Make        string `json:"make" binding:"required,min=1,max=100"`
	Model       string `json:"model" binding:"required,min=1,max=100"`
	Generation  string `json:"generation" binding:"max=100"`
	YearFrom    int    `json:"year_from" binding:"required,min=1900,max=2100"`
	YearTo      int    `json:"year_to" binding:"omitempty,min=1900,max=2100"`
	Description string `json:"description" binding:"required,min=1,max=2000"`
}

type ListModelTrainingRequestsQuery struct {
	Limit  int `form:"limit,default=20" binding:"omitempty,min=1,max=100"`
	Offset int `form:"offset,default=0" binding:"omitempty,min=0"`
}

type AdminListModelTrainingRequestsQuery struct {
	Status string `form:"status" binding:"omitempty,oneof=pending approved rejected in_progress completed"`
	Limit  int    `form:"limit,default=20" binding:"omitempty,min=1,max=100"`
	Offset int    `form:"offset,default=0" binding:"omitempty,min=0"`
}

type UpdateModelTrainingRequestStatusRequest struct {
	Status         domain.ModelTrainingRequestStatus `json:"status" binding:"required,oneof=pending approved rejected in_progress completed"`
	AdminComment   string                            `json:"admin_comment" binding:"max=2000"`
	CreatedModelID *uuid.UUID                        `json:"created_model_id,omitempty"`
}

type ModelTrainingRequestResponse struct {
	ID              uuid.UUID                         `json:"id"`
	InitiatorUserID uuid.UUID                         `json:"initiator_user_id"`
	InitiatorRole   domain.Role                       `json:"initiator_role"`
	Make            string                            `json:"make"`
	Model           string                            `json:"model"`
	Generation      string                            `json:"generation"`
	YearFrom        int                               `json:"year_from"`
	YearTo          int                               `json:"year_to,omitempty"`
	Description     string                            `json:"description"`
	Status          domain.ModelTrainingRequestStatus `json:"status"`
	AdminComment    string                            `json:"admin_comment,omitempty"`
	ReviewedBy      *uuid.UUID                        `json:"reviewed_by,omitempty"`
	ReviewedAt      *time.Time                        `json:"reviewed_at,omitempty"`
	CreatedModelID  *uuid.UUID                        `json:"created_model_id,omitempty"`
	CreatedAt       time.Time                         `json:"created_at"`
	UpdatedAt       time.Time                         `json:"updated_at"`
}

type CreateModelTrainingRequestResponse struct {
	Request ModelTrainingRequestResponse `json:"request"`
}

type ListModelTrainingRequestsResponse struct {
	Items  []ModelTrainingRequestResponse `json:"items"`
	Limit  int                            `json:"limit"`
	Offset int                            `json:"offset"`
	Meta   ListMeta                       `json:"meta"`
}

func ToModelTrainingRequestResponse(request *domain.ModelTrainingRequest) ModelTrainingRequestResponse {
	if request == nil {
		return ModelTrainingRequestResponse{}
	}

	return ModelTrainingRequestResponse{
		ID:              request.ID,
		InitiatorUserID: request.InitiatorUserID,
		InitiatorRole:   request.InitiatorRole,
		Make:            request.Make,
		Model:           request.Model,
		Generation:      request.Generation,
		YearFrom:        request.YearFrom,
		YearTo:          request.YearTo,
		Description:     request.Description,
		Status:          request.Status,
		AdminComment:    request.AdminComment,
		ReviewedBy:      request.ReviewedBy,
		ReviewedAt:      request.ReviewedAt,
		CreatedModelID:  request.CreatedModelID,
		CreatedAt:       request.CreatedAt,
		UpdatedAt:       request.UpdatedAt,
	}
}

func ToModelTrainingRequestResponseList(requests []*domain.ModelTrainingRequest) []ModelTrainingRequestResponse {
	out := make([]ModelTrainingRequestResponse, 0, len(requests))

	for _, request := range requests {
		if request == nil {
			continue
		}
		out = append(out, ToModelTrainingRequestResponse(request))
	}

	return out
}

func NewModelTrainingRequestListResponse(
	items []ModelTrainingRequestResponse,
	limit, offset int,
) ListModelTrainingRequestsResponse {
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

	return ListModelTrainingRequestsResponse{
		Items:  items,
		Limit:  limit,
		Offset: offset,
		Meta:   meta,
	}
}
