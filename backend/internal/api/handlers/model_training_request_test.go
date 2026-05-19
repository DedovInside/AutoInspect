package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DedovInside/AutoInspect/backend/internal/api/middleware"
	"github.com/DedovInside/AutoInspect/backend/internal/domain"
	"github.com/DedovInside/AutoInspect/backend/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestModelTrainingRequestHandlerCreate(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	repo := newHandlerModelTrainingRequestRepo()
	handler := NewModelTrainingRequestHandler(service.NewModelTrainingRequestService(repo, nil))
	userID := uuid.New()

	router := gin.New()
	router.POST("/v1/model-training-requests", withTestUser(userID, domain.RoleUser), handler.Create)

	body := map[string]any{
		"make":        "Volkswagen",
		"model":       "Golf",
		"generation":  "5",
		"year_from":   2008,
		"year_to":     2013,
		"description": "Нужна модель под Golf 5",
	}
	payload, err := json.Marshal(body)
	require.NoError(t, err)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/model-training-requests", bytes.NewReader(payload))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", "training-handler-key")

	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusCreated, recorder.Code)
	require.Contains(t, recorder.Body.String(), `"status":"pending"`)
	require.Contains(t, recorder.Body.String(), `"make":"Volkswagen"`)
	require.Len(t, repo.items, 1)
	require.Equal(t, userID, repo.items[0].InitiatorUserID)
	require.Equal(t, domain.RoleUser, repo.items[0].InitiatorRole)
	require.Equal(t, "training-handler-key", *repo.items[0].IdempotencyKey)
}

func TestModelTrainingRequestHandlerCreateRejectsMissingUserContext(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	handler := NewModelTrainingRequestHandler(service.NewModelTrainingRequestService(newHandlerModelTrainingRequestRepo(), nil))
	router := gin.New()
	router.POST("/v1/model-training-requests", handler.Create)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/model-training-requests", bytes.NewReader([]byte(`{}`)))
	request.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusUnauthorized, recorder.Code)
	require.Contains(t, recorder.Body.String(), "missing user context")
}

func TestModelTrainingRequestHandlerAdminUpdateStatus(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	repo := newHandlerModelTrainingRequestRepo()
	requestID := uuid.New()
	repo.items = append(repo.items, &domain.ModelTrainingRequest{
		ID:     requestID,
		Status: domain.ModelTrainingRequestStatusPending,
	})
	handler := NewModelTrainingRequestHandler(service.NewModelTrainingRequestService(repo, nil))
	adminID := uuid.New()

	router := gin.New()
	router.PATCH("/v1/admin/model-training-requests/:id/status", withTestUser(adminID, domain.RoleAdmin), handler.AdminUpdateStatus)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodPatch,
		"/v1/admin/model-training-requests/"+requestID.String()+"/status",
		bytes.NewReader([]byte(`{"status":"approved","admin_comment":"Берём в работу"}`)),
	)
	request.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), `"status":"approved"`)
	require.Contains(t, recorder.Body.String(), `"admin_comment":"Берём в работу"`)
	require.Equal(t, domain.ModelTrainingRequestStatusApproved, repo.items[0].Status)
	require.Equal(t, adminID, *repo.items[0].ReviewedBy)
}

func withTestUser(userID uuid.UUID, role domain.Role) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := context.WithValue(c.Request.Context(), middleware.UserIDContextKey, userID)
		ctx = context.WithValue(ctx, middleware.UserRoleContextKey, role)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

type handlerModelTrainingRequestRepo struct {
	items []*domain.ModelTrainingRequest
}

func newHandlerModelTrainingRequestRepo() *handlerModelTrainingRequestRepo {
	return &handlerModelTrainingRequestRepo{items: make([]*domain.ModelTrainingRequest, 0)}
}

func (r *handlerModelTrainingRequestRepo) Create(_ context.Context, request *domain.ModelTrainingRequest) error {
	r.items = append(r.items, request)
	return nil
}

func (r *handlerModelTrainingRequestRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.ModelTrainingRequest, error) {
	for _, item := range r.items {
		if item.ID == id {
			return item, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (r *handlerModelTrainingRequestRepo) GetByUserAndIdempotencyKey(
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

func (r *handlerModelTrainingRequestRepo) ListByUserID(
	_ context.Context,
	userID uuid.UUID,
	_, _ int,
) ([]*domain.ModelTrainingRequest, error) {
	out := make([]*domain.ModelTrainingRequest, 0)
	for _, item := range r.items {
		if item.InitiatorUserID == userID {
			out = append(out, item)
		}
	}
	return out, nil
}

func (r *handlerModelTrainingRequestRepo) ListForAdmin(
	_ context.Context,
	status *domain.ModelTrainingRequestStatus,
	_, _ int,
) ([]*domain.ModelTrainingRequest, error) {
	out := make([]*domain.ModelTrainingRequest, 0, len(r.items))
	for _, item := range r.items {
		if status == nil || item.Status == *status {
			out = append(out, item)
		}
	}
	return out, nil
}

func (r *handlerModelTrainingRequestRepo) CountActiveByUserID(context.Context, uuid.UUID) (int, error) {
	return 0, nil
}

func (r *handlerModelTrainingRequestRepo) UpdateStatus(
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
