package service

import (
	"context"
	"testing"

	"github.com/DedovInside/AutoInspect/backend/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestCarServiceReviewServiceCreateCompletedRequest(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	requestID := uuid.New()
	profileID := uuid.New()
	requestRepo := newFakeRepairRequestRepo()
	requestRepo.items = append(requestRepo.items, &domain.RepairRequest{
		ID:                  requestID,
		UserID:              userID,
		CarServiceProfileID: profileID,
		Status:              domain.RepairRequestStatusCompleted,
		CustomerName:        ptr("Имя из заявки"),
	})
	reviewRepo := newFakeCarServiceReviewRepo()
	svc := NewCarServiceReviewService(
		reviewRepo,
		requestRepo,
		&fakeReviewUserRepo{
			byID: map[uuid.UUID]*domain.User{
				userID: {
					ID:          userID,
					Username:    "ivan",
					Email:       "ivan@example.com",
					ContactName: ptr(" Иван Дедов "),
				},
			},
		},
	)

	got, err := svc.Create(context.Background(), &domain.CreateCarServiceReviewInput{
		UserID:          userID,
		RepairRequestID: requestID,
		Rating:          5,
		Comment:         ptr(" Всё отлично "),
	})

	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, got.ID)
	require.Equal(t, requestID, got.RepairRequestID)
	require.Equal(t, profileID, got.CarServiceProfileID)
	require.Equal(t, userID, got.UserID)
	require.Equal(t, 5, got.Rating)
	require.Equal(t, "Иван Дедов", *got.AuthorName)
	require.Equal(t, "Всё отлично", *got.Comment)
	require.Len(t, reviewRepo.items, 1)
}

func TestCarServiceReviewServiceCreateFallsBackToRequestCustomerName(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	requestID := uuid.New()
	requestRepo := newFakeRepairRequestRepo()
	requestRepo.items = append(requestRepo.items, &domain.RepairRequest{
		ID:           requestID,
		UserID:       userID,
		Status:       domain.RepairRequestStatusCompleted,
		CustomerName: ptr(" Клиент из заявки "),
	})
	svc := NewCarServiceReviewService(
		newFakeCarServiceReviewRepo(),
		requestRepo,
		&fakeReviewUserRepo{byID: map[uuid.UUID]*domain.User{userID: {ID: userID, Email: "u@example.com"}}},
	)

	got, err := svc.Create(context.Background(), &domain.CreateCarServiceReviewInput{
		UserID:          userID,
		RepairRequestID: requestID,
		Rating:          4,
	})

	require.NoError(t, err)
	require.Equal(t, "Клиент из заявки", *got.AuthorName)
}

func TestCarServiceReviewServiceCreateRejectsNotCompletedRequest(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	requestID := uuid.New()
	requestRepo := newFakeRepairRequestRepo()
	requestRepo.items = append(requestRepo.items, &domain.RepairRequest{
		ID:     requestID,
		UserID: userID,
		Status: domain.RepairRequestStatusAccepted,
	})
	svc := NewCarServiceReviewService(newFakeCarServiceReviewRepo(), requestRepo, nil)

	_, err := svc.Create(context.Background(), &domain.CreateCarServiceReviewInput{
		UserID:          userID,
		RepairRequestID: requestID,
		Rating:          5,
	})

	require.ErrorIs(t, err, domain.ErrInvalidInput)
}

func TestCarServiceReviewServiceCreateRejectsDuplicateReview(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	requestID := uuid.New()
	requestRepo := newFakeRepairRequestRepo()
	requestRepo.items = append(requestRepo.items, &domain.RepairRequest{
		ID:     requestID,
		UserID: userID,
		Status: domain.RepairRequestStatusCompleted,
	})
	reviewRepo := newFakeCarServiceReviewRepo()
	existing := &domain.CarServiceReview{
		ID:              uuid.New(),
		RepairRequestID: requestID,
		UserID:          userID,
		Rating:          5,
	}
	reviewRepo.items = append(reviewRepo.items, existing)
	svc := NewCarServiceReviewService(reviewRepo, requestRepo, nil)

	got, err := svc.Create(context.Background(), &domain.CreateCarServiceReviewInput{
		UserID:          userID,
		RepairRequestID: requestID,
		Rating:          4,
	})

	require.ErrorIs(t, err, domain.ErrAlreadyExists)
	require.Equal(t, existing, got)
}

func TestCarServiceReviewServiceUpdateOwnCompletedRequestReview(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	requestID := uuid.New()
	profileID := uuid.New()
	requestRepo := newFakeRepairRequestRepo()
	requestRepo.items = append(requestRepo.items, &domain.RepairRequest{
		ID:                  requestID,
		UserID:              userID,
		CarServiceProfileID: profileID,
		Status:              domain.RepairRequestStatusCompleted,
		CustomerName:        ptr("Клиент"),
	})
	reviewRepo := newFakeCarServiceReviewRepo()
	reviewRepo.items = append(reviewRepo.items, &domain.CarServiceReview{
		ID:                  uuid.New(),
		RepairRequestID:     requestID,
		CarServiceProfileID: profileID,
		UserID:              userID,
		Rating:              3,
		AuthorName:          ptr("Старое имя"),
		Comment:             ptr("Старый текст"),
	})
	svc := NewCarServiceReviewService(reviewRepo, requestRepo, nil)

	got, err := svc.Update(context.Background(), &domain.UpdateCarServiceReviewInput{
		UserID:          userID,
		RepairRequestID: requestID,
		Rating:          5,
		AuthorName:      ptr(" Новое имя "),
		Comment:         ptr(" Новый текст "),
	})

	require.NoError(t, err)
	require.Equal(t, 5, got.Rating)
	require.Equal(t, "Новое имя", *got.AuthorName)
	require.Equal(t, "Новый текст", *got.Comment)
}

func TestCarServiceReviewServiceDeleteOwnReview(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	requestID := uuid.New()
	reviewRepo := newFakeCarServiceReviewRepo()
	reviewRepo.items = append(reviewRepo.items, &domain.CarServiceReview{
		ID:              uuid.New(),
		RepairRequestID: requestID,
		UserID:          userID,
		Rating:          4,
	})
	svc := NewCarServiceReviewService(reviewRepo, newFakeRepairRequestRepo(), nil)

	err := svc.Delete(context.Background(), userID, requestID)

	require.NoError(t, err)
	require.Empty(t, reviewRepo.items)
}

type fakeCarServiceReviewRepo struct {
	items []*domain.CarServiceReview
}

func newFakeCarServiceReviewRepo() *fakeCarServiceReviewRepo {
	return &fakeCarServiceReviewRepo{items: make([]*domain.CarServiceReview, 0)}
}

func (r *fakeCarServiceReviewRepo) Create(_ context.Context, review *domain.CarServiceReview) error {
	for _, item := range r.items {
		if item.RepairRequestID == review.RepairRequestID {
			return domain.ErrAlreadyExists
		}
	}
	r.items = append(r.items, review)
	return nil
}

func (r *fakeCarServiceReviewRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.CarServiceReview, error) {
	for _, item := range r.items {
		if item.ID == id {
			return item, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (r *fakeCarServiceReviewRepo) GetByRepairRequestID(
	_ context.Context,
	repairRequestID uuid.UUID,
) (*domain.CarServiceReview, error) {
	for _, item := range r.items {
		if item.RepairRequestID == repairRequestID {
			return item, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (r *fakeCarServiceReviewRepo) ListByCarServiceProfileID(
	_ context.Context,
	carServiceProfileID uuid.UUID,
	_, _ int,
) ([]*domain.CarServiceReview, error) {
	out := make([]*domain.CarServiceReview, 0)
	for _, item := range r.items {
		if item.CarServiceProfileID == carServiceProfileID {
			out = append(out, item)
		}
	}
	return out, nil
}

func (r *fakeCarServiceReviewRepo) ListByUserID(
	_ context.Context,
	userID uuid.UUID,
	_, _ int,
) ([]*domain.CarServiceReview, error) {
	out := make([]*domain.CarServiceReview, 0)
	for _, item := range r.items {
		if item.UserID == userID {
			out = append(out, item)
		}
	}
	return out, nil
}

func (r *fakeCarServiceReviewRepo) UpdateByRepairRequestIDAndUserID(
	_ context.Context,
	input *domain.UpdateCarServiceReviewInput,
) (*domain.CarServiceReview, error) {
	for _, item := range r.items {
		if item.RepairRequestID == input.RepairRequestID && item.UserID == input.UserID {
			item.Rating = input.Rating
			item.AuthorName = input.AuthorName
			item.Comment = input.Comment
			return item, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (r *fakeCarServiceReviewRepo) DeleteByRepairRequestIDAndUserID(
	_ context.Context,
	repairRequestID, userID uuid.UUID,
) error {
	for i, item := range r.items {
		if item.RepairRequestID == repairRequestID && item.UserID == userID {
			r.items = append(r.items[:i], r.items[i+1:]...)
			return nil
		}
	}
	return domain.ErrNotFound
}

type fakeReviewUserRepo struct {
	fakeRepairRequestUserRepo
	byID map[uuid.UUID]*domain.User
}

func (r *fakeReviewUserRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.User, error) {
	if r.byID != nil {
		if user, ok := r.byID[id]; ok {
			return user, nil
		}
	}
	return nil, domain.ErrNotFound
}
