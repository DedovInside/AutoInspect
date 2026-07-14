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
	defaultCarServiceReviewLimit = 20
	maxCarServiceReviewLimit     = 100
	maxReviewAuthorNameLength    = 255
)

type CarServiceReviewService struct {
	reviewRepo  repository.CarServiceReviewRepository
	requestRepo repository.RepairRequestRepository
	userRepo    repository.UserRepository
}

func NewCarServiceReviewService(
	reviewRepo repository.CarServiceReviewRepository,
	requestRepo repository.RepairRequestRepository,
	userRepo repository.UserRepository,
) *CarServiceReviewService {
	return &CarServiceReviewService{
		reviewRepo:  reviewRepo,
		requestRepo: requestRepo,
		userRepo:    userRepo,
	}
}

func (s *CarServiceReviewService) Create(
	ctx context.Context,
	input *domain.CreateCarServiceReviewInput,
) (*domain.CarServiceReview, error) {
	if err := validateCreateCarServiceReviewInput(input); err != nil {
		return nil, err
	}

	request, err := s.requestRepo.GetByID(ctx, input.RepairRequestID)
	if err != nil {
		return nil, err
	}

	if request.UserID != input.UserID {
		return nil, domain.ErrForbidden
	}

	if request.Status != domain.RepairRequestStatusCompleted {
		return nil, domain.ErrInvalidInput
	}

	existing, err := s.reviewRepo.GetByRepairRequestID(ctx, request.ID)
	if err == nil {
		return existing, domain.ErrAlreadyExists
	}
	if !errors.Is(err, domain.ErrNotFound) {
		return nil, err
	}

	authorName, err := s.reviewAuthorName(ctx, input, request)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	review := &domain.CarServiceReview{
		ID:                  uuid.New(),
		RepairRequestID:     request.ID,
		CarServiceProfileID: request.CarServiceProfileID,
		UserID:              input.UserID,
		Rating:              input.Rating,
		AuthorName:          authorName,
		Comment:             normalizedOptionalString(input.Comment),
		CreatedAt:           now,
		UpdatedAt:           now,
	}

	if err := s.reviewRepo.Create(ctx, review); err != nil {
		if errors.Is(err, domain.ErrAlreadyExists) {
			existing, getErr := s.reviewRepo.GetByRepairRequestID(ctx, request.ID)
			if getErr == nil {
				return existing, domain.ErrAlreadyExists
			}
		}
		return nil, err
	}

	return review, nil
}

func (s *CarServiceReviewService) GetByRepairRequestID(
	ctx context.Context,
	repairRequestID uuid.UUID,
) (*domain.CarServiceReview, error) {
	if repairRequestID == uuid.Nil {
		return nil, domain.ErrInvalidInput
	}

	return s.reviewRepo.GetByRepairRequestID(ctx, repairRequestID)
}

func (s *CarServiceReviewService) Update(
	ctx context.Context,
	input *domain.UpdateCarServiceReviewInput,
) (*domain.CarServiceReview, error) {
	if err := validateUpdateCarServiceReviewInput(input); err != nil {
		return nil, err
	}

	request, err := s.requestRepo.GetByID(ctx, input.RepairRequestID)
	if err != nil {
		return nil, err
	}

	if request.UserID != input.UserID {
		return nil, domain.ErrForbidden
	}

	if request.Status != domain.RepairRequestStatusCompleted {
		return nil, domain.ErrInvalidInput
	}

	authorName, err := s.reviewAuthorName(ctx, &domain.CreateCarServiceReviewInput{
		UserID:          input.UserID,
		RepairRequestID: input.RepairRequestID,
		Rating:          input.Rating,
		AuthorName:      input.AuthorName,
		Comment:         input.Comment,
	}, request)
	if err != nil {
		return nil, err
	}

	updateInput := *input
	updateInput.AuthorName = authorName
	updateInput.Comment = normalizedOptionalString(input.Comment)

	return s.reviewRepo.UpdateByRepairRequestIDAndUserID(ctx, &updateInput)
}

func (s *CarServiceReviewService) Delete(
	ctx context.Context,
	userID, repairRequestID uuid.UUID,
) error {
	if userID == uuid.Nil || repairRequestID == uuid.Nil {
		return domain.ErrInvalidInput
	}

	return s.reviewRepo.DeleteByRepairRequestIDAndUserID(ctx, repairRequestID, userID)
}

func (s *CarServiceReviewService) ListByCarServiceProfileID(
	ctx context.Context,
	carServiceProfileID uuid.UUID,
	limit, offset int,
) ([]*domain.CarServiceReview, error) {
	if carServiceProfileID == uuid.Nil {
		return nil, domain.ErrInvalidInput
	}

	limit, offset, err := normalizeCarServiceReviewPagination(limit, offset)
	if err != nil {
		return nil, err
	}

	return s.reviewRepo.ListByCarServiceProfileID(ctx, carServiceProfileID, limit, offset)
}

func (s *CarServiceReviewService) ListMine(
	ctx context.Context,
	userID uuid.UUID,
	limit, offset int,
) ([]*domain.CarServiceReview, error) {
	if userID == uuid.Nil {
		return nil, domain.ErrInvalidInput
	}

	limit, offset, err := normalizeCarServiceReviewPagination(limit, offset)
	if err != nil {
		return nil, err
	}

	return s.reviewRepo.ListByUserID(ctx, userID, limit, offset)
}

func (s *CarServiceReviewService) reviewAuthorName(
	ctx context.Context,
	input *domain.CreateCarServiceReviewInput,
	request *domain.RepairRequest,
) (*string, error) {
	if authorName := normalizedOptionalString(input.AuthorName); authorName != nil {
		if len(*authorName) > maxReviewAuthorNameLength {
			return nil, domain.ErrInvalidInput
		}
		return authorName, nil
	}

	var user *domain.User
	if s.userRepo != nil {
		foundUser, err := s.userRepo.GetByID(ctx, input.UserID)
		if err != nil && !errors.Is(err, domain.ErrNotFound) {
			return nil, err
		}
		user = foundUser
	}

	for _, value := range []string{
		stringFromPtr(userContactName(user)),
		stringFromPtr(request.CustomerName),
		userDisplayName(user),
		stringFromPtr(userEmail(user)),
	} {
		normalized := normalizedOptionalString(&value)
		if normalized != nil {
			if len(*normalized) > maxReviewAuthorNameLength {
				return nil, domain.ErrInvalidInput
			}
			return normalized, nil
		}
	}

	return nil, nil
}

func validateCreateCarServiceReviewInput(input *domain.CreateCarServiceReviewInput) error {
	if input == nil ||
		input.UserID == uuid.Nil ||
		input.RepairRequestID == uuid.Nil ||
		!domain.IsValidReviewRating(input.Rating) {
		return domain.ErrInvalidInput
	}

	if input.AuthorName != nil && len(strings.TrimSpace(*input.AuthorName)) > maxReviewAuthorNameLength {
		return domain.ErrInvalidInput
	}

	return nil
}

func validateUpdateCarServiceReviewInput(input *domain.UpdateCarServiceReviewInput) error {
	if input == nil ||
		input.UserID == uuid.Nil ||
		input.RepairRequestID == uuid.Nil ||
		!domain.IsValidReviewRating(input.Rating) {
		return domain.ErrInvalidInput
	}

	if input.AuthorName != nil && len(strings.TrimSpace(*input.AuthorName)) > maxReviewAuthorNameLength {
		return domain.ErrInvalidInput
	}

	return nil
}

func normalizeCarServiceReviewPagination(limit, offset int) (normalizedLimit, normalizedOffset int, err error) {
	if limit <= 0 {
		limit = defaultCarServiceReviewLimit
	}

	if limit > maxCarServiceReviewLimit {
		limit = maxCarServiceReviewLimit
	}

	if offset < 0 {
		return 0, 0, domain.ErrInvalidInput
	}

	return limit, offset, nil
}

func userContactName(user *domain.User) *string {
	if user == nil {
		return nil
	}
	return user.ContactName
}

func userEmail(user *domain.User) *string {
	if user == nil {
		return nil
	}
	return &user.Email
}

func userDisplayName(user *domain.User) string {
	if user == nil {
		return ""
	}

	if user.DisplayName != nil {
		if displayName := strings.TrimSpace(*user.DisplayName); displayName != "" {
			return displayName
		}
	}

	parts := make([]string, 0, 2)
	if user.FirstName != nil {
		if firstName := strings.TrimSpace(*user.FirstName); firstName != "" {
			parts = append(parts, firstName)
		}
	}
	if user.LastName != nil {
		if lastName := strings.TrimSpace(*user.LastName); lastName != "" {
			parts = append(parts, lastName)
		}
	}
	if len(parts) > 0 {
		return strings.Join(parts, " ")
	}

	return user.Username
}

func stringFromPtr(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
