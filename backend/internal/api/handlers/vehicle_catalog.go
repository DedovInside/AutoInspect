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

type VehicleCatalogHandler struct {
	service *service.VehicleCatalogService
}

func NewVehicleCatalogHandler(svc *service.VehicleCatalogService) *VehicleCatalogHandler {
	return &VehicleCatalogHandler{service: svc}
}

func (h *VehicleCatalogHandler) ListMakes(c *gin.Context) {
	makes, err := h.service.ListMakes(c.Request.Context())
	if err != nil {
		handleVehicleCatalogError(c, err)
		return
	}

	items := dto.ToVehicleMakeResponseList(makes)
	writeJSON(c, http.StatusOK, dto.NewVehicleMakeListResponse(items))
}

func (h *VehicleCatalogHandler) ListModels(c *gin.Context) {
	makeID, err := uuid.Parse(c.Param("make_id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid_make_id", "Make ID must be a valid UUID")
		return
	}

	models, err := h.service.ListModels(c.Request.Context(), makeID)
	if err != nil {
		handleVehicleCatalogError(c, err)
		return
	}

	items := dto.ToVehicleModelResponseList(models)
	writeJSON(c, http.StatusOK, dto.NewVehicleModelListResponse(items))
}

func (h *VehicleCatalogHandler) ListGenerations(c *gin.Context) {
	modelID, err := uuid.Parse(c.Param("model_id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid_model_id", "Model ID must be a valid UUID")
		return
	}

	generations, err := h.service.ListGenerations(c.Request.Context(), modelID)
	if err != nil {
		handleVehicleCatalogError(c, err)
		return
	}

	items := make([]dto.VehicleGenerationResponse, 0, len(generations))
	for _, generation := range generations {
		if generation == nil {
			continue
		}
		items = append(items, dto.ToVehicleGenerationResponse(generation, h.service.YearOptions(generation)))
	}

	writeJSON(c, http.StatusOK, dto.NewVehicleGenerationListResponse(items))
}

func (h *VehicleCatalogHandler) AdminListMakes(c *gin.Context) {
	makes, err := h.service.AdminListMakes(c.Request.Context())
	if err != nil {
		handleVehicleCatalogError(c, err)
		return
	}

	items := dto.ToAdminVehicleMakeResponseList(makes)
	writeJSON(c, http.StatusOK, dto.NewAdminVehicleMakeListResponse(items))
}

func (h *VehicleCatalogHandler) AdminCreateMake(c *gin.Context) {
	req, ok := bindJSONOrAbort[dto.CreateVehicleMakeRequest](c)
	if !ok {
		return
	}

	vehicleMake, err := h.service.CreateMake(c.Request.Context(), &domain.CreateVehicleMakeInput{
		Name: req.Name,
		Slug: req.Slug,
	})
	if err != nil {
		handleVehicleCatalogError(c, err)
		return
	}

	writeJSON(c, http.StatusCreated, dto.ToAdminVehicleMakeResponse(vehicleMake))
}

func (h *VehicleCatalogHandler) AdminUpdateMake(c *gin.Context) {
	id, ok := uuidParamOrAbort(c, "id", "invalid_make_id", "Make ID must be a valid UUID")
	if !ok {
		return
	}

	req, ok := bindJSONOrAbort[dto.UpdateVehicleMakeRequest](c)
	if !ok {
		return
	}

	vehicleMake, err := h.service.UpdateMake(c.Request.Context(), &domain.UpdateVehicleMakeInput{
		ID:   id,
		Name: req.Name,
		Slug: req.Slug,
	})
	if err != nil {
		handleVehicleCatalogError(c, err)
		return
	}

	writeJSON(c, http.StatusOK, dto.ToAdminVehicleMakeResponse(vehicleMake))
}

func (h *VehicleCatalogHandler) AdminSetMakeActive(c *gin.Context) {
	id, ok := uuidParamOrAbort(c, "id", "invalid_make_id", "Make ID must be a valid UUID")
	if !ok {
		return
	}

	req, ok := bindJSONOrAbort[dto.SetVehicleCatalogActiveRequest](c)
	if !ok {
		return
	}

	err := h.service.SetMakeActive(c.Request.Context(), domain.SetVehicleMakeActiveInput{
		ID:       id,
		IsActive: req.IsActive,
	})
	if err != nil {
		handleVehicleCatalogError(c, err)
		return
	}

	writeJSON(c, http.StatusOK, gin.H{"status": "ok"})
}

func (h *VehicleCatalogHandler) AdminListModels(c *gin.Context) {
	makeID, ok := uuidParamOrAbort(c, "make_id", "invalid_make_id", "Make ID must be a valid UUID")
	if !ok {
		return
	}

	models, err := h.service.AdminListModels(c.Request.Context(), makeID)
	if err != nil {
		handleVehicleCatalogError(c, err)
		return
	}

	items := dto.ToAdminVehicleModelResponseList(models)
	writeJSON(c, http.StatusOK, dto.NewAdminVehicleModelListResponse(items))
}

func (h *VehicleCatalogHandler) AdminCreateModel(c *gin.Context) {
	req, ok := bindJSONOrAbort[dto.CreateVehicleModelRequest](c)
	if !ok {
		return
	}

	model, err := h.service.CreateModel(c.Request.Context(), &domain.CreateVehicleModelInput{
		MakeID: req.MakeID,
		Name:   req.Name,
		Slug:   req.Slug,
	})
	if err != nil {
		handleVehicleCatalogError(c, err)
		return
	}

	writeJSON(c, http.StatusCreated, dto.ToAdminVehicleModelResponse(model))
}

func (h *VehicleCatalogHandler) AdminUpdateModel(c *gin.Context) {
	id, ok := uuidParamOrAbort(c, "id", "invalid_model_id", "Model ID must be a valid UUID")
	if !ok {
		return
	}

	req, ok := bindJSONOrAbort[dto.UpdateVehicleModelRequest](c)
	if !ok {
		return
	}

	model, err := h.service.UpdateModel(c.Request.Context(), &domain.UpdateVehicleModelInput{
		ID:     id,
		MakeID: req.MakeID,
		Name:   req.Name,
		Slug:   req.Slug,
	})
	if err != nil {
		handleVehicleCatalogError(c, err)
		return
	}

	writeJSON(c, http.StatusOK, dto.ToAdminVehicleModelResponse(model))
}

func (h *VehicleCatalogHandler) AdminSetModelActive(c *gin.Context) {
	id, ok := uuidParamOrAbort(c, "id", "invalid_model_id", "Model ID must be a valid UUID")
	if !ok {
		return
	}

	req, ok := bindJSONOrAbort[dto.SetVehicleCatalogActiveRequest](c)
	if !ok {
		return
	}

	err := h.service.SetModelActive(c.Request.Context(), domain.SetVehicleModelActiveInput{
		ID:       id,
		IsActive: req.IsActive,
	})
	if err != nil {
		handleVehicleCatalogError(c, err)
		return
	}

	writeJSON(c, http.StatusOK, gin.H{"status": "ok"})
}

func (h *VehicleCatalogHandler) AdminListGenerations(c *gin.Context) {
	modelID, ok := uuidParamOrAbort(c, "model_id", "invalid_model_id", "Model ID must be a valid UUID")
	if !ok {
		return
	}

	generations, err := h.service.AdminListGenerations(c.Request.Context(), modelID)
	if err != nil {
		handleVehicleCatalogError(c, err)
		return
	}

	items := dto.ToAdminVehicleGenerationResponseList(generations)
	writeJSON(c, http.StatusOK, dto.NewAdminVehicleGenerationListResponse(items))
}

func (h *VehicleCatalogHandler) AdminCreateGeneration(c *gin.Context) {
	req, ok := bindJSONOrAbort[dto.CreateVehicleGenerationRequest](c)
	if !ok {
		return
	}

	generation, err := h.service.CreateGeneration(c.Request.Context(), &domain.CreateVehicleGenerationInput{
		ModelID:  req.ModelID,
		Name:     req.Name,
		Slug:     req.Slug,
		YearFrom: req.YearFrom,
		YearTo:   req.YearTo,
	})
	if err != nil {
		handleVehicleCatalogError(c, err)
		return
	}

	writeJSON(c, http.StatusCreated, dto.ToAdminVehicleGenerationResponse(generation))
}

func (h *VehicleCatalogHandler) AdminUpdateGeneration(c *gin.Context) {
	id, ok := uuidParamOrAbort(c, "id", "invalid_generation_id", "Generation ID must be a valid UUID")
	if !ok {
		return
	}

	req, ok := bindJSONOrAbort[dto.UpdateVehicleGenerationRequest](c)
	if !ok {
		return
	}

	generation, err := h.service.UpdateGeneration(c.Request.Context(), &domain.UpdateVehicleGenerationInput{
		ID:       id,
		ModelID:  req.ModelID,
		Name:     req.Name,
		Slug:     req.Slug,
		YearFrom: req.YearFrom,
		YearTo:   req.YearTo,
	})
	if err != nil {
		handleVehicleCatalogError(c, err)
		return
	}

	writeJSON(c, http.StatusOK, dto.ToAdminVehicleGenerationResponse(generation))
}

func (h *VehicleCatalogHandler) AdminSetGenerationActive(c *gin.Context) {
	id, ok := uuidParamOrAbort(c, "id", "invalid_generation_id", "Generation ID must be a valid UUID")
	if !ok {
		return
	}

	req, ok := bindJSONOrAbort[dto.SetVehicleCatalogActiveRequest](c)
	if !ok {
		return
	}

	err := h.service.SetGenerationActive(c.Request.Context(), domain.SetVehicleGenerationActiveInput{
		ID:       id,
		IsActive: req.IsActive,
	})
	if err != nil {
		handleVehicleCatalogError(c, err)
		return
	}

	writeJSON(c, http.StatusOK, gin.H{"status": "ok"})
}

func uuidParamOrAbort(c *gin.Context, name, code, message string) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param(name))
	if err != nil {
		writeError(c, http.StatusBadRequest, code, message)
		return uuid.Nil, false
	}

	return id, true
}

func handleVehicleCatalogError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidInput):
		writeError(c, http.StatusBadRequest, "invalid_input", err.Error())
	case errors.Is(err, domain.ErrAlreadyExists):
		writeError(c, http.StatusConflict, "already_exists", err.Error())
	case errors.Is(err, domain.ErrNotFound):
		writeError(c, http.StatusNotFound, "not_found", err.Error())
	default:
		writeError(c, http.StatusInternalServerError, "internal_error", "Failed to process vehicle catalog")
	}
}
