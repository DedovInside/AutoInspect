package service

import (
	"context"
	"time"

	"github.com/DedovInside/AutoInspect/backend/internal/config"
	"github.com/DedovInside/AutoInspect/backend/internal/domain"
	"github.com/DedovInside/AutoInspect/backend/internal/repository"
	"github.com/google/uuid"
)

type CarServiceMatchingService struct {
	jobRepo   repository.AnalysisJobRepository
	matchRepo repository.CarServiceMatchRepository
	imageRepo repository.CarServiceImageRepository
	fileRepo  repository.FileRepository
	s3Cfg     *config.S3Config
}

func NewCarServiceMatchingService(
	jobRepo repository.AnalysisJobRepository,
	matchRepo repository.CarServiceMatchRepository,
	imageRepo repository.CarServiceImageRepository,
	fileRepo repository.FileRepository,
	s3Cfg *config.S3Config,
) *CarServiceMatchingService {
	return &CarServiceMatchingService{
		jobRepo:   jobRepo,
		matchRepo: matchRepo,
		imageRepo: imageRepo,
		fileRepo:  fileRepo,
		s3Cfg:     s3Cfg,
	}
}

func (s *CarServiceMatchingService) FindMatchingCarServices(
	ctx context.Context,
	jobID, userID uuid.UUID,
	limit, offset int,
) ([]*domain.CarServiceMatchWithImageURL, error) {
	if jobID == uuid.Nil || userID == uuid.Nil || limit <= 0 || offset < 0 {
		return nil, domain.ErrInvalidInput
	}

	if s.matchRepo == nil {
		return nil, domain.ErrInternal
	}

	job, err := s.jobRepo.GetByID(ctx, jobID)
	if err != nil {
		return nil, err
	}

	if job.UserID != userID {
		return nil, domain.ErrForbidden
	}

	criteria, err := matchCriteriaFromAnalysisJob(job)
	if err != nil {
		return nil, err
	}

	if len(criteria) == 0 {
		return []*domain.CarServiceMatchWithImageURL{}, nil
	}

	matches, err := s.matchRepo.FindMatching(ctx, criteria, limit, offset)
	if err != nil {
		return nil, err
	}

	for _, match := range matches {
		match.RequiredCount = len(criteria)
		if match.RequiredCount > 0 {
			match.Score = float64(match.MatchCount) / float64(match.RequiredCount)
		}
		if err := s.attachPrimaryImage(ctx, match); err != nil {
			return nil, err
		}
	}

	return s.carServiceMatchesWithImageURLs(ctx, matches)
}

func (s *CarServiceMatchingService) attachPrimaryImage(ctx context.Context, match *domain.CarServiceMatch) error {
	if match == nil || match.Profile == nil || s.imageRepo == nil {
		return nil
	}

	images, err := s.imageRepo.ListByProfileID(ctx, match.Profile.ID)
	if err != nil {
		return err
	}

	if len(images) == 0 {
		return nil
	}

	match.PrimaryImage = images[0]

	return nil
}

func matchCriteriaFromAnalysisJob(job *domain.AnalysisJob) ([]domain.CarServiceMatchCriterion, error) {
	if err := validateJobReadyForMatching(job); err != nil {
		return nil, err
	}

	criteria := make([]domain.CarServiceMatchCriterion, 0)
	seen := make(map[string]struct{})
	for resultIdx := range job.Result.Results {
		criteria = appendCriteriaFromImageResult(criteria, seen, &job.Result.Results[resultIdx])
	}

	return criteria, nil
}

func validateJobReadyForMatching(job *domain.AnalysisJob) error {
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

func appendCriteriaFromImageResult(
	criteria []domain.CarServiceMatchCriterion,
	seen map[string]struct{},
	imageResult *domain.ImageAnalysisResult,
) []domain.CarServiceMatchCriterion {
	if imageResult == nil {
		return criteria
	}

	for summaryIdx := range imageResult.PartsSummary {
		criteria = appendCriteriaFromPartSummary(criteria, seen, &imageResult.PartsSummary[summaryIdx])
	}

	return criteria
}

func appendCriteriaFromPartSummary(
	criteria []domain.CarServiceMatchCriterion,
	seen map[string]struct{},
	summary *domain.PartSummary,
) []domain.CarServiceMatchCriterion {
	partCategoryCode := matchPartCategoryCode(summary)
	if partCategoryCode == "" {
		return criteria
	}

	for damageTypeIdx := range summary.DamageTypes {
		damageTypeCode := normalizeDamageTypeCode(summary.DamageTypes[damageTypeIdx].Code)
		if damageTypeCode == "" {
			continue
		}

		criteria = appendUniqueMatchCriterion(criteria, seen, damageTypeCode, partCategoryCode)
	}

	return criteria
}

func appendUniqueMatchCriterion(
	criteria []domain.CarServiceMatchCriterion,
	seen map[string]struct{},
	damageTypeCode, partCategoryCode string,
) []domain.CarServiceMatchCriterion {
	key := damageTypeCode + "\x00" + partCategoryCode
	if _, ok := seen[key]; ok {
		return criteria
	}

	seen[key] = struct{}{}
	return append(criteria, domain.CarServiceMatchCriterion{
		DamageTypeCode:   damageTypeCode,
		PartCategoryCode: partCategoryCode,
	})
}

func matchPartCategoryCode(summary *domain.PartSummary) string {
	if summary == nil {
		return ""
	}

	if code := normalizePartCode(summary.ParentName); code != "" {
		return code
	}

	return normalizePartCode(summary.Name)
}

func (s *CarServiceMatchingService) carServiceMatchesWithImageURLs(
	ctx context.Context,
	matches []*domain.CarServiceMatch,
) ([]*domain.CarServiceMatchWithImageURL, error) {
	out := make([]*domain.CarServiceMatchWithImageURL, 0, len(matches))
	for _, match := range matches {
		item, err := s.carServiceMatchWithImageURL(ctx, match)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}

	return out, nil
}

func (s *CarServiceMatchingService) carServiceMatchWithImageURL(
	ctx context.Context,
	match *domain.CarServiceMatch,
) (*domain.CarServiceMatchWithImageURL, error) {
	item := &domain.CarServiceMatchWithImageURL{Match: match}
	if match == nil || match.PrimaryImage == nil {
		return item, nil
	}

	if s.fileRepo == nil || s.s3Cfg == nil {
		return nil, domain.ErrInternal
	}

	expiresAt := time.Now().Add(s.s3Cfg.PresignedURLTTL)
	url, err := s.fileRepo.GetPresignedURL(
		ctx,
		s.s3Cfg.BucketUploads,
		match.PrimaryImage.S3Key,
		s.s3Cfg.PresignedURLTTL,
	)
	if err != nil {
		return nil, err
	}

	item.PrimaryImageURL = url
	item.PrimaryImageExpiresAt = &expiresAt

	return item, nil
}
