package handlers

import (
	"errors"
	"net/http"

	"github.com/DedovInside/AutoInspect/backend/internal/api/dto"
	"github.com/DedovInside/AutoInspect/backend/internal/api/middleware"
	"github.com/DedovInside/AutoInspect/backend/internal/domain"
	"github.com/DedovInside/AutoInspect/backend/internal/service"
	"github.com/gin-gonic/gin"
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
	userID, ok := userIDOrAbort(c)

	if !ok {
		return
	}

	var query dto.ListCarServiceApplicationsQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		writeError(c, http.StatusBadRequest, "invalid_query", err.Error())
		return
	}

	applications, err := h.service.ListMine(c.Request.Context(), userID, query.Limit, query.Offset)
	if err != nil {
		handleCarServiceApplicationError(c, err)
		return
	}

	items := dto.ToCarServiceApplicationResponseList(applications)
	writeJSON(c, http.StatusOK, dto.NewCarServiceApplicationListResponse(items, query.Limit, query.Offset))
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
