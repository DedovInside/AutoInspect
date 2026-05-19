//go:build integration

package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/DedovInside/AutoInspect/backend/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestCarModelRepoCreateFindListAndDeactivate(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	dsn := postgresContainerDSN(t, ctx)
	db, err := New(ctx, dsn, 2, time.Minute)
	require.NoError(t, err)
	t.Cleanup(db.Close)

	prepareCarModelsSchema(t, ctx, db)

	repo := NewCarModelRepo(db)
	specialized := &domain.CarModel{
		ID:                uuid.New(),
		Make:              "Volkswagen",
		Model:             "Golf",
		Generation:        "5",
		YearFrom:          2008,
		YearTo:            2013,
		PartsModelS3Key:   "models/vw-golf-5/parts_segmentation.pt",
		PartsConfigS3Key:  "models/vw-golf-5/parts_inference_config.json",
		PartsCatalogS3Key: "models/vw-golf-5/parts_catalog.json",
		ModelVersion:      "v1",
		IsActive:          true,
		CreatedAt:         time.Now().UTC(),
	}
	universal := &domain.CarModel{
		ID:                uuid.New(),
		Make:              "general",
		Model:             "general",
		Generation:        "general",
		YearFrom:          1900,
		YearTo:            2100,
		PartsModelS3Key:   "models/general/parts_segmentation.pt",
		PartsConfigS3Key:  "models/general/parts_inference_config.json",
		PartsCatalogS3Key: "models/general/parts_catalog.json",
		ModelVersion:      "v1",
		IsUniversal:       true,
		IsActive:          true,
		CreatedAt:         time.Now().UTC(),
	}

	require.NoError(t, repo.CreateModel(ctx, specialized))
	require.NoError(t, repo.CreateModel(ctx, universal))

	found, err := repo.FindActiveModel(ctx, "Volkswagen", "Golf", "5", 2010)
	require.NoError(t, err)
	require.Equal(t, specialized.ID, found.ID)
	require.Equal(t, "models/vw-golf-5/parts_catalog.json", found.PartsCatalogS3Key)

	universalFound, err := repo.GetUniversalModel(ctx)
	require.NoError(t, err)
	require.Equal(t, universal.ID, universalFound.ID)

	models, err := repo.ListModels(ctx, 10, 0)
	require.NoError(t, err)
	require.Len(t, models, 2)

	require.NoError(t, repo.DeactivateModel(ctx, specialized.ID))
	_, err = repo.FindActiveModel(ctx, "Volkswagen", "Golf", "5", 2010)
	require.ErrorIs(t, err, domain.ErrInvalidModel)
}

func prepareCarModelsSchema(t *testing.T, ctx context.Context, db *DB) {
	t.Helper()

	_, err := db.Exec(ctx, `
		CREATE TABLE car_models
		(
			id UUID PRIMARY KEY,
			make VARCHAR(100) NOT NULL,
			model VARCHAR(100) NOT NULL,
			generation VARCHAR(100),
			year_from INTEGER NOT NULL,
			year_to INTEGER,
			parts_model_s3_key VARCHAR(500) NOT NULL,
			parts_config_s3_key VARCHAR(500) NOT NULL,
			parts_catalog_s3_key VARCHAR(500) NOT NULL,
			model_version VARCHAR(50) NOT NULL,
			is_universal BOOLEAN DEFAULT FALSE,
			is_active BOOLEAN DEFAULT TRUE,
			created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
		)
	`)
	require.NoError(t, err)

	_, err = db.Exec(ctx, `
		CREATE UNIQUE INDEX idx_car_models_unique_active
		ON car_models (make, model, generation, year_from, COALESCE(year_to, 9999), model_version)
		WHERE is_active = true
	`)
	require.NoError(t, err)
}
