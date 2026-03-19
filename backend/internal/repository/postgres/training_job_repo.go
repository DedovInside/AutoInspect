package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/DedovInside/AutoInspect/backend/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type TrainingJobRepo struct {
	db *DB
}

func NewTrainingJobRepo(db *DB) *TrainingJobRepo {
	return &TrainingJobRepo{db: db}
}

const trainingJobSelectCols = `
	id, dataset_id, base_model_id,
	status,
	params_json,
	result_model_id,
	logs_path, metrics_json,
	created_at, started_at, completed_at,
	worker_id, error_message,
	webhook_url, notified_at`

// Create создаёт новую задачу обучения.
func (r *TrainingJobRepo) Create(ctx context.Context, job *domain.TrainingJob) error {
	query := `
		INSERT INTO training_jobs (
			id, dataset_id, base_model_id,
			status,
			params_json,
			created_at,
			webhook_url
		) VALUES (
			$1, $2, $3,
			$4,
			$5,
			$6,
			$7
		)`

	if job.ID == uuid.Nil {
		job.ID = uuid.New()
	}
	job.CreatedAt = time.Now()

	_, err := r.db.pool.Exec(ctx, query,
		job.ID, job.DatasetID, job.BaseModelID,
		job.Status,
		job.Params,
		job.CreatedAt,
		job.WebhookURL,
	)
	if err != nil {
		return fmt.Errorf("TrainingJobRepo.Create: %w", err)
	}
	return nil
}

// GetByID возвращает задачу обучения по UUID.
func (r *TrainingJobRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.TrainingJob, error) {
	query := `SELECT ` + trainingJobSelectCols + ` FROM training_jobs WHERE id = $1`
	row := r.db.pool.QueryRow(ctx, query, id)
	return scanTrainingJob(row)
}

// Update обновляет поля задачи обучения (результат, метрики, ошибки).
func (r *TrainingJobRepo) Update(ctx context.Context, job *domain.TrainingJob) error {
	query := `
		UPDATE training_jobs SET
			status          = $1,
			result_model_id = $2,
			logs_path       = $3,
			metrics_json    = $4,
			started_at      = $5,
			completed_at    = $6,
			worker_id       = $7,
			error_message   = $8,
			notified_at     = $9
		WHERE id = $10`

	tag, err := r.db.pool.Exec(ctx, query,
		job.Status,
		job.ResultModelID,
		job.LogsPath,
		job.Metrics,
		job.StartedAt,
		job.CompletedAt,
		job.WorkerID,
		job.ErrorMessage,
		job.NotifiedAt,
		job.ID,
	)
	if err != nil {
		return fmt.Errorf("TrainingJobRepo.Update: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("TrainingJobRepo.Update: %w", domain.ErrNotFound)
	}
	return nil
}

// UpdateStatus обновляет только статус задачи.
func (r *TrainingJobRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.JobStatus) error {
	tag, err := r.db.pool.Exec(ctx,
		`UPDATE training_jobs SET status = $1 WHERE id = $2`, status, id,
	)
	if err != nil {
		return fmt.Errorf("TrainingJobRepo.UpdateStatus: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("TrainingJobRepo.UpdateStatus: %w", domain.ErrNotFound)
	}
	return nil
}

// GetPending возвращает задачи в статусе queued для воркера.
func (r *TrainingJobRepo) GetPending(ctx context.Context, limit int) ([]*domain.TrainingJob, error) {
	query := `
		SELECT ` + trainingJobSelectCols + `
		FROM training_jobs
		WHERE status = $1
		ORDER BY created_at ASC
		LIMIT $2`

	rows, err := r.db.pool.Query(ctx, query, domain.JobStatusQueued, limit)
	if err != nil {
		return nil, fmt.Errorf("TrainingJobRepo.GetPending: %w", err)
	}
	defer rows.Close()

	return collectTrainingJobs(rows)
}

// GetByDatasetID возвращает все задачи, связанные с датасетом.
func (r *TrainingJobRepo) GetByDatasetID(ctx context.Context, datasetID uuid.UUID) ([]*domain.TrainingJob, error) {
	query := `
		SELECT ` + trainingJobSelectCols + `
		FROM training_jobs
		WHERE dataset_id = $1
		ORDER BY created_at DESC`

	rows, err := r.db.pool.Query(ctx, query, datasetID)
	if err != nil {
		return nil, fmt.Errorf("TrainingJobRepo.GetByDatasetID: %w", err)
	}
	defer rows.Close()

	return collectTrainingJobs(rows)
}

// List возвращает все задачи обучения с пагинацией.
func (r *TrainingJobRepo) List(ctx context.Context, limit, offset int) ([]*domain.TrainingJob, error) {
	query := `
		SELECT ` + trainingJobSelectCols + `
		FROM training_jobs
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`

	rows, err := r.db.pool.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("TrainingJobRepo.List: %w", err)
	}
	defer rows.Close()

	return collectTrainingJobs(rows)
}

// --- вспомогательные функции сканирования ---

func scanTrainingJob(row pgx.Row) (*domain.TrainingJob, error) {
	j := &domain.TrainingJob{}
	err := row.Scan(
		&j.ID, &j.DatasetID, &j.BaseModelID,
		&j.Status,
		&j.Params,
		&j.ResultModelID,
		&j.LogsPath, &j.Metrics,
		&j.CreatedAt, &j.StartedAt, &j.CompletedAt,
		&j.WorkerID, &j.ErrorMessage,
		&j.WebhookURL, &j.NotifiedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("scanTrainingJob: %w", err)
	}
	return j, nil
}

func collectTrainingJobs(rows pgx.Rows) ([]*domain.TrainingJob, error) {
	var jobs []*domain.TrainingJob
	for rows.Next() {
		j := &domain.TrainingJob{}
		err := rows.Scan(
			&j.ID, &j.DatasetID, &j.BaseModelID,
			&j.Status,
			&j.Params,
			&j.ResultModelID,
			&j.LogsPath, &j.Metrics,
			&j.CreatedAt, &j.StartedAt, &j.CompletedAt,
			&j.WorkerID, &j.ErrorMessage,
			&j.WebhookURL, &j.NotifiedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("collectTrainingJobs: %w", err)
		}
		jobs = append(jobs, j)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("collectTrainingJobs rows: %w", err)
	}
	return jobs, nil
}
