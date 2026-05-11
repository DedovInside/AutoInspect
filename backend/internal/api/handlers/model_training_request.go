package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/DedovInside/AutoInspect/backend/internal/api/dto"
	"github.com/DedovInside/AutoInspect/backend/internal/api/middleware"
	"github.com/DedovInside/AutoInspect/backend/internal/domain"
	"github.com/DedovInside/AutoInspect/backend/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ModelTrainingRequestHandler struct {
	service *service.ModelTrainingRequestService
}

func NewModelTrainingRequestHandler(svc *service.ModelTrainingRequestService) *ModelTrainingRequestHandler {
	return &ModelTrainingRequestHandler{service: svc}
}

func (h *ModelTrainingRequestHandler) Create(c *gin.Context) {
	userID, ok := middleware.UserIDFromContext(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "unauthorized", "missing user context")
		return
	}

	role, ok := middleware.UserRoleFromContext(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "unauthorized", "missing user role")
		return
	}

	var req dto.CreateModelTrainingRequestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	idempotencyKey := optionalHeader(c.Request, "Idempotency-Key")
	request, err := h.service.Create(c.Request.Context(), &domain.CreateModelTrainingRequestInput{
		InitiatorUserID: userID,
		InitiatorRole:   role,
		Make:            req.Make,
		Model:           req.Model,
		Generation:      req.Generation,
		YearFrom:        req.YearFrom,
		YearTo:          req.YearTo,
		Description:     req.Description,
		IdempotencyKey:  idempotencyKey,
	})
	if err != nil {
		handleModelTrainingRequestError(c, err)
		return
	}

	writeJSON(c, http.StatusCreated, dto.CreateModelTrainingRequestResponse{
		Request: dto.ToModelTrainingRequestResponse(request),
	})
}

func (h *ModelTrainingRequestHandler) ListMine(c *gin.Context) {
	handleUserQueryList(c, h.listMine, writeModelTrainingRequestList, handleModelTrainingRequestError)
}

func (h *ModelTrainingRequestHandler) AdminList(c *gin.Context) {
	handleQueryList(c, h.adminList, writeAdminModelTrainingRequestList, handleModelTrainingRequestError)
}

func (h *ModelTrainingRequestHandler) AdminUpdateStatus(c *gin.Context) {
	adminID, ok := middleware.UserIDFromContext(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "unauthorized", "missing user context")
		return
	}

	requestID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid_id", "Request ID must be a valid UUID")
		return
	}

	var req dto.UpdateModelTrainingRequestStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	err = h.service.UpdateStatus(c.Request.Context(), &domain.UpdateModelTrainingRequestStatusInput{
		ID:             requestID,
		Status:         req.Status,
		AdminComment:   req.AdminComment,
		ReviewedBy:     adminID,
		CreatedModelID: req.CreatedModelID,
	})
	if err != nil {
		handleModelTrainingRequestError(c, err)
		return
	}

	writeJSON(c, http.StatusOK, gin.H{"status": "ok"})
}

func handleModelTrainingRequestError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidInput):
		writeError(c, http.StatusBadRequest, "invalid_input", err.Error())
	case errors.Is(err, domain.ErrAlreadyExists):
		writeError(c, http.StatusConflict, "already_exists", err.Error())
	case errors.Is(err, domain.ErrTrainingRequestLimitExceeded):
		writeError(c, http.StatusConflict, "training_request_limit_exceeded", err.Error())
	case errors.Is(err, domain.ErrNotFound):
		writeError(c, http.StatusNotFound, "not_found", err.Error())
	default:
		writeError(c, http.StatusInternalServerError, "internal_error", "Failed to process model training request")
	}
}

func (h *ModelTrainingRequestHandler) listMine(
	ctx context.Context,
	userID uuid.UUID,
	query dto.ListModelTrainingRequestsQuery,
) ([]*domain.ModelTrainingRequest, error) {
	return h.service.ListMine(ctx, userID, query.Limit, query.Offset)
}

func (h *ModelTrainingRequestHandler) adminList(
	ctx context.Context,
	query dto.AdminListModelTrainingRequestsQuery,
) ([]*domain.ModelTrainingRequest, error) {
	var status *domain.ModelTrainingRequestStatus
	if query.Status != "" {
		s := domain.ModelTrainingRequestStatus(query.Status)
		status = &s
	}

	return h.service.ListForAdmin(ctx, status, query.Limit, query.Offset)
}

func writeModelTrainingRequestList(
	c *gin.Context,
	requests []*domain.ModelTrainingRequest,
	query dto.ListModelTrainingRequestsQuery,
) {
	items := dto.ToModelTrainingRequestResponseList(requests)
	writeJSON(c, http.StatusOK, dto.NewModelTrainingRequestListResponse(items, query.Limit, query.Offset))
}

func writeAdminModelTrainingRequestList(
	c *gin.Context,
	requests []*domain.ModelTrainingRequest,
	query dto.AdminListModelTrainingRequestsQuery,
) {
	items := dto.ToModelTrainingRequestResponseList(requests)
	writeJSON(c, http.StatusOK, dto.NewModelTrainingRequestListResponse(items, query.Limit, query.Offset))
}
