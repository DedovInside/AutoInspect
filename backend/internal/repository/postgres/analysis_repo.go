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

type AnalysisRepo struct {
	db *DB
}

func NewAnalysisRepo(db *DB) *AnalysisRepo {
	return &AnalysisRepo{db: db}
}

const analysisSelectCols = `
	id, user_id, status,
	image_key, image_metadata,
	model_version, model_id,
	result_json,
	error_message, error_code, retry_count,
	created_at, updated_at, processed_at`

// Create создаёт новую запись анализа.
func (r *AnalysisRepo) Create(ctx context.Context, a *domain.Analysis) error {
	query := `
		INSERT INTO analyses (
			id, user_id, status,
			image_key, image_metadata,
			model_version, model_id,
			retry_count, created_at
		) VALUES (
			$1, $2, $3,
			$4, $5,
			$6, $7,
			$8, $9
		)`

	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	a.CreatedAt = time.Now()

	_, err := r.db.pool.Exec(ctx, query,
		a.ID, a.UserID, a.Status,
		a.ImageKey, a.ImageMetadata,
		a.ModelVersion, a.ModelID,
		a.RetryCount, a.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("AnalysisRepo.Create: %w", err)
	}
	return nil
}

// GetByID возвращает анализ по UUID.
func (r *AnalysisRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Analysis, error) {
	query := `SELECT ` + analysisSelectCols + ` FROM analyses WHERE id = $1`
	row := r.db.pool.QueryRow(ctx, query, id)
	return scanAnalysis(row)
}

// GetByUserID возвращает список анализов конкретного пользователя с пагинацией.
func (r *AnalysisRepo) GetByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.Analysis, error) {
	query := `
		SELECT ` + analysisSelectCols + `
		FROM analyses
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.db.pool.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("AnalysisRepo.GetByUserID: %w", err)
	}
	defer rows.Close()

	return collectAnalyses(rows)
}

// Update обновляет все поля анализа.
func (r *AnalysisRepo) Update(ctx context.Context, a *domain.Analysis) error {
	query := `
		UPDATE analyses SET
			status        = $1,
			image_key     = $2,
			image_metadata = $3,
			model_version = $4,
			model_id      = $5,
			result_json   = $6,
			error_message = $7,
			error_code    = $8,
			retry_count   = $9,
			processed_at  = $10
		WHERE id = $11`

	tag, err := r.db.pool.Exec(ctx, query,
		a.Status, a.ImageKey, a.ImageMetadata,
		a.ModelVersion, a.ModelID,
		a.Result,
		a.ErrorMessage, a.ErrorCode, a.RetryCount,
		a.ProcessedAt,
		a.ID,
	)
	if err != nil {
		return fmt.Errorf("AnalysisRepo.Update: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("AnalysisRepo.Update: %w", domain.ErrNotFound)
	}
	return nil
}

// UpdateStatus обновляет только статус анализа.
func (r *AnalysisRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.AnalysisStatus) error {
	tag, err := r.db.pool.Exec(ctx,
		`UPDATE analyses SET status = $1 WHERE id = $2`, status, id,
	)
	if err != nil {
		return fmt.Errorf("AnalysisRepo.UpdateStatus: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("AnalysisRepo.UpdateStatus: %w", domain.ErrNotFound)
	}
	return nil
}

// UpdateResult сохраняет результат анализа и переводит статус в completed.
func (r *AnalysisRepo) UpdateResult(ctx context.Context, id uuid.UUID, result *domain.AnalysisResult) error {
	now := time.Now()
	tag, err := r.db.pool.Exec(ctx,
		`UPDATE analyses
		 SET result_json = $1, status = $2, processed_at = $3
		 WHERE id = $4`,
		result, domain.AnalysisStatusCompleted, now, id,
	)
	if err != nil {
		return fmt.Errorf("AnalysisRepo.UpdateResult: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("AnalysisRepo.UpdateResult: %w", domain.ErrNotFound)
	}
	return nil
}

// Delete удаляет анализ по UUID.
func (r *AnalysisRepo) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.db.pool.Exec(ctx, `DELETE FROM analyses WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("AnalysisRepo.Delete: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("AnalysisRepo.Delete: %w", domain.ErrNotFound)
	}
	return nil
}

// GetPending возвращает анализы в статусе queued для обработки воркером.
func (r *AnalysisRepo) GetPending(ctx context.Context, limit int) ([]*domain.Analysis, error) {
	query := `
		SELECT ` + analysisSelectCols + `
		FROM analyses
		WHERE status = $1
		ORDER BY created_at ASC
		LIMIT $2`

	rows, err := r.db.pool.Query(ctx, query, domain.AnalysisStatusQueued, limit)
	if err != nil {
		return nil, fmt.Errorf("AnalysisRepo.GetPending: %w", err)
	}
	defer rows.Close()

	return collectAnalyses(rows)
}

// --- вспомогательные функции сканирования ---

func scanAnalysis(row pgx.Row) (*domain.Analysis, error) {
	a := &domain.Analysis{}
	err := row.Scan(
		&a.ID, &a.UserID, &a.Status,
		&a.ImageKey, &a.ImageMetadata,
		&a.ModelVersion, &a.ModelID,
		&a.Result,
		&a.ErrorMessage, &a.ErrorCode, &a.RetryCount,
		&a.CreatedAt, &a.UpdatedAt, &a.ProcessedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("scanAnalysis: %w", err)
	}
	return a, nil
}

func collectAnalyses(rows pgx.Rows) ([]*domain.Analysis, error) {
	var analyses []*domain.Analysis
	for rows.Next() {
		a := &domain.Analysis{}
		err := rows.Scan(
			&a.ID, &a.UserID, &a.Status,
			&a.ImageKey, &a.ImageMetadata,
			&a.ModelVersion, &a.ModelID,
			&a.Result,
			&a.ErrorMessage, &a.ErrorCode, &a.RetryCount,
			&a.CreatedAt, &a.UpdatedAt, &a.ProcessedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("collectAnalyses: %w", err)
		}
		analyses = append(analyses, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("collectAnalyses rows: %w", err)
	}
	return analyses, nil
}
