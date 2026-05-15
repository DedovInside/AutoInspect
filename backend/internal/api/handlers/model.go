package handlers

import (
	"errors"
	"net/http"

	"github.com/DedovInside/AutoInspect/backend/internal/api/dto"
	"github.com/DedovInside/AutoInspect/backend/internal/domain"
	"github.com/DedovInside/AutoInspect/backend/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	partsModelFormField   = "parts_model_file"
	partsConfigFormField  = "parts_config_file"
	partsCatalogFormField = "parts_catalog_file"
)

type ModelHandler struct {
	service *service.ModelService
}

func NewModelHandler(svc *service.ModelService) *ModelHandler {
	return &ModelHandler{service: svc}
}

func (h *ModelHandler) Upload(c *gin.Context) {
	var req dto.UploadModelArtifactsRequest
	if err := c.ShouldBind(&req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	partsModel, err := openModelArtifact(c, partsModelFormField)
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid_parts_model", err.Error())
		return
	}
	defer closeModelArtifact(partsModel)

	partsConfig, err := openModelArtifact(c, partsConfigFormField)
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid_parts_config", err.Error())
		return
	}
	defer closeModelArtifact(partsConfig)

	partsCatalog, err := openModelArtifact(c, partsCatalogFormField)
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid_parts_catalog", err.Error())
		return
	}
	defer closeModelArtifact(partsCatalog)

	model, err := h.service.UploadModelArtifacts(c.Request.Context(), &service.UploadModelArtifactsInput{
		IdempotencyKey: c.GetHeader("Idempotency-Key"),
		Make:           req.Make,
		Model:          req.Model,
		Generation:     req.Generation,
		YearFrom:       req.YearFrom,
		YearTo:         req.YearTo,
		Version:        req.Version,
		IsUniversal:    req.IsUniversal,
		PartsModel:     partsModel,
		PartsConfig:    partsConfig,
		PartsCatalog:   partsCatalog,
	})
	if err != nil {
		handleModelError(c, err)
		return
	}

	writeJSON(c, http.StatusCreated, dto.UploadModelArtifactsResponse{
		Model: dto.ToCarModelResponse(model),
	})
}

func (h *ModelHandler) List(c *gin.Context) {
	var query dto.ListModelsQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		writeError(c, http.StatusBadRequest, "invalid_query", err.Error())
		return
	}

	models, err := h.service.ListModels(c.Request.Context(), query.Limit, query.Offset)
	if err != nil {
		handleModelError(c, err)
		return
	}

	items := dto.ToCarModelResponseList(models)
	writeJSON(c, http.StatusOK, dto.NewModelListResponse(items, query.Limit, query.Offset))
}

func (h *ModelHandler) ListAvailableSpecialized(c *gin.Context) {
	models, err := h.service.ListActiveSpecializedModels(c.Request.Context())
	if err != nil {
		handleModelError(c, err)
		return
	}

	items := dto.ToPublicCarModelResponseList(models)
	writeJSON(c, http.StatusOK, dto.NewPublicModelListResponse(items))
}

func (h *ModelHandler) Deactivate(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid_id", "Model ID must be a valid UUID")
		return
	}

	if err := h.service.DeactivateModel(c.Request.Context(), id); err != nil {
		handleModelError(c, err)
		return
	}

	writeJSON(c, http.StatusOK, gin.H{"status": "ok"})
}

func openModelArtifact(c *gin.Context, field string) (service.ModelArtifactFile, error) {
	header, err := c.FormFile(field)
	if err != nil {
		return service.ModelArtifactFile{}, err
	}

	file, err := header.Open()
	if err != nil {
		return service.ModelArtifactFile{}, err
	}

	return service.ModelArtifactFile{
		File:     file,
		Filename: header.Filename,
	}, nil
}

func closeModelArtifact(file service.ModelArtifactFile) {
	if file.File != nil {
		_ = file.File.Close()
	}
}

func handleModelError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidInput):
		writeError(c, http.StatusBadRequest, "invalid_input", err.Error())
	case errors.Is(err, domain.ErrAlreadyExists):
		writeError(c, http.StatusConflict, "already_exists", err.Error())
	case errors.Is(err, domain.ErrNotFound):
		writeError(c, http.StatusNotFound, "not_found", err.Error())
	default:
		writeError(c, http.StatusInternalServerError, "internal_error", "Failed to process model artifacts")
	}
}
