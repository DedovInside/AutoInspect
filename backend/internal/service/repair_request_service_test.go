package service

import (
	"context"
	"testing"
	"time"

	"github.com/DedovInside/AutoInspect/backend/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestRepairSummaryFromAnalysisResultAggregatesByPartSideAndDamageType(t *testing.T) {
	t.Parallel()

	result := &domain.AnalysisResult{
		Results: []domain.ImageAnalysisResult{
			{
				PartsSummary: []domain.PartSummary{
					{
						Name:         "hood",
						NameRU:       "капот",
						ParentName:   "hood",
						ParentNameRU: "капот",
						DamageCount:  1,
						DamageTypes: []domain.DamageTypeSummary{
							{Code: "Scratch", NameRU: "царапина", Count: 1},
						},
					},
					{
						Name:         "hood",
						NameRU:       "капот",
						ParentName:   "hood",
						ParentNameRU: "капот",
						DamageCount:  2,
						DamageTypes: []domain.DamageTypeSummary{
							{Code: " scratch ", Count: 2},
							{Code: "dent", NameRU: "вмятина", Count: 1},
						},
					},
				},
			},
			{
				PartsSummary: []domain.PartSummary{
					{
						Name:         "front-door",
						NameRU:       "передняя дверь",
						ParentName:   "front-door",
						ParentNameRU: "передняя дверь",
						IsPair:       true,
						Side:         "left",
						SideRU:       "слева",
						DamageTypes: []domain.DamageTypeSummary{
							{Code: "dent", NameRU: "вмятина", Count: 0},
						},
					},
				},
			},
		},
	}

	got := repairSummaryFromAnalysisResult(result)

	require.Equal(t, []domain.RepairSummaryItem{
		{
			PartName:     "hood",
			PartNameRU:   "капот",
			ParentName:   "hood",
			ParentNameRU: "капот",
			DamageCount:  3,
			DamageTypes: []domain.RepairDamageTypeSummary{
				{Code: "scratch", NameRU: "царапина", Count: 3},
				{Code: "dent", NameRU: "вмятина", Count: 1},
			},
		},
		{
			PartName:     "front-door",
			PartNameRU:   "передняя дверь",
			ParentName:   "front-door",
			ParentNameRU: "передняя дверь",
			IsPair:       true,
			Side:         "left",
			SideRU:       "слева",
			DamageCount:  1,
			DamageTypes: []domain.RepairDamageTypeSummary{
				{Code: "dent", NameRU: "вмятина", Count: 1},
			},
		},
	}, got)
}

func TestNormalizeRepairEstimateItemTrimsAndValidatesPriceRange(t *testing.T) {
	t.Parallel()

	priceMin := 5000.0
	priceMax := 8000.0
	comment := "  После осмотра уточним  "
	got, err := normalizeRepairEstimateItem(&domain.RepairEstimateItem{
		PartName:     " hood ",
		PartNameRU:   " капот ",
		ParentName:   " hood ",
		ParentNameRU: " капот ",
		DamageCode:   " Scratch ",
		DamageNameRU: " царапина ",
		Quantity:     2,
		PriceMin:     &priceMin,
		PriceMax:     &priceMax,
		Comment:      &comment,
	})

	require.NoError(t, err)
	require.Equal(t, domain.RepairEstimateItem{
		PartName:     "hood",
		PartNameRU:   "капот",
		ParentName:   "hood",
		ParentNameRU: "капот",
		DamageCode:   "scratch",
		DamageNameRU: "царапина",
		Quantity:     2,
		PriceMin:     &priceMin,
		PriceMax:     &priceMax,
		Comment:      ptr("После осмотра уточним"),
	}, got)

	invalidMin := 9000.0
	invalidMax := 8000.0
	_, err = normalizeRepairEstimateItem(&domain.RepairEstimateItem{
		PartName:   "hood",
		DamageCode: "scratch",
		Quantity:   1,
		PriceMin:   &invalidMin,
		PriceMax:   &invalidMax,
	})
	require.ErrorIs(t, err, domain.ErrInvalidInput)
}

func TestValidateAcceptRepairRequestInputRequiresPrices(t *testing.T) {
	t.Parallel()

	minPrice := 1000.0
	maxPrice := 2000.0
	valid := &domain.AcceptRepairRequestInput{
		ID:                uuid.New(),
		CarServiceUserID:  uuid.New(),
		EstimatedPriceMin: &minPrice,
		EstimatedPriceMax: &maxPrice,
	}

	require.NoError(t, validateAcceptRepairRequestInput(valid))
	require.ErrorIs(t, validateAcceptRepairRequestInput(&domain.AcceptRepairRequestInput{
		ID:               uuid.New(),
		CarServiceUserID: uuid.New(),
	}), domain.ErrInvalidInput)
}

func TestRepairRequestServiceCreateBuildsRepairSummaryAndStoresRequest(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	jobID := uuid.New()
	profileID := uuid.New()
	job := completedRepairAnalysisJob(userID, jobID)
	profile := &domain.CarServiceProfile{ID: profileID, UserID: uuid.New(), IsActive: true}
	requestRepo := newFakeRepairRequestRepo()
	svc := NewRepairRequestService(
		requestRepo,
		&fakeAnalysisJobRepo{byID: map[uuid.UUID]*domain.AnalysisJob{jobID: job}},
		&fakeCarServiceProfileRepo{byID: map[uuid.UUID]*domain.CarServiceProfile{profileID: profile}},
		nil,
		nil,
	)

	got, err := svc.Create(context.Background(), &domain.CreateRepairRequestInput{
		UserID:              userID,
		AnalysisJobID:       jobID,
		CarServiceProfileID: profileID,
		CustomerPhone:       ptr("+79990000000"),
	})

	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, got.ID)
	require.Equal(t, domain.RepairRequestStatusPending, got.Status)
	require.Equal(t, job, got.Analysis)
	require.Equal(t, profile, got.CarServiceProfile)
	require.Equal(t, "+79990000000", *got.CustomerPhone)
	require.Len(t, got.RepairSummary, 1)
	require.Equal(t, "hood", got.RepairSummary[0].PartName)
	require.Len(t, requestRepo.items, 1)
}

func TestRepairRequestServiceAcceptIncomingStoresEstimate(t *testing.T) {
	t.Parallel()

	serviceUserID := uuid.New()
	profileID := uuid.New()
	requestID := uuid.New()
	minPrice := 1000.0
	maxPrice := 2000.0
	requestRepo := newFakeRepairRequestRepo()
	requestRepo.items = append(requestRepo.items, &domain.RepairRequest{
		ID:                  requestID,
		UserID:              uuid.New(),
		AnalysisJobID:       uuid.New(),
		CarServiceProfileID: profileID,
		Status:              domain.RepairRequestStatusPending,
	})
	svc := NewRepairRequestService(
		requestRepo,
		&fakeAnalysisJobRepo{},
		&fakeCarServiceProfileRepo{
			byUserID: map[uuid.UUID]*domain.CarServiceProfile{
				serviceUserID: {ID: profileID, UserID: serviceUserID, IsActive: true},
			},
		},
		nil,
		nil,
	)

	got, err := svc.AcceptIncoming(context.Background(), &domain.AcceptRepairRequestInput{
		ID:                requestID,
		CarServiceUserID:  serviceUserID,
		ServiceComment:    ptr("Готовы принять"),
		EstimatedPriceMin: &minPrice,
		EstimatedPriceMax: &maxPrice,
		ServiceEstimate: []domain.RepairEstimateItem{
			{PartName: " hood ", DamageCode: " Scratch ", Quantity: 1, PriceMin: &minPrice, PriceMax: &maxPrice},
		},
	})

	require.NoError(t, err)
	require.Equal(t, domain.RepairRequestStatusAccepted, got.Status)
	require.Equal(t, "Готовы принять", *got.ServiceComment)
	require.Equal(t, minPrice, *got.EstimatedPriceMin)
	require.Equal(t, maxPrice, *got.EstimatedPriceMax)
	require.Len(t, got.ServiceEstimate, 1)
	require.Equal(t, "hood", got.ServiceEstimate[0].PartName)
	require.Equal(t, "scratch", got.ServiceEstimate[0].DamageCode)
}

func TestRepairRequestServiceRejectIncomingStoresComment(t *testing.T) {
	t.Parallel()

	serviceUserID := uuid.New()
	profileID := uuid.New()
	requestID := uuid.New()
	requestRepo := newFakeRepairRequestRepo()
	requestRepo.items = append(requestRepo.items, &domain.RepairRequest{
		ID:                  requestID,
		CarServiceProfileID: profileID,
		Status:              domain.RepairRequestStatusPending,
	})
	svc := NewRepairRequestService(
		requestRepo,
		&fakeAnalysisJobRepo{},
		&fakeCarServiceProfileRepo{
			byUserID: map[uuid.UUID]*domain.CarServiceProfile{
				serviceUserID: {ID: profileID, UserID: serviceUserID, IsActive: true},
			},
		},
		nil,
		nil,
	)

	got, err := svc.RejectIncoming(context.Background(), &domain.RejectRepairRequestInput{
		ID:               requestID,
		CarServiceUserID: serviceUserID,
		ServiceComment:   "Нет свободных мест",
	})

	require.NoError(t, err)
	require.Equal(t, domain.RepairRequestStatusRejected, got.Status)
	require.Equal(t, "Нет свободных мест", *got.ServiceComment)
}

func ptr(value string) *string {
	return &value
}

func completedRepairAnalysisJob(userID, jobID uuid.UUID) *domain.AnalysisJob {
	return &domain.AnalysisJob{
		ID:     jobID,
		UserID: userID,
		Status: domain.StatusCompleted,
		Result: &domain.AnalysisResult{
			Results: []domain.ImageAnalysisResult{
				{
					PartsSummary: []domain.PartSummary{
						{
							Name:         "hood",
							NameRU:       "капот",
							ParentName:   "hood",
							ParentNameRU: "капот",
							DamageCount:  1,
							DamageTypes: []domain.DamageTypeSummary{
								{Code: "dent", NameRU: "вмятина", Count: 1},
							},
						},
					},
				},
			},
		},
	}
}

type fakeRepairRequestRepo struct {
	items []*domain.RepairRequest
}

func newFakeRepairRequestRepo() *fakeRepairRequestRepo {
	return &fakeRepairRequestRepo{items: make([]*domain.RepairRequest, 0)}
}

func (r *fakeRepairRequestRepo) Create(_ context.Context, request *domain.RepairRequest) error {
	r.items = append(r.items, request)
	return nil
}

func (r *fakeRepairRequestRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.RepairRequest, error) {
	for _, item := range r.items {
		if item.ID == id {
			return item, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (r *fakeRepairRequestRepo) GetByIDAndCarServiceProfileID(
	_ context.Context,
	id, carServiceProfileID uuid.UUID,
) (*domain.RepairRequest, error) {
	item, err := r.GetByID(context.Background(), id)
	if err != nil {
		return nil, err
	}
	if item.CarServiceProfileID != carServiceProfileID {
		return nil, domain.ErrNotFound
	}
	return item, nil
}

func (r *fakeRepairRequestRepo) GetPendingByUserAnalysisAndService(
	_ context.Context,
	userID, analysisJobID, carServiceProfileID uuid.UUID,
) (*domain.RepairRequest, error) {
	for _, item := range r.items {
		if item.UserID == userID &&
			item.AnalysisJobID == analysisJobID &&
			item.CarServiceProfileID == carServiceProfileID &&
			item.Status == domain.RepairRequestStatusPending {
			return item, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (r *fakeRepairRequestRepo) ListByUserID(_ context.Context, userID uuid.UUID, _, _ int) ([]*domain.RepairRequest, error) {
	out := make([]*domain.RepairRequest, 0)
	for _, item := range r.items {
		if item.UserID == userID {
			out = append(out, item)
		}
	}
	return out, nil
}

func (r *fakeRepairRequestRepo) ListByCarServiceProfileID(
	_ context.Context,
	carServiceProfileID uuid.UUID,
	_, _ int,
) ([]*domain.RepairRequest, error) {
	out := make([]*domain.RepairRequest, 0)
	for _, item := range r.items {
		if item.CarServiceProfileID == carServiceProfileID {
			out = append(out, item)
		}
	}
	return out, nil
}

func (r *fakeRepairRequestRepo) CancelPendingByUserID(_ context.Context, id, userID uuid.UUID) error {
	item, err := r.GetByID(context.Background(), id)
	if err != nil {
		return err
	}
	if item.UserID != userID || item.Status != domain.RepairRequestStatusPending {
		return domain.ErrInvalidInput
	}
	item.Status = domain.RepairRequestStatusCanceled
	return nil
}

func (r *fakeRepairRequestRepo) RespondByCarServiceProfileID(
	_ context.Context,
	carServiceProfileID uuid.UUID,
	input *domain.RespondRepairRequestInput,
) error {
	item, err := r.GetByIDAndCarServiceProfileID(context.Background(), input.ID, carServiceProfileID)
	if err != nil {
		return err
	}
	item.Status = input.Status
	item.ServiceComment = input.ServiceComment
	item.ServiceEstimate = input.ServiceEstimate
	item.EstimatedPriceMin = input.EstimatedPriceMin
	item.EstimatedPriceMax = input.EstimatedPriceMax
	now := time.Now().UTC()
	item.RespondedAt = &now
	return nil
}

type fakeAnalysisJobRepo struct {
	byID map[uuid.UUID]*domain.AnalysisJob
}

func (r *fakeAnalysisJobRepo) Create(context.Context, *domain.AnalysisJob) error {
	return nil
}

func (r *fakeAnalysisJobRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.AnalysisJob, error) {
	if r.byID != nil {
		if job, ok := r.byID[id]; ok {
			return job, nil
		}
	}
	return nil, domain.ErrJobNotFound
}

func (r *fakeAnalysisJobRepo) GetByCorrelationID(context.Context, uuid.UUID) (*domain.AnalysisJob, error) {
	return nil, domain.ErrJobNotFound
}

func (r *fakeAnalysisJobRepo) GetByUserAndIdempotencyKey(context.Context, uuid.UUID, string) (*domain.AnalysisJob, error) {
	return nil, domain.ErrJobNotFound
}

func (r *fakeAnalysisJobRepo) GetByUserID(context.Context, uuid.UUID, int, int) ([]*domain.AnalysisJob, error) {
	return nil, nil
}

func (r *fakeAnalysisJobRepo) UpdateStatus(context.Context, uuid.UUID, domain.JobStatus, *string) error {
	return nil
}

func (r *fakeAnalysisJobRepo) UpdateStatusByCorrelationID(context.Context, uuid.UUID, domain.JobStatus, *string) error {
	return nil
}

func (r *fakeAnalysisJobRepo) UpdateResult(context.Context, uuid.UUID, *domain.AnalysisResult, string) error {
	return nil
}

func (r *fakeAnalysisJobRepo) UpdateResultByCorrelationID(context.Context, uuid.UUID, *domain.AnalysisResult, string) error {
	return nil
}

type fakeCarServiceProfileRepo struct {
	byID     map[uuid.UUID]*domain.CarServiceProfile
	byUserID map[uuid.UUID]*domain.CarServiceProfile
}

func (r *fakeCarServiceProfileRepo) Create(_ context.Context, profile *domain.CarServiceProfile) error {
	if r.byID == nil {
		r.byID = make(map[uuid.UUID]*domain.CarServiceProfile)
	}
	if r.byUserID == nil {
		r.byUserID = make(map[uuid.UUID]*domain.CarServiceProfile)
	}
	r.byID[profile.ID] = profile
	r.byUserID[profile.UserID] = profile
	return nil
}

func (r *fakeCarServiceProfileRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.CarServiceProfile, error) {
	if r.byID != nil {
		if profile, ok := r.byID[id]; ok {
			return profile, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (r *fakeCarServiceProfileRepo) GetByUserID(_ context.Context, userID uuid.UUID) (*domain.CarServiceProfile, error) {
	if r.byUserID != nil {
		if profile, ok := r.byUserID[userID]; ok {
			return profile, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (r *fakeCarServiceProfileRepo) Update(context.Context, *domain.UpdateCarServiceProfileInput) error {
	return nil
}

func (r *fakeCarServiceProfileRepo) SetActive(context.Context, uuid.UUID, bool) error {
	return nil
}
