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

type ModelRepo struct {
	db *DB
}

func NewModelRepo(db *DB) *ModelRepo {
	return &ModelRepo{db: db}
}

// Create создаёт новую ML-модель в базе данных.
func (r *ModelRepo) Create(ctx context.Context, model *domain.MLModel) error {
	query := `
		INSERT INTO models (
			id, version, name,
			weights_path, config_path,
			car_make, car_model,
			status, active,
			metrics_json,
			parent_model_id, trained_at, description,
			created_at, created_by
		) VALUES (
			$1, $2, $3,
			$4, $5,
			$6, $7,
			$8, $9,
			$10,
			$11, $12, $13,
			$14, $15
		)`

	if model.ID == uuid.Nil {
		model.ID = uuid.New()
	}
	model.CreatedAt = time.Now()

	_, err := r.db.pool.Exec(ctx, query,
		model.ID, model.Version, model.Name,
		model.WeightsPath, model.ConfigPath,
		model.CarMake, model.CarModel,
		model.Status, model.Active,
		model.Metrics,
		model.ParentModelID, model.TrainedAt, model.Description,
		model.CreatedAt, model.CreatedBy,
	)
	if err != nil {
		return fmt.Errorf("ModelRepo.Create: %w", err)
	}
	return nil
}

// GetByID возвращает модель по UUID.
func (r *ModelRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.MLModel, error) {
	query := `
		SELECT id, version, name,
		       weights_path, config_path,
		       car_make, car_model,
		       status, active,
		       metrics_json,
		       parent_model_id, trained_at, description,
		       created_at, created_by
		FROM models
		WHERE id = $1`

	row := r.db.pool.QueryRow(ctx, query, id)
	return scanModel(row)
}

// GetByVersion возвращает модель по версии.
func (r *ModelRepo) GetByVersion(ctx context.Context, version string) (*domain.MLModel, error) {
	query := `
		SELECT id, version, name,
		       weights_path, config_path,
		       car_make, car_model,
		       status, active,
		       metrics_json,
		       parent_model_id, trained_at, description,
		       created_at, created_by
		FROM models
		WHERE version = $1`

	row := r.db.pool.QueryRow(ctx, query, version)
	return scanModel(row)
}

// GetActive возвращает текущую активную модель.
func (r *ModelRepo) GetActive(ctx context.Context) (*domain.MLModel, error) {
	query := `
		SELECT id, version, name,
		       weights_path, config_path,
		       car_make, car_model,
		       status, active,
		       metrics_json,
		       parent_model_id, trained_at, description,
		       created_at, created_by
		FROM models
		WHERE active = TRUE
		LIMIT 1`

	row := r.db.pool.QueryRow(ctx, query)
	return scanModel(row)
}

// Update обновляет данные модели.
func (r *ModelRepo) Update(ctx context.Context, model *domain.MLModel) error {
	query := `
		UPDATE models SET
			version         = $1,
			name            = $2,
			weights_path    = $3,
			config_path     = $4,
			car_make        = $5,
			car_model       = $6,
			status          = $7,
			active          = $8,
			metrics_json    = $9,
			trained_at      = $10,
			description     = $11
		WHERE id = $12`

	tag, err := r.db.pool.Exec(ctx, query,
		model.Version, model.Name,
		model.WeightsPath, model.ConfigPath,
		model.CarMake, model.CarModel,
		model.Status, model.Active,
		model.Metrics,
		model.TrainedAt, model.Description,
		model.ID,
	)
	if err != nil {
		return fmt.Errorf("ModelRepo.Update: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("ModelRepo.Update: %w", domain.ErrNotFound)
	}
	return nil
}

// List возвращает список моделей с пагинацией.
func (r *ModelRepo) List(ctx context.Context, limit, offset int) ([]*domain.MLModel, error) {
	query := `
		SELECT id, version, name,
		       weights_path, config_path,
		       car_make, car_model,
		       status, active,
		       metrics_json,
		       parent_model_id, trained_at, description,
		       created_at, created_by
		FROM models
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`

	rows, err := r.db.pool.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("ModelRepo.List: %w", err)
	}
	defer rows.Close()

	return collectModels(rows)
}

// SetActive деактивирует все модели и активирует выбранную (транзакционно).
func (r *ModelRepo) SetActive(ctx context.Context, id uuid.UUID) error {
	tx, err := r.db.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("ModelRepo.SetActive begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	_, err = tx.Exec(ctx, `UPDATE models SET active = FALSE WHERE active = TRUE`)
	if err != nil {
		return fmt.Errorf("ModelRepo.SetActive deactivate all: %w", err)
	}

	tag, err := tx.Exec(ctx, `UPDATE models SET active = TRUE WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("ModelRepo.SetActive activate: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("ModelRepo.SetActive: %w", domain.ErrNotFound)
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("ModelRepo.SetActive commit: %w", err)
	}
	return nil
}

// ListByCarModel возвращает модели, привязанные к конкретной марке и модели авто.
func (r *ModelRepo) ListByCarModel(ctx context.Context, carMake, carModel string) ([]*domain.MLModel, error) {
	query := `
		SELECT id, version, name,
		       weights_path, config_path,
		       car_make, car_model,
		       status, active,
		       metrics_json,
		       parent_model_id, trained_at, description,
		       created_at, created_by
		FROM models
		WHERE car_make = $1 AND car_model = $2
		ORDER BY created_at DESC`

	rows, err := r.db.pool.Query(ctx, query, carMake, carModel)
	if err != nil {
		return nil, fmt.Errorf("ModelRepo.ListByCarModel: %w", err)
	}
	defer rows.Close()

	return collectModels(rows)
}

// --- вспомогательные функции сканирования ---

func scanModel(row pgx.Row) (*domain.MLModel, error) {
	m := &domain.MLModel{}
	err := row.Scan(
		&m.ID, &m.Version, &m.Name,
		&m.WeightsPath, &m.ConfigPath,
		&m.CarMake, &m.CarModel,
		&m.Status, &m.Active,
		&m.Metrics,
		&m.ParentModelID, &m.TrainedAt, &m.Description,
		&m.CreatedAt, &m.CreatedBy,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("scanModel: %w", err)
	}
	return m, nil
}

func collectModels(rows pgx.Rows) ([]*domain.MLModel, error) {
	var models []*domain.MLModel
	for rows.Next() {
		m := &domain.MLModel{}
		err := rows.Scan(
			&m.ID, &m.Version, &m.Name,
			&m.WeightsPath, &m.ConfigPath,
			&m.CarMake, &m.CarModel,
			&m.Status, &m.Active,
			&m.Metrics,
			&m.ParentModelID, &m.TrainedAt, &m.Description,
			&m.CreatedAt, &m.CreatedBy,
		)
		if err != nil {
			return nil, fmt.Errorf("collectModels: %w", err)
		}
		models = append(models, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("collectModels rows: %w", err)
	}
	return models, nil
}
