package service

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"

	"github.com/DedovInside/AutoInspect/backend/internal/config"
	"github.com/DedovInside/AutoInspect/backend/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestModelServiceListActiveSpecializedModelsFiltersUniversalAndInactive(t *testing.T) {
	t.Parallel()

	repo := &fakeCarModelRepo{
		models: []*domain.CarModel{
			{ID: uuid.New(), Make: "general", IsUniversal: true, IsActive: true},
			{ID: uuid.New(), Make: "Volkswagen", Model: "Golf", IsUniversal: false, IsActive: true},
			{ID: uuid.New(), Make: "BMW", Model: "3", IsUniversal: false, IsActive: false},
		},
	}
	svc := NewModelService(repo, nil, nil)

	got, err := svc.ListActiveSpecializedModels(context.Background())

	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, "Volkswagen", got[0].Make)
}

func TestModelServiceDeactivateModelValidatesIDAndExistingModel(t *testing.T) {
	t.Parallel()

	modelID := uuid.New()
	repo := &fakeCarModelRepo{
		byID: map[uuid.UUID]*domain.CarModel{
			modelID: {ID: modelID, IsActive: true},
		},
	}
	svc := NewModelService(repo, nil, nil)

	require.ErrorIs(t, svc.DeactivateModel(context.Background(), uuid.Nil), domain.ErrInvalidInput)
	require.NoError(t, svc.DeactivateModel(context.Background(), modelID))
	require.False(t, repo.byID[modelID].IsActive)
}

func TestModelServiceUploadModelArtifactsCleansUpUploadedFilesOnMetadataFailure(t *testing.T) {
	t.Parallel()

	repo := &failingCreateCarModelRepo{fakeCarModelRepo: fakeCarModelRepo{byID: make(map[uuid.UUID]*domain.CarModel)}}
	fileRepo := &fakeFileRepo{}
	svc := NewModelService(repo, fileRepo, &config.S3Config{BucketModels: "autoinspect-models"})

	got, err := svc.UploadModelArtifacts(context.Background(), &UploadModelArtifactsInput{
		IdempotencyKey: "vw-golf-v1",
		Make:           "Volkswagen",
		Model:          "Golf",
		Generation:     "5",
		YearFrom:       2008,
		YearTo:         2013,
		Version:        "v1",
		PartsModel:     ModelArtifactFile{File: nopMultipartFile("model"), Filename: "parts_segmentation.pt"},
		PartsConfig:    ModelArtifactFile{File: nopMultipartFile(`{"ok":true}`), Filename: "parts_inference_config.json"},
		PartsCatalog:   ModelArtifactFile{File: nopMultipartFile(`{"parts":[]}`), Filename: "parts_catalog.json"},
	})

	require.Nil(t, got)
	require.Error(t, err)
	require.Len(t, fileRepo.uploaded, 3)
	require.ElementsMatch(t, fileRepo.uploaded, fileRepo.deleted)
}

type failingCreateCarModelRepo struct {
	fakeCarModelRepo
}

func (r *failingCreateCarModelRepo) CreateModel(context.Context, *domain.CarModel) error {
	return domain.ErrInternal
}

type fakeFileRepo struct {
	uploaded     []string
	deleted      []string
	existsChecks []string
	missing      map[string]struct{}
	downloads    map[string]string
}

func (r *fakeFileRepo) Upload(_ context.Context, _, objectKey string, data io.Reader, _ string, _ int64) error {
	_, _ = io.ReadAll(data)
	r.uploaded = append(r.uploaded, objectKey)
	return nil
}

func (r *fakeFileRepo) Download(_ context.Context, _, objectKey string) (io.ReadCloser, error) {
	if r.downloads != nil {
		if content, ok := r.downloads[objectKey]; ok {
			return io.NopCloser(bytes.NewReader([]byte(content))), nil
		}
	}
	return io.NopCloser(bytes.NewReader(nil)), nil
}

func (r *fakeFileRepo) Exists(_ context.Context, _, objectKey string) (bool, error) {
	r.existsChecks = append(r.existsChecks, objectKey)
	if r.missing != nil {
		if _, ok := r.missing[objectKey]; ok {
			return false, nil
		}
	}
	return true, nil
}

func (r *fakeFileRepo) Delete(_ context.Context, _, objectKey string) error {
	r.deleted = append(r.deleted, objectKey)
	return nil
}

func (r *fakeFileRepo) GetPresignedURL(context.Context, string, string, time.Duration) (string, error) {
	return "https://example.test/image.jpg", nil
}

type testMultipartFile struct {
	*bytes.Reader
}

func (f testMultipartFile) Close() error {
	return nil
}

func nopMultipartFile(value string) testMultipartFile {
	return testMultipartFile{Reader: bytes.NewReader([]byte(value))}
}
