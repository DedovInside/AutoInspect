package handlers

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DedovInside/AutoInspect/backend/internal/domain"
	"github.com/DedovInside/AutoInspect/backend/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestRepairRequestHandlerCreate(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	userID := uuid.New()
	jobID := uuid.New()
	profileID := uuid.New()
	requestRepo := newHandlerRepairRequestRepo()
	handler := NewRepairRequestHandler(service.NewRepairRequestService(
		requestRepo,
		&handlerAnalysisJobRepo{byID: map[uuid.UUID]*domain.AnalysisJob{jobID: handlerCompletedRepairAnalysisJob(userID, jobID)}},
		&handlerCarServiceProfileRepo{byID: map[uuid.UUID]*domain.CarServiceProfile{
			profileID: {ID: profileID, IsActive: true},
		}},
		nil,
		nil,
		nil,
	))

	router := gin.New()
	router.POST("/v1/repair-requests", withTestUser(userID, domain.RoleUser), handler.Create)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/repair-requests", bytes.NewReader(mustJSON(t, map[string]any{
		"analysis_job_id":        jobID,
		"car_service_profile_id": profileID,
		"customer_phone":         "+79990000000",
		"customer_comment":       "Хочу записаться",
	})))
	request.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusCreated, recorder.Code)
	require.Contains(t, recorder.Body.String(), `"status":"pending"`)
	require.Contains(t, recorder.Body.String(), `"repair_summary"`)
	require.Len(t, requestRepo.items, 1)
}

func TestRepairRequestHandlerAcceptIncoming(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	serviceUserID := uuid.New()
	profileID := uuid.New()
	requestID := uuid.New()
	minPrice := 1000.0
	maxPrice := 2000.0
	requestRepo := newHandlerRepairRequestRepo()
	requestRepo.items = append(requestRepo.items, &domain.RepairRequest{
		ID:                  requestID,
		CarServiceProfileID: profileID,
		Status:              domain.RepairRequestStatusPending,
		CreatedAt:           time.Now().UTC(),
		UpdatedAt:           time.Now().UTC(),
	})
	handler := NewRepairRequestHandler(service.NewRepairRequestService(
		requestRepo,
		&handlerAnalysisJobRepo{},
		&handlerCarServiceProfileRepo{byUserID: map[uuid.UUID]*domain.CarServiceProfile{
			serviceUserID: {ID: profileID, UserID: serviceUserID, IsActive: true},
		}},
		nil,
		nil,
		nil,
	))

	router := gin.New()
	router.POST("/v1/repair-requests/incoming/:id/accept", withTestUser(serviceUserID, domain.RoleCarService), handler.AcceptIncoming)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodPost,
		"/v1/repair-requests/incoming/"+requestID.String()+"/accept",
		bytes.NewReader(mustJSON(t, map[string]any{
			"service_comment":     "Готовы принять",
			"estimated_price_min": minPrice,
			"estimated_price_max": maxPrice,
			"service_estimate": []map[string]any{
				{
					"part_name":   "hood",
					"damage_code": "dent",
					"quantity":    1,
					"price_min":   minPrice,
					"price_max":   maxPrice,
				},
			},
		})),
	)
	request.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), `"status":"accepted"`)
	require.Contains(t, recorder.Body.String(), `"service_comment":"Готовы принять"`)
	require.Contains(t, recorder.Body.String(), `"service_estimate"`)
}

func TestRepairRequestHandlerRejectIncoming(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	serviceUserID := uuid.New()
	profileID := uuid.New()
	requestID := uuid.New()
	requestRepo := newHandlerRepairRequestRepo()
	requestRepo.items = append(requestRepo.items, &domain.RepairRequest{
		ID:                  requestID,
		CarServiceProfileID: profileID,
		Status:              domain.RepairRequestStatusPending,
	})
	handler := NewRepairRequestHandler(service.NewRepairRequestService(
		requestRepo,
		&handlerAnalysisJobRepo{},
		&handlerCarServiceProfileRepo{byUserID: map[uuid.UUID]*domain.CarServiceProfile{
			serviceUserID: {ID: profileID, UserID: serviceUserID, IsActive: true},
		}},
		nil,
		nil,
		nil,
	))

	router := gin.New()
	router.POST("/v1/repair-requests/incoming/:id/reject", withTestUser(serviceUserID, domain.RoleCarService), handler.RejectIncoming)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodPost,
		"/v1/repair-requests/incoming/"+requestID.String()+"/reject",
		bytes.NewReader([]byte(`{"service_comment":"Нет свободных мест"}`)),
	)
	request.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), `"status":"rejected"`)
	require.Contains(t, recorder.Body.String(), `"service_comment":"Нет свободных мест"`)
}

func TestRepairRequestHandlerCancelMine(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	userID := uuid.New()
	profileID := uuid.New()
	requestID := uuid.New()
	jobID := uuid.New()
	requestRepo := newHandlerRepairRequestRepo()
	requestRepo.items = append(requestRepo.items, &domain.RepairRequest{
		ID:                  requestID,
		UserID:              userID,
		AnalysisJobID:       jobID,
		CarServiceProfileID: profileID,
		Status:              domain.RepairRequestStatusPending,
	})
	handler := NewRepairRequestHandler(service.NewRepairRequestService(
		requestRepo,
		&handlerAnalysisJobRepo{byID: map[uuid.UUID]*domain.AnalysisJob{
			jobID: handlerCompletedRepairAnalysisJob(userID, jobID),
		}},
		&handlerCarServiceProfileRepo{byID: map[uuid.UUID]*domain.CarServiceProfile{
			profileID: {ID: profileID, IsActive: true},
		}},
		nil,
		nil,
		nil,
	))

	router := gin.New()
	router.POST("/v1/repair-requests/:id/cancel", withTestUser(userID, domain.RoleUser), handler.CancelMine)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/repair-requests/"+requestID.String()+"/cancel", http.NoBody)

	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), `"status":"canceled"`)
	require.Equal(t, domain.RepairRequestStatusCanceled, requestRepo.items[0].Status)
}

func handlerCompletedRepairAnalysisJob(userID, jobID uuid.UUID) *domain.AnalysisJob {
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

type handlerRepairRequestRepo struct {
	items []*domain.RepairRequest
}

func newHandlerRepairRequestRepo() *handlerRepairRequestRepo {
	return &handlerRepairRequestRepo{items: make([]*domain.RepairRequest, 0)}
}

func (r *handlerRepairRequestRepo) Create(_ context.Context, request *domain.RepairRequest) error {
	r.items = append(r.items, request)
	return nil
}

func (r *handlerRepairRequestRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.RepairRequest, error) {
	for _, item := range r.items {
		if item.ID == id {
			return item, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (r *handlerRepairRequestRepo) GetByIDAndCarServiceProfileID(
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

func (r *handlerRepairRequestRepo) GetPendingByUserAnalysisAndService(
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

func (r *handlerRepairRequestRepo) ListByUserID(_ context.Context, userID uuid.UUID, _, _ int) ([]*domain.RepairRequest, error) {
	out := make([]*domain.RepairRequest, 0)
	for _, item := range r.items {
		if item.UserID == userID {
			out = append(out, item)
		}
	}
	return out, nil
}

func (r *handlerRepairRequestRepo) ListByCarServiceProfileID(
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

func (r *handlerRepairRequestRepo) CancelPendingByUserID(_ context.Context, id, userID uuid.UUID) error {
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

func (r *handlerRepairRequestRepo) RespondByCarServiceProfileID(
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

type handlerAnalysisJobRepo struct {
	byID    map[uuid.UUID]*domain.AnalysisJob
	created *domain.AnalysisJob
}

func (r *handlerAnalysisJobRepo) Create(_ context.Context, job *domain.AnalysisJob) error {
	r.created = job
	if r.byID == nil {
		r.byID = make(map[uuid.UUID]*domain.AnalysisJob)
	}
	r.byID[job.ID] = job
	return nil
}

func (r *handlerAnalysisJobRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.AnalysisJob, error) {
	if r.byID != nil {
		if job, ok := r.byID[id]; ok {
			return job, nil
		}
	}
	return nil, domain.ErrJobNotFound
}

func (r *handlerAnalysisJobRepo) GetByCorrelationID(context.Context, uuid.UUID) (*domain.AnalysisJob, error) {
	return nil, domain.ErrJobNotFound
}

func (r *handlerAnalysisJobRepo) GetByUserAndIdempotencyKey(
	_ context.Context,
	userID uuid.UUID,
	idempotencyKey string,
) (*domain.AnalysisJob, error) {
	for _, job := range r.byID {
		if job.UserID == userID && job.IdempotencyKey != nil && *job.IdempotencyKey == idempotencyKey {
			return job, nil
		}
	}
	return nil, domain.ErrJobNotFound
}

func (r *handlerAnalysisJobRepo) GetByUserID(_ context.Context, userID uuid.UUID, _, _ int) ([]*domain.AnalysisJob, error) {
	out := make([]*domain.AnalysisJob, 0)
	for _, job := range r.byID {
		if job.UserID == userID {
			out = append(out, job)
		}
	}
	return out, nil
}

func (r *handlerAnalysisJobRepo) UpdateStatus(context.Context, uuid.UUID, domain.JobStatus, *string) error {
	return nil
}

func (r *handlerAnalysisJobRepo) UpdateStatusByCorrelationID(context.Context, uuid.UUID, domain.JobStatus, *string) error {
	return nil
}

func (r *handlerAnalysisJobRepo) UpdateResult(context.Context, uuid.UUID, *domain.AnalysisResult, string) error {
	return nil
}

func (r *handlerAnalysisJobRepo) UpdateResultByCorrelationID(context.Context, uuid.UUID, *domain.AnalysisResult, string) error {
	return nil
}

type handlerCarServiceProfileRepo struct {
	byID     map[uuid.UUID]*domain.CarServiceProfile
	byUserID map[uuid.UUID]*domain.CarServiceProfile
}

func (r *handlerCarServiceProfileRepo) Create(context.Context, *domain.CarServiceProfile) error {
	return nil
}

func (r *handlerCarServiceProfileRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.CarServiceProfile, error) {
	if r.byID != nil {
		if profile, ok := r.byID[id]; ok {
			return profile, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (r *handlerCarServiceProfileRepo) GetByUserID(_ context.Context, userID uuid.UUID) (*domain.CarServiceProfile, error) {
	if r.byUserID != nil {
		if profile, ok := r.byUserID[userID]; ok {
			return profile, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (r *handlerCarServiceProfileRepo) Update(context.Context, *domain.UpdateCarServiceProfileInput) error {
	return nil
}

func (r *handlerCarServiceProfileRepo) SetActive(context.Context, uuid.UUID, bool) error {
	return nil
}
