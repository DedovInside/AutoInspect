package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/DedovInside/AutoInspect/backend/internal/domain"
	"github.com/DedovInside/AutoInspect/backend/internal/repository"
	"github.com/google/uuid"
)

const (
	defaultCarServiceApplicationLimit = 20
	maxCarServiceApplicationLimit     = 100
)

type CarServiceApplicationService struct {
	applicationRepo repository.CarServiceApplicationRepository
}

func NewCarServiceApplicationService(
	applicationRepo repository.CarServiceApplicationRepository,
) *CarServiceApplicationService {
	return &CarServiceApplicationService{applicationRepo: applicationRepo}
}

func (s *CarServiceApplicationService) Create(
	ctx context.Context,
	input *domain.CreateCarServiceApplicationInput,
) (*domain.CarServiceApplication, error) {
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

func (s *CarServiceApplicationService) GetMyPending(
	ctx context.Context,
	userID uuid.UUID,
) (*domain.CarServiceApplication, error) {
	if userID == uuid.Nil {
		return nil, domain.ErrInvalidInput
	}

	return s.applicationRepo.GetPendingByUserID(ctx, userID)
}

func (s *CarServiceApplicationService) ListMine(
	ctx context.Context,
	userID uuid.UUID,
	limit, offset int,
) ([]*domain.CarServiceApplication, error) {
	if userID == uuid.Nil {
		return nil, domain.ErrInvalidInput
	}

	limit, offset, err := normalizeCarServiceApplicationPagination(limit, offset)
	if err != nil {
		return nil, err
	}

	return s.applicationRepo.ListByUserID(ctx, userID, limit, offset)
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
