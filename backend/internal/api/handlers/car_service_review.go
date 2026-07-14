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

type CarServiceReviewHandler struct {
	service *service.CarServiceReviewService
}

func NewCarServiceReviewHandler(svc *service.CarServiceReviewService) *CarServiceReviewHandler {
	return &CarServiceReviewHandler{service: svc}
}

func (h *CarServiceReviewHandler) CreateForRepairRequest(c *gin.Context) {
	userID, requestID, ok := userAndUUIDParamOrAbort(c, "id", "Repair request ID must be a valid UUID")
	if !ok {
		return
	}

	req, ok := bindJSONOrAbort[dto.CreateCarServiceReviewRequest](c)
	if !ok {
		return
	}

	review, err := h.service.Create(c.Request.Context(), &domain.CreateCarServiceReviewInput{
		UserID:          userID,
		RepairRequestID: requestID,
		Rating:          req.Rating,
		AuthorName:      req.AuthorName,
		Comment:         req.Comment,
	})
	if err != nil {
		handleCarServiceReviewError(c, err)
		return
	}

	writeJSON(c, http.StatusCreated, dto.CreateCarServiceReviewResponse{
		Review: dto.ToCarServiceReviewResponse(review),
	})
}

func (h *CarServiceReviewHandler) GetByRepairRequest(c *gin.Context) {
	_, requestID, ok := userAndUUIDParamOrAbort(c, "id", "Repair request ID must be a valid UUID")
	if !ok {
		return
	}

	review, err := h.service.GetByRepairRequestID(c.Request.Context(), requestID)
	if err != nil {
		handleCarServiceReviewError(c, err)
		return
	}

	writeJSON(c, http.StatusOK, dto.GetCarServiceReviewResponse{
		Review: dto.ToCarServiceReviewResponse(review),
	})
}

func (h *CarServiceReviewHandler) UpdateForRepairRequest(c *gin.Context) {
	userID, requestID, ok := userAndUUIDParamOrAbort(c, "id", "Repair request ID must be a valid UUID")
	if !ok {
		return
	}

	req, ok := bindJSONOrAbort[dto.UpdateCarServiceReviewRequest](c)
	if !ok {
		return
	}

	review, err := h.service.Update(c.Request.Context(), &domain.UpdateCarServiceReviewInput{
		UserID:          userID,
		RepairRequestID: requestID,
		Rating:          req.Rating,
		AuthorName:      req.AuthorName,
		Comment:         req.Comment,
	})
	if err != nil {
		handleCarServiceReviewError(c, err)
		return
	}

	writeJSON(c, http.StatusOK, dto.UpdateCarServiceReviewResponse{
		Review: dto.ToCarServiceReviewResponse(review),
	})
}

func (h *CarServiceReviewHandler) DeleteForRepairRequest(c *gin.Context) {
	userID, requestID, ok := userAndUUIDParamOrAbort(c, "id", "Repair request ID must be a valid UUID")
	if !ok {
		return
	}

	if err := h.service.Delete(c.Request.Context(), userID, requestID); err != nil {
		handleCarServiceReviewError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *CarServiceReviewHandler) ListByCarService(c *gin.Context) {
	profileID, ok := reviewUUIDParamOrAbort(c, "id", "Car service profile ID must be a valid UUID")
	if !ok {
		return
	}

	query, ok := bindQueryOrAbort[dto.ListCarServiceReviewsQuery](c)
	if !ok {
		return
	}

	reviews, err := h.service.ListByCarServiceProfileID(c.Request.Context(), profileID, query.Limit, query.Offset)
	if err != nil {
		handleCarServiceReviewError(c, err)
		return
	}

	writeCarServiceReviewList(c, reviews, query)
}

func (h *CarServiceReviewHandler) ListMine(c *gin.Context) {
	handleUserQueryList(c, h.listMine, writeCarServiceReviewList, handleCarServiceReviewError)
}

func (h *CarServiceReviewHandler) listMine(
	ctx context.Context,
	userID uuid.UUID,
	query dto.ListCarServiceReviewsQuery,
) ([]*domain.CarServiceReview, error) {
	return h.service.ListMine(ctx, userID, query.Limit, query.Offset)
}

func writeCarServiceReviewList(
	c *gin.Context,
	reviews []*domain.CarServiceReview,
	query dto.ListCarServiceReviewsQuery,
) {
	items := dto.ToCarServiceReviewResponseList(reviews)
	writeJSON(c, http.StatusOK, dto.NewCarServiceReviewListResponse(items, query.Limit, query.Offset))
}

func userAndUUIDParamOrAbort(c *gin.Context, paramName, message string) (userID, id uuid.UUID, ok bool) {
	userID, ok = userIDOrAbort(c)
	if !ok {
		return uuid.Nil, uuid.Nil, false
	}

	id, ok = reviewUUIDParamOrAbort(c, paramName, message)
	if !ok {
		return uuid.Nil, uuid.Nil, false
	}

	return userID, id, true
}

func reviewUUIDParamOrAbort(c *gin.Context, paramName, message string) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param(paramName))
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid_id", message)
		return uuid.Nil, false
	}

	return id, true
}

func handleCarServiceReviewError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidInput):
		writeError(c, http.StatusBadRequest, "invalid_input", err.Error())
	case errors.Is(err, domain.ErrAlreadyExists):
		writeError(c, http.StatusConflict, "already_exists", err.Error())
	case errors.Is(err, domain.ErrForbidden):
		writeError(c, http.StatusForbidden, "forbidden", err.Error())
	case errors.Is(err, domain.ErrNotFound):
		writeError(c, http.StatusNotFound, "not_found", err.Error())
	default:
		writeError(c, http.StatusInternalServerError, "internal_error", "Failed to process car service review")
	}
}
