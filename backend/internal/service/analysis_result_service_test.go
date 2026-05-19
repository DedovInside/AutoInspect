package service

import (
	"context"
	"testing"
	"time"

	"github.com/DedovInside/AutoInspect/backend/internal/broker"
	"github.com/DedovInside/AutoInspect/backend/internal/config"
	"github.com/DedovInside/AutoInspect/backend/internal/domain"
	"github.com/DedovInside/AutoInspect/backend/internal/notify"
	analysisv1 "github.com/DedovInside/AutoInspect/backend/internal/proto/gen/go/analysis/v1"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

const (
	testAnalysisBucketModels = "autoinspect-models"
	testCatalogKey           = "models/general/parts_catalog.json"
	testDamageDent           = "dent"
	testDamageDentRU         = "вмятина"
	testPartHood             = "hood"
	testPartHoodRU           = "капот"
)

func TestAnalysisServiceHandleAnalysisResultMarksJobFailed(t *testing.T) {
	t.Parallel()

	correlationID := uuid.New()
	job := pendingAnalysisJob(correlationID)
	jobRepo := &analysisResultJobRepo{job: job}
	notifier := &fakeJobNotifier{}
	svc := NewAnalysisService(nil, jobRepo, nil, nil, nil, notifier, nil, nil)

	message := analysisResultMessage(t, &analysisv1.AnalysisResult{
		CorrelationId: correlationID.String(),
		Status:        string(domain.StatusFailed),
		ErrorMessage:  "model artifact not found",
	})

	err := svc.HandleAnalysisResult(context.Background(), message)

	require.NoError(t, err)
	require.Equal(t, domain.StatusFailed, jobRepo.status)
	require.NotNil(t, jobRepo.errorMessage)
	require.Equal(t, "model artifact not found", *jobRepo.errorMessage)
	require.Len(t, notifier.events, 1)
	require.Equal(t, notify.EventAnalysisFailed, notifier.events[0].Type)
}

func TestAnalysisServiceHandleAnalysisResultStoresEnrichedCompletedResult(t *testing.T) {
	t.Parallel()

	correlationID := uuid.New()
	job := pendingAnalysisJob(correlationID)
	jobRepo := &analysisResultJobRepo{job: job}
	modelRepo := &fakeCarModelRepo{
		universalModel: &domain.CarModel{
			Make:              "general",
			Model:             "general",
			Generation:        "general",
			PartsCatalogS3Key: testCatalogKey,
			IsUniversal:       true,
			IsActive:          true,
		},
	}
	fileRepo := &fakeFileRepo{
		downloads: map[string]string{
			testCatalogKey: `{
				"model_id": "general",
				"parts": [
					{"name": "hood", "name_ru": "капот", "is_pair": false}
				]
			}`,
		},
	}
	damageTypeRepo := &fakeDamageTypeRepo{
		all: []*domain.DamageType{
			{Code: testDamageDent, NameRU: testDamageDentRU, IsActive: true},
		},
	}
	notifier := &fakeJobNotifier{}
	svc := NewAnalysisService(
		fileRepo,
		jobRepo,
		modelRepo,
		damageTypeRepo,
		nil,
		notifier,
		&config.S3Config{BucketModels: testAnalysisBucketModels},
		nil,
	)

	message := analysisResultMessage(t, &analysisv1.AnalysisResult{
		CorrelationId: correlationID.String(),
		Status:        string(domain.StatusCompleted),
		ModelId:       "general",
		ModelVersion:  "v1.3.0",
		BatchId:       correlationID.String(),
		Results: []*analysisv1.ImageAnalysisResult{
			{
				ImageId:  "1",
				ImageUri: "s3://autoinspect-uploads/uploads/user/car.jpg",
				Image:    &analysisv1.ImageInfo{Width: 800, Height: 600},
				DamageInstances: []*analysisv1.DamageInstance{
					{
						Id:         "damage-1",
						DamageType: testDamageDent,
						Parts: []*analysisv1.PartAssociation{
							{Name: testPartHood, Confidence: 0.91},
						},
					},
				},
				PartsSummary: []*analysisv1.PartSummary{
					{
						Name:        testPartHood,
						DamageCount: 1,
						DamageTypes: map[string]int32{testDamageDent: 1},
					},
				},
			},
		},
	})

	err := svc.HandleAnalysisResult(context.Background(), message)

	require.NoError(t, err)
	require.Equal(t, domain.StatusCompleted, jobRepo.status)
	require.Equal(t, "v1.3.0", jobRepo.modelVersion)
	require.NotNil(t, jobRepo.result)
	require.Equal(t, testDamageDentRU, jobRepo.result.Results[0].DamageInstances[0].DamageNameRU)
	require.Equal(t, testPartHoodRU, jobRepo.result.Results[0].DamageInstances[0].Parts[0].NameRU)
	require.Equal(t, testPartHood, jobRepo.result.Results[0].DamageInstances[0].Parts[0].ParentName)
	require.Equal(t, testPartHoodRU, jobRepo.result.Results[0].PartsSummary[0].NameRU)
	require.Equal(t, testDamageDentRU, jobRepo.result.Results[0].PartsSummary[0].DamageTypes[0].NameRU)
	require.Len(t, notifier.events, 1)
	require.Equal(t, notify.EventAnalysisCompleted, notifier.events[0].Type)
}

func pendingAnalysisJob(correlationID uuid.UUID) *domain.AnalysisJob {
	return &domain.AnalysisJob{
		ID:            uuid.New(),
		UserID:        uuid.New(),
		CarMake:       "Volkswagen",
		CarModel:      "Golf",
		CarGeneration: "5",
		CarYear:       2015,
		CorrelationID: correlationID,
		Status:        domain.StatusPending,
		RequestedAt:   time.Now().UTC(),
	}
}

func analysisResultMessage(t *testing.T, result *analysisv1.AnalysisResult) broker.Message {
	t.Helper()

	data, err := proto.Marshal(result)
	require.NoError(t, err)
	return broker.Message{
		Topic: "autoinspect.analysis.result",
		Key:   []byte(result.CorrelationId),
		Value: data,
	}
}

type analysisResultJobRepo struct {
	job          *domain.AnalysisJob
	status       domain.JobStatus
	errorMessage *string
	result       *domain.AnalysisResult
	modelVersion string
}

func (r *analysisResultJobRepo) Create(context.Context, *domain.AnalysisJob) error {
	return nil
}

func (r *analysisResultJobRepo) GetByID(context.Context, uuid.UUID) (*domain.AnalysisJob, error) {
	return nil, domain.ErrJobNotFound
}

func (r *analysisResultJobRepo) GetByCorrelationID(_ context.Context, correlationID uuid.UUID) (*domain.AnalysisJob, error) {
	if r.job != nil && r.job.CorrelationID == correlationID {
		return r.job, nil
	}
	return nil, domain.ErrJobNotFound
}

func (r *analysisResultJobRepo) GetByUserAndIdempotencyKey(context.Context, uuid.UUID, string) (*domain.AnalysisJob, error) {
	return nil, domain.ErrJobNotFound
}

func (r *analysisResultJobRepo) GetByUserID(context.Context, uuid.UUID, int, int) ([]*domain.AnalysisJob, error) {
	return nil, nil
}

func (r *analysisResultJobRepo) UpdateStatus(context.Context, uuid.UUID, domain.JobStatus, *string) error {
	return nil
}

func (r *analysisResultJobRepo) UpdateStatusByCorrelationID(
	_ context.Context,
	correlationID uuid.UUID,
	status domain.JobStatus,
	errorMessage *string,
) error {
	if r.job == nil || r.job.CorrelationID != correlationID {
		return domain.ErrJobNotFound
	}
	r.status = status
	r.errorMessage = errorMessage
	r.job.Status = status
	r.job.ErrorMessage = errorMessage
	return nil
}

func (r *analysisResultJobRepo) UpdateResult(context.Context, uuid.UUID, *domain.AnalysisResult, string) error {
	return nil
}

func (r *analysisResultJobRepo) UpdateResultByCorrelationID(
	_ context.Context,
	correlationID uuid.UUID,
	result *domain.AnalysisResult,
	modelVersion string,
) error {
	if r.job == nil || r.job.CorrelationID != correlationID {
		return domain.ErrJobNotFound
	}
	r.status = domain.StatusCompleted
	r.result = result
	r.modelVersion = modelVersion
	r.job.Status = domain.StatusCompleted
	r.job.Result = result
	r.job.UsedModelVersion = modelVersion
	return nil
}

type fakeJobNotifier struct {
	events []*notify.JobEvent
}

func (n *fakeJobNotifier) NotifyJobEvent(_ context.Context, event *notify.JobEvent) error {
	n.events = append(n.events, event)
	return nil
}
