package service

import (
	"context"
	"testing"

	"github.com/DedovInside/AutoInspect/backend/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestAuthServiceUpdateContactProfileNormalizesAndStores(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	userRepo := &fakeAuthUserRepo{}
	svc := &AuthService{users: userRepo}

	got, err := svc.UpdateContactProfile(context.Background(), domain.UpdateUserContactProfileInput{
		UserID:       userID,
		ContactName:  ptr(" Иван "),
		ContactPhone: ptr(" +79990000000 "),
		ContactEmail: ptr(" ivan@example.com "),
	})

	require.NoError(t, err)
	require.Equal(t, userID, got.ID)
	require.Equal(t, userID, userRepo.lastInput.UserID)
	require.Equal(t, "Иван", *userRepo.lastInput.ContactName)
	require.Equal(t, "+79990000000", *userRepo.lastInput.ContactPhone)
	require.Equal(t, "ivan@example.com", *userRepo.lastInput.ContactEmail)
}

func TestAuthServiceUpdateContactProfileTurnsEmptyValuesIntoNil(t *testing.T) {
	t.Parallel()

	userRepo := &fakeAuthUserRepo{}
	svc := &AuthService{users: userRepo}

	_, err := svc.UpdateContactProfile(context.Background(), domain.UpdateUserContactProfileInput{
		UserID:       uuid.New(),
		ContactName:  ptr(" "),
		ContactPhone: ptr(""),
		ContactEmail: nil,
	})

	require.NoError(t, err)
	require.Nil(t, userRepo.lastInput.ContactName)
	require.Nil(t, userRepo.lastInput.ContactPhone)
	require.Nil(t, userRepo.lastInput.ContactEmail)
}

func TestAuthServiceUpdateContactProfileRejectsInvalidInput(t *testing.T) {
	t.Parallel()

	svc := &AuthService{users: &fakeAuthUserRepo{}}

	_, err := svc.UpdateContactProfile(context.Background(), domain.UpdateUserContactProfileInput{
		UserID:       uuid.New(),
		ContactEmail: ptr("bad-email"),
	})
	require.ErrorIs(t, err, domain.ErrInvalidInput)

	_, err = svc.UpdateContactProfile(context.Background(), domain.UpdateUserContactProfileInput{
		UserID: uuid.Nil,
	})
	require.ErrorIs(t, err, domain.ErrInvalidInput)
}

type fakeAuthUserRepo struct {
	lastInput domain.UpdateUserContactProfileInput
}

func (r *fakeAuthUserRepo) Create(context.Context, *domain.User) error {
	return nil
}

func (r *fakeAuthUserRepo) GetByID(context.Context, uuid.UUID) (*domain.User, error) {
	return nil, domain.ErrNotFound
}

func (r *fakeAuthUserRepo) GetByEmail(context.Context, string) (*domain.User, error) {
	return nil, domain.ErrNotFound
}

func (r *fakeAuthUserRepo) Update(context.Context, *domain.User) error {
	return nil
}

func (r *fakeAuthUserRepo) UpdateContactProfile(
	_ context.Context,
	input domain.UpdateUserContactProfileInput,
) (*domain.User, error) {
	r.lastInput = input
	return &domain.User{ID: input.UserID}, nil
}

func (r *fakeAuthUserRepo) UpdateRole(context.Context, uuid.UUID, domain.Role) error {
	return nil
}

func (r *fakeAuthUserRepo) Delete(context.Context, uuid.UUID) error {
	return nil
}

func (r *fakeAuthUserRepo) List(context.Context, int, int) ([]*domain.User, error) {
	return nil, nil
}

func (r *fakeAuthUserRepo) UpdateLastLogin(context.Context, uuid.UUID) error {
	return nil
}
