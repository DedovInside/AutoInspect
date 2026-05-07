package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/DedovInside/AutoInspect/backend/internal/api/dto"
	"github.com/DedovInside/AutoInspect/backend/internal/domain"
	"github.com/DedovInside/AutoInspect/backend/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const carServiceImageFormField = "image"

type CarServiceProfileHandler struct {
	service *service.CarServiceProfileService
}

func NewCarServiceProfileHandler(svc *service.CarServiceProfileService) *CarServiceProfileHandler {
	return &CarServiceProfileHandler{service: svc}
}

func (h *CarServiceProfileHandler) GetMine(c *gin.Context) {
	userID, ok := userIDOrAbort(c)
	if !ok {
		return
	}

	profile, err := h.service.GetMyProfile(c.Request.Context(), userID)
	if err != nil {
		handleCarServiceProfileError(c, err)
		return
	}

	writeJSON(c, http.StatusOK, dto.GetCarServiceProfileResponse{
		Profile: dto.ToCarServiceProfileResponse(profile),
	})
}

func (h *CarServiceProfileHandler) UpdateMine(c *gin.Context) {
	userID, ok := userIDOrAbort(c)
	if !ok {
		return
	}

	var req dto.UpdateCarServiceProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	profile, err := h.service.UpdateMyProfile(c.Request.Context(), &domain.UpdateCarServiceProfileInput{
		UserID:           userID,
		OrganizationName: req.OrganizationName,
		City:             req.City,
		Address:          req.Address,
		Phone:            req.Phone,
		Email:            req.Email,
		WebsiteURL:       req.WebsiteURL,
		ContactInfo:      req.ContactInfo,
		Description:      req.Description,
		IsActive:         *req.IsActive,
	})
	if err != nil {
		handleCarServiceProfileError(c, err)
		return
	}

	writeJSON(c, http.StatusOK, dto.UpdateCarServiceProfileResponse{
		Profile: dto.ToCarServiceProfileResponse(profile),
	})
}

func (h *CarServiceProfileHandler) SetActive(c *gin.Context) {
	userID, ok := userIDOrAbort(c)
	if !ok {
		return
	}

	var req dto.SetCarServiceProfileActiveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	profile, err := h.service.SetMyProfileActive(c.Request.Context(), userID, *req.IsActive)
	if err != nil {
		handleCarServiceProfileError(c, err)
		return
	}

	writeJSON(c, http.StatusOK, dto.SetCarServiceProfileActiveResponse{
		Profile: dto.ToCarServiceProfileResponse(profile),
	})
}

func (h *CarServiceProfileHandler) UploadImage(c *gin.Context) {
	userID, ok := userIDOrAbort(c)
	if !ok {
		return
	}

	header, err := c.FormFile(carServiceImageFormField)
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid_image", err.Error())
		return
	}

	file, err := header.Open()
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid_image", err.Error())
		return
	}
	defer func() { _ = file.Close() }()

	image, err := h.service.UploadImage(c.Request.Context(), &service.UploadCarServiceImageInput{
		UserID:           userID,
		File:             file,
		OriginalFilename: header.Filename,
		IsPrimary:        parseOptionalBool(c.PostForm("is_primary")),
	})
	if err != nil {
		handleCarServiceProfileError(c, err)
		return
	}

	writeJSON(c, http.StatusCreated, dto.UploadCarServiceImageResponse{
		Image: dto.ToCarServiceImageResponse(image.Image, image.URL, image.ExpiresAt),
	})
}

func (h *CarServiceProfileHandler) ListImages(c *gin.Context) {
	userID, ok := userIDOrAbort(c)
	if !ok {
		return
	}

	images, err := h.service.ListImages(c.Request.Context(), userID)
	if err != nil {
		handleCarServiceProfileError(c, err)
		return
	}

	items := make([]dto.CarServiceImageResponse, 0, len(images))
	for _, image := range images {
		if image == nil {
			continue
		}
		items = append(items, dto.ToCarServiceImageResponse(image.Image, image.URL, image.ExpiresAt))
	}

	writeJSON(c, http.StatusOK, dto.NewCarServiceImagesResponse(items))
}

func (h *CarServiceProfileHandler) SetPrimaryImage(c *gin.Context) {
	userID, imageID, ok := carServiceImageIDFromRequest(c)
	if !ok {
		return
	}

	image, err := h.service.SetPrimaryImage(c.Request.Context(), userID, imageID)
	if err != nil {
		handleCarServiceProfileError(c, err)
		return
	}

	writeJSON(c, http.StatusOK, dto.SetPrimaryCarServiceImageResponse{
		Image: dto.ToCarServiceImageResponse(image.Image, image.URL, image.ExpiresAt),
	})
}

func (h *CarServiceProfileHandler) DeleteImage(c *gin.Context) {
	userID, imageID, ok := carServiceImageIDFromRequest(c)
	if !ok {
		return
	}

	if err := h.service.DeleteImage(c.Request.Context(), userID, imageID); err != nil {
		handleCarServiceProfileError(c, err)
		return
	}

	writeJSON(c, http.StatusOK, gin.H{"status": "ok"})
}

func (h *CarServiceProfileHandler) ListSpecializationOptions(c *gin.Context) {
	options, err := h.service.ListSpecializationOptions(c.Request.Context())
	if err != nil {
		handleCarServiceProfileError(c, err)
		return
	}

	writeJSON(c, http.StatusOK, dto.ToSpecializationOptionsResponse(options))
}

func (h *CarServiceProfileHandler) ListSpecializations(c *gin.Context) {
	userID, ok := userIDOrAbort(c)
	if !ok {
		return
	}

	specializations, err := h.service.ListMySpecializations(c.Request.Context(), userID)
	if err != nil {
		handleCarServiceProfileError(c, err)
		return
	}

	items := dto.ToCarServiceSpecializationResponseList(specializations)
	writeJSON(c, http.StatusOK, dto.NewCarServiceSpecializationsResponse(items))
}

func (h *CarServiceProfileHandler) ReplaceSpecializations(c *gin.Context) {
	userID, ok := userIDOrAbort(c)
	if !ok {
		return
	}

	var req dto.ReplaceCarServiceSpecializationsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	inputs := dto.ToCarServiceSpecializationInputList(req.Items)
	specializations, err := h.service.ReplaceMySpecializations(c.Request.Context(), userID, inputs)
	if err != nil {
		handleCarServiceProfileError(c, err)
		return
	}

	items := dto.ToCarServiceSpecializationResponseList(specializations)
	writeJSON(c, http.StatusOK, dto.NewReplaceCarServiceSpecializationsResponse(items))
}

func carServiceImageIDFromRequest(c *gin.Context) (userID, imageID uuid.UUID, ok bool) {
	userID, ok = userIDOrAbort(c)
	if !ok {
		return uuid.Nil, uuid.Nil, false
	}

	imageID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid_id", "Image ID must be a valid UUID")
		return uuid.Nil, uuid.Nil, false
	}

	return userID, imageID, true
}

func parseOptionalBool(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}

	parsed, err := strconv.ParseBool(value)

	return err == nil && parsed
}

func handleCarServiceProfileError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidInput):
		writeError(c, http.StatusBadRequest, "invalid_input", err.Error())
	case errors.Is(err, domain.ErrInvalidImage):
		writeError(c, http.StatusBadRequest, "invalid_image", err.Error())
	case errors.Is(err, domain.ErrForbidden):
		writeError(c, http.StatusForbidden, "forbidden", err.Error())
	case errors.Is(err, domain.ErrAlreadyExists):
		writeError(c, http.StatusConflict, "already_exists", err.Error())
	case errors.Is(err, domain.ErrNotFound):
		writeError(c, http.StatusNotFound, "not_found", err.Error())
	default:
		writeError(c, http.StatusInternalServerError, "internal_error", "Failed to process car service profile")
	}
}
