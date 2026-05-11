package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"github.com/DedovInside/AutoInspect/backend/internal/broker"
	"github.com/DedovInside/AutoInspect/backend/internal/config"
	"github.com/DedovInside/AutoInspect/backend/internal/domain"
	"github.com/DedovInside/AutoInspect/backend/internal/notify"
	analysisv1 "github.com/DedovInside/AutoInspect/backend/internal/proto/gen/go/analysis/v1"
	"github.com/DedovInside/AutoInspect/backend/internal/repository"
	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
)

type AnalysisService struct {
	fileRepo       repository.FileRepository
	jobRepo        repository.AnalysisJobRepository
	modelRepo      repository.CarModelRepository
	damageTypeRepo repository.DamageTypeRepository
	publisher      broker.Publisher
	notifier       notify.Notifier
	s3Cfg          *config.S3Config
	kafkaCfg       *config.KafkaConfig
}

type Broadcaster interface {
	BroadcastToJob(jobID uuid.UUID, event string, payload any)
}

func NewAnalysisService(
	fileRepo repository.FileRepository,
	jobRepo repository.AnalysisJobRepository,
	modelRepo repository.CarModelRepository,
	damageTypeRepo repository.DamageTypeRepository,
	publisher broker.Publisher,
	notifier notify.Notifier,
	s3Cfg *config.S3Config,
	kafkaCfg *config.KafkaConfig,
) *AnalysisService {
	return &AnalysisService{
		fileRepo:       fileRepo,
		jobRepo:        jobRepo,
		modelRepo:      modelRepo,
		damageTypeRepo: damageTypeRepo,
		publisher:      publisher,
		notifier:       notifier,
		s3Cfg:          s3Cfg,
		kafkaCfg:       kafkaCfg,
	}
}

func (s *AnalysisService) SubmitAnalysis(ctx context.Context,
	userID uuid.UUID, idempotencyKey *string, carInfo domain.CarInfo,
	images []multipart.File) (*domain.AnalysisJob, error) {
	existing, err := s.findIdempotentJob(ctx, userID, idempotencyKey)
	if err != nil {
		return nil, err
	}

	if existing != nil {
		if err := s.ensurePublished(ctx, existing); err != nil {
			return nil, fmt.Errorf("republish existing job: %w", err)
		}
		return existing, nil
	}

	model, err := s.findModelForCar(ctx, carInfo)
	if err != nil {
		return nil, err
	}

	if err := s.ensureModelArtifactsExist(ctx, model); err != nil {
		return nil, err
	}

	imageKeys, err := s.uploadImages(ctx, userID, images)
	if err != nil {
		return nil, err
	}

	job := newAnalysisJob(userID, idempotencyKey, carInfo, imageKeys, model.ModelVersion)
	published, err := s.createAnalysisJob(ctx, job, imageKeys, idempotencyKey)
	if err != nil {
		return nil, err
	}

	if published {
		return job, nil
	}

	if err := s.publishAnalysisRequest(ctx, job, model); err != nil {
		// Задача уже создана. На retry с тем же idempotency-key service переопубликует pending задачу.
		return nil, fmt.Errorf("publish to Kafka: %w", err)
	}

	return job, nil
}

func (s *AnalysisService) findIdempotentJob(ctx context.Context,
	userID uuid.UUID, idempotencyKey *string) (*domain.AnalysisJob, error) {
	if idempotencyKey == nil || *idempotencyKey == "" {
		return nil, nil
	}

	existing, err := s.jobRepo.GetByUserAndIdempotencyKey(ctx, userID, *idempotencyKey)
	if err == nil {
		return existing, nil
	}

	if !errors.Is(err, domain.ErrJobNotFound) {
		return nil, fmt.Errorf("check idempotency: %w", err)
	}

	return nil, nil
}

func newAnalysisJob(
	userID uuid.UUID,
	idempotencyKey *string,
	carInfo domain.CarInfo,
	imageKeys []string,
	modelVersion string,
) *domain.AnalysisJob {
	return &domain.AnalysisJob{
		ID:               uuid.New(),
		UserID:           userID,
		IdempotencyKey:   idempotencyKey,
		CarMake:          carInfo.Make,
		CarModel:         carInfo.Model,
		CarGeneration:    carInfo.Generation,
		CarYear:          carInfo.Year,
		ImageKeys:        imageKeys,
		CorrelationID:    uuid.New(),
		Status:           domain.StatusPending,
		UsedModelVersion: modelVersion,
		RequestedAt:      time.Now(),
	}
}

func (s *AnalysisService) createAnalysisJob(ctx context.Context,
	job *domain.AnalysisJob, imageKeys []string, idempotencyKey *string) (bool, error) {
	if err := s.jobRepo.Create(ctx, job); err != nil {
		existing, handled, handleErr := s.handleCreateConflict(ctx, err, job.UserID, idempotencyKey, imageKeys)
		if handleErr != nil {
			return false, handleErr
		}

		if handled {
			*job = *existing
			return true, nil
		}

		s.cleanupUploaded(ctx, imageKeys)
		return false, fmt.Errorf("create job in DB: %w", err)
	}

	return false, nil
}

func (s *AnalysisService) handleCreateConflict(ctx context.Context,
	createErr error, userID uuid.UUID, idempotencyKey *string, imageKeys []string) (*domain.AnalysisJob, bool, error) {
	if !errors.Is(createErr, domain.ErrAlreadyExists) || idempotencyKey == nil || *idempotencyKey == "" {
		return nil, false, nil
	}

	existing, getErr := s.jobRepo.GetByUserAndIdempotencyKey(ctx, userID, *idempotencyKey)
	if getErr != nil {
		if errors.Is(getErr, domain.ErrJobNotFound) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("read existing job after create conflict: %w", getErr)
	}

	s.cleanupUploaded(ctx, imageKeys)
	if republishErr := s.ensurePublished(ctx, existing); republishErr != nil {
		return nil, false, fmt.Errorf("republish existing job after create conflict: %w", republishErr)
	}

	return existing, true, nil
}

func (s *AnalysisService) HandleAnalysisResult(ctx context.Context, msg broker.Message) error {
	log.Printf("analysis result received: topic=%s key=%s payload_bytes=%d", msg.Topic, string(msg.Key), len(msg.Value))

	protoResult, jobCorrID, err := parseAnalysisResultMessage(msg)
	if err != nil {
		log.Printf("analysis result parse failed: topic=%s key=%s error=%v", msg.Topic, string(msg.Key), err)
		return err
	}

	log.Printf(
		"analysis result parsed: correlation_id=%s status=%q model_id=%q model_version=%q images=%d error=%q",
		jobCorrID,
		protoResult.Status,
		protoResult.ModelId,
		protoResult.ModelVersion,
		len(protoResult.Results),
		protoResult.ErrorMessage,
	)

	existingJob, err := s.jobRepo.GetByCorrelationID(ctx, jobCorrID)
	if err != nil {
		if errors.Is(err, domain.ErrJobNotFound) {
			log.Printf("analysis result ignored: job not found correlation_id=%s", jobCorrID)
			return nil
		}
		log.Printf("analysis result job lookup failed: correlation_id=%s error=%v", jobCorrID, err)
		return fmt.Errorf("check existing job: %w", err)
	}

	if isTerminalStatus(existingJob.Status) {
		log.Printf(
			"analysis result ignored: job already terminal correlation_id=%s job_id=%s status=%s",
			jobCorrID,
			existingJob.ID,
			existingJob.Status,
		)
		return nil
	}

	status := strings.ToLower(strings.TrimSpace(protoResult.Status))
	switch status {
	case string(domain.StatusFailed):
		log.Printf("analysis result handling failed status: correlation_id=%s job_id=%s", jobCorrID, existingJob.ID)
		return s.handleFailedAnalysisResult(ctx, jobCorrID, existingJob, protoResult.ErrorMessage)
	case string(domain.StatusCompleted):
		log.Printf("analysis result handling completed status: correlation_id=%s job_id=%s", jobCorrID, existingJob.ID)
		return s.handleCompletedAnalysisResult(ctx, jobCorrID, existingJob, protoResult)
	default:
		log.Printf("analysis result unsupported status: correlation_id=%s status=%q", jobCorrID, protoResult.Status)
		return fmt.Errorf("unsupported analysis status: %q", protoResult.Status)
	}
}

func parseAnalysisResultMessage(msg broker.Message) (*analysisv1.AnalysisResult, uuid.UUID, error) {
	protoResult := &analysisv1.AnalysisResult{}
	if err := proto.Unmarshal(msg.Value, protoResult); err != nil {
		return nil, uuid.Nil, fmt.Errorf("unmarshal protobuf: %w", err)
	}

	jobCorrID, err := uuid.Parse(protoResult.CorrelationId)
	if err != nil {
		return nil, uuid.Nil, fmt.Errorf("invalid correlation_id: %w", err)
	}

	return protoResult, jobCorrID, nil
}

func isTerminalStatus(status domain.JobStatus) bool {
	return status == domain.StatusCompleted || status == domain.StatusFailed
}

func (s *AnalysisService) handleFailedAnalysisResult(
	ctx context.Context,
	correlationID uuid.UUID,
	job *domain.AnalysisJob,
	errorMessage string,
) error {
	errMsg := strings.TrimSpace(errorMessage)
	if errMsg == "" {
		errMsg = "analysis failed"
	}

	if err := s.jobRepo.UpdateStatusByCorrelationID(ctx, correlationID, domain.StatusFailed, &errMsg); err != nil {
		if errors.Is(err, domain.ErrJobNotFound) {
			log.Printf("analysis job failed update skipped: job not found correlation_id=%s", correlationID)
			return nil
		}
		log.Printf("analysis job failed update error: correlation_id=%s error=%v", correlationID, err)
		return fmt.Errorf("update job status to failed: %w", err)
	}

	log.Printf("analysis job marked failed: correlation_id=%s job_id=%s error=%q", correlationID, job.ID, errMsg)

	s.notifyJobEvent(ctx, &notify.JobEvent{
		JobID:     job.ID,
		UserID:    job.UserID,
		Type:      notify.EventAnalysisFailed,
		Status:    string(domain.StatusFailed),
		Timestamp: time.Now().UTC(),
		Payload:   notify.AnalysisFailedPayload{Error: errMsg},
	})

	return nil
}

func (s *AnalysisService) handleCompletedAnalysisResult(
	ctx context.Context,
	correlationID uuid.UUID,
	job *domain.AnalysisJob,
	protoResult *analysisv1.AnalysisResult,
) error {
	domainResult := ProtoToDomainResult(protoResult)
	model, err := s.findModelForCar(ctx, domain.CarInfo{
		Make:       job.CarMake,
		Model:      job.CarModel,
		Generation: job.CarGeneration,
		Year:       job.CarYear,
	})
	if err != nil {
		return fmt.Errorf("find model for analysis result enrichment: %w", err)
	}

	if err := s.enrichAnalysisResult(ctx, domainResult, model); err != nil {
		log.Printf("analysis result enrichment failed: correlation_id=%s job_id=%s error=%v", correlationID, job.ID, err)
		return fmt.Errorf("enrich analysis result: %w", err)
	}

	if err := s.jobRepo.UpdateResultByCorrelationID(ctx, correlationID, domainResult, protoResult.ModelVersion); err != nil {
		if errors.Is(err, domain.ErrJobNotFound) {
			log.Printf("analysis job completed update skipped: job not found correlation_id=%s", correlationID)
			return nil
		}
		log.Printf("analysis job completed update error: correlation_id=%s error=%v", correlationID, err)
		return fmt.Errorf("update job result: %w", err)
	}

	log.Printf(
		"analysis job marked completed: correlation_id=%s job_id=%s model_version=%q images=%d damages=%d",
		correlationID,
		job.ID,
		protoResult.ModelVersion,
		len(protoResult.Results),
		countDamageInstances(domainResult),
	)

	s.notifyJobEvent(ctx, &notify.JobEvent{
		JobID:     job.ID,
		UserID:    job.UserID,
		Type:      notify.EventAnalysisCompleted,
		Status:    string(domain.StatusCompleted),
		Timestamp: time.Now().UTC(),
		Payload: notify.AnalysisCompletedPayload{
			ModelVersion: protoResult.ModelVersion,
			DamageCount:  countDamageInstances(domainResult),
		},
	})

	return nil
}

func countDamageInstances(result *domain.AnalysisResult) int {
	if result == nil {
		return 0
	}

	count := 0
	for _, imageResult := range result.Results {
		count += len(imageResult.DamageInstances)
	}

	return count
}

func (s *AnalysisService) notifyJobEvent(ctx context.Context, event *notify.JobEvent) {
	if s.notifier != nil {
		_ = s.notifier.NotifyJobEvent(ctx, event)
	}
}

func (s *AnalysisService) GetJobStatus(ctx context.Context, jobID, userID uuid.UUID) (*domain.AnalysisJob, error) {
	job, err := s.jobRepo.GetByID(ctx, jobID)
	if err != nil {
		return nil, err
	}

	if job.UserID != userID {
		return nil, domain.ErrForbidden
	}

	return job, nil
}

func (s *AnalysisService) GetPresignedImageURL(ctx context.Context,
	jobID, userID uuid.UUID, idx int) (string, time.Time, error) {
	job, err := s.jobRepo.GetByID(ctx, jobID)
	if err != nil {
		return "", time.Time{}, err
	}

	if job.UserID != userID {
		return "", time.Time{}, domain.ErrForbidden
	}

	if idx < 0 || idx >= len(job.ImageKeys) {
		return "", time.Time{}, fmt.Errorf("image index %d out of range", idx)
	}

	objectKey := job.ImageKeys[idx]
	expiresAt := time.Now().Add(s.s3Cfg.PresignedURLTTL)

	url, err := s.fileRepo.GetPresignedURL(ctx, s.s3Cfg.BucketUploads, objectKey, s.s3Cfg.PresignedURLTTL)
	if err != nil {
		return "", time.Time{}, err
	}

	return url, expiresAt, nil
}

func (s *AnalysisService) ListUserJobs(ctx context.Context,
	userID uuid.UUID, limit, offset int) ([]*domain.AnalysisJob, error) {
	return s.jobRepo.GetByUserID(ctx, userID, limit, offset)
}

func (s *AnalysisService) ensurePublished(ctx context.Context, job *domain.AnalysisJob) error {

	if job == nil {
		return domain.ErrInvalidInput
	}

	if job.Status == domain.StatusCompleted || job.Status == domain.StatusFailed {
		return nil
	}

	carInfo := domain.CarInfo{
		Make:       job.CarMake,
		Model:      job.CarModel,
		Generation: job.CarGeneration,
		Year:       job.CarYear,
	}

	model, err := s.findModelForCar(ctx, carInfo)
	if err != nil {
		return err
	}

	return s.publishAnalysisRequest(ctx, job, model)
}

func (s *AnalysisService) publishAnalysisRequest(ctx context.Context,
	job *domain.AnalysisJob, model *domain.CarModel) error {
	protoReq := DomainToProtoRequest(job, model)
	data, err := proto.Marshal(protoReq)

	if err != nil {
		return fmt.Errorf("marshal protobuf: %w", err)
	}

	topic := s.kafkaCfg.TopicAnalysisRequest
	if topic == "" {
		topic = "autoinspect.analysis.request"
	}

	headers := map[string]string{
		"content-type":     "application/protobuf",
		"x-correlation-id": job.CorrelationID.String(),
		"x-user-id":        job.UserID.String(),
		"x-event-type":     "AnalysisRequested",
	}

	if err := s.publisher.Publish(ctx, broker.Message{
		Topic:   topic,
		Key:     []byte(job.CorrelationID.String()),
		Value:   data,
		Headers: headers,
	}); err != nil {
		return err
	}

	return nil
}

func (s *AnalysisService) ensureModelArtifactsExist(ctx context.Context, model *domain.CarModel) error {
	if model == nil ||
		strings.TrimSpace(model.PartsModelS3Key) == "" ||
		strings.TrimSpace(model.PartsConfigS3Key) == "" ||
		strings.TrimSpace(model.PartsCatalogS3Key) == "" {
		return domain.ErrInvalidModel
	}

	if s.fileRepo == nil {
		return nil
	}

	if err := s.ensureModelObjectExists(ctx, model.PartsModelS3Key); err != nil {
		return err
	}

	if err := s.ensureModelObjectExists(ctx, model.PartsConfigS3Key); err != nil {
		return err
	}

	return s.ensureModelObjectExists(ctx, model.PartsCatalogS3Key)
}

func (s *AnalysisService) ensureModelObjectExists(ctx context.Context, objectKey string) error {
	exists, err := s.fileRepo.Exists(ctx, s.s3Cfg.BucketModels, objectKey)
	if err != nil {
		return fmt.Errorf("check model artifact %q: %w", objectKey, err)
	}

	if !exists {
		return fmt.Errorf("%w: model artifact %q not found", domain.ErrInvalidModel, objectKey)
	}

	return nil
}

func (s *AnalysisService) findModelForCar(ctx context.Context, carInfo domain.CarInfo) (*domain.CarModel, error) {
	var (
		model *domain.CarModel
		err   error
	)

	model, err = s.modelRepo.FindActiveModel(ctx, carInfo.Make, carInfo.Model, carInfo.Generation, carInfo.Year)
	if err == nil {
		return model, nil
	}

	if !errors.Is(err, domain.ErrInvalidModel) {
		return nil, fmt.Errorf("find model: %w", err)
	}

	model, err = s.modelRepo.GetUniversalModel(ctx)
	if err != nil {
		return nil, fmt.Errorf("no suitable model found: %w", err)
	}

	return model, nil
}

func (s *AnalysisService) uploadImages(ctx context.Context,
	userID uuid.UUID, images []multipart.File) ([]string, error) {
	imageKeys := make([]string, 0, len(images))

	for i, img := range images {
		contentType, size, err := detectUploadMeta(img)

		if err != nil {
			s.cleanupUploaded(ctx, imageKeys)
			_ = img.Close()
			return nil, fmt.Errorf("inspect image %d: %w", i, err)
		}

		if !strings.HasPrefix(contentType, "image/") {
			s.cleanupUploaded(ctx, imageKeys)
			_ = img.Close()
			return nil, domain.ErrInvalidImage
		}

		objectKey := fmt.Sprintf(
			"uploads/%s/%d/%d%s",
			userID,
			time.Now().UnixNano(),
			i,
			extensionByContentType(contentType),
		)

		if err := s.fileRepo.Upload(ctx, s.s3Cfg.BucketUploads, objectKey, img, contentType, size); err != nil {
			s.cleanupUploaded(ctx, imageKeys)
			_ = img.Close()
			return nil, fmt.Errorf("upload to S3: %w", err)
		}
		_ = img.Close()
		imageKeys = append(imageKeys, objectKey)
	}

	return imageKeys, nil
}

func (s *AnalysisService) cleanupUploaded(ctx context.Context, imageKeys []string) {
	for _, key := range imageKeys {
		_ = s.fileRepo.Delete(ctx, s.s3Cfg.BucketUploads, key)
	}
}

func detectUploadMeta(file multipart.File) (contentType string, size int64, err error) {
	header := make([]byte, 512)
	n, readErr := file.Read(header)

	if readErr != nil && !errors.Is(readErr, io.EOF) {
		return "", 0, readErr
	}

	contentType = http.DetectContentType(header[:n])

	if _, err = file.Seek(0, io.SeekStart); err != nil {
		return "", 0, err
	}

	size, err = file.Seek(0, io.SeekEnd)
	if err != nil {
		return contentType, -1, nil
	}

	_, _ = file.Seek(0, io.SeekStart)
	return contentType, size, nil
}

func extensionByContentType(contentType string) string {
	switch contentType {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/webp":
		return ".webp"
	default:
		return ".bin"
	}
}
