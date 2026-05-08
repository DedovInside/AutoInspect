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
	UpdateRole(ctx context.Context, id uuid.UUID, role domain.Role) error
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
	ListModels(ctx context.Context, limit, offset int) ([]*domain.CarModel, error)
	GetModelByID(ctx context.Context, id uuid.UUID) (*domain.CarModel, error)
	DeactivateModel(ctx context.Context, id uuid.UUID) error
}

type ModelTrainingRequestRepository interface {
	Create(ctx context.Context, request *domain.ModelTrainingRequest) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.ModelTrainingRequest, error)
	GetByUserAndIdempotencyKey(ctx context.Context, userID uuid.UUID, idempotencyKey string) (*domain.ModelTrainingRequest, error)
	ListByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.ModelTrainingRequest, error)
	ListForAdmin(ctx context.Context, status *domain.ModelTrainingRequestStatus, limit, offset int) ([]*domain.ModelTrainingRequest, error)
	CountActiveByUserID(ctx context.Context, userID uuid.UUID) (int, error)
	UpdateStatus(ctx context.Context, input domain.UpdateModelTrainingRequestStatusInput) error
}

type CarServiceApplicationRepository interface {
	Create(ctx context.Context, application *domain.CarServiceApplication) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.CarServiceApplication, error)
	GetPendingByUserID(ctx context.Context, userID uuid.UUID) (*domain.CarServiceApplication, error)
	ListByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.CarServiceApplication, error)
	ListForAdmin(ctx context.Context, status *domain.CarServiceApplicationStatus, limit, offset int) ([]*domain.CarServiceApplication, error)
	Approve(ctx context.Context, input domain.ApproveCarServiceApplicationInput) error
	Reject(ctx context.Context, input domain.RejectCarServiceApplicationInput) error
}

type CarServiceProfileRepository interface {
	Create(ctx context.Context, profile *domain.CarServiceProfile) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.CarServiceProfile, error)
	GetByUserID(ctx context.Context, userID uuid.UUID) (*domain.CarServiceProfile, error)
	Update(ctx context.Context, input *domain.UpdateCarServiceProfileInput) error
	SetActive(ctx context.Context, userID uuid.UUID, isActive bool) error
}

type CarServiceImageRepository interface {
	Create(ctx context.Context, image *domain.CarServiceImage) error
	ListByProfileID(ctx context.Context, profileID uuid.UUID) ([]*domain.CarServiceImage, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.CarServiceImage, error)
	Delete(ctx context.Context, id uuid.UUID) error
	ClearPrimary(ctx context.Context, profileID uuid.UUID) error
	SetPrimary(ctx context.Context, id uuid.UUID) error
	NextSortOrder(ctx context.Context, profileID uuid.UUID) (int, error)
}

type DamageTypeRepository interface {
	ListActive(ctx context.Context) ([]*domain.DamageType, error)
	ListAll(ctx context.Context) ([]*domain.DamageType, error)
	ExistsActive(ctx context.Context, code string) (bool, error)
}

type PartCategoryRepository interface {
	ListActive(ctx context.Context) ([]*domain.PartCategory, error)
	ExistsActive(ctx context.Context, code string) (bool, error)
}

type CarServiceSpecializationRepository interface {
	Create(ctx context.Context, specialization *domain.CarServiceSpecialization) error
	ListByProfileID(ctx context.Context, profileID uuid.UUID) ([]*domain.CarServiceSpecialization, error)
	DeleteByProfileID(ctx context.Context, profileID uuid.UUID) error
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
}

type FileRepository interface {
	Upload(ctx context.Context, bucket, objectKey string, data io.Reader, contentType string, size int64) error
	Download(ctx context.Context, bucket, objectKey string) (io.ReadCloser, error)
	Exists(ctx context.Context, bucket, objectKey string) (bool, error)
	Delete(ctx context.Context, bucket, objectKey string) error
	GetPresignedURL(ctx context.Context, bucket, objectKey string, expires time.Duration) (string, error)
}
