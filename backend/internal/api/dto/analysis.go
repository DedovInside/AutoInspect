package dto

import (
	"time"

	"github.com/DedovInside/AutoInspect/backend/internal/domain"
	"github.com/google/uuid"
)

type CreateAnalysisRequest struct {
	Make       string `form:"make" json:"make" binding:"required,min=1,max=100"`
	Model      string `form:"model" json:"model" binding:"required,min=1,max=100"`
	Generation string `form:"generation" json:"generation" binding:"max=100"`
	Year       int    `form:"year" json:"year" binding:"omitempty,min=1900,max=2100"`
}

type CreateAnalysisResponse struct {
	Job AnalysisJobResponse `json:"job"`
}

type AnalysisJobResponse struct {
	ID               uuid.UUID               `json:"id"`
	UserID           uuid.UUID               `json:"user_id"`
	IdempotencyKey   *string                 `json:"idempotency_key,omitempty"`
	CarMake          string                  `json:"car_make"`
	CarModel         string                  `json:"car_model"`
	CarGeneration    string                  `json:"car_generation"`
	CarYear          int                     `json:"car_year"`
	ImageCount       int                     `json:"image_count"`
	CorrelationID    uuid.UUID               `json:"-"`
	Status           domain.JobStatus        `json:"status"`
	ErrorMessage     *string                 `json:"error_message,omitempty"`
	UsedModelVersion string                  `json:"used_model_version,omitempty"`
	RequestedAt      time.Time               `json:"requested_at"`
	StartedAt        *time.Time              `json:"started_at,omitempty"`
	CompletedAt      *time.Time              `json:"completed_at,omitempty"`
	Result           *AnalysisResultResponse `json:"result,omitempty"`
}

type ListResponse struct {
	Items  []AnalysisJobResponse `json:"items"`
	Limit  int                   `json:"limit"`
	Offset int                   `json:"offset"`
	Meta   ListMeta              `json:"meta"`
}

type ListMeta struct {
	Count      int  `json:"count"`
	HasNext    bool `json:"has_next"`
	NextOffset *int `json:"next_offset,omitempty"`
}

type ListAnalysisQuery struct {
	Limit  int `form:"limit,default=20" binding:"omitempty,min=1,max=100"`
	Offset int `form:"offset,default=0" binding:"omitempty,min=0"`
}

type PresignedImageURLResponse struct {
	URL       string    `json:"url"`
	ExpiresAt time.Time `json:"expires_at"`
}

type AnalysisResultResponse struct {
	ImageInfo       ImageMetaResponse        `json:"image"`
	ModelVersion    string                   `json:"model_version"`
	DamageInstances []DamageInstanceResponse `json:"damage_instances"`
	PartsSummary    []PartSummaryResponse    `json:"parts_summary"`
}

type ImageMetaResponse struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

type DamageInstanceResponse struct {
	ID         string                    `json:"id"`
	DamageType string                    `json:"damage_type"`
	Polygon    [][]int                   `json:"polygon"`
	BBox       []int                     `json:"bbox"`
	Confidence float64                   `json:"confidence"`
	Parts      []PartAssociationResponse `json:"parts"`
}

type PartAssociationResponse struct {
	PartName   string  `json:"part_name"`
	Confidence float64 `json:"confidence"`
}

type PartSummaryResponse struct {
	PartName    string         `json:"part_name"`
	DamageCount int            `json:"damage_count"`
	DamageTypes map[string]int `json:"damage_types"`
}

func ToAnalysisJobResponse(job *domain.AnalysisJob) AnalysisJobResponse {
	resp := AnalysisJobResponse{
		ID:               job.ID,
		UserID:           job.UserID,
		IdempotencyKey:   job.IdempotencyKey,
		CarMake:          job.CarMake,
		CarModel:         job.CarModel,
		CarGeneration:    job.CarGeneration,
		CarYear:          job.CarYear,
		ImageCount:       len(job.ImageKeys),
		CorrelationID:    job.CorrelationID,
		Status:           job.Status,
		ErrorMessage:     job.ErrorMessage,
		UsedModelVersion: job.UsedModelVersion,
		RequestedAt:      job.RequestedAt,
		StartedAt:        job.StartedAt,
		CompletedAt:      job.CompletedAt,
		Result:           toAnalysisResultResponse(job.Result),
	}
	return resp
}

func ToAnalysisJobResponseList(jobs []*domain.AnalysisJob) []AnalysisJobResponse {
	out := make([]AnalysisJobResponse, 0, len(jobs))
	for _, j := range jobs {
		if j == nil {
			continue
		}
		out = append(out, ToAnalysisJobResponse(j))
	}
	return out
}

func NewListResponse(items []AnalysisJobResponse, limit, offset int) ListResponse {
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

	return ListResponse{
		Items:  items,
		Limit:  limit,
		Offset: offset,
		Meta:   meta,
	}
}

func toAnalysisResultResponse(result *domain.AnalysisResult) *AnalysisResultResponse {
	if result == nil {
		return nil
	}

	out := &AnalysisResultResponse{
		ImageInfo: ImageMetaResponse{
			Width:  result.ImageInfo.Width,
			Height: result.ImageInfo.Height,
		},
		ModelVersion:    result.ModelVersion,
		DamageInstances: make([]DamageInstanceResponse, 0, len(result.DamageInstances)),
		PartsSummary:    make([]PartSummaryResponse, 0, len(result.PartsSummary)),
	}

	for _, d := range result.DamageInstances {
		dto := DamageInstanceResponse{
			ID:         d.ID,
			DamageType: d.DamageType,
			Polygon:    d.Polygon,
			BBox:       d.BBox,
			Confidence: d.Confidence,
			Parts:      make([]PartAssociationResponse, 0, len(d.Parts)),
		}

		for _, p := range d.Parts {
			dto.Parts = append(dto.Parts, PartAssociationResponse{
				PartName:   p.PartName,
				Confidence: p.Confidence,
			})
		}

		out.DamageInstances = append(out.DamageInstances, dto)
	}

	for _, s := range result.PartsSummary {
		out.PartsSummary = append(out.PartsSummary, PartSummaryResponse{
			PartName:    s.PartName,
			DamageCount: s.DamageCount,
			DamageTypes: s.DamageTypes,
		})
	}

	return out
}
