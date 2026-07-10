package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/DedovInside/AutoInspect/backend/internal/domain"
	"github.com/DedovInside/AutoInspect/backend/internal/repository/postgres/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

type VehicleCatalogRepo struct {
	queries *db.Queries
}

func NewVehicleCatalogRepo(tx DBTX) *VehicleCatalogRepo {
	return &VehicleCatalogRepo{queries: db.New(tx)}
}

func (r *VehicleCatalogRepo) ListMakes(ctx context.Context) ([]*domain.VehicleMake, error) {
	dbMakes, err := r.queries.ListVehicleMakes(ctx)
	if err != nil {
		return nil, domain.ErrInternal
	}

	makes := make([]*domain.VehicleMake, 0, len(dbMakes))
	for i := range dbMakes {
		makes = append(makes, toDomainVehicleMake(&dbMakes[i]))
	}

	return makes, nil
}

func (r *VehicleCatalogRepo) ListMakesForAdmin(ctx context.Context) ([]*domain.VehicleMake, error) {
	dbMakes, err := r.queries.ListVehicleMakesForAdmin(ctx)
	if err != nil {
		return nil, domain.ErrInternal
	}

	makes := make([]*domain.VehicleMake, 0, len(dbMakes))
	for i := range dbMakes {
		makes = append(makes, toDomainVehicleMake(&dbMakes[i]))
	}

	return makes, nil
}

func (r *VehicleCatalogRepo) GetMakeByID(ctx context.Context, id uuid.UUID) (*domain.VehicleMake, error) {
	dbMake, err := r.queries.GetVehicleMakeByID(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, domain.ErrInternal
	}

	return toDomainVehicleMake(&dbMake), nil
}

func (r *VehicleCatalogRepo) CreateMake(ctx context.Context, vehicleMake *domain.VehicleMake) error {
	if vehicleMake == nil {
		return domain.ErrInvalidInput
	}

	err := r.queries.CreateVehicleMake(ctx, db.CreateVehicleMakeParams{
		ID:        pgtype.UUID{Bytes: vehicleMake.ID, Valid: true},
		Name:      vehicleMake.Name,
		Slug:      vehicleMake.Slug,
		IsActive:  vehicleMake.IsActive,
		CreatedAt: pgtype.Timestamptz{Time: vehicleMake.CreatedAt, Valid: true},
		UpdatedAt: pgtype.Timestamptz{Time: vehicleMake.UpdatedAt, Valid: true},
	})

	return vehicleCatalogWriteError(err)
}

func (r *VehicleCatalogRepo) UpdateMake(ctx context.Context, input domain.UpdateVehicleMakeInput) error {
	rowsAffected, err := r.queries.UpdateVehicleMake(ctx, db.UpdateVehicleMakeParams{
		ID:        pgtype.UUID{Bytes: input.ID, Valid: true},
		Name:      input.Name,
		Slug:      input.Slug,
		UpdatedAt: pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
	})
	if err != nil {
		return vehicleCatalogWriteError(err)
	}

	return vehicleCatalogRowsAffectedError(rowsAffected)
}

func (r *VehicleCatalogRepo) SetMakeActive(ctx context.Context, input domain.SetVehicleMakeActiveInput) error {
	rowsAffected, err := r.queries.SetVehicleMakeActive(ctx, db.SetVehicleMakeActiveParams{
		ID:        pgtype.UUID{Bytes: input.ID, Valid: true},
		IsActive:  input.IsActive,
		UpdatedAt: pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
	})
	if err != nil {
		return vehicleCatalogWriteError(err)
	}

	return vehicleCatalogRowsAffectedError(rowsAffected)
}

func (r *VehicleCatalogRepo) ListModelsByMakeID(ctx context.Context, makeID uuid.UUID) ([]*domain.VehicleModel, error) {
	dbModels, err := r.queries.ListVehicleModelsByMakeID(ctx, pgtype.UUID{Bytes: makeID, Valid: true})
	if err != nil {
		return nil, domain.ErrInternal
	}

	models := make([]*domain.VehicleModel, 0, len(dbModels))
	for i := range dbModels {
		models = append(models, toDomainVehicleModel(&dbModels[i]))
	}

	return models, nil
}

func (r *VehicleCatalogRepo) ListModelsByMakeIDForAdmin(ctx context.Context,
	makeID uuid.UUID) ([]*domain.VehicleModel, error) {
	dbModels, err := r.queries.ListVehicleModelsByMakeIDForAdmin(ctx, pgtype.UUID{Bytes: makeID, Valid: true})
	if err != nil {
		return nil, domain.ErrInternal
	}

	models := make([]*domain.VehicleModel, 0, len(dbModels))
	for i := range dbModels {
		models = append(models, toDomainVehicleModel(&dbModels[i]))
	}

	return models, nil
}

func (r *VehicleCatalogRepo) GetModelByID(ctx context.Context, id uuid.UUID) (*domain.VehicleModel, error) {
	dbModel, err := r.queries.GetVehicleModelByID(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, domain.ErrInternal
	}

	return toDomainVehicleModel(&dbModel), nil
}

func (r *VehicleCatalogRepo) CreateModel(ctx context.Context, model *domain.VehicleModel) error {
	if model == nil {
		return domain.ErrInvalidInput
	}

	err := r.queries.CreateVehicleModel(ctx, db.CreateVehicleModelParams{
		ID:        pgtype.UUID{Bytes: model.ID, Valid: true},
		MakeID:    pgtype.UUID{Bytes: model.MakeID, Valid: true},
		Name:      model.Name,
		Slug:      model.Slug,
		IsActive:  model.IsActive,
		CreatedAt: pgtype.Timestamptz{Time: model.CreatedAt, Valid: true},
		UpdatedAt: pgtype.Timestamptz{Time: model.UpdatedAt, Valid: true},
	})

	return vehicleCatalogWriteError(err)
}

func (r *VehicleCatalogRepo) UpdateModel(ctx context.Context, input domain.UpdateVehicleModelInput) error {
	rowsAffected, err := r.queries.UpdateVehicleModel(ctx, db.UpdateVehicleModelParams{
		ID:        pgtype.UUID{Bytes: input.ID, Valid: true},
		MakeID:    pgtype.UUID{Bytes: input.MakeID, Valid: true},
		Name:      input.Name,
		Slug:      input.Slug,
		UpdatedAt: pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
	})
	if err != nil {
		return vehicleCatalogWriteError(err)
	}

	return vehicleCatalogRowsAffectedError(rowsAffected)
}

func (r *VehicleCatalogRepo) SetModelActive(ctx context.Context, input domain.SetVehicleModelActiveInput) error {
	rowsAffected, err := r.queries.SetVehicleModelActive(ctx, db.SetVehicleModelActiveParams{
		ID:        pgtype.UUID{Bytes: input.ID, Valid: true},
		IsActive:  input.IsActive,
		UpdatedAt: pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
	})
	if err != nil {
		return vehicleCatalogWriteError(err)
	}

	return vehicleCatalogRowsAffectedError(rowsAffected)
}

func (r *VehicleCatalogRepo) ListGenerationsByModelID(ctx context.Context,
	modelID uuid.UUID) ([]*domain.VehicleGeneration, error) {
	dbGenerations, err := r.queries.ListVehicleGenerationsByModelID(ctx, pgtype.UUID{Bytes: modelID, Valid: true})
	if err != nil {
		return nil, domain.ErrInternal
	}

	generations := make([]*domain.VehicleGeneration, 0, len(dbGenerations))
	for i := range dbGenerations {
		generations = append(generations, toDomainVehicleGeneration(&dbGenerations[i]))
	}

	return generations, nil
}

func (r *VehicleCatalogRepo) ListGenerationsByModelIDForAdmin(ctx context.Context,
	modelID uuid.UUID) ([]*domain.VehicleGeneration, error) {
	dbGenerations, err := r.queries.ListVehicleGenerationsByModelIDForAdmin(ctx, pgtype.UUID{Bytes: modelID, Valid: true})
	if err != nil {
		return nil, domain.ErrInternal
	}

	generations := make([]*domain.VehicleGeneration, 0, len(dbGenerations))
	for i := range dbGenerations {
		generations = append(generations, toDomainVehicleGeneration(&dbGenerations[i]))
	}

	return generations, nil
}

func (r *VehicleCatalogRepo) GetGenerationByID(ctx context.Context, id uuid.UUID) (*domain.VehicleGeneration, error) {
	dbGeneration, err := r.queries.GetVehicleGenerationByID(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, domain.ErrInternal
	}

	return toDomainVehicleGeneration(&dbGeneration), nil
}

func (r *VehicleCatalogRepo) GetGenerationDetailsByID(ctx context.Context,
	generationID uuid.UUID) (*domain.VehicleGenerationDetails, error) {
	row, err := r.queries.GetVehicleGenerationDetailsByID(ctx, pgtype.UUID{Bytes: generationID, Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, domain.ErrInternal
	}

	return &domain.VehicleGenerationDetails{
		Make:       vehicleMakeFromDetailsRow(&row),
		Model:      vehicleModelFromDetailsRow(&row),
		Generation: vehicleGenerationFromDetailsRow(&row),
	}, nil
}

func (r *VehicleCatalogRepo) CreateGeneration(ctx context.Context, generation *domain.VehicleGeneration) error {
	if generation == nil {
		return domain.ErrInvalidInput
	}

	yearFrom, err := intToInt32Checked(generation.YearFrom)
	if err != nil {
		return domain.ErrInvalidInput
	}

	err = r.queries.CreateVehicleGeneration(ctx, db.CreateVehicleGenerationParams{
		ID:        pgtype.UUID{Bytes: generation.ID, Valid: true},
		ModelID:   pgtype.UUID{Bytes: generation.ModelID, Valid: true},
		Name:      generation.Name,
		Slug:      generation.Slug,
		YearFrom:  yearFrom,
		YearTo:    int32PtrOrNil(generation.YearTo),
		IsActive:  generation.IsActive,
		CreatedAt: pgtype.Timestamptz{Time: generation.CreatedAt, Valid: true},
		UpdatedAt: pgtype.Timestamptz{Time: generation.UpdatedAt, Valid: true},
	})

	return vehicleCatalogWriteError(err)
}

func (r *VehicleCatalogRepo) UpdateGeneration(ctx context.Context,
	input *domain.UpdateVehicleGenerationInput) error {
	if input == nil {
		return domain.ErrInvalidInput
	}

	yearFrom, err := intToInt32Checked(input.YearFrom)
	if err != nil {
		return domain.ErrInvalidInput
	}

	rowsAffected, err := r.queries.UpdateVehicleGeneration(ctx, db.UpdateVehicleGenerationParams{
		ID:        pgtype.UUID{Bytes: input.ID, Valid: true},
		ModelID:   pgtype.UUID{Bytes: input.ModelID, Valid: true},
		Name:      input.Name,
		Slug:      input.Slug,
		YearFrom:  yearFrom,
		YearTo:    int32PtrOrNil(input.YearTo),
		UpdatedAt: pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
	})
	if err != nil {
		return vehicleCatalogWriteError(err)
	}

	return vehicleCatalogRowsAffectedError(rowsAffected)
}

func (r *VehicleCatalogRepo) SetGenerationActive(ctx context.Context,
	input domain.SetVehicleGenerationActiveInput) error {
	rowsAffected, err := r.queries.SetVehicleGenerationActive(ctx, db.SetVehicleGenerationActiveParams{
		ID:        pgtype.UUID{Bytes: input.ID, Valid: true},
		IsActive:  input.IsActive,
		UpdatedAt: pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
	})
	if err != nil {
		return vehicleCatalogWriteError(err)
	}

	return vehicleCatalogRowsAffectedError(rowsAffected)
}

func toDomainVehicleMake(dbMake *db.VehicleMake) *domain.VehicleMake {
	return &domain.VehicleMake{
		ID:        fromPgUUID(dbMake.ID),
		Name:      dbMake.Name,
		Slug:      dbMake.Slug,
		IsActive:  dbMake.IsActive,
		CreatedAt: dbMake.CreatedAt.Time,
		UpdatedAt: dbMake.UpdatedAt.Time,
	}
}

func toDomainVehicleModel(dbModel *db.VehicleModel) *domain.VehicleModel {
	return &domain.VehicleModel{
		ID:        fromPgUUID(dbModel.ID),
		MakeID:    fromPgUUID(dbModel.MakeID),
		Name:      dbModel.Name,
		Slug:      dbModel.Slug,
		IsActive:  dbModel.IsActive,
		CreatedAt: dbModel.CreatedAt.Time,
		UpdatedAt: dbModel.UpdatedAt.Time,
	}
}

func toDomainVehicleGeneration(dbGeneration *db.VehicleGeneration) *domain.VehicleGeneration {
	return &domain.VehicleGeneration{
		ID:        fromPgUUID(dbGeneration.ID),
		ModelID:   fromPgUUID(dbGeneration.ModelID),
		Name:      dbGeneration.Name,
		Slug:      dbGeneration.Slug,
		YearFrom:  int(dbGeneration.YearFrom),
		YearTo:    int32Value(dbGeneration.YearTo),
		IsActive:  dbGeneration.IsActive,
		CreatedAt: dbGeneration.CreatedAt.Time,
		UpdatedAt: dbGeneration.UpdatedAt.Time,
	}
}

func vehicleMakeFromDetailsRow(row *db.GetVehicleGenerationDetailsByIDRow) *domain.VehicleMake {
	return &domain.VehicleMake{
		ID:        fromPgUUID(row.MakeID),
		Name:      row.MakeName,
		Slug:      row.MakeSlug,
		IsActive:  row.MakeIsActive,
		CreatedAt: row.MakeCreatedAt.Time,
		UpdatedAt: row.MakeUpdatedAt.Time,
	}
}

func vehicleModelFromDetailsRow(row *db.GetVehicleGenerationDetailsByIDRow) *domain.VehicleModel {
	return &domain.VehicleModel{
		ID:        fromPgUUID(row.ModelID),
		MakeID:    fromPgUUID(row.ModelMakeID),
		Name:      row.ModelName,
		Slug:      row.ModelSlug,
		IsActive:  row.ModelIsActive,
		CreatedAt: row.ModelCreatedAt.Time,
		UpdatedAt: row.ModelUpdatedAt.Time,
	}
}

func vehicleGenerationFromDetailsRow(row *db.GetVehicleGenerationDetailsByIDRow) *domain.VehicleGeneration {
	return &domain.VehicleGeneration{
		ID:        fromPgUUID(row.GenerationID),
		ModelID:   fromPgUUID(row.GenerationModelID),
		Name:      row.GenerationName,
		Slug:      row.GenerationSlug,
		YearFrom:  int(row.GenerationYearFrom),
		YearTo:    int32Value(row.GenerationYearTo),
		IsActive:  row.GenerationIsActive,
		CreatedAt: row.GenerationCreatedAt.Time,
		UpdatedAt: row.GenerationUpdatedAt.Time,
	}
}

func vehicleCatalogWriteError(err error) error {
	if err == nil {
		return nil
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return domain.ErrAlreadyExists
		case "23503", "23514":
			return domain.ErrInvalidInput
		}
	}

	return domain.ErrInternal
}

func vehicleCatalogRowsAffectedError(rowsAffected int64) error {
	if rowsAffected == 0 {
		return domain.ErrNotFound
	}

	return nil
}
