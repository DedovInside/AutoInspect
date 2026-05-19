package handlers

import (
	"bytes"
	"context"
	"encoding/json"
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

func TestCarServiceApplicationHandlerCreate(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	repo := newHandlerCarServiceApplicationRepo()
	handler := NewCarServiceApplicationHandler(service.NewCarServiceApplicationService(nil, repo, nil, nil))
	userID := uuid.New()

	router := gin.New()
	router.POST("/v1/car-service-applications", withTestUser(userID, domain.RoleUser), handler.Create)

	payload := mustJSON(t, map[string]any{
		"organization_name": "Detail Lab",
		"city":              "Москва",
		"address":           "Тестовая улица, 1",
		"phone":             "+79990000000",
		"description":       "Кузовной ремонт",
	})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/car-service-applications", bytes.NewReader(payload))
	request.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusCreated, recorder.Code)
	require.Contains(t, recorder.Body.String(), `"organization_name":"Detail Lab"`)
	require.Contains(t, recorder.Body.String(), `"status":"pending"`)
	require.Len(t, repo.items, 1)
	require.Equal(t, userID, repo.items[0].UserID)
	require.Equal(t, "Detail Lab", repo.items[0].OrganizationName)
}

func TestCarServiceApplicationHandlerCreateRejectsCarServiceRole(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	handler := NewCarServiceApplicationHandler(service.NewCarServiceApplicationService(
		nil,
		newHandlerCarServiceApplicationRepo(),
		nil,
		nil,
	))

	router := gin.New()
	router.POST("/v1/car-service-applications", withTestUser(uuid.New(), domain.RoleCarService), handler.Create)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/car-service-applications", bytes.NewReader(mustJSON(t, map[string]any{
		"organization_name": "Detail Lab",
		"city":              "Москва",
		"address":           "Тестовая улица, 1",
		"phone":             "+79990000000",
		"description":       "Кузовной ремонт",
	})))
	request.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.Contains(t, recorder.Body.String(), "invalid_input")
}

func TestCarServiceApplicationHandlerAdminReject(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	repo := newHandlerCarServiceApplicationRepo()
	applicationID := uuid.New()
	repo.items = append(repo.items, &domain.CarServiceApplication{
		ID:               applicationID,
		UserID:           uuid.New(),
		OrganizationName: "Detail Lab",
		City:             "Москва",
		Address:          "Тестовая улица, 1",
		Description:      "Кузовной ремонт",
		Status:           domain.CarServiceApplicationStatusPending,
		CreatedAt:        time.Now().UTC(),
		UpdatedAt:        time.Now().UTC(),
	})
	handler := NewCarServiceApplicationHandler(service.NewCarServiceApplicationService(nil, repo, nil, nil))
	adminID := uuid.New()

	router := gin.New()
	router.POST("/v1/admin/car-service-applications/:id/reject", withTestUser(adminID, domain.RoleAdmin), handler.AdminReject)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodPost,
		"/v1/admin/car-service-applications/"+applicationID.String()+"/reject",
		bytes.NewReader([]byte(`{"rejection_reason":"Недостаточно данных"}`)),
	)
	request.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), `"status":"rejected"`)
	require.Contains(t, recorder.Body.String(), `"rejection_reason":"Недостаточно данных"`)
	require.Equal(t, domain.CarServiceApplicationStatusRejected, repo.items[0].Status)
	require.Equal(t, adminID, *repo.items[0].ReviewedBy)
	require.Equal(t, "Недостаточно данных", *repo.items[0].RejectionReason)
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()

	data, err := json.Marshal(value)
	require.NoError(t, err)
	return data
}

type handlerCarServiceApplicationRepo struct {
	items []*domain.CarServiceApplication
}

func newHandlerCarServiceApplicationRepo() *handlerCarServiceApplicationRepo {
	return &handlerCarServiceApplicationRepo{items: make([]*domain.CarServiceApplication, 0)}
}

func (r *handlerCarServiceApplicationRepo) Create(_ context.Context, application *domain.CarServiceApplication) error {
	r.items = append(r.items, application)
	return nil
}

func (r *handlerCarServiceApplicationRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.CarServiceApplication, error) {
	for _, item := range r.items {
		if item.ID == id {
			return item, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (r *handlerCarServiceApplicationRepo) GetPendingByUserID(
	_ context.Context,
	userID uuid.UUID,
) (*domain.CarServiceApplication, error) {
	for _, item := range r.items {
		if item.UserID == userID && item.Status == domain.CarServiceApplicationStatusPending {
			return item, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (r *handlerCarServiceApplicationRepo) ListByUserID(
	_ context.Context,
	userID uuid.UUID,
	_, _ int,
) ([]*domain.CarServiceApplication, error) {
	out := make([]*domain.CarServiceApplication, 0)
	for _, item := range r.items {
		if item.UserID == userID {
			out = append(out, item)
		}
	}
	return out, nil
}

func (r *handlerCarServiceApplicationRepo) ListForAdmin(
	_ context.Context,
	status *domain.CarServiceApplicationStatus,
	_, _ int,
) ([]*domain.CarServiceApplication, error) {
	out := make([]*domain.CarServiceApplication, 0, len(r.items))
	for _, item := range r.items {
		if status == nil || item.Status == *status {
			out = append(out, item)
		}
	}
	return out, nil
}

func (r *handlerCarServiceApplicationRepo) Approve(
	_ context.Context,
	input domain.ApproveCarServiceApplicationInput,
) error {
	item, err := r.GetByID(context.Background(), input.ID)
	if err != nil {
		return err
	}
	item.Status = domain.CarServiceApplicationStatusApproved
	item.ReviewedBy = &input.ReviewedBy
	item.CreatedProfileID = input.CreatedProfileID
	return nil
}

func (r *handlerCarServiceApplicationRepo) Reject(
	_ context.Context,
	input domain.RejectCarServiceApplicationInput,
) error {
	item, err := r.GetByID(context.Background(), input.ID)
	if err != nil {
		return err
	}
	reason := input.RejectionReason
	item.Status = domain.CarServiceApplicationStatusRejected
	item.ReviewedBy = &input.ReviewedBy
	item.RejectionReason = &reason
	now := time.Now().UTC()
	item.ReviewedAt = &now
	return nil
}
