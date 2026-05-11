package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/DedovInside/AutoInspect/backend/internal/domain"
	"github.com/DedovInside/AutoInspect/backend/internal/repository"
	"github.com/DedovInside/AutoInspect/backend/internal/repository/postgres"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const (
	defaultCarServiceApplicationLimit = 20
	maxCarServiceApplicationLimit     = 100
)

type CarServiceApplicationService struct {
	applicationRepo repository.CarServiceApplicationRepository
	profileRepo     repository.CarServiceProfileRepository
	userRepo        repository.UserRepository
	db              *postgres.DB
}

type ApproveCarServiceApplicationResult struct {
	Application *domain.CarServiceApplication
	Profile     *domain.CarServiceProfile
}

func NewCarServiceApplicationService(
	db *postgres.DB,
	applicationRepo repository.CarServiceApplicationRepository,
	profileRepo repository.CarServiceProfileRepository,
	userRepo repository.UserRepository,
) *CarServiceApplicationService {
	return &CarServiceApplicationService{
		applicationRepo: applicationRepo,
		profileRepo:     profileRepo,
		userRepo:        userRepo,
		db:              db,
	}
}

func (s *CarServiceApplicationService) Create(ctx context.Context,
	input *domain.CreateCarServiceApplicationInput) (*domain.CarServiceApplication, error) {
	if err := validateCreateCarServiceApplicationInput(input); err != nil {
		return nil, err
	}

	existing, err := s.applicationRepo.GetPendingByUserID(ctx, input.UserID)
	if err == nil {
		return existing, domain.ErrAlreadyExists
	}

	if !errors.Is(err, domain.ErrNotFound) {
		return nil, err
	}

	now := time.Now().UTC()
	application := &domain.CarServiceApplication{
		ID:               uuid.New(),
		UserID:           input.UserID,
		OrganizationName: strings.TrimSpace(input.OrganizationName),
		City:             strings.TrimSpace(input.City),
		Address:          strings.TrimSpace(input.Address),
		Phone:            normalizedOptionalString(input.Phone),
		Email:            normalizedOptionalString(input.Email),
		ContactInfo:      normalizedOptionalString(input.ContactInfo),
		Description:      strings.TrimSpace(input.Description),
		Status:           domain.CarServiceApplicationStatusPending,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	if err := s.applicationRepo.Create(ctx, application); err != nil {
		if errors.Is(err, domain.ErrAlreadyExists) {
			existing, getErr := s.applicationRepo.GetPendingByUserID(ctx, input.UserID)
			if getErr == nil {
				return existing, domain.ErrAlreadyExists
			}
		}
		return nil, err
	}

	return application, nil
}

func (s *CarServiceApplicationService) GetMyPending(ctx context.Context,
	userID uuid.UUID) (*domain.CarServiceApplication, error) {
	if userID == uuid.Nil {
		return nil, domain.ErrInvalidInput
	}

	return s.applicationRepo.GetPendingByUserID(ctx, userID)
}

func (s *CarServiceApplicationService) ListMine(ctx context.Context,
	userID uuid.UUID, limit, offset int) ([]*domain.CarServiceApplication, error) {
	if userID == uuid.Nil {
		return nil, domain.ErrInvalidInput
	}

	limit, offset, err := normalizeCarServiceApplicationPagination(limit, offset)
	if err != nil {
		return nil, err
	}

	return s.applicationRepo.ListByUserID(ctx, userID, limit, offset)
}

func (s *CarServiceApplicationService) ListForAdmin(ctx context.Context,
	status *domain.CarServiceApplicationStatus, limit, offset int) ([]*domain.CarServiceApplication, error) {
	if status != nil && !status.IsValid() {
		return nil, domain.ErrInvalidInput
	}

	limit, offset, err := normalizeCarServiceApplicationPagination(limit, offset)
	if err != nil {
		return nil, err
	}

	return s.applicationRepo.ListForAdmin(ctx, status, limit, offset)
}

func (s *CarServiceApplicationService) Approve(ctx context.Context,
	input *domain.ApproveCarServiceApplicationInput) (*ApproveCarServiceApplicationResult, error) {
	if err := validateApproveCarServiceApplicationInput(input); err != nil {
		return nil, err
	}

	tx, err := s.beginTx(ctx)
	if err != nil {
		return nil, err
	}

	defer func() { _ = tx.Rollback(ctx) }()

	applicationRepo := postgres.NewCarServiceApplicationRepo(tx)
	profileRepo := postgres.NewCarServiceProfileRepo(tx)
	userRepo := postgres.NewUserRepo(tx)

	result, err := approveCarServiceApplicationInTx(ctx, input, applicationRepo, profileRepo, userRepo)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, domain.ErrInternal
	}

	return result, nil
}

func (s *CarServiceApplicationService) beginTx(ctx context.Context) (pgx.Tx, error) {
	if s.db == nil {
		return nil, domain.ErrInternal
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, domain.ErrInternal
	}

	return tx, nil
}

func approveCarServiceApplicationInTx(ctx context.Context,
	input *domain.ApproveCarServiceApplicationInput,
	applicationRepo repository.CarServiceApplicationRepository,
	profileRepo repository.CarServiceProfileRepository,
	userRepo repository.UserRepository,
) (*ApproveCarServiceApplicationResult, error) {
	application, err := applicationRepo.GetByID(ctx, input.ID)
	if err != nil {
		return nil, err
	}

	if err := ensurePendingCarServiceApplication(application); err != nil {
		return nil, err
	}

	if err := ensureApplicationUserCanBecomeCarService(ctx, userRepo, application.UserID); err != nil {
		return nil, err
	}

	profile := newCarServiceProfileFromApplication(application)
	if err := createApprovedCarServiceProfile(ctx, profileRepo, userRepo, applicationRepo, input, profile); err != nil {
		return nil, err
	}

	updatedApplication, err := applicationRepo.GetByID(ctx, input.ID)
	if err != nil {
		return nil, err
	}

	return &ApproveCarServiceApplicationResult{
		Application: updatedApplication,
		Profile:     profile,
	}, nil
}

func ensurePendingCarServiceApplication(application *domain.CarServiceApplication) error {
	if application == nil || application.Status != domain.CarServiceApplicationStatusPending {
		return domain.ErrInvalidInput
	}

	return nil
}

func ensureApplicationUserCanBecomeCarService(ctx context.Context,
	userRepo repository.UserRepository, userID uuid.UUID) error {
	user, err := userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	if !user.IsActive || user.Role != domain.RoleUser {
		return domain.ErrInvalidInput
	}

	return nil
}

func createApprovedCarServiceProfile(ctx context.Context,
	profileRepo repository.CarServiceProfileRepository,
	userRepo repository.UserRepository,
	applicationRepo repository.CarServiceApplicationRepository,
	input *domain.ApproveCarServiceApplicationInput,
	profile *domain.CarServiceProfile,
) error {
	if err := profileRepo.Create(ctx, profile); err != nil {
		return err
	}

	if err := userRepo.UpdateRole(ctx, profile.UserID, domain.RoleCarService); err != nil {
		return err
	}

	profileID := profile.ID
	return applicationRepo.Approve(ctx, domain.ApproveCarServiceApplicationInput{
		ID:               input.ID,
		ReviewedBy:       input.ReviewedBy,
		CreatedProfileID: &profileID,
	})
}

func (s *CarServiceApplicationService) Reject(ctx context.Context,
	input *domain.RejectCarServiceApplicationInput) (*domain.CarServiceApplication, error) {
	if input == nil ||
		input.ID == uuid.Nil ||
		input.ReviewedBy == uuid.Nil ||
		strings.TrimSpace(input.RejectionReason) == "" {
		return nil, domain.ErrInvalidInput
	}

	application, err := s.applicationRepo.GetByID(ctx, input.ID)
	if err != nil {
		return nil, err
	}

	if application.Status != domain.CarServiceApplicationStatusPending {
		return nil, domain.ErrInvalidInput
	}

	rejectionReason := strings.TrimSpace(input.RejectionReason)
	if err := s.applicationRepo.Reject(ctx, domain.RejectCarServiceApplicationInput{
		ID:              input.ID,
		ReviewedBy:      input.ReviewedBy,
		RejectionReason: rejectionReason,
	}); err != nil {
		return nil, err
	}

	return s.applicationRepo.GetByID(ctx, input.ID)
}

func validateCreateCarServiceApplicationInput(input *domain.CreateCarServiceApplicationInput) error {
	if input == nil ||
		input.UserID == uuid.Nil ||
		input.UserRole != domain.RoleUser ||
		strings.TrimSpace(input.OrganizationName) == "" ||
		strings.TrimSpace(input.City) == "" ||
		strings.TrimSpace(input.Address) == "" ||
		strings.TrimSpace(input.Description) == "" {
		return domain.ErrInvalidInput
	}

	if normalizedOptionalString(input.Phone) == nil &&
		normalizedOptionalString(input.Email) == nil &&
		normalizedOptionalString(input.ContactInfo) == nil {
		return domain.ErrInvalidInput
	}

	return nil
}

func validateApproveCarServiceApplicationInput(input *domain.ApproveCarServiceApplicationInput) error {
	if input == nil || input.ID == uuid.Nil || input.ReviewedBy == uuid.Nil {
		return domain.ErrInvalidInput
	}
	return nil
}

func normalizeCarServiceApplicationPagination(limit, offset int) (normalizedLimit, normalizedOffset int, err error) {
	if limit <= 0 {
		limit = defaultCarServiceApplicationLimit
	}

	if limit > maxCarServiceApplicationLimit {
		limit = maxCarServiceApplicationLimit
	}

	if offset < 0 {
		return 0, 0, domain.ErrInvalidInput
	}

	return limit, offset, nil
}

func newCarServiceProfileFromApplication(application *domain.CarServiceApplication) *domain.CarServiceProfile {
	now := time.Now().UTC()
	description := strings.TrimSpace(application.Description)

	return &domain.CarServiceProfile{
		ID:               uuid.New(),
		UserID:           application.UserID,
		OrganizationName: strings.TrimSpace(application.OrganizationName),
		City:             strings.TrimSpace(application.City),
		Address:          strings.TrimSpace(application.Address),
		Phone:            normalizedOptionalString(application.Phone),
		Email:            normalizedOptionalString(application.Email),
		ContactInfo:      normalizedOptionalString(application.ContactInfo),
		Description:      normalizedOptionalString(&description),
		IsActive:         true,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
}
