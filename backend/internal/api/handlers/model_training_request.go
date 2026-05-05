package handlers

import (
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
	userID, ok := userIDOrAbort(c)
	if !ok {
		return
	}

	var query dto.ListModelTrainingRequestsQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		writeError(c, http.StatusBadRequest, "invalid_query", err.Error())
		return
	}

	requests, err := h.service.ListMine(c.Request.Context(), userID, query.Limit, query.Offset)
	if err != nil {
		handleModelTrainingRequestError(c, err)
		return
	}

	items := dto.ToModelTrainingRequestResponseList(requests)
	writeJSON(c, http.StatusOK, dto.NewModelTrainingRequestListResponse(items, query.Limit, query.Offset))
}

func (h *ModelTrainingRequestHandler) AdminList(c *gin.Context) {
	var query dto.AdminListModelTrainingRequestsQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		writeError(c, http.StatusBadRequest, "invalid_query", err.Error())
		return
	}

	var status *domain.ModelTrainingRequestStatus
	if query.Status != "" {
		s := domain.ModelTrainingRequestStatus(query.Status)
		status = &s
	}

	requests, err := h.service.ListForAdmin(c.Request.Context(), status, query.Limit, query.Offset)
	if err != nil {
		handleModelTrainingRequestError(c, err)
		return
	}

	items := dto.ToModelTrainingRequestResponseList(requests)
	writeJSON(c, http.StatusOK, dto.NewModelTrainingRequestListResponse(items, query.Limit, query.Offset))
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
