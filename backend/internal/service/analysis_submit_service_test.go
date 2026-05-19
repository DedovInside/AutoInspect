package service

import (
	"bytes"
	"context"
	"mime/multipart"
	"strings"
	"testing"

	"github.com/DedovInside/AutoInspect/backend/internal/broker"
	"github.com/DedovInside/AutoInspect/backend/internal/config"
	"github.com/DedovInside/AutoInspect/backend/internal/domain"
	analysisv1 "github.com/DedovInside/AutoInspect/backend/internal/proto/gen/go/analysis/v1"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

const (
	testAnalysisBucketUploads = "autoinspect-uploads"
	testAnalysisRequestTopic  = "autoinspect.analysis.request"
	testPartsModelKey         = "models/general/parts_segmentation.pt"
	testPartsConfigKey        = "models/general/parts_inference_config.json"
)

func TestAnalysisServiceSubmitAnalysisCreatesJobUploadsImageAndPublishesRequest(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	idempotencyKey := "analysis-submit-key"
	jobRepo := &submitAnalysisJobRepo{}
	fileRepo := &fakeFileRepo{}
	publisher := &fakeAnalysisPublisher{}
	modelRepo := &fakeCarModelRepo{
		universalModel: &domain.CarModel{
			Make:              "general",
			Model:             "general",
			Generation:        "general",
			YearFrom:          1900,
			YearTo:            2100,
			PartsModelS3Key:   testPartsModelKey,
			PartsConfigS3Key:  testPartsConfigKey,
			PartsCatalogS3Key: testCatalogKey,
			ModelVersion:      "v1",
			IsUniversal:       true,
			IsActive:          true,
		},
	}
	svc := NewAnalysisService(
		fileRepo,
		jobRepo,
		modelRepo,
		nil,
		publisher,
		nil,
		&config.S3Config{BucketUploads: testAnalysisBucketUploads, BucketModels: testAnalysisBucketModels},
		&config.KafkaConfig{TopicAnalysisRequest: testAnalysisRequestTopic},
	)

	job, err := svc.SubmitAnalysis(context.Background(), userID, &idempotencyKey, domain.CarInfo{
		Make:       "Volkswagen",
		Model:      "Golf",
		Generation: "5",
		Year:       2015,
	}, []multipart.File{jpegMultipartFile()})

	require.NoError(t, err)
	require.Equal(t, domain.StatusPending, job.Status)
	require.Same(t, job, jobRepo.created)
	require.Len(t, fileRepo.uploaded, 1)
	require.True(t, strings.HasSuffix(fileRepo.uploaded[0], ".jpg"))
	require.Len(t, publisher.messages, 1)

	var request analysisv1.AnalysisRequest
	require.NoError(t, proto.Unmarshal(publisher.messages[0].Value, &request))
	require.Equal(t, testAnalysisRequestTopic, publisher.messages[0].Topic)
	require.Equal(t, job.CorrelationID.String(), request.CorrelationId)
	require.Equal(t, userID.String(), request.UserId)
	require.Equal(t, "Volkswagen", request.CarInfo.Make)
	require.Equal(t, int32(2015), request.CarInfo.Year)
	require.Equal(t, fileRepo.uploaded, request.ImageS3Keys)
	require.Equal(t, testPartsModelKey, request.PartsModelS3Key)
	require.Equal(t, testPartsConfigKey, request.PartsConfigS3Key)
}

func TestAnalysisServiceSubmitAnalysisReturnsExistingIdempotentJob(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	idempotencyKey := "same-analysis-key"
	existing := pendingAnalysisJob(uuid.New())
	existing.UserID = userID
	existing.IdempotencyKey = &idempotencyKey
	jobRepo := &submitAnalysisJobRepo{existing: existing}
	publisher := &fakeAnalysisPublisher{}
	modelRepo := &fakeCarModelRepo{universalModel: &domain.CarModel{
		PartsModelS3Key:   testPartsModelKey,
		PartsConfigS3Key:  testPartsConfigKey,
		PartsCatalogS3Key: testCatalogKey,
		ModelVersion:      "v1",
		IsActive:          true,
	}}
	svc := NewAnalysisService(
		&fakeFileRepo{},
		jobRepo,
		modelRepo,
		nil,
		publisher,
		nil,
		&config.S3Config{BucketUploads: testAnalysisBucketUploads, BucketModels: testAnalysisBucketModels},
		&config.KafkaConfig{TopicAnalysisRequest: testAnalysisRequestTopic},
	)

	got, err := svc.SubmitAnalysis(context.Background(), userID, &idempotencyKey, domain.CarInfo{Make: "Volkswagen", Year: 2015}, nil)

	require.NoError(t, err)
	require.Same(t, existing, got)
	require.Nil(t, jobRepo.created)
	require.Len(t, publisher.messages, 1)
}

func jpegMultipartFile() testMultipartFile {
	return testMultipartFile{Reader: bytes.NewReader([]byte("\xff\xd8\xff\xe0\x00\x10JFIF\x00\x01\x01"))}
}

type submitAnalysisJobRepo struct {
	created  *domain.AnalysisJob
	existing *domain.AnalysisJob
}

func (r *submitAnalysisJobRepo) Create(_ context.Context, job *domain.AnalysisJob) error {
	r.created = job
	return nil
}

func (r *submitAnalysisJobRepo) GetByID(context.Context, uuid.UUID) (*domain.AnalysisJob, error) {
	return nil, domain.ErrJobNotFound
}

func (r *submitAnalysisJobRepo) GetByCorrelationID(context.Context, uuid.UUID) (*domain.AnalysisJob, error) {
	return nil, domain.ErrJobNotFound
}

func (r *submitAnalysisJobRepo) GetByUserAndIdempotencyKey(
	_ context.Context,
	userID uuid.UUID,
	idempotencyKey string,
) (*domain.AnalysisJob, error) {
	if r.existing != nil &&
		r.existing.UserID == userID &&
		r.existing.IdempotencyKey != nil &&
		*r.existing.IdempotencyKey == idempotencyKey {
		return r.existing, nil
	}
	return nil, domain.ErrJobNotFound
}

func (r *submitAnalysisJobRepo) GetByUserID(context.Context, uuid.UUID, int, int) ([]*domain.AnalysisJob, error) {
	return nil, nil
}

func (r *submitAnalysisJobRepo) UpdateStatus(context.Context, uuid.UUID, domain.JobStatus, *string) error {
	return nil
}

func (r *submitAnalysisJobRepo) UpdateStatusByCorrelationID(context.Context, uuid.UUID, domain.JobStatus, *string) error {
	return nil
}

func (r *submitAnalysisJobRepo) UpdateResult(context.Context, uuid.UUID, *domain.AnalysisResult, string) error {
	return nil
}

func (r *submitAnalysisJobRepo) UpdateResultByCorrelationID(context.Context, uuid.UUID, *domain.AnalysisResult, string) error {
	return nil
}

type fakeAnalysisPublisher struct {
	messages []broker.Message
}

func (p *fakeAnalysisPublisher) Publish(_ context.Context, message broker.Message) error {
	p.messages = append(p.messages, message)
	return nil
}

func (p *fakeAnalysisPublisher) Close() error {
	return nil
}
