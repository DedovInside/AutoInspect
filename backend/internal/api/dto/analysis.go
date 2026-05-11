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
	Year       int    `form:"year" json:"year" binding:"required,min=1900,max=2100"`
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

type ListMatchedCarServicesResponse struct {
	Items  []MatchedCarServiceResponse `json:"items"`
	Limit  int                         `json:"limit"`
	Offset int                         `json:"offset"`
	Meta   ListMeta                    `json:"meta"`
}

type MatchedCarServiceResponse struct {
	ID               uuid.UUID                `json:"id"`
	OrganizationName string                   `json:"organization_name"`
	City             string                   `json:"city"`
	Address          string                   `json:"address"`
	Phone            *string                  `json:"phone,omitempty"`
	Email            *string                  `json:"email,omitempty"`
	WebsiteURL       *string                  `json:"website_url,omitempty"`
	ContactInfo      *string                  `json:"contact_info,omitempty"`
	Description      *string                  `json:"description,omitempty"`
	MatchCount       int                      `json:"match_count"`
	RequiredCount    int                      `json:"required_count"`
	Score            float64                  `json:"score"`
	PrimaryImage     *CarServiceImageResponse `json:"primary_image,omitempty"`
}

type AnalysisResultResponse struct {
	ModelID      string                        `json:"model_id"`
	ModelVersion string                        `json:"model_version"`
	BatchID      string                        `json:"batch_id"`
	Results      []ImageAnalysisResultResponse `json:"results"`
}

type ImageAnalysisResultResponse struct {
	ImageID         string                   `json:"image_id"`
	ImageURI        string                   `json:"image_uri"`
	ImageInfo       ImageMetaResponse        `json:"image"`
	DamageInstances []DamageInstanceResponse `json:"damage_instances"`
	PartsSummary    []PartSummaryResponse    `json:"parts_summary"`
}

type ImageMetaResponse struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

type DamageInstanceResponse struct {
	ID           string                    `json:"id"`
	DamageType   string                    `json:"damage_type"`
	DamageNameRU string                    `json:"damage_name_ru,omitempty"`
	Polygon      [][]int                   `json:"polygon"`
	BBox         []int                     `json:"bbox"`
	Confidence   float64                   `json:"confidence"`
	Parts        []PartAssociationResponse `json:"parts"`
}

type PartAssociationResponse struct {
	Name         string  `json:"name"`
	NameRU       string  `json:"name_ru,omitempty"`
	ParentName   string  `json:"parent_name,omitempty"`
	ParentNameRU string  `json:"parent_name_ru,omitempty"`
	IsPair       bool    `json:"is_pair"`
	Side         string  `json:"side,omitempty"`
	SideRU       string  `json:"side_ru,omitempty"`
	Confidence   float64 `json:"confidence"`
}

type PartSummaryResponse struct {
	Name         string                      `json:"name"`
	NameRU       string                      `json:"name_ru,omitempty"`
	ParentName   string                      `json:"parent_name,omitempty"`
	ParentNameRU string                      `json:"parent_name_ru,omitempty"`
	IsPair       bool                        `json:"is_pair"`
	Side         string                      `json:"side,omitempty"`
	SideRU       string                      `json:"side_ru,omitempty"`
	DamageCount  int                         `json:"damage_count"`
	DamageTypes  []DamageTypeSummaryResponse `json:"damage_types"`
}

type DamageTypeSummaryResponse struct {
	Code   string `json:"code"`
	NameRU string `json:"name_ru,omitempty"`
	Count  int    `json:"count"`
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

func NewMatchedCarServicesResponse(
	items []MatchedCarServiceResponse,
	limit, offset int,
) ListMatchedCarServicesResponse {
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

	return ListMatchedCarServicesResponse{
		Items:  items,
		Limit:  limit,
		Offset: offset,
		Meta:   meta,
	}
}

func ToMatchedCarServiceResponseList(
	matches []*domain.CarServiceMatchWithImageURL,
) []MatchedCarServiceResponse {
	out := make([]MatchedCarServiceResponse, 0, len(matches))
	for _, match := range matches {
		item := ToMatchedCarServiceResponse(match)
		if item.ID == uuid.Nil {
			continue
		}
		out = append(out, item)
	}

	return out
}

func ToMatchedCarServiceResponse(match *domain.CarServiceMatchWithImageURL) MatchedCarServiceResponse {
	if match == nil || match.Match == nil || match.Match.Profile == nil {
		return MatchedCarServiceResponse{}
	}

	profile := match.Match.Profile
	resp := MatchedCarServiceResponse{
		ID:               profile.ID,
		OrganizationName: profile.OrganizationName,
		City:             profile.City,
		Address:          profile.Address,
		Phone:            profile.Phone,
		Email:            profile.Email,
		WebsiteURL:       profile.WebsiteURL,
		ContactInfo:      profile.ContactInfo,
		Description:      profile.Description,
		MatchCount:       match.Match.MatchCount,
		RequiredCount:    match.Match.RequiredCount,
		Score:            match.Match.Score,
	}

	if match.Match.PrimaryImage != nil && match.PrimaryImageExpiresAt != nil {
		image := ToCarServiceImageResponse(match.Match.PrimaryImage, match.PrimaryImageURL, *match.PrimaryImageExpiresAt)
		resp.PrimaryImage = &image
	}

	return resp
}

func toAnalysisResultResponse(result *domain.AnalysisResult) *AnalysisResultResponse {
	if result == nil {
		return nil
	}

	out := &AnalysisResultResponse{
		ModelID:      result.ModelID,
		ModelVersion: result.ModelVersion,
		BatchID:      result.BatchID,
		Results:      make([]ImageAnalysisResultResponse, 0, len(result.Results)),
	}

	for i := range result.Results {
		out.Results = append(out.Results, toImageAnalysisResultResponse(&result.Results[i]))
	}

	return out
}

func toImageAnalysisResultResponse(result *domain.ImageAnalysisResult) ImageAnalysisResultResponse {
	out := ImageAnalysisResultResponse{
		ImageID:  result.ImageID,
		ImageURI: result.ImageURI,
		ImageInfo: ImageMetaResponse{
			Width:  result.ImageInfo.Width,
			Height: result.ImageInfo.Height,
		},
		DamageInstances: make([]DamageInstanceResponse, 0, len(result.DamageInstances)),
		PartsSummary:    make([]PartSummaryResponse, 0, len(result.PartsSummary)),
	}

	for i := range result.DamageInstances {
		out.DamageInstances = append(out.DamageInstances, toDamageInstanceResponse(&result.DamageInstances[i]))
	}

	for i := range result.PartsSummary {
		out.PartsSummary = append(out.PartsSummary, toPartSummaryResponse(&result.PartsSummary[i]))
	}

	return out
}

func toDamageInstanceResponse(damage *domain.DamageInstance) DamageInstanceResponse {
	out := DamageInstanceResponse{
		ID:           damage.ID,
		DamageType:   damage.DamageType,
		DamageNameRU: damage.DamageNameRU,
		Polygon:      damage.Polygon,
		BBox:         damage.BBox,
		Confidence:   damage.Confidence,
		Parts:        make([]PartAssociationResponse, 0, len(damage.Parts)),
	}
	for _, part := range damage.Parts {
		out.Parts = append(out.Parts, PartAssociationResponse{
			Name:         part.Name,
			NameRU:       part.NameRU,
			ParentName:   part.ParentName,
			ParentNameRU: part.ParentNameRU,
			IsPair:       part.IsPair,
			Side:         part.Side,
			SideRU:       part.SideRU,
			Confidence:   part.Confidence,
		})
	}

	return out
}

func toPartSummaryResponse(summary *domain.PartSummary) PartSummaryResponse {
	return PartSummaryResponse{
		Name:         summary.Name,
		NameRU:       summary.NameRU,
		ParentName:   summary.ParentName,
		ParentNameRU: summary.ParentNameRU,
		IsPair:       summary.IsPair,
		Side:         summary.Side,
		SideRU:       summary.SideRU,
		DamageCount:  summary.DamageCount,
		DamageTypes:  toDamageTypeSummaryResponseList(summary.DamageTypes),
	}
}

func toDamageTypeSummaryResponseList(items []domain.DamageTypeSummary) []DamageTypeSummaryResponse {
	out := make([]DamageTypeSummaryResponse, 0, len(items))
	for _, item := range items {
		out = append(out, DamageTypeSummaryResponse{
			Code:   item.Code,
			NameRU: item.NameRU,
			Count:  item.Count,
		})
	}

	return out
}
