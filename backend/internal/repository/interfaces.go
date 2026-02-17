package repository

import (
	"context"

	"github.com/DedovInside/AutoInspect/backend/internal/domain"
	"github.com/google/uuid"
)

// UserRepository - интерфейс для работы с пользователями
type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error

	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)

	GetByEmail(ctx context.Context, email string) (*domain.User, error)

	GetByUsername(ctx context.Context, username string) (*domain.User, error)

	Update(ctx context.Context, user *domain.User) error

	Delete(ctx context.Context, id uuid.UUID) error

	List(ctx context.Context, limit, offset int) ([]*domain.User, error)

	UpdateLastLogin(ctx context.Context, id uuid.UUID) error
}

// ModelRepository - интерфейс для работы с ML моделями
type ModelRepository interface {
	Create(ctx context.Context, model *domain.MLModel) error

	GetByID(ctx context.Context, id uuid.UUID) (*domain.MLModel, error)

	GetByVersion(ctx context.Context, version string) (*domain.MLModel, error)

	GetActive(ctx context.Context) (*domain.MLModel, error)

	Update(ctx context.Context, model *domain.MLModel) error

	List(ctx context.Context, limit, offset int) ([]*domain.MLModel, error)

	SetActive(ctx context.Context, id uuid.UUID) error

	ListByCarModel(ctx context.Context, carMake, carModel string) ([]*domain.MLModel, error)
}

// AnalysisRepository - интерфейс для работы с анализами изображений
type AnalysisRepository interface {
	Create(ctx context.Context, analysis *domain.Analysis) error

	GetByID(ctx context.Context, id uuid.UUID) (*domain.Analysis, error)

	GetByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.Analysis, error)

	Update(ctx context.Context, analysis *domain.Analysis) error

	UpdateStatus(ctx context.Context, id uuid.UUID, status domain.AnalysisStatus) error

	UpdateResult(ctx context.Context, id uuid.UUID, result *domain.AnalysisResult) error

	Delete(ctx context.Context, id uuid.UUID) error

	GetPending(ctx context.Context, limit int) ([]*domain.Analysis, error)
}

// DatasetRepository - интерфейс для работы с наборами данных
type DatasetRepository interface {
	Create(ctx context.Context, dataset *domain.Dataset) error

	GetByID(ctx context.Context, id uuid.UUID) (*domain.Dataset, error)

	GetByOwnerID(ctx context.Context, ownerID uuid.UUID, limit, offset int) ([]*domain.Dataset, error)

	Update(ctx context.Context, dataset *domain.Dataset) error

	Delete(ctx context.Context, id uuid.UUID) error

	List(ctx context.Context, limit, offset int) ([]*domain.Dataset, error)

	UpdateStatus(ctx context.Context, id uuid.UUID, status domain.DatasetStatus) error
}

// TrainingJobRepository - интерфейс для работы с задачами обучения
type TrainingJobRepository interface {
	Create(ctx context.Context, job *domain.TrainingJob) error

	GetByID(ctx context.Context, id uuid.UUID) (*domain.TrainingJob, error)

	Update(ctx context.Context, job *domain.TrainingJob) error

	UpdateStatus(ctx context.Context, id uuid.UUID, status domain.JobStatus) error

	GetPending(ctx context.Context, limit int) ([]*domain.TrainingJob, error)

	GetByDatasetID(ctx context.Context, datasetID uuid.UUID) ([]*domain.TrainingJob, error)

	List(ctx context.Context, limit, offset int) ([]*domain.TrainingJob, error)
}

// AuditLogRepository - интерфейс для работы с журналом аудита
type AuditLogRepository interface {
	Create(ctx context.Context, log *domain.AuditLog) error

	GetByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.AuditLog, error)

	GetByAction(ctx context.Context, action string, limit, offset int) ([]*domain.AuditLog, error)

	List(ctx context.Context, limit, offset int) ([]*domain.AuditLog, error)
}
