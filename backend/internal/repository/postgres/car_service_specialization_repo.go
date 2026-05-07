package postgres

import (
	"context"
	"errors"

	"github.com/DedovInside/AutoInspect/backend/internal/domain"
	"github.com/DedovInside/AutoInspect/backend/internal/repository/postgres/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

type DamageTypeRepo struct {
	queries *db.Queries
}

func NewDamageTypeRepo(tx DBTX) *DamageTypeRepo {
	return &DamageTypeRepo{queries: db.New(tx)}
}

func (r *DamageTypeRepo) ListActive(ctx context.Context) ([]*domain.DamageType, error) {
	dbTypes, err := r.queries.ListActiveDamageTypes(ctx)
	if err != nil {
		return nil, domain.ErrInternal
	}
	return toDomainDamageTypes(dbTypes), nil
}

func (r *DamageTypeRepo) ExistsActive(ctx context.Context, code string) (bool, error) {
	exists, err := r.queries.ExistsActiveDamageType(ctx, code)
	if err != nil {
		return false, domain.ErrInternal
	}
	return exists, nil
}

type PartCategoryRepo struct {
	queries *db.Queries
}

func NewPartCategoryRepo(tx DBTX) *PartCategoryRepo {
	return &PartCategoryRepo{queries: db.New(tx)}
}

func (r *PartCategoryRepo) ListActive(ctx context.Context) ([]*domain.PartCategory, error) {
	dbCategories, err := r.queries.ListActivePartCategories(ctx)
	if err != nil {
		return nil, domain.ErrInternal
	}
	return toDomainPartCategories(dbCategories), nil
}

func (r *PartCategoryRepo) ExistsActive(ctx context.Context, code string) (bool, error) {
	exists, err := r.queries.ExistsActivePartCategory(ctx, code)
	if err != nil {
		return false, domain.ErrInternal
	}
	return exists, nil
}

type CarServiceSpecializationRepo struct {
	queries *db.Queries
}

func NewCarServiceSpecializationRepo(tx DBTX) *CarServiceSpecializationRepo {
	return &CarServiceSpecializationRepo{queries: db.New(tx)}
}

func (r *CarServiceSpecializationRepo) Create(
	ctx context.Context,
	specialization *domain.CarServiceSpecialization,
) error {
	params := db.CreateCarServiceSpecializationParams{
		ID:               pgtype.UUID{Bytes: specialization.ID, Valid: true},
		ProfileID:        pgtype.UUID{Bytes: specialization.ProfileID, Valid: true},
		DamageTypeCode:   specialization.DamageTypeCode,
		PartCategoryCode: specialization.PartCategoryCode,
		CreatedAt:        pgtype.Timestamptz{Time: specialization.CreatedAt, Valid: true},
	}

	if err := r.queries.CreateCarServiceSpecialization(ctx, params); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.ErrAlreadyExists
		}
		return domain.ErrInternal
	}
	return nil
}

func (r *CarServiceSpecializationRepo) ListByProfileID(
	ctx context.Context,
	profileID uuid.UUID,
) ([]*domain.CarServiceSpecialization, error) {
	dbSpecializations, err := r.queries.ListCarServiceSpecializationsByProfileID(ctx, pgtype.UUID{Bytes: profileID, Valid: true})
	if err != nil {
		return nil, domain.ErrInternal
	}
	return toDomainCarServiceSpecializations(dbSpecializations), nil
}

func (r *CarServiceSpecializationRepo) DeleteByProfileID(ctx context.Context, profileID uuid.UUID) error {
	if err := r.queries.DeleteCarServiceSpecializationsByProfileID(ctx, pgtype.UUID{Bytes: profileID, Valid: true}); err != nil {
		return domain.ErrInternal
	}
	return nil
}

func toDomainDamageTypes(dbTypes []db.DamageType) []*domain.DamageType {
	items := make([]*domain.DamageType, 0, len(dbTypes))
	for i := range dbTypes {
		items = append(items, toDomainDamageType(&dbTypes[i]))
	}
	return items
}

func toDomainDamageType(dbType *db.DamageType) *domain.DamageType {
	return &domain.DamageType{
		Code:      dbType.Code,
		NameRU:    dbType.NameRu,
		IsActive:  dbType.IsActive,
		CreatedAt: dbType.CreatedAt.Time,
		UpdatedAt: dbType.UpdatedAt.Time,
	}
}

func toDomainPartCategories(dbCategories []db.PartCategory) []*domain.PartCategory {
	items := make([]*domain.PartCategory, 0, len(dbCategories))
	for i := range dbCategories {
		items = append(items, toDomainPartCategory(&dbCategories[i]))
	}
	return items
}

func toDomainPartCategory(dbCategory *db.PartCategory) *domain.PartCategory {
	return &domain.PartCategory{
		Code:      dbCategory.Code,
		NameRU:    dbCategory.NameRu,
		IsPair:    dbCategory.IsPair,
		IsActive:  dbCategory.IsActive,
		CreatedAt: dbCategory.CreatedAt.Time,
		UpdatedAt: dbCategory.UpdatedAt.Time,
	}
}

func toDomainCarServiceSpecializations(
	dbSpecializations []db.CarServiceSpecialization,
) []*domain.CarServiceSpecialization {
	items := make([]*domain.CarServiceSpecialization, 0, len(dbSpecializations))
	for i := range dbSpecializations {
		items = append(items, toDomainCarServiceSpecialization(&dbSpecializations[i]))
	}
	return items
}

func toDomainCarServiceSpecialization(
	dbSpecialization *db.CarServiceSpecialization,
) *domain.CarServiceSpecialization {
	return &domain.CarServiceSpecialization{
		ID:               fromPgUUID(dbSpecialization.ID),
		ProfileID:        fromPgUUID(dbSpecialization.ProfileID),
		DamageTypeCode:   dbSpecialization.DamageTypeCode,
		PartCategoryCode: dbSpecialization.PartCategoryCode,
		CreatedAt:        dbSpecialization.CreatedAt.Time,
	}
}
