package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/DedovInside/AutoInspect/backend/internal/domain"
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
}

func NewRepairRequestService(
	requestRepo repository.RepairRequestRepository,
	jobRepo repository.AnalysisJobRepository,
	profileRepo repository.CarServiceProfileRepository,
) *RepairRequestService {
	return &RepairRequestService{
		requestRepo: requestRepo,
		jobRepo:     jobRepo,
		profileRepo: profileRepo,
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

	return s.requestRepo.ListByUserID(ctx, userID, limit, offset)
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

	return s.requestRepo.CancelPendingByUserID(ctx, requestID, userID)
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
