package postgres

import (
	"context"
	"errors"

	"github.com/DedovInside/AutoInspect/backend/internal/domain"
	"github.com/DedovInside/AutoInspect/backend/internal/repository/postgres/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

type CarModelRepo struct {
	queries *db.Queries
}

func NewCarModelRepo(tx DBTX) *CarModelRepo {
	return &CarModelRepo{queries: db.New(tx)}
}

func (r *CarModelRepo) FindActiveModel(ctx context.Context,
	carMake, model, generation string, year int) (*domain.CarModel, error) {
	generationPtr := stringPtrOrNil(generation)
	year32, err := intToInt32Checked(year)
	if err != nil {
		return nil, domain.ErrInvalidInput
	}

	dbModel, err := r.queries.FindActiveCarModel(ctx, db.FindActiveCarModelParams{
		Make:       carMake,
		Model:      model,
		Generation: generationPtr,
		YearFrom:   year32,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrInvalidModel
		}
		return nil, domain.ErrInternal
	}

	return toDomainCarModel(&dbModel), nil
}

func (r *CarModelRepo) GetUniversalModel(ctx context.Context) (*domain.CarModel, error) {
	dbModel, err := r.queries.GetUniversalCarModel(ctx)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrInvalidModel
		}
		return nil, domain.ErrInternal
	}

	return toDomainCarModel(&dbModel), nil
}

func (r *CarModelRepo) CreateModel(ctx context.Context, cm *domain.CarModel) error {
	yearFrom32, err := intToInt32Checked(cm.YearFrom)
	if err != nil {
		return domain.ErrInvalidInput
	}

	params := db.CreateCarModelParams{
		ID:                pgtype.UUID{Bytes: cm.ID, Valid: true},
		Make:              cm.Make,
		Model:             cm.Model,
		Generation:        stringPtrOrNil(cm.Generation),
		YearFrom:          yearFrom32,
		PartsModelS3Key:   cm.PartsModelS3Key,
		PartsConfigS3Key:  cm.PartsConfigS3Key,
		PartsCatalogS3Key: cm.PartsCatalogS3Key,
		ModelVersion:      cm.ModelVersion,
		IsUniversal:       boolPtr(cm.IsUniversal),
		IsActive:          boolPtr(cm.IsActive),
		CreatedAt:         pgtype.Timestamptz{Time: cm.CreatedAt, Valid: true},
	}

	if cm.YearTo > 0 {
		y, err := intToInt32Checked(cm.YearTo)
		if err != nil {
			return domain.ErrInvalidInput
		}
		params.YearTo = &y
	}

	err = r.queries.CreateCarModel(ctx, params)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.ErrAlreadyExists
		}
		return domain.ErrInternal
	}

	return nil
}

func (r *CarModelRepo) ListModels(ctx context.Context, limit, offset int) ([]*domain.CarModel, error) {
	limit32, err := intToInt32Checked(limit)
	if err != nil {
		return nil, domain.ErrInvalidInput
	}

	offset32, err := intToInt32Checked(offset)
	if err != nil {
		return nil, domain.ErrInvalidInput
	}

	dbModels, err := r.queries.ListCarModels(ctx, db.ListCarModelsParams{
		Limit:  limit32,
		Offset: offset32,
	})
	if err != nil {
		return nil, domain.ErrInternal
	}

	models := make([]*domain.CarModel, 0, len(dbModels))
	for i := range dbModels {
		models = append(models, toDomainCarModel(&dbModels[i]))
	}

	return models, nil
}

func (r *CarModelRepo) GetModelByID(ctx context.Context, id uuid.UUID) (*domain.CarModel, error) {
	dbModel, err := r.queries.GetCarModelByID(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, domain.ErrInternal
	}

	return toDomainCarModel(&dbModel), nil
}

func (r *CarModelRepo) DeactivateModel(ctx context.Context, id uuid.UUID) error {
	if err := r.queries.DeactivateCarModel(ctx, pgtype.UUID{Bytes: id, Valid: true}); err != nil {
		return domain.ErrInternal
	}

	return nil
}

func toDomainCarModel(dbModel *db.CarModel) *domain.CarModel {
	yearTo := 0
	if dbModel.YearTo != nil {
		yearTo = int(*dbModel.YearTo)
	}

	generation := ""
	if dbModel.Generation != nil {
		generation = *dbModel.Generation
	}

	isUniversal := false
	if dbModel.IsUniversal != nil {
		isUniversal = *dbModel.IsUniversal
	}

	isActive := true
	if dbModel.IsActive != nil {
		isActive = *dbModel.IsActive
	}

	return &domain.CarModel{
		ID:                fromPgUUID(dbModel.ID),
		Make:              dbModel.Make,
		Model:             dbModel.Model,
		Generation:        generation,
		YearFrom:          int(dbModel.YearFrom),
		YearTo:            yearTo,
		PartsModelS3Key:   dbModel.PartsModelS3Key,
		PartsConfigS3Key:  dbModel.PartsConfigS3Key,
		PartsCatalogS3Key: dbModel.PartsCatalogS3Key,
		ModelVersion:      dbModel.ModelVersion,
		IsUniversal:       isUniversal,
		IsActive:          isActive,
		CreatedAt:         dbModel.CreatedAt.Time,
	}
}
