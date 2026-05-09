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

type CarServiceImageRepo struct {
	queries *db.Queries
}

func NewCarServiceImageRepo(tx DBTX) *CarServiceImageRepo {
	return &CarServiceImageRepo{queries: db.New(tx)}
}

func (r *CarServiceImageRepo) Create(ctx context.Context, image *domain.CarServiceImage) error {
	sortOrder, err := intToInt32Checked(image.SortOrder)
	if err != nil {
		return domain.ErrInvalidInput
	}

	params := db.CreateCarServiceImageParams{
		ID:               pgtype.UUID{Bytes: image.ID, Valid: true},
		ProfileID:        pgtype.UUID{Bytes: image.ProfileID, Valid: true},
		S3Key:            image.S3Key,
		IsPrimary:        image.IsPrimary,
		SortOrder:        sortOrder,
		OriginalFilename: image.OriginalFilename,
		ContentType:      image.ContentType,
		SizeBytes:        image.SizeBytes,
		CreatedAt:        pgtype.Timestamptz{Time: image.CreatedAt, Valid: true},
	}

	if err := r.queries.CreateCarServiceImage(ctx, params); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.ErrAlreadyExists
		}
		return domain.ErrInternal
	}

	return nil
}

func (r *CarServiceImageRepo) ListByProfileID(ctx context.Context,
	profileID uuid.UUID) ([]*domain.CarServiceImage, error) {
	dbImages, err := r.queries.ListCarServiceImagesByProfileID(ctx, pgtype.UUID{Bytes: profileID, Valid: true})
	if err != nil {
		return nil, domain.ErrInternal
	}

	return toDomainCarServiceImages(dbImages), nil
}

func (r *CarServiceImageRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.CarServiceImage, error) {
	dbImage, err := r.queries.GetCarServiceImageByID(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, domain.ErrInternal
	}
	return toDomainCarServiceImage(&dbImage), nil
}

func (r *CarServiceImageRepo) Delete(ctx context.Context, id uuid.UUID) error {
	rowsAffected, err := r.queries.DeleteCarServiceImage(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		return domain.ErrInternal
	}

	if rowsAffected == 0 {
		return domain.ErrNotFound
	}

	return nil
}

func (r *CarServiceImageRepo) ClearPrimary(ctx context.Context, profileID uuid.UUID) error {
	if err := r.queries.ClearPrimaryCarServiceImage(ctx, pgtype.UUID{Bytes: profileID, Valid: true}); err != nil {
		return domain.ErrInternal
	}

	return nil
}

func (r *CarServiceImageRepo) SetPrimary(ctx context.Context, id uuid.UUID) error {
	rowsAffected, err := r.queries.SetPrimaryCarServiceImage(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		return domain.ErrInternal
	}

	if rowsAffected == 0 {
		return domain.ErrNotFound
	}

	return nil
}

func (r *CarServiceImageRepo) NextSortOrder(ctx context.Context, profileID uuid.UUID) (int, error) {
	sortOrder, err := r.queries.NextCarServiceImageSortOrder(ctx, pgtype.UUID{Bytes: profileID, Valid: true})
	if err != nil {
		return 0, domain.ErrInternal
	}

	return int(sortOrder), nil
}

func toDomainCarServiceImages(dbImages []db.CarServiceImage) []*domain.CarServiceImage {
	images := make([]*domain.CarServiceImage, 0, len(dbImages))
	for i := range dbImages {
		images = append(images, toDomainCarServiceImage(&dbImages[i]))
	}

	return images
}

func toDomainCarServiceImage(dbImage *db.CarServiceImage) *domain.CarServiceImage {
	return &domain.CarServiceImage{
		ID:               fromPgUUID(dbImage.ID),
		ProfileID:        fromPgUUID(dbImage.ProfileID),
		S3Key:            dbImage.S3Key,
		IsPrimary:        dbImage.IsPrimary,
		SortOrder:        int(dbImage.SortOrder),
		OriginalFilename: dbImage.OriginalFilename,
		ContentType:      dbImage.ContentType,
		SizeBytes:        dbImage.SizeBytes,
		CreatedAt:        dbImage.CreatedAt.Time,
	}
}
