package service

import (
	"testing"

	"github.com/DedovInside/AutoInspect/backend/internal/broker"
	"github.com/DedovInside/AutoInspect/backend/internal/domain"
	analysisv1 "github.com/DedovInside/AutoInspect/backend/internal/proto/gen/go/analysis/v1"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestAnalysisRequestProtoContractRoundTrip(t *testing.T) {
	t.Parallel()

	job := &domain.AnalysisJob{
		UserID:        uuid.New(),
		CorrelationID: uuid.New(),
		CarMake:       "Volkswagen",
		CarModel:      "Golf",
		CarGeneration: "5",
		CarYear:       2015,
		ImageKeys:     []string{"uploads/user/0.jpg"},
	}
	model := &domain.CarModel{
		PartsModelS3Key:  "models/general/parts_segmentation.pt",
		PartsConfigS3Key: "models/general/parts_inference_config.json",
	}

	request := DomainToProtoRequest(job, model)
	payload, err := proto.Marshal(request)
	require.NoError(t, err)
	require.NotEmpty(t, payload)

	var decoded analysisv1.AnalysisRequest
	require.NoError(t, proto.Unmarshal(payload, &decoded))

	require.Equal(t, job.CorrelationID.String(), decoded.GetCorrelationId())
	require.Equal(t, job.UserID.String(), decoded.GetUserId())
	require.Equal(t, job.ImageKeys, decoded.GetImageS3Keys())
	require.Equal(t, "models/general/parts_segmentation.pt", decoded.GetPartsModelS3Key())
	require.Equal(t, "models/general/parts_inference_config.json", decoded.GetPartsConfigS3Key())
	require.Equal(t, "Volkswagen", decoded.GetCarInfo().GetMake())
	require.Equal(t, "Golf", decoded.GetCarInfo().GetModel())
	require.Equal(t, "5", decoded.GetCarInfo().GetGeneration())
	require.Equal(t, int32(2015), decoded.GetCarInfo().GetYear())
}

func TestAnalysisResultProtoContractParsesCompletedAndFailedMessages(t *testing.T) {
	t.Parallel()

	correlationID := uuid.New()
	completedPayload, err := proto.Marshal(&analysisv1.AnalysisResult{
		CorrelationId: correlationID.String(),
		Status:        "completed",
		ModelId:       "general",
		ModelVersion:  "v1.3.0",
		BatchId:       "batch-1",
		Results: []*analysisv1.ImageAnalysisResult{
			{
				ImageId:  "image-1",
				ImageUri: "s3://autoinspect-uploads/uploads/user/0.jpg",
				Image:    &analysisv1.ImageInfo{Width: 886, Height: 530},
				DamageInstances: []*analysisv1.DamageInstance{
					{
						Id:         "damage-1",
						DamageType: "dent",
						Bbox:       &analysisv1.BBox{XMin: 1, YMin: 2, XMax: 3, YMax: 4},
						Parts: []*analysisv1.PartAssociation{
							{Name: "hood", Confidence: 0.9},
						},
					},
				},
				PartsSummary: []*analysisv1.PartSummary{
					{Name: "hood", DamageCount: 1, DamageTypes: map[string]int32{"dent": 1}},
				},
			},
		},
	})
	require.NoError(t, err)

	parsed, parsedCorrelationID, err := parseAnalysisResultMessage(broker.Message{
		Key:   []byte(correlationID.String()),
		Value: completedPayload,
	})
	require.NoError(t, err)
	require.Equal(t, correlationID, parsedCorrelationID)
	require.Equal(t, "completed", parsed.GetStatus())
	require.Equal(t, "general", parsed.GetModelId())
	require.Len(t, parsed.GetResults(), 1)

	failedPayload, err := proto.Marshal(&analysisv1.AnalysisResult{
		CorrelationId: correlationID.String(),
		Status:        "failed",
		ErrorMessage:  "artifact not found",
	})
	require.NoError(t, err)

	failed, failedCorrelationID, err := parseAnalysisResultMessage(broker.Message{
		Value: failedPayload,
	})
	require.NoError(t, err)
	require.Equal(t, correlationID, failedCorrelationID)
	require.Equal(t, "failed", failed.GetStatus())
	require.Equal(t, "artifact not found", failed.GetErrorMessage())
}
