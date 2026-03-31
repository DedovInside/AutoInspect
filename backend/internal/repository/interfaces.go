package repository

import (
	"context"
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

// ModelRepository - интерфейс для работы с реестром inference-моделей.
//type ModelRepository interface {
//	Create(ctx context.Context, model *domain.MLModel) error
//	GetByID(ctx context.Context, id uuid.UUID) (*domain.MLModel, error)
//	GetByVersion(ctx context.Context, version string) (*domain.MLModel, error)
//
//	// ResolveForCarSpec выбирает подходящую модель по спецификации авто.
//	ResolveForCarSpec(ctx context.Context, carMake, carModel string, carGeneration *string, carYear *int) (*domain.MLModel, error)
//
//	Update(ctx context.Context, model *domain.MLModel) error
//	List(ctx context.Context, limit, offset int) ([]*domain.MLModel, error)
//
//	// SetActiveForCarSpec активирует модель и деактивирует другие только в той же car-spec группе.
//	SetActiveForCarSpec(ctx context.Context, id uuid.UUID) error
//	ListByCarModel(ctx context.Context, carMake, carModel string) ([]*domain.MLModel, error)
//}

// AnalysisRepository - интерфейс для работы с анализами изображений.
//type AnalysisRepository interface {
//	Create(ctx context.Context, analysis *domain.Analysis) error
//	GetByID(ctx context.Context, id uuid.UUID) (*domain.Analysis, error)
//	GetByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.Analysis, error)
//	Update(ctx context.Context, analysis *domain.Analysis) error
//	UpdateStatus(ctx context.Context, id uuid.UUID, status domain.AnalysisStatus) error
//	UpdateResult(ctx context.Context, id uuid.UUID, result *domain.AnalysisResult) error
//	Delete(ctx context.Context, id uuid.UUID) error
//	GetPending(ctx context.Context, limit int) ([]*domain.Analysis, error)
//}

// AuditLogRepository - интерфейс для работы с журналом аудита.
//type AuditLogRepository interface {
//	Create(ctx context.Context, log *domain.AuditLog) error
//	GetByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.AuditLog, error)
//	GetByAction(ctx context.Context, action string, limit, offset int) ([]*domain.AuditLog, error)
//	List(ctx context.Context, limit, offset int) ([]*domain.AuditLog, error)
//}

type AuthSessionRepository interface {
	Create(ctx context.Context, session *domain.AuthRefreshSession) error
	GetByTokenHash(ctx context.Context, tokenHash string) (*domain.AuthRefreshSession, error)
	Revoke(ctx context.Context, id uuid.UUID, revokedReason string, replacedByID *uuid.UUID) error
	TouchLastUsed(ctx context.Context, id uuid.UUID, at time.Time) error
	RevokeFamily(ctx context.Context, familyID uuid.UUID, revokedReason string) error
}

type OAuthIdentityRepository interface {
	Create(ctx context.Context, identity *domain.OAuthIdentity) error
	GetByProviderSubject(ctx context.Context, provider, providerUserID string) (*domain.OAuthIdentity, error)
}
