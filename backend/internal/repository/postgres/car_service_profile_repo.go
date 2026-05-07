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

type CarServiceProfileRepo struct {
	queries *db.Queries
}

func NewCarServiceProfileRepo(tx DBTX) *CarServiceProfileRepo {
	return &CarServiceProfileRepo{queries: db.New(tx)}
}

func (r *CarServiceProfileRepo) Create(ctx context.Context, profile *domain.CarServiceProfile) error {
	params := db.CreateCarServiceProfileParams{
		ID:               pgtype.UUID{Bytes: profile.ID, Valid: true},
		UserID:           pgtype.UUID{Bytes: profile.UserID, Valid: true},
		OrganizationName: profile.OrganizationName,
		City:             profile.City,
		Address:          profile.Address,
		Phone:            profile.Phone,
		Email:            profile.Email,
		WebsiteUrl:       profile.WebsiteURL,
		ContactInfo:      profile.ContactInfo,
		Description:      profile.Description,
		IsActive:         profile.IsActive,
		CreatedAt:        pgtype.Timestamptz{Time: profile.CreatedAt, Valid: true},
		UpdatedAt:        pgtype.Timestamptz{Time: profile.UpdatedAt, Valid: true},
	}

	if err := r.queries.CreateCarServiceProfile(ctx, params); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.ErrAlreadyExists
		}
		return domain.ErrInternal
	}
	return nil
}

func (r *CarServiceProfileRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.CarServiceProfile, error) {
	dbProfile, err := r.queries.GetCarServiceProfileByID(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, domain.ErrInternal
	}
	return toDomainCarServiceProfile(&dbProfile), nil
}

func (r *CarServiceProfileRepo) GetByUserID(ctx context.Context, userID uuid.UUID) (*domain.CarServiceProfile, error) {
	dbProfile, err := r.queries.GetCarServiceProfileByUserID(ctx, pgtype.UUID{Bytes: userID, Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, domain.ErrInternal
	}
	return toDomainCarServiceProfile(&dbProfile), nil
}

func (r *CarServiceProfileRepo) Update(ctx context.Context, input *domain.UpdateCarServiceProfileInput) error {
	if input == nil {
		return domain.ErrInvalidInput
	}

	params := db.UpdateCarServiceProfileParams{
		UserID:           pgtype.UUID{Bytes: input.UserID, Valid: true},
		OrganizationName: input.OrganizationName,
		City:             input.City,
		Address:          input.Address,
		Phone:            input.Phone,
		Email:            input.Email,
		WebsiteUrl:       input.WebsiteURL,
		ContactInfo:      input.ContactInfo,
		Description:      input.Description,
		IsActive:         input.IsActive,
		UpdatedAt:        pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
	}

	rowsAffected, err := r.queries.UpdateCarServiceProfile(ctx, params)
	if err != nil {
		return domain.ErrInternal
	}
	if rowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *CarServiceProfileRepo) SetActive(ctx context.Context, userID uuid.UUID, isActive bool) error {
	rowsAffected, err := r.queries.SetCarServiceProfileActive(ctx, db.SetCarServiceProfileActiveParams{
		UserID:    pgtype.UUID{Bytes: userID, Valid: true},
		IsActive:  isActive,
		UpdatedAt: pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
	})
	if err != nil {
		return domain.ErrInternal
	}
	if rowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func toDomainCarServiceProfile(dbProfile *db.CarServiceProfile) *domain.CarServiceProfile {
	return &domain.CarServiceProfile{
		ID:               fromPgUUID(dbProfile.ID),
		UserID:           fromPgUUID(dbProfile.UserID),
		OrganizationName: dbProfile.OrganizationName,
		City:             dbProfile.City,
		Address:          dbProfile.Address,
		Phone:            dbProfile.Phone,
		Email:            dbProfile.Email,
		WebsiteURL:       dbProfile.WebsiteUrl,
		ContactInfo:      dbProfile.ContactInfo,
		Description:      dbProfile.Description,
		IsActive:         dbProfile.IsActive,
		CreatedAt:        dbProfile.CreatedAt.Time,
		UpdatedAt:        dbProfile.UpdatedAt.Time,
	}
}
