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

type CarServiceApplicationHandler struct {
	service *service.CarServiceApplicationService
}

func NewCarServiceApplicationHandler(svc *service.CarServiceApplicationService) *CarServiceApplicationHandler {
	return &CarServiceApplicationHandler{service: svc}
}

func (h *CarServiceApplicationHandler) Create(c *gin.Context) {
	userID, ok := userIDOrAbort(c)
	if !ok {
		return
	}

	role, ok := middleware.UserRoleFromContext(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "unauthorized", "missing user role")
		return
	}

	var req dto.CreateCarServiceApplicationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	application, err := h.service.Create(c.Request.Context(), &domain.CreateCarServiceApplicationInput{
		UserID:           userID,
		UserRole:         role,
		OrganizationName: req.OrganizationName,
		City:             req.City,
		Address:          req.Address,
		Phone:            req.Phone,
		Email:            req.Email,
		ContactInfo:      req.ContactInfo,
		Description:      req.Description,
	})
	if err != nil {
		handleCarServiceApplicationError(c, err)
		return
	}

	writeJSON(c, http.StatusCreated, dto.CreateCarServiceApplicationResponse{
		Application: dto.ToCarServiceApplicationResponse(application),
	})
}

func (h *CarServiceApplicationHandler) GetCurrent(c *gin.Context) {
	userID, ok := userIDOrAbort(c)
	if !ok {
		return
	}

	application, err := h.service.GetMyPending(c.Request.Context(), userID)
	if err != nil {
		handleCarServiceApplicationError(c, err)
		return
	}

	writeJSON(c, http.StatusOK, dto.GetCarServiceApplicationResponse{
		Application: dto.ToCarServiceApplicationResponse(application),
	})
}

func (h *CarServiceApplicationHandler) ListMine(c *gin.Context) {
	handleUserQueryList(c, h.listMine, writeCarServiceApplicationList, handleCarServiceApplicationError)
}

func (h *CarServiceApplicationHandler) AdminList(c *gin.Context) {
	handleQueryList(c, h.adminList, writeAdminCarServiceApplicationList, handleCarServiceApplicationError)
}

func (h *CarServiceApplicationHandler) AdminApprove(c *gin.Context) {
	adminID, ok := userIDOrAbort(c)
	if !ok {
		return
	}

	applicationID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid_id", "Application ID must be a valid UUID")
		return
	}

	result, err := h.service.Approve(c.Request.Context(), &domain.ApproveCarServiceApplicationInput{
		ID:         applicationID,
		ReviewedBy: adminID,
	})
	if err != nil {
		handleCarServiceApplicationError(c, err)
		return
	}

	writeJSON(c, http.StatusOK, dto.ApproveCarServiceApplicationResponse{
		Application: dto.ToCarServiceApplicationResponse(result.Application),
		Profile:     dto.ToCarServiceProfileResponse(result.Profile),
	})
}

func (h *CarServiceApplicationHandler) AdminReject(c *gin.Context) {
	adminID, ok := userIDOrAbort(c)
	if !ok {
		return
	}

	applicationID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid_id", "Application ID must be a valid UUID")
		return
	}

	req, ok := bindJSONOrAbort[dto.RejectCarServiceApplicationRequest](c)
	if !ok {
		return
	}

	application, err := h.service.Reject(c.Request.Context(), &domain.RejectCarServiceApplicationInput{
		ID:              applicationID,
		ReviewedBy:      adminID,
		RejectionReason: req.RejectionReason,
	})
	if err != nil {
		handleCarServiceApplicationError(c, err)
		return
	}

	writeJSON(c, http.StatusOK, dto.RejectCarServiceApplicationResponse{
		Application: dto.ToCarServiceApplicationResponse(application),
	})
}

func handleCarServiceApplicationError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidInput):
		writeError(c, http.StatusBadRequest, "invalid_input", err.Error())
	case errors.Is(err, domain.ErrAlreadyExists):
		writeError(c, http.StatusConflict, "already_exists", err.Error())
	case errors.Is(err, domain.ErrNotFound):
		writeError(c, http.StatusNotFound, "not_found", err.Error())
	default:
		writeError(c, http.StatusInternalServerError, "internal_error", "Failed to process car service application")
	}
}

func (h *CarServiceApplicationHandler) listMine(
	ctx context.Context,
	userID uuid.UUID,
	query dto.ListCarServiceApplicationsQuery,
) ([]*domain.CarServiceApplication, error) {
	return h.service.ListMine(ctx, userID, query.Limit, query.Offset)
}

func (h *CarServiceApplicationHandler) adminList(
	ctx context.Context,
	query dto.AdminListCarServiceApplicationsQuery,
) ([]*domain.CarServiceApplication, error) {
	var status *domain.CarServiceApplicationStatus
	if query.Status != "" {
		s := domain.CarServiceApplicationStatus(query.Status)
		status = &s
	}

	return h.service.ListForAdmin(ctx, status, query.Limit, query.Offset)
}

func writeCarServiceApplicationList(
	c *gin.Context,
	applications []*domain.CarServiceApplication,
	query dto.ListCarServiceApplicationsQuery,
) {
	items := dto.ToCarServiceApplicationResponseList(applications)
	writeJSON(c, http.StatusOK, dto.NewCarServiceApplicationListResponse(items, query.Limit, query.Offset))
}

func writeAdminCarServiceApplicationList(
	c *gin.Context,
	applications []*domain.CarServiceApplication,
	query dto.AdminListCarServiceApplicationsQuery,
) {
	items := dto.ToCarServiceApplicationResponseList(applications)
	writeJSON(c, http.StatusOK, dto.NewCarServiceApplicationListResponse(items, query.Limit, query.Offset))
}
