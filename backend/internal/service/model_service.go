package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/DedovInside/AutoInspect/backend/internal/config"
	"github.com/DedovInside/AutoInspect/backend/internal/domain"
	"github.com/DedovInside/AutoInspect/backend/internal/repository"
	"github.com/google/uuid"
)

const (
	partsModelFileName   = "parts_segmentation.pt"
	partsConfigFileName  = "parts_inference_config.json"
	partsCatalogFileName = "parts_catalog.json"
	defaultSlug          = "default"
)

type ModelService struct {
	modelRepo repository.CarModelRepository
	fileRepo  repository.FileRepository
	s3Cfg     *config.S3Config
}

type ModelArtifactFile struct {
	File     multipart.File
	Filename string
}

type UploadModelArtifactsInput struct {
	IdempotencyKey string

	Make       string
	Model      string
	Generation string
	YearFrom   int
	YearTo     int
	Version    string

	IsUniversal bool

	PartsModel   ModelArtifactFile
	PartsConfig  ModelArtifactFile
	PartsCatalog ModelArtifactFile
}

func NewModelService(
	modelRepo repository.CarModelRepository,
	fileRepo repository.FileRepository,
	s3Cfg *config.S3Config,
) *ModelService {
	return &ModelService{
		modelRepo: modelRepo,
		fileRepo:  fileRepo,
		s3Cfg:     s3Cfg,
	}
}

func (s *ModelService) UploadModelArtifacts(ctx context.Context,
	input *UploadModelArtifactsInput) (*domain.CarModel, error) {
	if err := validateModelUploadInput(input); err != nil {
		return nil, err
	}

	modelID := modelUploadID(input)
	objectKeys := modelArtifactKeys(input)
	uploadedKeys, err := s.uploadModelArtifacts(ctx, input, objectKeys)
	if err != nil {
		return nil, err
	}

	model := &domain.CarModel{
		ID:                modelID,
		Make:              strings.TrimSpace(input.Make),
		Model:             strings.TrimSpace(input.Model),
		Generation:        strings.TrimSpace(input.Generation),
		YearFrom:          input.YearFrom,
		YearTo:            input.YearTo,
		PartsModelS3Key:   objectKeys.partsModel,
		PartsConfigS3Key:  objectKeys.partsConfig,
		PartsCatalogS3Key: objectKeys.partsCatalog,
		ModelVersion:      strings.TrimSpace(input.Version),
		IsUniversal:       input.IsUniversal,
		IsActive:          true,
		CreatedAt:         time.Now().UTC(),
	}

	if err := s.modelRepo.CreateModel(ctx, model); err != nil {
		if errors.Is(err, domain.ErrAlreadyExists) {
			existing, getErr := s.modelRepo.GetModelByID(ctx, modelID)
			if getErr == nil {
				return existing, nil
			}
		}
		s.cleanupModelArtifacts(ctx, uploadedKeys)
		return nil, fmt.Errorf("create model metadata: %w", err)
	}

	return model, nil
}

func (s *ModelService) ListModels(ctx context.Context, limit, offset int) ([]*domain.CarModel, error) {
	if limit <= 0 {
		limit = 20
	}

	if limit > 100 {
		limit = 100
	}

	if offset < 0 {
		return nil, domain.ErrInvalidInput
	}
	return s.modelRepo.ListModels(ctx, limit, offset)
}

func (s *ModelService) ListActiveSpecializedModels(ctx context.Context) ([]*domain.CarModel, error) {
	models, err := s.ListModels(ctx, 100, 0)
	if err != nil {
		return nil, err
	}

	items := make([]*domain.CarModel, 0, len(models))
	for _, model := range models {
		if model == nil || model.IsUniversal || !model.IsActive {
			continue
		}
		items = append(items, model)
	}
	return items, nil
}

func (s *ModelService) DeactivateModel(ctx context.Context, id uuid.UUID) error {
	if id == uuid.Nil {
		return domain.ErrInvalidInput
	}

	if _, err := s.modelRepo.GetModelByID(ctx, id); err != nil {
		return err
	}
	return s.modelRepo.DeactivateModel(ctx, id)
}

type modelArtifactObjectKeys struct {
	partsModel   string
	partsConfig  string
	partsCatalog string
}

func (s *ModelService) uploadModelArtifacts(
	ctx context.Context,
	input *UploadModelArtifactsInput,
	keys modelArtifactObjectKeys,
) ([]string, error) {
	if s.fileRepo == nil || s.s3Cfg == nil {
		return nil, domain.ErrInternal
	}

	artifacts := []struct {
		file        ModelArtifactFile
		objectKey   string
		contentType string
	}{
		{file: input.PartsModel, objectKey: keys.partsModel, contentType: "application/octet-stream"},
		{file: input.PartsConfig, objectKey: keys.partsConfig, contentType: "application/json"},
		{file: input.PartsCatalog, objectKey: keys.partsCatalog, contentType: "application/json"},
	}

	uploaded := make([]string, 0, len(artifacts))
	for _, artifact := range artifacts {
		size, err := artifactSize(artifact.file.File)
		if err != nil {
			s.cleanupModelArtifacts(ctx, uploaded)
			return nil, fmt.Errorf("inspect model artifact %q: %w", artifact.file.Filename, err)
		}

		if err := s.fileRepo.Upload(ctx, s.s3Cfg.BucketModels, artifact.objectKey, artifact.file.File, artifact.contentType, size); err != nil {
			s.cleanupModelArtifacts(ctx, uploaded)
			return nil, fmt.Errorf("upload model artifact %q: %w", artifact.file.Filename, err)
		}
		uploaded = append(uploaded, artifact.objectKey)
	}

	return uploaded, nil
}

func (s *ModelService) cleanupModelArtifacts(ctx context.Context, objectKeys []string) {
	if s.fileRepo == nil || s.s3Cfg == nil {
		return
	}

	for _, key := range objectKeys {
		_ = s.fileRepo.Delete(ctx, s.s3Cfg.BucketModels, key)
	}
}

func validateModelUploadInput(input *UploadModelArtifactsInput) error {
	if input == nil {
		return domain.ErrInvalidInput
	}

	if strings.TrimSpace(input.Make) == "" ||
		strings.TrimSpace(input.Model) == "" ||
		strings.TrimSpace(input.Version) == "" ||
		input.YearFrom <= 0 {
		return domain.ErrInvalidInput
	}

	if input.YearTo > 0 && input.YearTo < input.YearFrom {
		return domain.ErrInvalidInput
	}

	if err := validateArtifactFile(input.PartsModel, ".pt", false); err != nil {
		return err
	}

	if err := validateArtifactFile(input.PartsConfig, ".json", true); err != nil {
		return err
	}
	return validateArtifactFile(input.PartsCatalog, ".json", true)
}

func validateArtifactFile(file ModelArtifactFile, expectedExt string, requireJSON bool) error {
	if file.File == nil {
		return domain.ErrInvalidInput
	}

	if strings.ToLower(filepath.Ext(file.Filename)) != expectedExt {
		return domain.ErrInvalidInput
	}

	size, err := artifactSize(file.File)
	if err != nil || size <= 0 {
		return domain.ErrInvalidInput
	}

	if !requireJSON {
		return nil
	}

	data, err := ioReadAllAndReset(file.File)
	if err != nil || !json.Valid(data) {
		return domain.ErrInvalidInput
	}
	return nil
}

func artifactSize(file multipart.File) (int64, error) {
	if file == nil {
		return 0, domain.ErrInvalidInput
	}

	current, err := file.Seek(0, 1)

	if err != nil {
		return 0, err
	}

	size, err := file.Seek(0, 2)
	if err != nil {
		return 0, err
	}

	_, seekErr := file.Seek(current, 0)
	if seekErr != nil {
		return 0, seekErr
	}
	return size, nil
}

func ioReadAllAndReset(file multipart.File) ([]byte, error) {
	if _, err := file.Seek(0, 0); err != nil {
		return nil, err
	}

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}
	_, seekErr := file.Seek(0, 0)
	return data, seekErr
}

func modelArtifactKeys(input *UploadModelArtifactsInput) modelArtifactObjectKeys {
	prefix := path.Join(
		"models",
		"parts",
		modelScope(input),
		slug(input.Make),
		slug(input.Model),
		slug(input.Generation),
		slug(input.Version),
	)

	return modelArtifactObjectKeys{
		partsModel:   path.Join(prefix, partsModelFileName),
		partsConfig:  path.Join(prefix, partsConfigFileName),
		partsCatalog: path.Join(prefix, partsCatalogFileName),
	}
}

func modelScope(input *UploadModelArtifactsInput) string {
	if input.IsUniversal {
		return "universal"
	}
	return "specialized"
}

func modelUploadID(input *UploadModelArtifactsInput) uuid.UUID {
	key := strings.TrimSpace(input.IdempotencyKey)
	if key == "" {
		key = strings.Join([]string{
			strings.ToLower(strings.TrimSpace(input.Make)),
			strings.ToLower(strings.TrimSpace(input.Model)),
			strings.ToLower(strings.TrimSpace(input.Generation)),
			strconv.Itoa(input.YearFrom),
			strconv.Itoa(input.YearTo),
			strings.ToLower(strings.TrimSpace(input.Version)),
			strconv.FormatBool(input.IsUniversal),
		}, "|")
	}
	return uuid.NewSHA1(uuid.NameSpaceURL, []byte("autoinspect:model-upload:"+key))
}

func slug(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return defaultSlug
	}

	var b strings.Builder
	lastDash := false
	for _, r := range value {
		switch {
		case unicode.IsLetter(r), unicode.IsDigit(r):
			b.WriteRune(r)
			lastDash = false
		case r == '.', r == '_':
			b.WriteRune(r)
			lastDash = false
		default:
			if !lastDash {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}

	out := strings.Trim(b.String(), "-")
	if out == "" {
		return defaultSlug
	}
	return out
}
