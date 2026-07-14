package handlers

import (
	"context"
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
	handleUserQueryList(c, h.listMine, writeRepairRequestList, handleRepairRequestError)
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

func (h *RepairRequestHandler) ListIncoming(c *gin.Context) {
	handleUserQueryList(c, h.listIncoming, writeRepairRequestList, handleRepairRequestError)
}

func (h *RepairRequestHandler) GetIncomingDetails(c *gin.Context) {
	userID, requestID, ok := repairRequestIDFromRequest(c)
	if !ok {
		return
	}

	details, err := h.service.GetIncomingDetails(c.Request.Context(), userID, requestID)
	if err != nil {
		handleRepairRequestError(c, err)
		return
	}

	writeJSON(c, http.StatusOK, dto.ToRepairRequestDetailsResponse(details))
}

func (h *RepairRequestHandler) AcceptIncoming(c *gin.Context) {
	userID, requestID, ok := repairRequestIDFromRequest(c)
	if !ok {
		return
	}

	req, ok := bindJSONOrAbort[dto.AcceptRepairRequestRequest](c)
	if !ok {
		return
	}

	request, err := h.service.AcceptIncoming(c.Request.Context(), &domain.AcceptRepairRequestInput{
		ID:                requestID,
		CarServiceUserID:  userID,
		ServiceComment:    req.ServiceComment,
		ServiceEstimate:   dto.ToRepairEstimateInputList(req.ServiceEstimate),
		EstimatedPriceMin: req.EstimatedPriceMin,
		EstimatedPriceMax: req.EstimatedPriceMax,
	})
	if err != nil {
		handleRepairRequestError(c, err)
		return
	}

	writeJSON(c, http.StatusOK, dto.RespondRepairRequestResponse{
		Request: dto.ToRepairRequestResponse(request),
	})
}

func (h *RepairRequestHandler) RejectIncoming(c *gin.Context) {
	userID, requestID, ok := repairRequestIDFromRequest(c)
	if !ok {
		return
	}

	req, ok := bindJSONOrAbort[dto.RejectRepairRequestRequest](c)
	if !ok {
		return
	}

	request, err := h.service.RejectIncoming(c.Request.Context(), &domain.RejectRepairRequestInput{
		ID:               requestID,
		CarServiceUserID: userID,
		ServiceComment:   req.ServiceComment,
	})
	if err != nil {
		handleRepairRequestError(c, err)
		return
	}

	writeJSON(c, http.StatusOK, dto.RespondRepairRequestResponse{
		Request: dto.ToRepairRequestResponse(request),
	})
}

func (h *RepairRequestHandler) CompleteIncoming(c *gin.Context) {
	userID, requestID, ok := repairRequestIDFromRequest(c)
	if !ok {
		return
	}

	request, err := h.service.CompleteIncoming(c.Request.Context(), &domain.CompleteRepairRequestInput{
		ID:               requestID,
		CarServiceUserID: userID,
	})
	if err != nil {
		handleRepairRequestError(c, err)
		return
	}

	writeJSON(c, http.StatusOK, dto.RespondRepairRequestResponse{
		Request: dto.ToRepairRequestResponse(request),
	})
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

func (h *RepairRequestHandler) listMine(
	ctx context.Context,
	userID uuid.UUID,
	query dto.ListRepairRequestsQuery,
) ([]*domain.RepairRequest, error) {
	return h.service.ListMine(ctx, userID, query.Limit, query.Offset)
}

func (h *RepairRequestHandler) listIncoming(
	ctx context.Context,
	userID uuid.UUID,
	query dto.ListRepairRequestsQuery,
) ([]*domain.RepairRequest, error) {
	return h.service.ListIncoming(ctx, userID, query.Limit, query.Offset)
}

func writeRepairRequestList(
	c *gin.Context,
	requests []*domain.RepairRequest,
	query dto.ListRepairRequestsQuery,
) {
	items := dto.ToRepairRequestResponseList(requests)
	writeJSON(c, http.StatusOK, dto.NewRepairRequestListResponse(items, query.Limit, query.Offset))
}
