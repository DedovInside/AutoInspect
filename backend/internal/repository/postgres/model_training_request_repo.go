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

type ModelTrainingRequestRepo struct {
	queries *db.Queries
}

func NewModelTrainingRequestRepo(tx DBTX) *ModelTrainingRequestRepo {
	return &ModelTrainingRequestRepo{queries: db.New(tx)}
}

func (r *ModelTrainingRequestRepo) Create(ctx context.Context, request *domain.ModelTrainingRequest) error {
	yearFrom32, err := intToInt32Checked(request.YearFrom)
	if err != nil {
		return domain.ErrInvalidInput
	}

	params := db.CreateModelTrainingRequestParams{
		ID:              pgtype.UUID{Bytes: request.ID, Valid: true},
		InitiatorUserID: pgtype.UUID{Bytes: request.InitiatorUserID, Valid: true},
		InitiatorRole:   string(request.InitiatorRole),
		Make:            request.Make,
		Model:           request.Model,
		Generation:      stringPtrOrNil(request.Generation),
		YearFrom:        yearFrom32,
		YearTo:          int32PtrOrNil(request.YearTo),
		Description:     request.Description,
		Status:          string(request.Status),
		AdminComment:    stringPtrOrNil(request.AdminComment),
		ReviewedBy:      toPgUUIDPtr(request.ReviewedBy),
		ReviewedAt:      toPgTimestamptzPtr(request.ReviewedAt),
		CreatedModelID:  toPgUUIDPtr(request.CreatedModelID),
		IdempotencyKey:  request.IdempotencyKey,
		CreatedAt:       pgtype.Timestamptz{Time: request.CreatedAt, Valid: true},
		UpdatedAt:       pgtype.Timestamptz{Time: request.UpdatedAt, Valid: true},
	}

	if err := r.queries.CreateModelTrainingRequest(ctx, params); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.ErrAlreadyExists
		}
		return domain.ErrInternal
	}
	return nil
}

func (r *ModelTrainingRequestRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.ModelTrainingRequest, error) {
	dbRequest, err := r.queries.GetModelTrainingRequestByID(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, domain.ErrInternal
	}
	return toDomainModelTrainingRequest(&dbRequest), nil
}

func (r *ModelTrainingRequestRepo) GetByUserAndIdempotencyKey(
	ctx context.Context,
	userID uuid.UUID,
	idempotencyKey string,
) (*domain.ModelTrainingRequest, error) {
	params := db.GetModelTrainingRequestByUserAndIdempotencyKeyParams{
		InitiatorUserID: pgtype.UUID{Bytes: userID, Valid: true},
		IdempotencyKey:  &idempotencyKey,
	}

	dbRequest, err := r.queries.GetModelTrainingRequestByUserAndIdempotencyKey(ctx, params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, domain.ErrInternal
	}
	return toDomainModelTrainingRequest(&dbRequest), nil
}

func (r *ModelTrainingRequestRepo) ListByUserID(
	ctx context.Context,
	userID uuid.UUID,
	limit, offset int,
) ([]*domain.ModelTrainingRequest, error) {
	limit32, offset32, err := checkedLimitOffset(limit, offset)
	if err != nil {
		return nil, err
	}

	dbRequests, err := r.queries.ListModelTrainingRequestsByUserID(ctx, db.ListModelTrainingRequestsByUserIDParams{
		InitiatorUserID: pgtype.UUID{Bytes: userID, Valid: true},
		Limit:           limit32,
		Offset:          offset32,
	})
	if err != nil {
		return nil, domain.ErrInternal
	}

	return toDomainModelTrainingRequests(dbRequests), nil
}

func (r *ModelTrainingRequestRepo) ListForAdmin(
	ctx context.Context,
	status *domain.ModelTrainingRequestStatus,
	limit, offset int,
) ([]*domain.ModelTrainingRequest, error) {
	limit32, offset32, err := checkedLimitOffset(limit, offset)
	if err != nil {
		return nil, err
	}

	var statusPtr *string
	if status != nil {
		s := string(*status)
		statusPtr = &s
	}

	dbRequests, err := r.queries.ListModelTrainingRequestsForAdmin(ctx, db.ListModelTrainingRequestsForAdminParams{
		Limit:  limit32,
		Offset: offset32,
		Status: statusPtr,
	})
	if err != nil {
		return nil, domain.ErrInternal
	}

	return toDomainModelTrainingRequests(dbRequests), nil
}

func (r *ModelTrainingRequestRepo) CountActiveByUserID(ctx context.Context, userID uuid.UUID) (int, error) {
	count, err := r.queries.CountActiveModelTrainingRequestsByUserID(ctx, pgtype.UUID{Bytes: userID, Valid: true})
	if err != nil {
		return 0, domain.ErrInternal
	}
	return int(count), nil
}

func (r *ModelTrainingRequestRepo) UpdateStatus(
	ctx context.Context,
	input domain.UpdateModelTrainingRequestStatusInput,
) error {
	reviewedAt := pgtype.Timestamptz{Time: time.Now().UTC(), Valid: input.ReviewedBy != uuid.Nil}

	params := db.UpdateModelTrainingRequestStatusParams{
		ID:             pgtype.UUID{Bytes: input.ID, Valid: true},
		Status:         string(input.Status),
		AdminComment:   stringPtrOrNil(input.AdminComment),
		ReviewedBy:     pgtype.UUID{Bytes: input.ReviewedBy, Valid: input.ReviewedBy != uuid.Nil},
		ReviewedAt:     reviewedAt,
		CreatedModelID: toPgUUIDPtr(input.CreatedModelID),
		UpdatedAt:      pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
	}

	rowsAffected, err := r.queries.UpdateModelTrainingRequestStatus(ctx, params)
	if err != nil {
		return domain.ErrInternal
	}
	if rowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func checkedLimitOffset(limit, offset int) (limit32, offset32 int32, err error) {
	limit32, err = intToInt32Checked(limit)
	if err != nil {
		return 0, 0, domain.ErrInvalidInput
	}
	offset32, err = intToInt32Checked(offset)
	if err != nil {
		return 0, 0, domain.ErrInvalidInput
	}
	return limit32, offset32, nil
}

func toDomainModelTrainingRequests(dbRequests []db.ModelTrainingRequest) []*domain.ModelTrainingRequest {
	requests := make([]*domain.ModelTrainingRequest, 0, len(dbRequests))
	for i := range dbRequests {
		requests = append(requests, toDomainModelTrainingRequest(&dbRequests[i]))
	}
	return requests
}

func toDomainModelTrainingRequest(dbRequest *db.ModelTrainingRequest) *domain.ModelTrainingRequest {
	return &domain.ModelTrainingRequest{
		ID:              fromPgUUID(dbRequest.ID),
		InitiatorUserID: fromPgUUID(dbRequest.InitiatorUserID),
		InitiatorRole:   domain.Role(dbRequest.InitiatorRole),
		Make:            dbRequest.Make,
		Model:           dbRequest.Model,
		Generation:      stringValue(dbRequest.Generation),
		YearFrom:        int(dbRequest.YearFrom),
		YearTo:          int32Value(dbRequest.YearTo),
		Description:     dbRequest.Description,
		Status:          domain.ModelTrainingRequestStatus(dbRequest.Status),
		AdminComment:    stringValue(dbRequest.AdminComment),
		ReviewedBy:      fromPgUUIDPtr(dbRequest.ReviewedBy),
		ReviewedAt:      fromPgTimestamptzPtr(dbRequest.ReviewedAt),
		CreatedModelID:  fromPgUUIDPtr(dbRequest.CreatedModelID),
		IdempotencyKey:  dbRequest.IdempotencyKey,
		CreatedAt:       dbRequest.CreatedAt.Time,
		UpdatedAt:       dbRequest.UpdatedAt.Time,
	}
}
