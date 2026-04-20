package repository

import (
	"context"
	"io"
	"time"

	"github.com/DedovInside/AutoInspect/backend/internal/domain"
	"github.com/google/uuid"
)

type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	Update(ctx context.Context, user *domain.User) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, limit, offset int) ([]*domain.User, error)
	UpdateLastLogin(ctx context.Context, id uuid.UUID) error
}

type AuthSessionRepository interface {
	Create(ctx context.Context, session *domain.AuthSession) error
	GetByTokenHash(ctx context.Context, tokenHash string) (*domain.AuthSession, error)
	Revoke(ctx context.Context, id uuid.UUID, revokedReason string, replacedByID *uuid.UUID) error
	TouchLastUsed(ctx context.Context, id uuid.UUID, at time.Time) error
	RevokeFamily(ctx context.Context, familyID uuid.UUID, revokedReason string) error
}

type OAuthIdentityRepository interface {
	Create(ctx context.Context, identity *domain.OAuthIdentity) error
	GetByProviderSubject(ctx context.Context, provider, providerUserID string) (*domain.OAuthIdentity, error)
}

type CarModelRepository interface {
	FindActiveModel(ctx context.Context, carMake, model, generation string, year int) (*domain.CarModel, error)
	GetUniversalModel(ctx context.Context) (*domain.CarModel, error)
	CreateModel(ctx context.Context, cm *domain.CarModel) error
}

type AnalysisJobRepository interface {
	Create(ctx context.Context, job *domain.AnalysisJob) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.AnalysisJob, error)
	GetByCorrelationID(ctx context.Context, correlationID uuid.UUID) (*domain.AnalysisJob, error)
	GetByUserAndIdempotencyKey(ctx context.Context, userID uuid.UUID, idempotencyKey string) (*domain.AnalysisJob, error)
	GetByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.AnalysisJob, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status domain.JobStatus, errorMessage *string) error
	UpdateStatusByCorrelationID(ctx context.Context, correlationID uuid.UUID, status domain.JobStatus, errorMessage *string) error
	UpdateResult(ctx context.Context, id uuid.UUID, result *domain.AnalysisResult, modelVersion string) error
	UpdateResultByCorrelationID(ctx context.Context, correlationID uuid.UUID, result *domain.AnalysisResult, modelVersion string) error
	MarkStarted(ctx context.Context, id uuid.UUID) error
}

type FileRepository interface {
	// Upload загружает файл и возвращает его S3-ключ
	Upload(ctx context.Context, bucket, objectKey string, data io.Reader, contentType string, size int64) error
	// Download возвращает читатель для скачивания файла
	Download(ctx context.Context, bucket, objectKey string) (io.ReadCloser, error)
	// Exists проверяет наличие объекта
	Exists(ctx context.Context, bucket, objectKey string) (bool, error)
	// Delete удаляет объект
	Delete(ctx context.Context, bucket, objectKey string) error
	// GetPresignedURL генерирует временную ссылку для публичного доступа (для фронтенда)
	GetPresignedURL(ctx context.Context, bucket, objectKey string, expires time.Duration) (string, error)
}
