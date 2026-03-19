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

type DatasetRepo struct {
	db *DB
}

func NewDatasetRepo(db *DB) *DatasetRepo {
	return &DatasetRepo{db: db}
}

const datasetSelectCols = `
	id, owner_id,
	name, description,
	dataset_type, status,
	file_key, total_size_bytes,
	annotation_format, images_count, annotations_count, classes_json,
	car_make, car_model,
	created_at, updated_at, validated_at`

// Create создаёт новый датасет.
func (r *DatasetRepo) Create(ctx context.Context, d *domain.Dataset) error {
	query := `
		INSERT INTO datasets (
			id, owner_id,
			name, description,
			dataset_type, status,
			file_key, total_size_bytes,
			annotation_format, images_count, annotations_count, classes_json,
			car_make, car_model,
			created_at
		) VALUES (
			$1, $2,
			$3, $4,
			$5, $6,
			$7, $8,
			$9, $10, $11, $12,
			$13, $14,
			$15
		)`

	if d.ID == uuid.Nil {
		d.ID = uuid.New()
	}
	d.CreatedAt = time.Now()

	_, err := r.db.pool.Exec(ctx, query,
		d.ID, d.OwnerID,
		d.Name, d.Description,
		d.DatasetType, d.Status,
		d.FileKey, d.TotalSizeBytes,
		d.AnnotationFormat, d.ImagesCount, d.AnnotationsCount, d.Classes,
		d.CarMake, d.CarModel,
		d.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("DatasetRepo.Create: %w", err)
	}
	return nil
}

// GetByID возвращает датасет по UUID.
func (r *DatasetRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Dataset, error) {
	query := `SELECT ` + datasetSelectCols + ` FROM datasets WHERE id = $1`
	row := r.db.pool.QueryRow(ctx, query, id)
	return scanDataset(row)
}

// GetByOwnerID возвращает датасеты пользователя с пагинацией.
func (r *DatasetRepo) GetByOwnerID(ctx context.Context, ownerID uuid.UUID, limit, offset int) ([]*domain.Dataset, error) {
	query := `
		SELECT ` + datasetSelectCols + `
		FROM datasets
		WHERE owner_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.db.pool.Query(ctx, query, ownerID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("DatasetRepo.GetByOwnerID: %w", err)
	}
	defer rows.Close()

	return collectDatasets(rows)
}

// Update обновляет данные датасета.
func (r *DatasetRepo) Update(ctx context.Context, d *domain.Dataset) error {
	query := `
		UPDATE datasets SET
			name              = $1,
			description       = $2,
			dataset_type      = $3,
			status            = $4,
			file_key          = $5,
			total_size_bytes  = $6,
			annotation_format = $7,
			images_count      = $8,
			annotations_count = $9,
			classes_json      = $10,
			car_make          = $11,
			car_model         = $12,
			validated_at      = $13
		WHERE id = $14`

	tag, err := r.db.pool.Exec(ctx, query,
		d.Name, d.Description,
		d.DatasetType, d.Status,
		d.FileKey, d.TotalSizeBytes,
		d.AnnotationFormat, d.ImagesCount, d.AnnotationsCount, d.Classes,
		d.CarMake, d.CarModel,
		d.ValidatedAt,
		d.ID,
	)
	if err != nil {
		return fmt.Errorf("DatasetRepo.Update: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("DatasetRepo.Update: %w", domain.ErrNotFound)
	}
	return nil
}

// Delete удаляет датасет по UUID.
func (r *DatasetRepo) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.db.pool.Exec(ctx, `DELETE FROM datasets WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("DatasetRepo.Delete: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("DatasetRepo.Delete: %w", domain.ErrNotFound)
	}
	return nil
}

// List возвращает все датасеты с пагинацией.
func (r *DatasetRepo) List(ctx context.Context, limit, offset int) ([]*domain.Dataset, error) {
	query := `
		SELECT ` + datasetSelectCols + `
		FROM datasets
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`

	rows, err := r.db.pool.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("DatasetRepo.List: %w", err)
	}
	defer rows.Close()

	return collectDatasets(rows)
}

// UpdateStatus обновляет только статус датасета.
func (r *DatasetRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.DatasetStatus) error {
	tag, err := r.db.pool.Exec(ctx,
		`UPDATE datasets SET status = $1 WHERE id = $2`, status, id,
	)
	if err != nil {
		return fmt.Errorf("DatasetRepo.UpdateStatus: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("DatasetRepo.UpdateStatus: %w", domain.ErrNotFound)
	}
	return nil
}

// --- вспомогательные функции сканирования ---

func scanDataset(row pgx.Row) (*domain.Dataset, error) {
	d := &domain.Dataset{}
	err := row.Scan(
		&d.ID, &d.OwnerID,
		&d.Name, &d.Description,
		&d.DatasetType, &d.Status,
		&d.FileKey, &d.TotalSizeBytes,
		&d.AnnotationFormat, &d.ImagesCount, &d.AnnotationsCount, &d.Classes,
		&d.CarMake, &d.CarModel,
		&d.CreatedAt, &d.UpdatedAt, &d.ValidatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("scanDataset: %w", err)
	}
	return d, nil
}

func collectDatasets(rows pgx.Rows) ([]*domain.Dataset, error) {
	var datasets []*domain.Dataset
	for rows.Next() {
		d := &domain.Dataset{}
		err := rows.Scan(
			&d.ID, &d.OwnerID,
			&d.Name, &d.Description,
			&d.DatasetType, &d.Status,
			&d.FileKey, &d.TotalSizeBytes,
			&d.AnnotationFormat, &d.ImagesCount, &d.AnnotationsCount, &d.Classes,
			&d.CarMake, &d.CarModel,
			&d.CreatedAt, &d.UpdatedAt, &d.ValidatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("collectDatasets: %w", err)
		}
		datasets = append(datasets, d)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("collectDatasets rows: %w", err)
	}
	return datasets, nil
}
