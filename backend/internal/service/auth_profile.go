package service

import (
	"context"
	"net/mail"

	"github.com/DedovInside/AutoInspect/backend/internal/domain"
	"github.com/google/uuid"
)

func (s *AuthService) GetMe(ctx context.Context, userID uuid.UUID) (*domain.User, error) {
	user, err := s.users.GetByID(ctx, userID)

	if err != nil {
		return nil, err
	}

	return user, nil
}

func (s *AuthService) UpdateContactProfile(
	ctx context.Context,
	input domain.UpdateUserContactProfileInput,
) (*domain.User, error) {
	if input.UserID == uuid.Nil {
		return nil, domain.ErrInvalidInput
	}

	normalized := domain.UpdateUserContactProfileInput{
		UserID:       input.UserID,
		ContactName:  normalizedOptionalString(input.ContactName),
		ContactPhone: normalizedOptionalString(input.ContactPhone),
		ContactEmail: normalizedOptionalString(input.ContactEmail),
	}

	if err := validateUserContactProfile(normalized); err != nil {
		return nil, err
	}

	return s.users.UpdateContactProfile(ctx, normalized)
}

func validateUserContactProfile(input domain.UpdateUserContactProfileInput) error {
	if input.ContactName != nil && len(*input.ContactName) > 255 {
		return domain.ErrInvalidInput
	}

	if input.ContactPhone != nil && len(*input.ContactPhone) > 50 {
		return domain.ErrInvalidInput
	}

	if input.ContactEmail != nil {
		if len(*input.ContactEmail) > 255 {
			return domain.ErrInvalidInput
		}
		address, err := mail.ParseAddress(*input.ContactEmail)
		if err != nil || address.Address != *input.ContactEmail {
			return domain.ErrInvalidInput
		}
	}

	return nil
}
