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

type RepairRequestHandler struct {
	service *service.RepairRequestService
}

func NewRepairRequestHandler(svc *service.RepairRequestService) *RepairRequestHandler {
	return &RepairRequestHandler{service: svc}
}

func (h *RepairRequestHandler) Create(c *gin.Context) {
	userID, ok := userIDOrAbort(c)
	if !ok {
		return
	}

	var req dto.CreateRepairRequestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	request, err := h.service.Create(c.Request.Context(), &domain.CreateRepairRequestInput{
		UserID:              userID,
		AnalysisJobID:       req.AnalysisJobID,
		CarServiceProfileID: req.CarServiceProfileID,
		CustomerName:        req.CustomerName,
		CustomerPhone:       req.CustomerPhone,
		CustomerEmail:       req.CustomerEmail,
		CustomerComment:     req.CustomerComment,
	})
	if err != nil {
		handleRepairRequestError(c, err)
		return
	}

	writeJSON(c, http.StatusCreated, dto.CreateRepairRequestResponse{
		Request: dto.ToRepairRequestResponse(request),
	})
}

func (h *RepairRequestHandler) GetMine(c *gin.Context) {
	userID, requestID, ok := repairRequestIDFromRequest(c)
	if !ok {
		return
	}

	request, err := h.service.GetMine(c.Request.Context(), userID, requestID)
	if err != nil {
		handleRepairRequestError(c, err)
		return
	}

	writeJSON(c, http.StatusOK, dto.GetRepairRequestResponse{
		Request: dto.ToRepairRequestResponse(request),
	})
}

func (h *RepairRequestHandler) ListMine(c *gin.Context) {
	userID, ok := userIDOrAbort(c)
	if !ok {
		return
	}

	var query dto.ListRepairRequestsQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		writeError(c, http.StatusBadRequest, "invalid_query", err.Error())
		return
	}

	requests, err := h.service.ListMine(c.Request.Context(), userID, query.Limit, query.Offset)
	if err != nil {
		handleRepairRequestError(c, err)
		return
	}

	items := dto.ToRepairRequestResponseList(requests)
	writeJSON(c, http.StatusOK, dto.NewRepairRequestListResponse(items, query.Limit, query.Offset))
}

func (h *RepairRequestHandler) CancelMine(c *gin.Context) {
	userID, requestID, ok := repairRequestIDFromRequest(c)
	if !ok {
		return
	}

	if err := h.service.CancelMine(c.Request.Context(), userID, requestID); err != nil {
		handleRepairRequestError(c, err)
		return
	}

	writeJSON(c, http.StatusOK, dto.CancelRepairRequestResponse{Status: string(domain.RepairRequestStatusCanceled)})
}

func repairRequestIDFromRequest(c *gin.Context) (userID, requestID uuid.UUID, ok bool) {
	userID, ok = userIDOrAbort(c)
	if !ok {
		return uuid.Nil, uuid.Nil, false
	}

	requestID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid_id", "Repair request ID must be a valid UUID")
		return uuid.Nil, uuid.Nil, false
	}

	return userID, requestID, true
}

func handleRepairRequestError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidInput):
		writeError(c, http.StatusBadRequest, "invalid_input", err.Error())
	case errors.Is(err, domain.ErrAlreadyExists):
		writeError(c, http.StatusConflict, "already_exists", err.Error())
	case errors.Is(err, domain.ErrForbidden):
		writeError(c, http.StatusForbidden, "forbidden", err.Error())
	case errors.Is(err, domain.ErrJobNotFound), errors.Is(err, domain.ErrNotFound):
		writeError(c, http.StatusNotFound, "not_found", err.Error())
	case errors.Is(err, domain.ErrJobNotReady):
		writeError(c, http.StatusConflict, "job_not_ready", err.Error())
	case errors.Is(err, domain.ErrJobFailed):
		writeError(c, http.StatusConflict, "job_failed", err.Error())
	default:
		writeError(c, http.StatusInternalServerError, "internal_error", "Failed to process repair request")
	}
}
