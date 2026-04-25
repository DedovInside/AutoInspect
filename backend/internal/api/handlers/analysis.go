package handlers

import (
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"strconv"

	"github.com/DedovInside/AutoInspect/backend/internal/api/dto"
	"github.com/DedovInside/AutoInspect/backend/internal/api/middleware"
	"github.com/DedovInside/AutoInspect/backend/internal/broadcast"
	"github.com/DedovInside/AutoInspect/backend/internal/domain"
	"github.com/DedovInside/AutoInspect/backend/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type AnalysisHandler struct {
	service        *service.AnalysisService
	broadcastMgr   *broadcast.Manager
	maxImages      int
	maxImageSize   int64
	allowedTypes   map[string]bool
	allowedOrigins map[string]bool
}

func NewAnalysisHandler(
	svc *service.AnalysisService,
	broadcastMgr *broadcast.Manager,
	maxImages int,
	maxImageSizeMB int,
	allowedTypes []string,
	allowedOrigins []string,
) *AnalysisHandler {
	allowed := make(map[string]bool, len(allowedTypes))
	for _, t := range allowedTypes {
		allowed[t] = true
	}
	origins := make(map[string]bool, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		if origin != "" {
			origins[origin] = true
		}
	}
	return &AnalysisHandler{
		service:        svc,
		broadcastMgr:   broadcastMgr,
		maxImages:      maxImages,
		maxImageSize:   int64(maxImageSizeMB) * 1024 * 1024,
		allowedTypes:   allowed,
		allowedOrigins: origins,
	}
}

func (h *AnalysisHandler) Submit(c *gin.Context) {
	userID, ok := middleware.UserIDFromContext(c)

	if !ok {
		writeError(c, http.StatusUnauthorized, "unauthorized", "missing user context")
		return
	}

	var req dto.CreateAnalysisRequest

	if err := c.ShouldBind(&req); err != nil {
		writeError(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	if err := c.Request.ParseMultipartForm(32 << 20); err != nil {
		writeError(c, http.StatusBadRequest, "invalid_form", "failed to parse multipart form")
		return
	}

	images, err := h.parseAndValidateImages(c)

	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid_images", err.Error())
		return
	}
	defer func() {
		for _, img := range images {
			_ = img.Close()
		}
	}()

	var idempotencyKey *string

	if key := c.GetHeader("Idempotency-Key"); key != "" {
		idempotencyKey = &key
	}

	carInfo := domain.CarInfo{
		Make:       req.Make,
		Model:      req.Model,
		Generation: req.Generation,
		Year:       req.Year,
	}

	job, err := h.service.SubmitAnalysis(c.Request.Context(), userID, idempotencyKey, carInfo, images)

	if err != nil {
		h.handleServiceError(c, err)
		return
	}

	resp := dto.CreateAnalysisResponse{
		Job: dto.ToAnalysisJobResponse(job),
	}
	writeJSON(c, http.StatusAccepted, resp)
}

func (h *AnalysisHandler) GetByID(c *gin.Context) {
	userID, ok := middleware.UserIDFromContext(c)

	if !ok {
		writeError(c, http.StatusUnauthorized, "unauthorized", "missing user context")
		return
	}

	jobID, err := uuid.Parse(c.Param("id"))

	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid_id", "Job ID must be a valid UUID")
		return
	}

	job, err := h.service.GetJobStatus(c.Request.Context(), jobID, userID)

	if err != nil {
		h.handleJobError(c, err)
		return
	}

	resp := dto.ToAnalysisJobResponse(job)
	writeJSON(c, http.StatusOK, resp)
}

func (h *AnalysisHandler) ListMine(c *gin.Context) {
	userID, ok := middleware.UserIDFromContext(c)

	if !ok {
		writeError(c, http.StatusUnauthorized, "unauthorized", "missing user context")
		return
	}

	var query dto.ListAnalysisQuery

	if err := c.ShouldBindQuery(&query); err != nil {
		writeError(c, http.StatusBadRequest, "invalid_query", err.Error())
		return
	}

	jobs, err := h.service.ListUserJobs(c.Request.Context(), userID, query.Limit, query.Offset)

	if err != nil {
		writeError(c, http.StatusInternalServerError, "internal_error", "Failed to list jobs")
		return
	}

	items := dto.ToAnalysisJobResponseList(jobs)
	resp := dto.NewListResponse(items, query.Limit, query.Offset)
	writeJSON(c, http.StatusOK, resp)
}

func (h *AnalysisHandler) GetPresignedImageURL(c *gin.Context) {
	userID, ok := middleware.UserIDFromContext(c)

	if !ok {
		writeError(c, http.StatusUnauthorized, "unauthorized", "missing user context")
		return
	}

	jobID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid_id", "Job ID must be a valid UUID")
		return
	}

	idx, err := strconv.Atoi(c.Param("idx"))
	if err != nil || idx < 0 {
		writeError(c, http.StatusBadRequest, "invalid_index", "Image index must be non-negative")
		return
	}

	url, expiresAt, err := h.service.GetPresignedImageURL(c.Request.Context(), jobID, userID, idx)

	if err != nil {
		h.handleJobError(c, err)
		return
	}

	resp := dto.PresignedImageURLResponse{
		URL:       url,
		ExpiresAt: expiresAt,
	}

	writeJSON(c, http.StatusOK, resp)
}

func (h *AnalysisHandler) parseAndValidateImages(c *gin.Context) ([]multipart.File, error) {
	form, err := c.MultipartForm()

	if err != nil {
		return nil, fmt.Errorf("parse multipart form: %w", err)
	}

	files := form.File["images"]

	if len(files) == 0 {
		return nil, errors.New("at least one image is required")
	}

	if len(files) > h.maxImages {
		return nil, fmt.Errorf("too many images: max %d allowed", h.maxImages)
	}

	images := make([]multipart.File, 0, len(files))
	for i, fh := range files {
		if fh.Size > h.maxImageSize {
			closeMultipartFiles(images)
			return nil, fmt.Errorf("image %d too large: max %d MB", i, h.maxImageSize/(1024*1024))
		}

		contentType := fh.Header.Get("Content-Type")

		if !h.allowedTypes[contentType] {
			closeMultipartFiles(images)
			return nil, fmt.Errorf("image %d has unsupported type: %s", i, contentType)
		}

		file, err := fh.Open()

		if err != nil {
			closeMultipartFiles(images)
			return nil, fmt.Errorf("open image %d: %w", i, err)
		}

		images = append(images, file)
	}

	return images, nil
}

func closeMultipartFiles(files []multipart.File) {
	for _, file := range files {
		_ = file.Close()
	}
}

func (h *AnalysisHandler) handleServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidModel):
		writeError(c, http.StatusBadRequest, "no_suitable_model", "No model found for specified car")
	case errors.Is(err, domain.ErrInvalidImage):
		writeError(c, http.StatusBadRequest, "invalid_image_type", "Only JPEG/PNG/WebP allowed")
	default:
		writeError(c, http.StatusInternalServerError, "internal_error", "Failed to create analysis job")
	}
}

func (h *AnalysisHandler) handleJobError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrJobNotFound):
		writeError(c, http.StatusNotFound, "not_found", "Analysis job not found")
	case errors.Is(err, domain.ErrForbidden):
		writeError(c, http.StatusForbidden, "forbidden", "Access denied to this job")
	default:
		writeError(c, http.StatusInternalServerError, "internal_error", "Failed to fetch job")
	}
}
