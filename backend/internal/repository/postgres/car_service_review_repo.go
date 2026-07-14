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

type CarServiceReviewRepo struct {
	queries *db.Queries
}

func NewCarServiceReviewRepo(tx DBTX) *CarServiceReviewRepo {
	return &CarServiceReviewRepo{queries: db.New(tx)}
}

func (r *CarServiceReviewRepo) Create(ctx context.Context, review *domain.CarServiceReview) error {
	rating, err := checkedReviewRating(review.Rating)
	if err != nil {
		return err
	}

	params := db.CreateCarServiceReviewParams{
		ID:                  pgtype.UUID{Bytes: review.ID, Valid: true},
		RepairRequestID:     pgtype.UUID{Bytes: review.RepairRequestID, Valid: true},
		CarServiceProfileID: pgtype.UUID{Bytes: review.CarServiceProfileID, Valid: true},
		UserID:              pgtype.UUID{Bytes: review.UserID, Valid: true},
		Rating:              rating,
		AuthorName:          review.AuthorName,
		Comment:             review.Comment,
		CreatedAt:           pgtype.Timestamptz{Time: review.CreatedAt, Valid: true},
		UpdatedAt:           pgtype.Timestamptz{Time: review.UpdatedAt, Valid: true},
	}

	if err := r.queries.CreateCarServiceReview(ctx, params); err != nil {
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

	return nil
}

func (r *CarServiceReviewRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.CarServiceReview, error) {
	dbReview, err := r.queries.GetCarServiceReviewByID(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, domain.ErrInternal
	}

	return toDomainCarServiceReview(&dbReview), nil
}

func (r *CarServiceReviewRepo) GetByRepairRequestID(ctx context.Context,
	repairRequestID uuid.UUID) (*domain.CarServiceReview, error) {
	dbReview, err := r.queries.GetCarServiceReviewByRepairRequestID(
		ctx,
		pgtype.UUID{Bytes: repairRequestID, Valid: true},
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, domain.ErrInternal
	}

	return toDomainCarServiceReview(&dbReview), nil
}

func (r *CarServiceReviewRepo) ListByCarServiceProfileID(ctx context.Context,
	carServiceProfileID uuid.UUID, limit, offset int) ([]*domain.CarServiceReview, error) {
	limit32, offset32, err := checkedLimitOffset(limit, offset)
	if err != nil {
		return nil, err
	}

	dbReviews, err := r.queries.ListCarServiceReviewsByProfileID(ctx, db.ListCarServiceReviewsByProfileIDParams{
		CarServiceProfileID: pgtype.UUID{Bytes: carServiceProfileID, Valid: true},
		Limit:               limit32,
		Offset:              offset32,
	})
	if err != nil {
		return nil, domain.ErrInternal
	}

	return toDomainCarServiceReviews(dbReviews), nil
}

func (r *CarServiceReviewRepo) ListByUserID(ctx context.Context,
	userID uuid.UUID, limit, offset int) ([]*domain.CarServiceReview, error) {
	limit32, offset32, err := checkedLimitOffset(limit, offset)
	if err != nil {
		return nil, err
	}

	dbReviews, err := r.queries.ListCarServiceReviewsByUserID(ctx, db.ListCarServiceReviewsByUserIDParams{
		UserID: pgtype.UUID{Bytes: userID, Valid: true},
		Limit:  limit32,
		Offset: offset32,
	})
	if err != nil {
		return nil, domain.ErrInternal
	}

	return toDomainCarServiceReviews(dbReviews), nil
}

func (r *CarServiceReviewRepo) UpdateByRepairRequestIDAndUserID(
	ctx context.Context,
	input *domain.UpdateCarServiceReviewInput,
) (*domain.CarServiceReview, error) {
	rating, err := checkedReviewRating(input.Rating)
	if err != nil {
		return nil, err
	}

	dbReview, err := r.queries.UpdateCarServiceReviewByRepairRequestIDAndUserID(
		ctx,
		db.UpdateCarServiceReviewByRepairRequestIDAndUserIDParams{
			RepairRequestID: pgtype.UUID{Bytes: input.RepairRequestID, Valid: true},
			UserID:          pgtype.UUID{Bytes: input.UserID, Valid: true},
			Rating:          rating,
			AuthorName:      input.AuthorName,
			Comment:         input.Comment,
		},
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23514" {
			return nil, domain.ErrInvalidInput
		}
		return nil, domain.ErrInternal
	}

	return toDomainCarServiceReview(&dbReview), nil
}

func checkedReviewRating(rating int) (int32, error) {
	if !domain.IsValidReviewRating(rating) {
		return 0, domain.ErrInvalidInput
	}

	rating32, err := intToInt32Checked(rating)
	if err != nil {
		return 0, domain.ErrInvalidInput
	}

	return rating32, nil
}

func (r *CarServiceReviewRepo) DeleteByRepairRequestIDAndUserID(
	ctx context.Context,
	repairRequestID, userID uuid.UUID,
) error {
	rowsAffected, err := r.queries.DeleteCarServiceReviewByRepairRequestIDAndUserID(
		ctx,
		db.DeleteCarServiceReviewByRepairRequestIDAndUserIDParams{
			RepairRequestID: pgtype.UUID{Bytes: repairRequestID, Valid: true},
			UserID:          pgtype.UUID{Bytes: userID, Valid: true},
		},
	)
	if err != nil {
		return domain.ErrInternal
	}
	if rowsAffected == 0 {
		return domain.ErrNotFound
	}

	return nil
}

func toDomainCarServiceReviews(dbReviews []db.CarServiceReview) []*domain.CarServiceReview {
	reviews := make([]*domain.CarServiceReview, 0, len(dbReviews))
	for i := range dbReviews {
		reviews = append(reviews, toDomainCarServiceReview(&dbReviews[i]))
	}

	return reviews
}

func toDomainCarServiceReview(dbReview *db.CarServiceReview) *domain.CarServiceReview {
	return &domain.CarServiceReview{
		ID:                  fromPgUUID(dbReview.ID),
		RepairRequestID:     fromPgUUID(dbReview.RepairRequestID),
		CarServiceProfileID: fromPgUUID(dbReview.CarServiceProfileID),
		UserID:              fromPgUUID(dbReview.UserID),
		Rating:              int(dbReview.Rating),
		AuthorName:          dbReview.AuthorName,
		Comment:             dbReview.Comment,
		CreatedAt:           dbReview.CreatedAt.Time,
		UpdatedAt:           dbReview.UpdatedAt.Time,
	}
}
