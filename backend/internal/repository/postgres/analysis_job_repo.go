package postgres

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/DedovInside/AutoInspect/backend/internal/domain"
	"github.com/DedovInside/AutoInspect/backend/internal/repository/postgres/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

type AnalysisJobRepo struct {
	queries *db.Queries
}

func NewAnalysisJobRepo(tx DBTX) *AnalysisJobRepo {
	return &AnalysisJobRepo{queries: db.New(tx)}
}

func (r *AnalysisJobRepo) Create(ctx context.Context, job *domain.AnalysisJob) error {
	imageKeysJSON, err := json.Marshal(job.ImageKeys)
	if err != nil {
		return domain.ErrInternal
	}

	params := db.CreateAnalysisJobParams{
		ID:             pgtype.UUID{Bytes: job.ID, Valid: true},
		UserID:         pgtype.UUID{Bytes: job.UserID, Valid: true},
		IdempotencyKey: job.IdempotencyKey,
		CarMake:        stringPtrOrNil(job.CarMake),
		CarModel:       stringPtrOrNil(job.CarModel),
		CarGeneration:  stringPtrOrNil(job.CarGeneration),
		CarYear:        int32PtrOrNil(job.CarYear),
		ImageKeys:      imageKeysJSON,
		CorrelationID:  pgtype.UUID{Bytes: job.CorrelationID, Valid: true},
		Status:         string(job.Status),
		RequestedAt:    pgtype.Timestamptz{Time: job.RequestedAt, Valid: true},
	}

	err = r.queries.CreateAnalysisJob(ctx, params)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.ErrAlreadyExists
		}
		return domain.ErrInternal
	}
	return nil
}

func (r *AnalysisJobRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.AnalysisJob, error) {
	pgID := pgtype.UUID{Bytes: id, Valid: true}
	dbJob, err := r.queries.GetAnalysisJobByID(ctx, pgID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrJobNotFound
		}
		return nil, domain.ErrInternal
	}
	return toDomainAnalysisJob(&dbJob), nil
}

func (r *AnalysisJobRepo) GetByCorrelationID(ctx context.Context, correlationID uuid.UUID) (*domain.AnalysisJob, error) {
	pgCorrID := pgtype.UUID{Bytes: correlationID, Valid: true}
	dbJob, err := r.queries.GetAnalysisJobByCorrelationID(ctx, pgCorrID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrJobNotFound
		}
		return nil, domain.ErrInternal
	}
	return toDomainAnalysisJob(&dbJob), nil
}

func (r *AnalysisJobRepo) GetByUserAndIdempotencyKey(ctx context.Context, userID uuid.UUID, idempotencyKey string) (*domain.AnalysisJob, error) {
	params := db.GetAnalysisJobByUserAndIdempotencyKeyParams{
		UserID:         pgtype.UUID{Bytes: userID, Valid: true},
		IdempotencyKey: &idempotencyKey,
	}

	dbJob, err := r.queries.GetAnalysisJobByUserAndIdempotencyKey(ctx, params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrJobNotFound
		}
		return nil, domain.ErrInternal
	}

	return toDomainAnalysisJob(&dbJob), nil
}

func (r *AnalysisJobRepo) GetByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.AnalysisJob, error) {
	limit32, err := intToInt32Checked(limit)

	if err != nil {
		return nil, domain.ErrInvalidInput
	}

	offset32, err := intToInt32Checked(offset)

	if err != nil {
		return nil, domain.ErrInvalidInput
	}

	pgUserID := pgtype.UUID{Bytes: userID, Valid: true}
	params := db.ListAnalysisJobsByUserIDParams{
		UserID: pgUserID,
		Limit:  limit32,
		Offset: offset32,
	}

	dbJobs, err := r.queries.ListAnalysisJobsByUserID(ctx, params)

	if err != nil {
		return nil, domain.ErrInternal
	}

	jobs := make([]*domain.AnalysisJob, len(dbJobs))

	for i := range dbJobs {
		jobs[i] = toDomainAnalysisJob(&dbJobs[i])
	}
	return jobs, nil
}

func (r *AnalysisJobRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.JobStatus, errorMessage *string) error {
	pgID := pgtype.UUID{Bytes: id, Valid: true}
	params := db.UpdateAnalysisJobStatusParams{
		Status:       string(status),
		ErrorMessage: errorMessage,
		ID:           pgID,
	}

	rowsAffected, err := r.queries.UpdateAnalysisJobStatus(ctx, params)

	if err != nil {
		return domain.ErrInternal
	}

	if rowsAffected == 0 {
		return domain.ErrJobNotFound
	}
	return nil
}

func (r *AnalysisJobRepo) UpdateStatusByCorrelationID(ctx context.Context, correlationID uuid.UUID, status domain.JobStatus, errorMessage *string) error {
	params := db.UpdateAnalysisJobStatusByCorrelationIDParams{
		Status:        string(status),
		ErrorMessage:  errorMessage,
		CorrelationID: pgtype.UUID{Bytes: correlationID, Valid: true},
	}

	rowsAffected, err := r.queries.UpdateAnalysisJobStatusByCorrelationID(ctx, params)
	if err != nil {
		return domain.ErrInternal
	}
	if rowsAffected == 0 {
		return domain.ErrJobNotFound
	}
	return nil
}

func (r *AnalysisJobRepo) UpdateResult(ctx context.Context, id uuid.UUID, result *domain.AnalysisResult, modelVersion string) error {
	pgID := pgtype.UUID{Bytes: id, Valid: true}

	resultJSON, err := json.Marshal(result)

	if err != nil {
		return domain.ErrInternal
	}

	params := db.UpdateAnalysisJobResultParams{
		Result:           resultJSON,
		UsedModelVersion: stringPtrOrNil(modelVersion),
		ID:               pgID,
	}

	rowsAffected, err := r.queries.UpdateAnalysisJobResult(ctx, params)

	if err != nil {
		return domain.ErrInternal
	}

	if rowsAffected == 0 {
		return domain.ErrJobNotFound
	}
	return nil
}

func (r *AnalysisJobRepo) UpdateResultByCorrelationID(ctx context.Context, correlationID uuid.UUID, result *domain.AnalysisResult, modelVersion string) error {
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return domain.ErrInternal
	}

	params := db.UpdateAnalysisJobResultByCorrelationIDParams{
		Result:           resultJSON,
		UsedModelVersion: stringPtrOrNil(modelVersion),
		CorrelationID:    pgtype.UUID{Bytes: correlationID, Valid: true},
	}

	rowsAffected, err := r.queries.UpdateAnalysisJobResultByCorrelationID(ctx, params)

	if err != nil {
		return domain.ErrInternal
	}

	if rowsAffected == 0 {
		return domain.ErrJobNotFound
	}
	return nil
}

func toDomainAnalysisJob(dbJob *db.AnalysisJob) *domain.AnalysisJob {
	job := &domain.AnalysisJob{
		ID:               fromPgUUID(dbJob.ID),
		UserID:           fromPgUUID(dbJob.UserID),
		IdempotencyKey:   dbJob.IdempotencyKey,
		CarMake:          stringValue(dbJob.CarMake),
		CarModel:         stringValue(dbJob.CarModel),
		CarGeneration:    stringValue(dbJob.CarGeneration),
		CarYear:          int32Value(dbJob.CarYear),
		CorrelationID:    fromPgUUID(dbJob.CorrelationID),
		Status:           domain.JobStatus(dbJob.Status),
		ErrorMessage:     dbJob.ErrorMessage,
		UsedModelVersion: stringValue(dbJob.UsedModelVersion),
		RequestedAt:      dbJob.RequestedAt.Time,
		StartedAt:        fromPgTimestamptzPtr(dbJob.StartedAt),
		CompletedAt:      fromPgTimestamptzPtr(dbJob.CompletedAt),
	}

	if len(dbJob.ImageKeys) > 0 {
		var keys []string
		if err := json.Unmarshal(dbJob.ImageKeys, &keys); err == nil {
			job.ImageKeys = keys
		}
	}

	if len(dbJob.Result) > 0 {
		var res domain.AnalysisResult
		if err := json.Unmarshal(dbJob.Result, &res); err == nil {
			job.Result = &res
		}
	}

	return job
}
