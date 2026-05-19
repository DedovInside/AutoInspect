package dto

import (
	"testing"
	"time"

	"github.com/DedovInside/AutoInspect/backend/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestToAnalysisJobResponseMapsEnrichedResult(t *testing.T) {
	t.Parallel()

	jobID := uuid.New()
	userID := uuid.New()
	correlationID := uuid.New()
	idempotencyKey := "analysis-key"
	requestedAt := time.Date(2026, 5, 19, 12, 0, 0, 0, time.UTC)
	completedAt := requestedAt.Add(time.Second)
	job := &domain.AnalysisJob{
		ID:               jobID,
		UserID:           userID,
		IdempotencyKey:   &idempotencyKey,
		CarMake:          "Volkswagen",
		CarModel:         "Golf",
		CarGeneration:    "5",
		CarYear:          2015,
		ImageKeys:        []string{"uploads/user/0.jpg"},
		CorrelationID:    correlationID,
		Status:           domain.StatusCompleted,
		UsedModelVersion: "v1.3.0",
		RequestedAt:      requestedAt,
		CompletedAt:      &completedAt,
		Result: &domain.AnalysisResult{
			ModelID:      "general",
			ModelVersion: "v1.3.0",
			BatchID:      "batch-1",
			Results: []domain.ImageAnalysisResult{
				{
					ImageID:   "image-1",
					ImageURI:  "s3://autoinspect-uploads/uploads/user/0.jpg",
					ImageInfo: domain.ImageMeta{Width: 886, Height: 530},
					DamageInstances: []domain.DamageInstance{
						{
							ID:           "damage-1",
							DamageType:   "dent",
							DamageNameRU: "вмятина",
							Polygon:      [][]int{{1, 2}, {3, 4}, {5, 6}},
							BBox:         []int{1, 2, 5, 6},
							Confidence:   0.87,
							Parts: []domain.PartAssociation{
								{
									Name:         "front-fender",
									NameRU:       "переднее крыло",
									ParentName:   "fender",
									ParentNameRU: "крыло",
									IsPair:       true,
									Side:         "left",
									SideRU:       "слева",
									Confidence:   0.91,
								},
							},
						},
					},
					PartsSummary: []domain.PartSummary{
						{
							Name:         "front-fender",
							NameRU:       "переднее крыло",
							ParentName:   "fender",
							ParentNameRU: "крыло",
							IsPair:       true,
							Side:         "left",
							SideRU:       "слева",
							DamageCount:  2,
							DamageTypes: []domain.DamageTypeSummary{
								{Code: "dent", NameRU: "вмятина", Count: 2},
							},
						},
					},
				},
			},
		},
	}

	got := ToAnalysisJobResponse(job)

	require.Equal(t, jobID, got.ID)
	require.Equal(t, userID, got.UserID)
	require.Equal(t, idempotencyKey, *got.IdempotencyKey)
	require.Equal(t, "Volkswagen", got.CarMake)
	require.Equal(t, "Golf", got.CarModel)
	require.Equal(t, "5", got.CarGeneration)
	require.Equal(t, 2015, got.CarYear)
	require.Equal(t, 1, got.ImageCount)
	require.Equal(t, correlationID, got.CorrelationID)
	require.Equal(t, domain.StatusCompleted, got.Status)
	require.Equal(t, "v1.3.0", got.UsedModelVersion)
	require.Equal(t, requestedAt, got.RequestedAt)
	require.Equal(t, completedAt, *got.CompletedAt)

	require.NotNil(t, got.Result)
	require.Equal(t, "general", got.Result.ModelID)
	require.Len(t, got.Result.Results, 1)
	image := got.Result.Results[0]
	require.Equal(t, "image-1", image.ImageID)
	require.Equal(t, ImageMetaResponse{Width: 886, Height: 530}, image.ImageInfo)
	require.Len(t, image.DamageInstances, 1)
	require.Len(t, image.PartsSummary, 1)

	damage := image.DamageInstances[0]
	require.Equal(t, "damage-1", damage.ID)
	require.Equal(t, "dent", damage.DamageType)
	require.Equal(t, "вмятина", damage.DamageNameRU)
	require.Equal(t, [][]int{{1, 2}, {3, 4}, {5, 6}}, damage.Polygon)
	require.Equal(t, []int{1, 2, 5, 6}, damage.BBox)
	require.Len(t, damage.Parts, 1)
	require.Equal(t, "front-fender", damage.Parts[0].Name)
	require.Equal(t, "переднее крыло", damage.Parts[0].NameRU)
	require.Equal(t, "fender", damage.Parts[0].ParentName)

	summary := image.PartsSummary[0]
	require.Equal(t, "front-fender", summary.Name)
	require.Equal(t, "переднее крыло", summary.NameRU)
	require.Equal(t, "fender", summary.ParentName)
	require.Equal(t, 2, summary.DamageCount)
	require.Equal(t, []DamageTypeSummaryResponse{{Code: "dent", NameRU: "вмятина", Count: 2}}, summary.DamageTypes)
}

func TestListResponsePaginationMeta(t *testing.T) {
	t.Parallel()

	full := NewListResponse([]AnalysisJobResponse{{}, {}}, 2, 4)
	require.Equal(t, 2, full.Meta.Count)
	require.True(t, full.Meta.HasNext)
	require.NotNil(t, full.Meta.NextOffset)
	require.Equal(t, 6, *full.Meta.NextOffset)

	last := NewListResponse([]AnalysisJobResponse{{}}, 2, 4)
	require.Equal(t, 1, last.Meta.Count)
	require.False(t, last.Meta.HasNext)
	require.Nil(t, last.Meta.NextOffset)
}
