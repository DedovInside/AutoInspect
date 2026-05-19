package handlers

import (
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

const (
	handlerModelMake       = "Volkswagen"
	handlerModelName       = "Golf"
	handlerModelGeneration = "5"
)

func TestModelHandlerList(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	repo := newHandlerCarModelRepo()
	repo.models = append(repo.models,
		handlerCarModel(handlerModelMake, handlerModelName, true, false),
		handlerCarModel("general", "general", true, true),
	)
	handler := NewModelHandler(service.NewModelService(repo, nil, nil))

	router := gin.New()
	router.GET("/v1/admin/models", handler.List)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v1/admin/models?limit=20&offset=0", http.NoBody)

	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), `"make":"Volkswagen"`)
	require.Contains(t, recorder.Body.String(), `"is_universal":true`)
}

func TestModelHandlerListAvailableSpecializedFiltersUniversalAndInactive(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	repo := newHandlerCarModelRepo()
	repo.models = append(repo.models,
		handlerCarModel(handlerModelMake, handlerModelName, true, false),
		handlerCarModel("BMW", "3", false, false),
		handlerCarModel("general", "general", true, true),
	)
	handler := NewModelHandler(service.NewModelService(repo, nil, nil))

	router := gin.New()
	router.GET("/v1/models/available", handler.ListAvailableSpecialized)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v1/models/available", http.NoBody)

	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), `"make":"Volkswagen"`)
	require.NotContains(t, recorder.Body.String(), `"make":"BMW"`)
	require.NotContains(t, recorder.Body.String(), `"make":"general"`)
}

func TestModelHandlerDeactivate(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	repo := newHandlerCarModelRepo()
	model := handlerCarModel(handlerModelMake, handlerModelName, true, false)
	repo.models = append(repo.models, model)
	repo.byID[model.ID] = model
	handler := NewModelHandler(service.NewModelService(repo, nil, nil))

	router := gin.New()
	router.PATCH("/v1/admin/models/:id/deactivate", handler.Deactivate)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodPatch,
		"/v1/admin/models/"+model.ID.String()+"/deactivate",
		http.NoBody,
	)

	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.False(t, model.IsActive)
	require.Contains(t, recorder.Body.String(), `"status":"ok"`)
}

func TestModelHandlerDeactivateRejectsInvalidID(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	handler := NewModelHandler(service.NewModelService(newHandlerCarModelRepo(), nil, nil))

	router := gin.New()
	router.PATCH("/v1/admin/models/:id/deactivate", handler.Deactivate)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPatch, "/v1/admin/models/not-a-uuid/deactivate", http.NoBody)

	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
	require.Contains(t, recorder.Body.String(), "invalid_id")
}

func handlerCarModel(makeName, modelName string, active, universal bool) *domain.CarModel {
	return &domain.CarModel{
		ID:                uuid.New(),
		Make:              makeName,
		Model:             modelName,
		Generation:        handlerModelGeneration,
		YearFrom:          2008,
		YearTo:            2013,
		PartsModelS3Key:   "models/test/parts_segmentation.pt",
		PartsConfigS3Key:  "models/test/parts_inference_config.json",
		PartsCatalogS3Key: "models/test/parts_catalog.json",
		ModelVersion:      "v1",
		IsUniversal:       universal,
		IsActive:          active,
		CreatedAt:         time.Now().UTC(),
	}
}

type handlerCarModelRepo struct {
	models []*domain.CarModel
	byID   map[uuid.UUID]*domain.CarModel
}

func newHandlerCarModelRepo() *handlerCarModelRepo {
	return &handlerCarModelRepo{
		models: make([]*domain.CarModel, 0),
		byID:   make(map[uuid.UUID]*domain.CarModel),
	}
}

func (r *handlerCarModelRepo) FindActiveModel(context.Context, string, string, string, int) (*domain.CarModel, error) {
	return nil, domain.ErrInvalidModel
}

func (r *handlerCarModelRepo) GetUniversalModel(context.Context) (*domain.CarModel, error) {
	for _, model := range r.models {
		if model.IsUniversal && model.IsActive {
			return model, nil
		}
	}
	return nil, domain.ErrInvalidModel
}

func (r *handlerCarModelRepo) CreateModel(_ context.Context, model *domain.CarModel) error {
	r.models = append(r.models, model)
	r.byID[model.ID] = model
	return nil
}

func (r *handlerCarModelRepo) ListModels(context.Context, int, int) ([]*domain.CarModel, error) {
	return append([]*domain.CarModel(nil), r.models...), nil
}

func (r *handlerCarModelRepo) GetModelByID(_ context.Context, id uuid.UUID) (*domain.CarModel, error) {
	if model, ok := r.byID[id]; ok {
		return model, nil
	}
	return nil, domain.ErrNotFound
}

func (r *handlerCarModelRepo) DeactivateModel(_ context.Context, id uuid.UUID) error {
	model, err := r.GetModelByID(context.Background(), id)
	if err != nil {
		return err
	}
	model.IsActive = false
	return nil
}
