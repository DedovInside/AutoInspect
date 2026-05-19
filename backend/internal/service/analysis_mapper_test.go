package service

import (
	"math"
	"testing"

	"github.com/DedovInside/AutoInspect/backend/internal/domain"
	analysisv1 "github.com/DedovInside/AutoInspect/backend/internal/proto/gen/go/analysis/v1"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestDomainToProtoRequestMapsAnalysisJobAndModel(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	correlationID := uuid.New()
	job := &domain.AnalysisJob{
		UserID:        userID,
		CorrelationID: correlationID,
		CarMake:       "Volkswagen",
		CarModel:      "Golf",
		CarGeneration: "5",
		CarYear:       2015,
		ImageKeys:     []string{"uploads/user/0.jpg", "uploads/user/1.jpg"},
	}
	model := &domain.CarModel{
		PartsModelS3Key:  "models/general/parts_segmentation.pt",
		PartsConfigS3Key: "models/general/parts_inference_config.json",
	}

	got := DomainToProtoRequest(job, model)

	require.NotNil(t, got)
	require.Equal(t, correlationID.String(), got.GetCorrelationId())
	require.Equal(t, userID.String(), got.GetUserId())
	require.Equal(t, []string{"uploads/user/0.jpg", "uploads/user/1.jpg"}, got.GetImageS3Keys())
	require.Equal(t, "models/general/parts_segmentation.pt", got.GetPartsModelS3Key())
	require.Equal(t, "models/general/parts_inference_config.json", got.GetPartsConfigS3Key())
	require.Equal(t, "Volkswagen", got.GetCarInfo().GetMake())
	require.Equal(t, "Golf", got.GetCarInfo().GetModel())
	require.Equal(t, "5", got.GetCarInfo().GetGeneration())
	require.Equal(t, int32(2015), got.GetCarInfo().GetYear())
}

func TestDomainToProtoRequestReturnsNilForMissingInput(t *testing.T) {
	t.Parallel()

	require.Nil(t, DomainToProtoRequest(nil, &domain.CarModel{}))
	require.Nil(t, DomainToProtoRequest(&domain.AnalysisJob{}, nil))
}

func TestDomainToProtoRequestClampsYearOutsideInt32(t *testing.T) {
	t.Parallel()

	got := DomainToProtoRequest(&domain.AnalysisJob{
		UserID:        uuid.New(),
		CorrelationID: uuid.New(),
		CarYear:       math.MaxInt32 + 1,
	}, &domain.CarModel{})

	require.NotNil(t, got)
	require.Zero(t, got.GetCarInfo().GetYear())
}

func TestProtoToDomainResultMapsCompletedAnalysisPayload(t *testing.T) {
	t.Parallel()

	protoResult := &analysisv1.AnalysisResult{
		ModelId:      "general",
		ModelVersion: "v1.3.0",
		BatchId:      "batch-1",
		Results: []*analysisv1.ImageAnalysisResult{
			nil,
			{
				ImageId:  "image-1",
				ImageUri: "s3://autoinspect-uploads/uploads/user/0.jpg",
				Image:    &analysisv1.ImageInfo{Width: 886, Height: 530},
				DamageInstances: []*analysisv1.DamageInstance{
					nil,
					{
						Id:         "image-1-damage-1",
						DamageType: "Dent",
						Polygon: []*analysisv1.Point{
							{X: 10, Y: 20},
							nil,
							{X: 30, Y: 40},
						},
						Bbox:       &analysisv1.BBox{XMin: 9, YMin: 19, XMax: 31, YMax: 41},
						Confidence: 0.87,
						Parts: []*analysisv1.PartAssociation{
							nil,
							{Name: "Hood", Side: "left", Confidence: 0.92},
						},
					},
				},
				PartsSummary: []*analysisv1.PartSummary{
					nil,
					{
						Name:        "Hood",
						Side:        "left",
						DamageCount: 2,
						DamageTypes: map[string]int32{
							"scratch": 1,
							"dent":    2,
						},
					},
				},
			},
		},
	}

	got := ProtoToDomainResult(protoResult)

	require.NotNil(t, got)
	require.Equal(t, "general", got.ModelID)
	require.Equal(t, "v1.3.0", got.ModelVersion)
	require.Equal(t, "batch-1", got.BatchID)
	require.Len(t, got.Results, 1)

	image := got.Results[0]
	require.Equal(t, "image-1", image.ImageID)
	require.Equal(t, "s3://autoinspect-uploads/uploads/user/0.jpg", image.ImageURI)
	require.Equal(t, domain.ImageMeta{Width: 886, Height: 530}, image.ImageInfo)
	require.Len(t, image.DamageInstances, 1)
	require.Len(t, image.PartsSummary, 1)

	damage := image.DamageInstances[0]
	require.Equal(t, "image-1-damage-1", damage.ID)
	require.Equal(t, "Dent", damage.DamageType)
	require.InEpsilon(t, 0.87, damage.Confidence, 0.0001)
	require.Equal(t, []int{9, 19, 31, 41}, damage.BBox)
	require.Equal(t, [][]int{{10, 20}, {30, 40}}, damage.Polygon)
	require.Len(t, damage.Parts, 1)
	require.Equal(t, domain.PartAssociation{Name: "Hood", Side: "left", Confidence: float64(float32(0.92))}, damage.Parts[0])

	summary := image.PartsSummary[0]
	require.Equal(t, "Hood", summary.Name)
	require.Equal(t, "left", summary.Side)
	require.Equal(t, 2, summary.DamageCount)
	require.Equal(t, []domain.DamageTypeSummary{
		{Code: "dent", Count: 2},
		{Code: "scratch", Count: 1},
	}, summary.DamageTypes)
}

func TestProtoToDomainResultReturnsNilForNilProto(t *testing.T) {
	t.Parallel()

	require.Nil(t, ProtoToDomainResult(nil))
}
