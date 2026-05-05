package dto

import (
	"time"

	"github.com/DedovInside/AutoInspect/backend/internal/domain"
	"github.com/google/uuid"
)

type UploadModelArtifactsRequest struct {
	Make        string `form:"make" binding:"required,min=1,max=100"`
	Model       string `form:"model" binding:"required,min=1,max=100"`
	Generation  string `form:"generation" binding:"max=100"`
	YearFrom    int    `form:"year_from" binding:"required,min=1900,max=2100"`
	YearTo      int    `form:"year_to" binding:"omitempty,min=1900,max=2100"`
	Version     string `form:"version" binding:"required,min=1,max=50"`
	IsUniversal bool   `form:"is_universal"`
}

type ListModelsQuery struct {
	Limit  int `form:"limit,default=20" binding:"omitempty,min=1,max=100"`
	Offset int `form:"offset,default=0" binding:"omitempty,min=0"`
}

type CarModelResponse struct {
	ID                uuid.UUID `json:"id"`
	Make              string    `json:"make"`
	Model             string    `json:"model"`
	Generation        string    `json:"generation"`
	YearFrom          int       `json:"year_from"`
	YearTo            int       `json:"year_to,omitempty"`
	PartsModelS3Key   string    `json:"parts_model_s3_key"`
	PartsConfigS3Key  string    `json:"parts_config_s3_key"`
	PartsCatalogS3Key string    `json:"parts_catalog_s3_key"`
	ModelVersion      string    `json:"model_version"`
	IsUniversal       bool      `json:"is_universal"`
	IsActive          bool      `json:"is_active"`
	CreatedAt         time.Time `json:"created_at"`
}

type UploadModelArtifactsResponse struct {
	Model CarModelResponse `json:"model"`
}

type ListModelsResponse struct {
	Items  []CarModelResponse `json:"items"`
	Limit  int                `json:"limit"`
	Offset int                `json:"offset"`
	Meta   ListMeta           `json:"meta"`
}

func ToCarModelResponse(model *domain.CarModel) CarModelResponse {
	if model == nil {
		return CarModelResponse{}
	}
	return CarModelResponse{
		ID:                model.ID,
		Make:              model.Make,
		Model:             model.Model,
		Generation:        model.Generation,
		YearFrom:          model.YearFrom,
		YearTo:            model.YearTo,
		PartsModelS3Key:   model.PartsModelS3Key,
		PartsConfigS3Key:  model.PartsConfigS3Key,
		PartsCatalogS3Key: model.PartsCatalogS3Key,
		ModelVersion:      model.ModelVersion,
		IsUniversal:       model.IsUniversal,
		IsActive:          model.IsActive,
		CreatedAt:         model.CreatedAt,
	}
}

func ToCarModelResponseList(models []*domain.CarModel) []CarModelResponse {
	out := make([]CarModelResponse, 0, len(models))
	for _, model := range models {
		if model == nil {
			continue
		}
		out = append(out, ToCarModelResponse(model))
	}
	return out
}

func NewModelListResponse(items []CarModelResponse, limit, offset int) ListModelsResponse {
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

	return ListModelsResponse{
		Items:  items,
		Limit:  limit,
		Offset: offset,
		Meta:   meta,
	}
}
