package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/DedovInside/AutoInspect/backend/internal/config"
	"github.com/DedovInside/AutoInspect/backend/internal/domain"
	"github.com/DedovInside/AutoInspect/backend/internal/observability"
	"github.com/DedovInside/AutoInspect/backend/internal/repository"
	"github.com/google/uuid"
)

const (
	defaultRepairRequestLimit = 20
	maxRepairRequestLimit     = 100
)

type RepairRequestService struct {
	requestRepo repository.RepairRequestRepository
	jobRepo     repository.AnalysisJobRepository
	profileRepo repository.CarServiceProfileRepository
	fileRepo    repository.FileRepository
	s3Cfg       *config.S3Config
}

func NewRepairRequestService(
	requestRepo repository.RepairRequestRepository,
	jobRepo repository.AnalysisJobRepository,
	profileRepo repository.CarServiceProfileRepository,
	fileRepo repository.FileRepository,
	s3Cfg *config.S3Config,
) *RepairRequestService {
	return &RepairRequestService{
		requestRepo: requestRepo,
		jobRepo:     jobRepo,
		profileRepo: profileRepo,
		fileRepo:    fileRepo,
		s3Cfg:       s3Cfg,
	}
}

func (s *RepairRequestService) Create(
	ctx context.Context,
	input *domain.CreateRepairRequestInput,
) (*domain.RepairRequest, error) {
	if err := validateCreateRepairRequestInput(input); err != nil {
		return nil, err
	}

	job, err := s.jobRepo.GetByID(ctx, input.AnalysisJobID)
	if err != nil {
		return nil, err
	}

	if job.UserID != input.UserID {
		return nil, domain.ErrForbidden
	}

	if err := validateJobReadyForRepairRequest(job); err != nil {
		return nil, err
	}

	profile, err := s.profileRepo.GetByID(ctx, input.CarServiceProfileID)
	if err != nil {
		return nil, err
	}

	if !profile.IsActive {
		return nil, domain.ErrInvalidInput
	}

	existing, err := s.requestRepo.GetPendingByUserAnalysisAndService(
		ctx,
		input.UserID,
		input.AnalysisJobID,
		input.CarServiceProfileID,
	)
	if err == nil {
		return existing, domain.ErrAlreadyExists
	}

	if !errors.Is(err, domain.ErrNotFound) {
		return nil, err
	}

	repairSummary := repairSummaryFromAnalysisResult(job.Result)
	if len(repairSummary) == 0 {
		return nil, domain.ErrInvalidInput
	}

	now := time.Now().UTC()
	request := &domain.RepairRequest{
		ID:                  uuid.New(),
		UserID:              input.UserID,
		AnalysisJobID:       input.AnalysisJobID,
		CarServiceProfileID: input.CarServiceProfileID,
		Status:              domain.RepairRequestStatusPending,
		Analysis:            job,
		CarServiceProfile:   profile,
		RepairSummary:       repairSummary,
		CustomerName:        normalizedOptionalString(input.CustomerName),
		CustomerPhone:       normalizedOptionalString(input.CustomerPhone),
		CustomerEmail:       normalizedOptionalString(input.CustomerEmail),
		CustomerComment:     normalizedOptionalString(input.CustomerComment),
		CreatedAt:           now,
		UpdatedAt:           now,
	}

	if err := s.requestRepo.Create(ctx, request); err != nil {
		if errors.Is(err, domain.ErrAlreadyExists) {
			existing, getErr := s.requestRepo.GetPendingByUserAnalysisAndService(
				ctx,
				input.UserID,
				input.AnalysisJobID,
				input.CarServiceProfileID,
			)
			if getErr == nil {
				return existing, domain.ErrAlreadyExists
			}
		}
		return nil, err
	}

	observability.RepairRequestsTotal.WithLabelValues("created").Inc()
	return request, nil
}

func (s *RepairRequestService) GetMine(
	ctx context.Context,
	userID, requestID uuid.UUID,
) (*domain.RepairRequest, error) {
	if userID == uuid.Nil || requestID == uuid.Nil {
		return nil, domain.ErrInvalidInput
	}

	request, err := s.requestRepo.GetByID(ctx, requestID)
	if err != nil {
		return nil, err
	}

	if request.UserID != userID {
		return nil, domain.ErrForbidden
	}

	if err := s.attachRepairRequestContext(ctx, request); err != nil {
		return nil, err
	}

	return request, nil
}

func (s *RepairRequestService) ListMine(
	ctx context.Context,
	userID uuid.UUID,
	limit, offset int,
) ([]*domain.RepairRequest, error) {
	if userID == uuid.Nil {
		return nil, domain.ErrInvalidInput
	}

	limit, offset, err := normalizeRepairRequestPagination(limit, offset)
	if err != nil {
		return nil, err
	}

	requests, err := s.requestRepo.ListByUserID(ctx, userID, limit, offset)
	if err != nil {
		return nil, err
	}

	if err := s.attachRepairRequestsContext(ctx, requests); err != nil {
		return nil, err
	}

	return requests, nil
}

func (s *RepairRequestService) CancelMine(
	ctx context.Context,
	userID, requestID uuid.UUID,
) error {
	request, err := s.GetMine(ctx, userID, requestID)
	if err != nil {
		return err
	}

	if request.Status != domain.RepairRequestStatusPending {
		return domain.ErrInvalidInput
	}

	if err := s.requestRepo.CancelPendingByUserID(ctx, requestID, userID); err != nil {
		return err
	}
	observability.RepairRequestsTotal.WithLabelValues("canceled").Inc()
	return nil
}

func (s *RepairRequestService) ListIncoming(
	ctx context.Context,
	carServiceUserID uuid.UUID,
	limit, offset int,
) ([]*domain.RepairRequest, error) {
	profile, err := s.getCarServiceProfile(ctx, carServiceUserID)
	if err != nil {
		return nil, err
	}

	limit, offset, err = normalizeRepairRequestPagination(limit, offset)
	if err != nil {
		return nil, err
	}

	requests, err := s.requestRepo.ListByCarServiceProfileID(ctx, profile.ID, limit, offset)
	if err != nil {
		return nil, err
	}

	for _, request := range requests {
		if request != nil {
			request.CarServiceProfile = profile
		}
	}

	if err := s.attachRepairRequestsContext(ctx, requests); err != nil {
		return nil, err
	}

	return requests, nil
}

func (s *RepairRequestService) GetIncomingDetails(
	ctx context.Context,
	carServiceUserID, requestID uuid.UUID,
) (*domain.RepairRequestDetails, error) {
	if requestID == uuid.Nil {
		return nil, domain.ErrInvalidInput
	}

	profile, err := s.getCarServiceProfile(ctx, carServiceUserID)
	if err != nil {
		return nil, err
	}

	request, err := s.requestRepo.GetByIDAndCarServiceProfileID(ctx, requestID, profile.ID)
	if err != nil {
		return nil, err
	}

	analysis, err := s.jobRepo.GetByID(ctx, request.AnalysisJobID)
	if err != nil {
		return nil, err
	}

	images, err := s.repairRequestImageLinks(ctx, analysis.ImageKeys)
	if err != nil {
		return nil, err
	}

	return &domain.RepairRequestDetails{
		Request:  request,
		Analysis: analysis,
		Images:   images,
	}, nil
}

func (s *RepairRequestService) AcceptIncoming(
	ctx context.Context,
	input *domain.AcceptRepairRequestInput,
) (*domain.RepairRequest, error) {
	if err := validateAcceptRepairRequestInput(input); err != nil {
		return nil, err
	}

	profile, request, err := s.getIncomingPendingRequest(ctx, input.CarServiceUserID, input.ID)
	if err != nil {
		return nil, err
	}

	estimate, err := normalizeRepairEstimate(input.ServiceEstimate)
	if err != nil {
		return nil, err
	}

	serviceComment := normalizedOptionalString(input.ServiceComment)
	if err := s.requestRepo.RespondByCarServiceProfileID(ctx, profile.ID, &domain.RespondRepairRequestInput{
		ID:                request.ID,
		Status:            domain.RepairRequestStatusAccepted,
		ServiceComment:    serviceComment,
		ServiceEstimate:   estimate,
		EstimatedPriceMin: input.EstimatedPriceMin,
		EstimatedPriceMax: input.EstimatedPriceMax,
	}); err != nil {
		return nil, err
	}

	observability.RepairRequestsTotal.WithLabelValues("accepted").Inc()
	return s.requestRepo.GetByIDAndCarServiceProfileID(ctx, request.ID, profile.ID)
}

func (s *RepairRequestService) RejectIncoming(
	ctx context.Context,
	input *domain.RejectRepairRequestInput,
) (*domain.RepairRequest, error) {
	if err := validateRejectRepairRequestInput(input); err != nil {
		return nil, err
	}

	profile, request, err := s.getIncomingPendingRequest(ctx, input.CarServiceUserID, input.ID)
	if err != nil {
		return nil, err
	}

	serviceComment := strings.TrimSpace(input.ServiceComment)
	if err := s.requestRepo.RespondByCarServiceProfileID(ctx, profile.ID, &domain.RespondRepairRequestInput{
		ID:             request.ID,
		Status:         domain.RepairRequestStatusRejected,
		ServiceComment: &serviceComment,
	}); err != nil {
		return nil, err
	}

	observability.RepairRequestsTotal.WithLabelValues("rejected").Inc()
	return s.requestRepo.GetByIDAndCarServiceProfileID(ctx, request.ID, profile.ID)
}

func (s *RepairRequestService) getIncomingPendingRequest(
	ctx context.Context,
	carServiceUserID, requestID uuid.UUID,
) (*domain.CarServiceProfile, *domain.RepairRequest, error) {
	if requestID == uuid.Nil {
		return nil, nil, domain.ErrInvalidInput
	}

	profile, err := s.getCarServiceProfile(ctx, carServiceUserID)
	if err != nil {
		return nil, nil, err
	}

	request, err := s.requestRepo.GetByIDAndCarServiceProfileID(ctx, requestID, profile.ID)
	if err != nil {
		return nil, nil, err
	}

	if request.Status != domain.RepairRequestStatusPending {
		return nil, nil, domain.ErrInvalidInput
	}

	return profile, request, nil
}

func (s *RepairRequestService) getCarServiceProfile(
	ctx context.Context,
	carServiceUserID uuid.UUID,
) (*domain.CarServiceProfile, error) {
	if carServiceUserID == uuid.Nil {
		return nil, domain.ErrInvalidInput
	}

	return s.profileRepo.GetByUserID(ctx, carServiceUserID)
}

func (s *RepairRequestService) attachRepairRequestsContext(
	ctx context.Context,
	requests []*domain.RepairRequest,
) error {
	for _, request := range requests {
		if err := s.attachRepairRequestContext(ctx, request); err != nil {
			return err
		}
	}

	return nil
}

func (s *RepairRequestService) attachRepairRequestContext(
	ctx context.Context,
	request *domain.RepairRequest,
) error {
	if request == nil {
		return nil
	}

	if request.Analysis == nil {
		analysis, err := s.jobRepo.GetByID(ctx, request.AnalysisJobID)
		if err != nil {
			return err
		}
		request.Analysis = analysis
	}

	if request.CarServiceProfile == nil {
		profile, err := s.profileRepo.GetByID(ctx, request.CarServiceProfileID)
		if err != nil {
			return err
		}
		request.CarServiceProfile = profile
	}

	return nil
}

func (s *RepairRequestService) repairRequestImageLinks(
	ctx context.Context,
	imageKeys []string,
) ([]domain.RepairRequestImageLink, error) {
	if len(imageKeys) == 0 {
		return []domain.RepairRequestImageLink{}, nil
	}

	if s.fileRepo == nil || s.s3Cfg == nil {
		return nil, domain.ErrInternal
	}

	images := make([]domain.RepairRequestImageLink, 0, len(imageKeys))
	for idx, objectKey := range imageKeys {
		objectKey = strings.TrimSpace(objectKey)
		if objectKey == "" {
			continue
		}

		expiresAt := time.Now().Add(s.s3Cfg.PresignedURLTTL)
		url, err := s.fileRepo.GetPresignedURL(ctx, s.s3Cfg.BucketUploads, objectKey, s.s3Cfg.PresignedURLTTL)
		if err != nil {
			return nil, err
		}

		images = append(images, domain.RepairRequestImageLink{
			Index:     idx,
			URL:       url,
			ExpiresAt: expiresAt,
		})
	}

	return images, nil
}

func validateCreateRepairRequestInput(input *domain.CreateRepairRequestInput) error {
	if input == nil ||
		input.UserID == uuid.Nil ||
		input.AnalysisJobID == uuid.Nil ||
		input.CarServiceProfileID == uuid.Nil {
		return domain.ErrInvalidInput
	}

	return nil
}

func validateAcceptRepairRequestInput(input *domain.AcceptRepairRequestInput) error {
	if input == nil || input.ID == uuid.Nil || input.CarServiceUserID == uuid.Nil {
		return domain.ErrInvalidInput
	}

	if input.EstimatedPriceMin == nil || input.EstimatedPriceMax == nil {
		return domain.ErrInvalidInput
	}

	if err := validatePriceRange(input.EstimatedPriceMin, input.EstimatedPriceMax); err != nil {
		return err
	}

	return nil
}

func validateRejectRepairRequestInput(input *domain.RejectRepairRequestInput) error {
	if input == nil ||
		input.ID == uuid.Nil ||
		input.CarServiceUserID == uuid.Nil ||
		strings.TrimSpace(input.ServiceComment) == "" {
		return domain.ErrInvalidInput
	}

	return nil
}

func validatePriceRange(minPrice, maxPrice *float64) error {
	if minPrice == nil && maxPrice == nil {
		return nil
	}

	if minPrice != nil && *minPrice < 0 {
		return domain.ErrInvalidInput
	}

	if maxPrice != nil && *maxPrice < 0 {
		return domain.ErrInvalidInput
	}

	if minPrice != nil && maxPrice != nil && *minPrice > *maxPrice {
		return domain.ErrInvalidInput
	}

	return nil
}

func normalizeRepairEstimate(items []domain.RepairEstimateItem) ([]domain.RepairEstimateItem, error) {
	out := make([]domain.RepairEstimateItem, 0, len(items))
	for idx := range items {
		item, err := normalizeRepairEstimateItem(&items[idx])
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}

	return out, nil
}

func normalizeRepairEstimateItem(item *domain.RepairEstimateItem) (domain.RepairEstimateItem, error) {
	if item == nil ||
		strings.TrimSpace(item.PartName) == "" ||
		strings.TrimSpace(item.DamageCode) == "" ||
		item.Quantity <= 0 {
		return domain.RepairEstimateItem{}, domain.ErrInvalidInput
	}
	if err := validatePriceRange(item.PriceMin, item.PriceMax); err != nil {
		return domain.RepairEstimateItem{}, err
	}

	return domain.RepairEstimateItem{
		PartName:     strings.TrimSpace(item.PartName),
		PartNameRU:   strings.TrimSpace(item.PartNameRU),
		ParentName:   strings.TrimSpace(item.ParentName),
		ParentNameRU: strings.TrimSpace(item.ParentNameRU),
		IsPair:       item.IsPair,
		Side:         strings.TrimSpace(item.Side),
		SideRU:       strings.TrimSpace(item.SideRU),
		DamageCode:   normalizeDamageTypeCode(item.DamageCode),
		DamageNameRU: strings.TrimSpace(item.DamageNameRU),
		Quantity:     item.Quantity,
		PriceMin:     item.PriceMin,
		PriceMax:     item.PriceMax,
		Comment:      normalizedOptionalString(item.Comment),
	}, nil
}

func validateJobReadyForRepairRequest(job *domain.AnalysisJob) error {
	if job == nil {
		return domain.ErrJobNotFound
	}

	if job.Status == domain.StatusFailed {
		return domain.ErrJobFailed
	}

	if job.Status != domain.StatusCompleted || job.Result == nil {
		return domain.ErrJobNotReady
	}

	return nil
}

func normalizeRepairRequestPagination(limit, offset int) (normalizedLimit, normalizedOffset int, err error) {
	if limit <= 0 {
		limit = defaultRepairRequestLimit
	}

	if limit > maxRepairRequestLimit {
		limit = maxRepairRequestLimit
	}

	if offset < 0 {
		return 0, 0, domain.ErrInvalidInput
	}

	return limit, offset, nil
}

type repairSummaryDraft struct {
	item       domain.RepairSummaryItem
	damageSeen map[string]int
}

func repairSummaryFromAnalysisResult(result *domain.AnalysisResult) []domain.RepairSummaryItem {
	if result == nil {
		return nil
	}

	drafts := make(map[string]*repairSummaryDraft)
	order := make([]string, 0)

	for resultIdx := range result.Results {
		for summaryIdx := range result.Results[resultIdx].PartsSummary {
			appendRepairSummary(drafts, &order, &result.Results[resultIdx].PartsSummary[summaryIdx])
		}
	}

	return repairSummaryItems(drafts, order)
}

func appendRepairSummary(
	drafts map[string]*repairSummaryDraft,
	order *[]string,
	summary *domain.PartSummary,
) {
	if summary == nil {
		return
	}

	partName := strings.TrimSpace(summary.Name)
	if partName == "" {
		partName = strings.TrimSpace(summary.ParentName)
	}

	if partName == "" {
		return
	}

	key := repairSummaryKey(partName, summary.Side, summary.ParentName)
	draft, ok := drafts[key]
	if !ok {
		draft = &repairSummaryDraft{
			item: domain.RepairSummaryItem{
				PartName:     partName,
				PartNameRU:   strings.TrimSpace(summary.NameRU),
				ParentName:   strings.TrimSpace(summary.ParentName),
				ParentNameRU: strings.TrimSpace(summary.ParentNameRU),
				IsPair:       summary.IsPair,
				Side:         strings.TrimSpace(summary.Side),
				SideRU:       strings.TrimSpace(summary.SideRU),
			},
			damageSeen: make(map[string]int),
		}
		drafts[key] = draft
		*order = append(*order, key)
	}

	draft.item.DamageCount += summary.DamageCount
	for damageIdx := range summary.DamageTypes {
		appendRepairDamageType(draft, &summary.DamageTypes[damageIdx])
	}
}

func appendRepairDamageType(draft *repairSummaryDraft, damageType *domain.DamageTypeSummary) {
	if draft == nil || damageType == nil {
		return
	}

	code := normalizeDamageTypeCode(damageType.Code)
	if code == "" {
		return
	}

	count := damageType.Count
	if count <= 0 {
		count = 1
	}

	idx, ok := draft.damageSeen[code]
	if !ok {
		draft.damageSeen[code] = len(draft.item.DamageTypes)
		draft.item.DamageTypes = append(draft.item.DamageTypes, domain.RepairDamageTypeSummary{
			Code:   code,
			NameRU: strings.TrimSpace(damageType.NameRU),
			Count:  count,
		})
		return
	}

	draft.item.DamageTypes[idx].Count += count
	if draft.item.DamageTypes[idx].NameRU == "" {
		draft.item.DamageTypes[idx].NameRU = strings.TrimSpace(damageType.NameRU)
	}
}

func repairSummaryItems(drafts map[string]*repairSummaryDraft, order []string) []domain.RepairSummaryItem {
	items := make([]domain.RepairSummaryItem, 0, len(order))
	for _, key := range order {
		draft := drafts[key]
		if draft == nil || len(draft.item.DamageTypes) == 0 {
			continue
		}
		if draft.item.DamageCount <= 0 {
			draft.item.DamageCount = repairDamageTypeCount(draft.item.DamageTypes)
		}
		items = append(items, draft.item)
	}

	return items
}

func repairDamageTypeCount(damageTypes []domain.RepairDamageTypeSummary) int {
	count := 0
	for _, damageType := range damageTypes {
		count += damageType.Count
	}

	return count
}

func repairSummaryKey(partName, side, parentName string) string {
	return strings.TrimSpace(partName) + "\x00" +
		strings.TrimSpace(side) + "\x00" +
		strings.TrimSpace(parentName)
}
