package service

import (
	"context"
	"testing"

	"github.com/DedovInside/AutoInspect/backend/internal/config"
	"github.com/DedovInside/AutoInspect/backend/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestAnalysisServiceFindModelForCarUsesSpecializedModelFirst(t *testing.T) {
	t.Parallel()

	specialized := &domain.CarModel{Make: "Volkswagen", Model: "Golf", IsUniversal: false}
	svc := NewAnalysisService(
		nil,
		nil,
		&fakeCarModelRepo{activeModel: specialized},
		nil,
		nil,
		nil,
		nil,
		nil,
	)

	got, err := svc.findModelForCar(context.Background(), domain.CarInfo{
		Make:       "Volkswagen",
		Model:      "Golf",
		Generation: "5",
		Year:       2015,
	})

	require.NoError(t, err)
	require.Same(t, specialized, got)
}

func TestAnalysisServiceFindModelForCarFallsBackToUniversalModel(t *testing.T) {
	t.Parallel()

	universal := &domain.CarModel{Make: "general", Model: "general", IsUniversal: true}
	svc := NewAnalysisService(
		nil,
		nil,
		&fakeCarModelRepo{universalModel: universal},
		nil,
		nil,
		nil,
		nil,
		nil,
	)

	got, err := svc.findModelForCar(context.Background(), domain.CarInfo{
		Make: "Unknown",
		Year: 2015,
	})

	require.NoError(t, err)
	require.Same(t, universal, got)
}

func TestAnalysisServiceEnsureModelArtifactsExistChecksAllRequiredKeys(t *testing.T) {
	t.Parallel()

	fileRepo := &fakeFileRepo{}
	svc := NewAnalysisService(
		fileRepo,
		nil,
		nil,
		nil,
		nil,
		nil,
		&config.S3Config{BucketModels: "autoinspect-models"},
		nil,
	)

	err := svc.ensureModelArtifactsExist(context.Background(), &domain.CarModel{
		PartsModelS3Key:   "models/general/parts_segmentation.pt",
		PartsConfigS3Key:  "models/general/parts_inference_config.json",
		PartsCatalogS3Key: "models/general/parts_catalog.json",
	})

	require.NoError(t, err)
	require.ElementsMatch(t, []string{
		"models/general/parts_segmentation.pt",
		"models/general/parts_inference_config.json",
		"models/general/parts_catalog.json",
	}, fileRepo.existsChecks)
}

func TestAnalysisServiceEnsureModelArtifactsExistFailsWhenArtifactIsMissing(t *testing.T) {
	t.Parallel()

	fileRepo := &fakeFileRepo{
		missing: map[string]struct{}{
			"models/general/parts_catalog.json": {},
		},
	}
	svc := NewAnalysisService(
		fileRepo,
		nil,
		nil,
		nil,
		nil,
		nil,
		&config.S3Config{BucketModels: "autoinspect-models"},
		nil,
	)

	err := svc.ensureModelArtifactsExist(context.Background(), &domain.CarModel{
		PartsModelS3Key:   "models/general/parts_segmentation.pt",
		PartsConfigS3Key:  "models/general/parts_inference_config.json",
		PartsCatalogS3Key: "models/general/parts_catalog.json",
	})

	require.ErrorIs(t, err, domain.ErrInvalidModel)
}
