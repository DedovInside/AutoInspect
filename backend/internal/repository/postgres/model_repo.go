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

const modelSelectCols = `
	id, version, name,
	car_make, car_model, car_generation, year_from, year_to,
	weights_path, config_path,
	status, active,
	created_at, updated_at`

// Create создаёт новую inference-модель в базе данных.
func (r *ModelRepo) Create(ctx context.Context, model *domain.MLModel) error {
	query := `
		INSERT INTO models (
			id, version, name,
			car_make, car_model, car_generation, year_from, year_to,
			weights_path, config_path,
			status, active,
			created_at
		) VALUES (
			$1, $2, $3,
			$4, $5, $6, $7, $8,
			$9, $10,
			$11, $12,
			$13
		)`

	if model.ID == uuid.Nil {
		model.ID = uuid.New()
	}
	model.CreatedAt = time.Now()

	_, err := r.db.pool.Exec(ctx, query,
		model.ID, model.Version, model.Name,
		model.CarMake, model.CarModel, model.CarGeneration, model.YearFrom, model.YearTo,
		model.WeightsPath, model.ConfigPath,
		model.Status, model.Active,
		model.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("ModelRepo.Create: %w", err)
	}
	return nil
}

// GetByID возвращает модель по UUID.
func (r *ModelRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.MLModel, error) {
	query := `SELECT ` + modelSelectCols + ` FROM models WHERE id = $1`
	row := r.db.pool.QueryRow(ctx, query, id)
	return scanModel(row)
}

// GetByVersion возвращает модель по версии.
func (r *ModelRepo) GetByVersion(ctx context.Context, version string) (*domain.MLModel, error) {
	query := `SELECT ` + modelSelectCols + ` FROM models WHERE version = $1`
	row := r.db.pool.QueryRow(ctx, query, version)
	return scanModel(row)
}

// ResolveForCarSpec выбирает наиболее подходящую модель для указанной спецификации автомобиля.
func (r *ModelRepo) ResolveForCarSpec(ctx context.Context, carMake, carModel string, carGeneration *string, carYear *int) (*domain.MLModel, error) {
	query := `
		SELECT ` + modelSelectCols + `
		FROM models
		WHERE car_make = $1
		  AND car_model = $2
		  AND status IN ('ready', 'active')
		  AND ($3::text IS NULL OR car_generation IS NULL OR car_generation = $3)
		  AND (
			$4::int IS NULL
			OR (
				(year_from IS NULL OR year_from <= $4)
				AND (year_to IS NULL OR year_to >= $4)
			)
		  )
		ORDER BY
		  active DESC,
		  CASE
			WHEN $3::text IS NOT NULL AND car_generation = $3 THEN 0
			WHEN car_generation IS NULL THEN 1
			ELSE 2
		  END,
		  CASE
			WHEN $4::int IS NOT NULL
				 AND year_from IS NOT NULL
				 AND year_to IS NOT NULL
				 AND $4 BETWEEN year_from AND year_to
			THEN 0
			ELSE 1
		  END,
		  created_at DESC
		LIMIT 1`
	row := r.db.pool.QueryRow(ctx, query, carMake, carModel, carGeneration, carYear)
	return scanModel(row)
}

// Update обновляет данные модели.
func (r *ModelRepo) Update(ctx context.Context, model *domain.MLModel) error {
	query := `
		UPDATE models SET
			version        = $1,
			name           = $2,
			car_make       = $3,
			car_model      = $4,
			car_generation = $5,
			year_from      = $6,
			year_to        = $7,
			weights_path   = $8,
			config_path    = $9,
			status         = $10,
			active         = $11
		WHERE id = $12`

	tag, err := r.db.pool.Exec(ctx, query,
		model.Version, model.Name,
		model.CarMake, model.CarModel, model.CarGeneration,
		model.YearFrom, model.YearTo,
		model.WeightsPath, model.ConfigPath,
		model.Status, model.Active,
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
		SELECT ` + modelSelectCols + `
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

// SetActiveForCarSpec активирует модель и деактивирует другие только в той же car-spec группе.
func (r *ModelRepo) SetActiveForCarSpec(ctx context.Context, id uuid.UUID) error {
	tx, err := r.db.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("ModelRepo.SetActiveForCarSpec begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // rollback error is not critical, already logged

	var carMake, carModel string
	var carGeneration *string
	var yearFrom, yearTo *int

	err = tx.QueryRow(ctx,
		`SELECT car_make, car_model, car_generation, year_from, year_to
		 FROM models
		 WHERE id = $1`,
		id,
	).Scan(&carMake, &carModel, &carGeneration, &yearFrom, &yearTo)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("ModelRepo.SetActiveForCarSpec: %w", domain.ErrNotFound)
		}
		return fmt.Errorf("ModelRepo.SetActiveForCarSpec load target spec: %w", err)
	}

	_, err = tx.Exec(ctx,
		`UPDATE models
		 SET active = FALSE
		 WHERE active = TRUE
		   AND car_make = $1
		   AND car_model = $2
		   AND car_generation IS NOT DISTINCT FROM $3
		   AND year_from IS NOT DISTINCT FROM $4
		   AND year_to IS NOT DISTINCT FROM $5`,
		carMake, carModel, carGeneration, yearFrom, yearTo,
	)
	if err != nil {
		return fmt.Errorf("ModelRepo.SetActiveForCarSpec deactivate group: %w", err)
	}

	tag, err := tx.Exec(ctx, `UPDATE models SET active = TRUE WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("ModelRepo.SetActiveForCarSpec activate: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("ModelRepo.SetActiveForCarSpec: %w", domain.ErrNotFound)
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("ModelRepo.SetActiveForCarSpec commit: %w", err)
	}
	return nil
}

// ListByCarModel возвращает модели, привязанные к конкретной марке и модели авто.
func (r *ModelRepo) ListByCarModel(ctx context.Context, carMake, carModel string) ([]*domain.MLModel, error) {
	query := `
		SELECT ` + modelSelectCols + `
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
		&m.CarMake, &m.CarModel, &m.CarGeneration, &m.YearFrom, &m.YearTo,
		&m.WeightsPath, &m.ConfigPath,
		&m.Status, &m.Active,
		&m.CreatedAt, &m.UpdatedAt,
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
			&m.CarMake, &m.CarModel, &m.CarGeneration, &m.YearFrom, &m.YearTo,
			&m.WeightsPath, &m.ConfigPath,
			&m.Status, &m.Active,
			&m.CreatedAt, &m.UpdatedAt,
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
