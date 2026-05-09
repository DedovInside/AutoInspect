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
	defaultModelTrainingRequestLimit = 20
	maxModelTrainingRequestLimit     = 100

	activeTrainingRequestLimitUser       = 3
	activeTrainingRequestLimitCarService = 10
	activeTrainingRequestLimitAdmin      = 20
)

type ModelTrainingRequestService struct {
	requestRepo repository.ModelTrainingRequestRepository
	modelRepo   repository.CarModelRepository
}

func NewModelTrainingRequestService(
	requestRepo repository.ModelTrainingRequestRepository,
	modelRepo repository.CarModelRepository,
) *ModelTrainingRequestService {
	return &ModelTrainingRequestService{
		requestRepo: requestRepo,
		modelRepo:   modelRepo,
	}
}

func (s *ModelTrainingRequestService) Create(ctx context.Context,
	input *domain.CreateModelTrainingRequestInput) (*domain.ModelTrainingRequest, error) {
	if err := validateCreateModelTrainingRequestInput(input); err != nil {
		return nil, err
	}

	idempotencyKey := normalizedOptionalString(input.IdempotencyKey)
	if idempotencyKey != nil {
		existing, err := s.requestRepo.GetByUserAndIdempotencyKey(ctx, input.InitiatorUserID, *idempotencyKey)
		if err == nil {
			return existing, nil
		}

		if !errors.Is(err, domain.ErrNotFound) {
			return nil, err
		}
	}

	if err := s.ensureActiveRequestLimit(ctx, input.InitiatorUserID, input.InitiatorRole); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	request := &domain.ModelTrainingRequest{
		ID:              uuid.New(),
		InitiatorUserID: input.InitiatorUserID,
		InitiatorRole:   input.InitiatorRole,
		Make:            strings.TrimSpace(input.Make),
		Model:           strings.TrimSpace(input.Model),
		Generation:      strings.TrimSpace(input.Generation),
		YearFrom:        input.YearFrom,
		YearTo:          input.YearTo,
		Description:     strings.TrimSpace(input.Description),
		Status:          domain.ModelTrainingRequestStatusPending,
		IdempotencyKey:  idempotencyKey,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := s.requestRepo.Create(ctx, request); err != nil {
		if errors.Is(err, domain.ErrAlreadyExists) && idempotencyKey != nil {
			existing, getErr := s.requestRepo.GetByUserAndIdempotencyKey(ctx, input.InitiatorUserID, *idempotencyKey)
			if getErr == nil {
				return existing, nil
			}
		}
		return nil, err
	}

	return request, nil
}

func (s *ModelTrainingRequestService) ListMine(ctx context.Context,
	userID uuid.UUID, limit, offset int) ([]*domain.ModelTrainingRequest, error) {
	if userID == uuid.Nil {
		return nil, domain.ErrInvalidInput
	}

	limit, offset, err := normalizeListPagination(limit, offset)
	if err != nil {
		return nil, err
	}

	return s.requestRepo.ListByUserID(ctx, userID, limit, offset)
}

func (s *ModelTrainingRequestService) ListForAdmin(
	ctx context.Context,
	status *domain.ModelTrainingRequestStatus,
	limit, offset int,
) ([]*domain.ModelTrainingRequest, error) {
	if status != nil && !status.IsValid() {
		return nil, domain.ErrInvalidInput
	}

	limit, offset, err := normalizeListPagination(limit, offset)
	if err != nil {
		return nil, err
	}

	return s.requestRepo.ListForAdmin(ctx, status, limit, offset)
}

func (s *ModelTrainingRequestService) UpdateStatus(
	ctx context.Context,
	input *domain.UpdateModelTrainingRequestStatusInput,
) error {
	if err := validateUpdateModelTrainingRequestStatusInput(input); err != nil {
		return err
	}

	if input.CreatedModelID != nil && s.modelRepo != nil {
		if _, err := s.modelRepo.GetModelByID(ctx, *input.CreatedModelID); err != nil {
			return err
		}
	}

	return s.requestRepo.UpdateStatus(ctx, *input)
}

func (s *ModelTrainingRequestService) ensureActiveRequestLimit(
	ctx context.Context,
	userID uuid.UUID,
	role domain.Role,
) error {
	limit := activeTrainingRequestLimit(role)
	if limit <= 0 {
		return nil
	}

	count, err := s.requestRepo.CountActiveByUserID(ctx, userID)
	if err != nil {
		return err
	}

	if count >= limit {
		return domain.ErrTrainingRequestLimitExceeded
	}

	return nil
}

func activeTrainingRequestLimit(role domain.Role) int {
	switch role {
	case domain.RoleUser:
		return activeTrainingRequestLimitUser
	case domain.RoleCarService:
		return activeTrainingRequestLimitCarService
	case domain.RoleAdmin:
		return activeTrainingRequestLimitAdmin
	default:
		return 0
	}
}

func validateCreateModelTrainingRequestInput(input *domain.CreateModelTrainingRequestInput) error {
	if input == nil ||
		input.InitiatorUserID == uuid.Nil ||
		!input.InitiatorRole.IsValid() ||
		strings.TrimSpace(input.Make) == "" ||
		strings.TrimSpace(input.Model) == "" ||
		strings.TrimSpace(input.Description) == "" ||
		input.YearFrom <= 0 {
		return domain.ErrInvalidInput
	}
	if input.YearTo > 0 && input.YearTo < input.YearFrom {
		return domain.ErrInvalidInput
	}

	return nil
}

func validateUpdateModelTrainingRequestStatusInput(input *domain.UpdateModelTrainingRequestStatusInput) error {
	if input == nil ||
		input.ID == uuid.Nil ||
		input.ReviewedBy == uuid.Nil ||
		!input.Status.IsValid() {
		return domain.ErrInvalidInput
	}

	return nil
}

func normalizeListPagination(limit, offset int) (normalizedLimit, normalizedOffset int, err error) {
	if limit <= 0 {
		limit = defaultModelTrainingRequestLimit
	}

	if limit > maxModelTrainingRequestLimit {
		limit = maxModelTrainingRequestLimit
	}

	if offset < 0 {
		return 0, 0, domain.ErrInvalidInput
	}

	return limit, offset, nil
}

func normalizedOptionalString(value *string) *string {
	if value == nil {
		return nil
	}

	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}

	return &trimmed
}
