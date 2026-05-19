package service

import (
	"context"
	"slices"
	"testing"

	"github.com/DedovInside/AutoInspect/backend/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestModelTrainingRequestServiceCreateTrimsInputAndStoresPendingRequest(t *testing.T) {
	t.Parallel()

	repo := newFakeModelTrainingRequestRepo()
	svc := NewModelTrainingRequestService(repo, nil)
	userID := uuid.New()
	idempotencyKey := " training-key "

	got, err := svc.Create(context.Background(), &domain.CreateModelTrainingRequestInput{
		InitiatorUserID: userID,
		InitiatorRole:   domain.RoleUser,
		Make:            " Volkswagen ",
		Model:           " Golf ",
		Generation:      " 5 ",
		YearFrom:        2008,
		YearTo:          2013,
		Description:     " Нужна специализированная модель ",
		IdempotencyKey:  &idempotencyKey,
	})

	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, got.ID)
	require.Equal(t, userID, got.InitiatorUserID)
	require.Equal(t, domain.RoleUser, got.InitiatorRole)
	require.Equal(t, "Volkswagen", got.Make)
	require.Equal(t, "Golf", got.Model)
	require.Equal(t, "5", got.Generation)
	require.Equal(t, 2008, got.YearFrom)
	require.Equal(t, 2013, got.YearTo)
	require.Equal(t, "Нужна специализированная модель", got.Description)
	require.Equal(t, domain.ModelTrainingRequestStatusPending, got.Status)
	require.Equal(t, "training-key", *got.IdempotencyKey)
	require.Len(t, repo.items, 1)
}

func TestModelTrainingRequestServiceCreateReturnsExistingByIdempotencyKey(t *testing.T) {
	t.Parallel()

	repo := newFakeModelTrainingRequestRepo()
	svc := NewModelTrainingRequestService(repo, nil)
	userID := uuid.New()
	idempotencyKey := "same-key"

	first, err := svc.Create(context.Background(), &domain.CreateModelTrainingRequestInput{
		InitiatorUserID: userID,
		InitiatorRole:   domain.RoleUser,
		Make:            "Volkswagen",
		Model:           "Golf",
		YearFrom:        2008,
		Description:     "first",
		IdempotencyKey:  &idempotencyKey,
	})
	require.NoError(t, err)

	second, err := svc.Create(context.Background(), &domain.CreateModelTrainingRequestInput{
		InitiatorUserID: userID,
		InitiatorRole:   domain.RoleUser,
		Make:            "Volkswagen",
		Model:           "Golf",
		YearFrom:        2008,
		Description:     "second",
		IdempotencyKey:  &idempotencyKey,
	})

	require.NoError(t, err)
	require.Same(t, first, second)
	require.Len(t, repo.items, 1)
}

func TestModelTrainingRequestServiceCreateEnforcesActiveLimitByRole(t *testing.T) {
	t.Parallel()

	repo := newFakeModelTrainingRequestRepo()
	repo.activeCount = activeTrainingRequestLimitUser
	svc := NewModelTrainingRequestService(repo, nil)

	got, err := svc.Create(context.Background(), &domain.CreateModelTrainingRequestInput{
		InitiatorUserID: uuid.New(),
		InitiatorRole:   domain.RoleUser,
		Make:            "Volkswagen",
		Model:           "Golf",
		YearFrom:        2008,
		Description:     "limit",
	})

	require.Nil(t, got)
	require.ErrorIs(t, err, domain.ErrTrainingRequestLimitExceeded)
}

func TestModelTrainingRequestServiceUpdateStatusValidatesCreatedModel(t *testing.T) {
	t.Parallel()

	requestID := uuid.New()
	modelID := uuid.New()
	repo := newFakeModelTrainingRequestRepo()
	repo.items = append(repo.items, &domain.ModelTrainingRequest{
		ID:     requestID,
		Status: domain.ModelTrainingRequestStatusPending,
	})
	modelRepo := &fakeCarModelRepo{
		byID: map[uuid.UUID]*domain.CarModel{
			modelID: {ID: modelID},
		},
	}
	svc := NewModelTrainingRequestService(repo, modelRepo)

	got, err := svc.UpdateStatus(context.Background(), &domain.UpdateModelTrainingRequestStatusInput{
		ID:             requestID,
		Status:         domain.ModelTrainingRequestStatusCompleted,
		ReviewedBy:     uuid.New(),
		CreatedModelID: &modelID,
		AdminComment:   "Готово",
	})

	require.NoError(t, err)
	require.Equal(t, domain.ModelTrainingRequestStatusCompleted, got.Status)
	require.Equal(t, "Готово", got.AdminComment)
	require.Equal(t, modelID, *got.CreatedModelID)
}

type fakeModelTrainingRequestRepo struct {
	items       []*domain.ModelTrainingRequest
	activeCount int
}

func newFakeModelTrainingRequestRepo() *fakeModelTrainingRequestRepo {
	return &fakeModelTrainingRequestRepo{items: make([]*domain.ModelTrainingRequest, 0)}
}

func (r *fakeModelTrainingRequestRepo) Create(_ context.Context, request *domain.ModelTrainingRequest) error {
	if request.IdempotencyKey != nil {
		for _, item := range r.items {
			if item.InitiatorUserID == request.InitiatorUserID &&
				item.IdempotencyKey != nil &&
				*item.IdempotencyKey == *request.IdempotencyKey {
				return domain.ErrAlreadyExists
			}
		}
	}
	r.items = append(r.items, request)
	return nil
}

func (r *fakeModelTrainingRequestRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.ModelTrainingRequest, error) {
	for _, item := range r.items {
		if item.ID == id {
			return item, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (r *fakeModelTrainingRequestRepo) GetByUserAndIdempotencyKey(
	_ context.Context,
	userID uuid.UUID,
	idempotencyKey string,
) (*domain.ModelTrainingRequest, error) {
	for _, item := range r.items {
		if item.InitiatorUserID == userID && item.IdempotencyKey != nil && *item.IdempotencyKey == idempotencyKey {
			return item, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (r *fakeModelTrainingRequestRepo) ListByUserID(
	_ context.Context,
	userID uuid.UUID,
	limit, offset int,
) ([]*domain.ModelTrainingRequest, error) {
	out := make([]*domain.ModelTrainingRequest, 0)
	for _, item := range r.items {
		if item.InitiatorUserID == userID {
			out = append(out, item)
		}
	}
	return sliceWindow(out, limit, offset), nil
}

func (r *fakeModelTrainingRequestRepo) ListForAdmin(
	_ context.Context,
	status *domain.ModelTrainingRequestStatus,
	limit, offset int,
) ([]*domain.ModelTrainingRequest, error) {
	out := make([]*domain.ModelTrainingRequest, 0, len(r.items))
	for _, item := range r.items {
		if status == nil || item.Status == *status {
			out = append(out, item)
		}
	}
	return sliceWindow(out, limit, offset), nil
}

func (r *fakeModelTrainingRequestRepo) CountActiveByUserID(_ context.Context, _ uuid.UUID) (int, error) {
	return r.activeCount, nil
}

func (r *fakeModelTrainingRequestRepo) UpdateStatus(
	_ context.Context,
	input domain.UpdateModelTrainingRequestStatusInput,
) error {
	item, err := r.GetByID(context.Background(), input.ID)
	if err != nil {
		return err
	}
	item.Status = input.Status
	item.AdminComment = input.AdminComment
	item.ReviewedBy = &input.ReviewedBy
	item.CreatedModelID = input.CreatedModelID
	return nil
}

type fakeCarModelRepo struct {
	models         []*domain.CarModel
	byID           map[uuid.UUID]*domain.CarModel
	activeModel    *domain.CarModel
	universalModel *domain.CarModel
	findErr        error
	universalErr   error
}

func (r *fakeCarModelRepo) FindActiveModel(context.Context, string, string, string, int) (*domain.CarModel, error) {
	if r.findErr != nil {
		return nil, r.findErr
	}
	if r.activeModel != nil {
		return r.activeModel, nil
	}
	return nil, domain.ErrInvalidModel
}

func (r *fakeCarModelRepo) GetUniversalModel(context.Context) (*domain.CarModel, error) {
	if r.universalErr != nil {
		return nil, r.universalErr
	}
	if r.universalModel != nil {
		return r.universalModel, nil
	}
	return nil, domain.ErrInvalidModel
}

func (r *fakeCarModelRepo) CreateModel(_ context.Context, model *domain.CarModel) error {
	r.models = append(r.models, model)
	if r.byID == nil {
		r.byID = make(map[uuid.UUID]*domain.CarModel)
	}
	r.byID[model.ID] = model
	return nil
}

func (r *fakeCarModelRepo) ListModels(context.Context, int, int) ([]*domain.CarModel, error) {
	return slices.Clone(r.models), nil
}

func (r *fakeCarModelRepo) GetModelByID(_ context.Context, id uuid.UUID) (*domain.CarModel, error) {
	if r.byID != nil {
		if model, ok := r.byID[id]; ok {
			return model, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (r *fakeCarModelRepo) DeactivateModel(_ context.Context, id uuid.UUID) error {
	model, err := r.GetModelByID(context.Background(), id)
	if err != nil {
		return err
	}
	model.IsActive = false
	return nil
}

func sliceWindow[T any](items []T, limit, offset int) []T {
	if offset >= len(items) {
		return []T{}
	}
	end := offset + limit
	if end > len(items) {
		end = len(items)
	}
	return items[offset:end]
}
