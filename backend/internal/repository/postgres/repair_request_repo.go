package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/DedovInside/AutoInspect/backend/internal/domain"
	"github.com/DedovInside/AutoInspect/backend/internal/repository/postgres/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

type RepairRequestRepo struct {
	queries *db.Queries
}

func NewRepairRequestRepo(tx DBTX) *RepairRequestRepo {
	return &RepairRequestRepo{queries: db.New(tx)}
}

func (r *RepairRequestRepo) Create(ctx context.Context, request *domain.RepairRequest) error {
	repairSummaryJSON, err := json.Marshal(request.RepairSummary)
	if err != nil {
		return domain.ErrInternal
	}

	serviceEstimateJSON, err := marshalNullableJSON(request.ServiceEstimate)
	if err != nil {
		return domain.ErrInternal
	}

	params := db.CreateRepairRequestParams{
		ID:                  pgtype.UUID{Bytes: request.ID, Valid: true},
		UserID:              pgtype.UUID{Bytes: request.UserID, Valid: true},
		AnalysisJobID:       pgtype.UUID{Bytes: request.AnalysisJobID, Valid: true},
		CarServiceProfileID: pgtype.UUID{Bytes: request.CarServiceProfileID, Valid: true},
		Status:              string(request.Status),
		RepairSummary:       repairSummaryJSON,
		ServiceEstimate:     serviceEstimateJSON,
		CustomerName:        request.CustomerName,
		CustomerPhone:       request.CustomerPhone,
		CustomerEmail:       request.CustomerEmail,
		CustomerComment:     request.CustomerComment,
		ServiceComment:      request.ServiceComment,
		EstimatedPriceMin:   floatPtrToPgNumeric(request.EstimatedPriceMin),
		EstimatedPriceMax:   floatPtrToPgNumeric(request.EstimatedPriceMax),
		CreatedAt:           pgtype.Timestamptz{Time: request.CreatedAt, Valid: true},
		UpdatedAt:           pgtype.Timestamptz{Time: request.UpdatedAt, Valid: true},
		RespondedAt:         toPgTimestamptzPtr(request.RespondedAt),
	}

	if err := r.queries.CreateRepairRequest(ctx, params); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.ErrAlreadyExists
		}
		return domain.ErrInternal
	}

	return nil
}

func (r *RepairRequestRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.RepairRequest, error) {
	dbRequest, err := r.queries.GetRepairRequestByID(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, domain.ErrInternal
	}

	return toDomainRepairRequest(&dbRequest)
}

func (r *RepairRequestRepo) GetByIDAndCarServiceProfileID(ctx context.Context,
	id, carServiceProfileID uuid.UUID) (*domain.RepairRequest, error) {
	params := db.GetRepairRequestByIDAndCarServiceProfileIDParams{
		ID:                  pgtype.UUID{Bytes: id, Valid: true},
		CarServiceProfileID: pgtype.UUID{Bytes: carServiceProfileID, Valid: true},
	}

	dbRequest, err := r.queries.GetRepairRequestByIDAndCarServiceProfileID(ctx, params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, domain.ErrInternal
	}

	return toDomainRepairRequest(&dbRequest)
}

func (r *RepairRequestRepo) GetPendingByUserAnalysisAndService(ctx context.Context,
	userID, analysisJobID, carServiceProfileID uuid.UUID) (*domain.RepairRequest, error) {
	params := db.GetPendingRepairRequestByUserAnalysisAndServiceParams{
		UserID:              pgtype.UUID{Bytes: userID, Valid: true},
		AnalysisJobID:       pgtype.UUID{Bytes: analysisJobID, Valid: true},
		CarServiceProfileID: pgtype.UUID{Bytes: carServiceProfileID, Valid: true},
	}

	dbRequest, err := r.queries.GetPendingRepairRequestByUserAnalysisAndService(ctx, params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, domain.ErrInternal
	}

	return toDomainRepairRequest(&dbRequest)
}

func (r *RepairRequestRepo) ListByUserID(ctx context.Context,
	userID uuid.UUID, limit, offset int) ([]*domain.RepairRequest, error) {
	limit32, offset32, err := checkedLimitOffset(limit, offset)
	if err != nil {
		return nil, err
	}

	dbRequests, err := r.queries.ListRepairRequestsByUserID(ctx, db.ListRepairRequestsByUserIDParams{
		UserID: pgtype.UUID{Bytes: userID, Valid: true},
		Limit:  limit32,
		Offset: offset32,
	})
	if err != nil {
		return nil, domain.ErrInternal
	}

	return toDomainRepairRequests(dbRequests)
}

func (r *RepairRequestRepo) ListByCarServiceProfileID(ctx context.Context,
	carServiceProfileID uuid.UUID, limit, offset int) ([]*domain.RepairRequest, error) {
	limit32, offset32, err := checkedLimitOffset(limit, offset)
	if err != nil {
		return nil, err
	}

	dbRequests, err := r.queries.ListRepairRequestsByCarServiceProfileID(
		ctx,
		db.ListRepairRequestsByCarServiceProfileIDParams{
			CarServiceProfileID: pgtype.UUID{Bytes: carServiceProfileID, Valid: true},
			Limit:               limit32,
			Offset:              offset32,
		},
	)
	if err != nil {
		return nil, domain.ErrInternal
	}

	return toDomainRepairRequests(dbRequests)
}

func (r *RepairRequestRepo) CancelPendingByUserID(ctx context.Context, id, userID uuid.UUID) error {
	params := db.CancelPendingRepairRequestByUserIDParams{
		ID:        pgtype.UUID{Bytes: id, Valid: true},
		UserID:    pgtype.UUID{Bytes: userID, Valid: true},
		UpdatedAt: pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
	}

	rowsAffected, err := r.queries.CancelPendingRepairRequestByUserID(ctx, params)
	if err != nil {
		return domain.ErrInternal
	}

	if rowsAffected == 0 {
		return domain.ErrNotFound
	}

	return nil
}

func (r *RepairRequestRepo) RespondByCarServiceProfileID(ctx context.Context,
	carServiceProfileID uuid.UUID, input *domain.RespondRepairRequestInput) error {
	serviceEstimateJSON, err := marshalNullableJSON(input.ServiceEstimate)
	if err != nil {
		return domain.ErrInternal
	}

	now := time.Now().UTC()
	params := db.RespondPendingRepairRequestByCarServiceProfileIDParams{
		ID:                  pgtype.UUID{Bytes: input.ID, Valid: true},
		CarServiceProfileID: pgtype.UUID{Bytes: carServiceProfileID, Valid: true},
		Status:              string(input.Status),
		ServiceComment:      input.ServiceComment,
		ServiceEstimate:     serviceEstimateJSON,
		EstimatedPriceMin:   floatPtrToPgNumeric(input.EstimatedPriceMin),
		EstimatedPriceMax:   floatPtrToPgNumeric(input.EstimatedPriceMax),
		RespondedAt:         pgtype.Timestamptz{Time: now, Valid: true},
	}

	rowsAffected, err := r.queries.RespondPendingRepairRequestByCarServiceProfileID(ctx, params)
	if err != nil {
		return domain.ErrInternal
	}

	if rowsAffected == 0 {
		return domain.ErrNotFound
	}

	return nil
}

func (r *RepairRequestRepo) CompleteAcceptedByCarServiceProfileID(
	ctx context.Context,
	id, carServiceProfileID uuid.UUID,
) error {
	params := db.CompleteAcceptedRepairRequestByCarServiceProfileIDParams{
		ID:                  pgtype.UUID{Bytes: id, Valid: true},
		CarServiceProfileID: pgtype.UUID{Bytes: carServiceProfileID, Valid: true},
		UpdatedAt:           pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true},
	}

	rowsAffected, err := r.queries.CompleteAcceptedRepairRequestByCarServiceProfileID(ctx, params)
	if err != nil {
		return domain.ErrInternal
	}
	if rowsAffected == 0 {
		return domain.ErrNotFound
	}

	return nil
}

func toDomainRepairRequests(dbRequests []db.RepairRequest) ([]*domain.RepairRequest, error) {
	requests := make([]*domain.RepairRequest, 0, len(dbRequests))
	for i := range dbRequests {
		request, err := toDomainRepairRequest(&dbRequests[i])
		if err != nil {
			return nil, err
		}

		requests = append(requests, request)
	}

	return requests, nil
}

func toDomainRepairRequest(dbRequest *db.RepairRequest) (*domain.RepairRequest, error) {
	repairSummary, err := unmarshalRepairSummary(dbRequest.RepairSummary)
	if err != nil {
		return nil, err
	}

	serviceEstimate, err := unmarshalServiceEstimate(dbRequest.ServiceEstimate)
	if err != nil {
		return nil, err
	}

	return &domain.RepairRequest{
		ID:                  fromPgUUID(dbRequest.ID),
		UserID:              fromPgUUID(dbRequest.UserID),
		AnalysisJobID:       fromPgUUID(dbRequest.AnalysisJobID),
		CarServiceProfileID: fromPgUUID(dbRequest.CarServiceProfileID),
		Status:              domain.RepairRequestStatus(dbRequest.Status),
		RepairSummary:       repairSummary,
		ServiceEstimate:     serviceEstimate,
		CustomerName:        dbRequest.CustomerName,
		CustomerPhone:       dbRequest.CustomerPhone,
		CustomerEmail:       dbRequest.CustomerEmail,
		CustomerComment:     dbRequest.CustomerComment,
		ServiceComment:      dbRequest.ServiceComment,
		EstimatedPriceMin:   pgNumericToFloatPtr(dbRequest.EstimatedPriceMin),
		EstimatedPriceMax:   pgNumericToFloatPtr(dbRequest.EstimatedPriceMax),
		CreatedAt:           dbRequest.CreatedAt.Time,
		UpdatedAt:           dbRequest.UpdatedAt.Time,
		RespondedAt:         fromPgTimestamptzPtr(dbRequest.RespondedAt),
	}, nil
}

func marshalNullableJSON[T any](value []T) ([]byte, error) {
	if len(value) == 0 {
		return nil, nil
	}

	return json.Marshal(value)
}

func unmarshalRepairSummary(data []byte) ([]domain.RepairSummaryItem, error) {
	var summary []domain.RepairSummaryItem
	if len(data) == 0 {
		return summary, nil
	}

	if err := json.Unmarshal(data, &summary); err != nil {
		return nil, domain.ErrInternal
	}

	return summary, nil
}

func unmarshalServiceEstimate(data []byte) ([]domain.RepairEstimateItem, error) {
	var estimate []domain.RepairEstimateItem
	if len(data) == 0 {
		return estimate, nil
	}

	if err := json.Unmarshal(data, &estimate); err != nil {
		return nil, domain.ErrInternal
	}

	return estimate, nil
}

func floatPtrToPgNumeric(value *float64) pgtype.Numeric {
	if value == nil {
		return pgtype.Numeric{}
	}

	var numeric pgtype.Numeric
	if err := numeric.Scan(strconv.FormatFloat(*value, 'f', -1, 64)); err != nil {
		return pgtype.Numeric{}
	}

	return numeric
}

func pgNumericToFloatPtr(value pgtype.Numeric) *float64 {
	if !value.Valid {
		return nil
	}

	floatValue, err := value.Float64Value()
	if err != nil || !floatValue.Valid {
		return nil
	}

	result := floatValue.Float64

	return &result
}
