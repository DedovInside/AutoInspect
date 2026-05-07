package service

import (
	"context"
	"fmt"
	"mime/multipart"
	"path"
	"strings"
	"time"

	"github.com/DedovInside/AutoInspect/backend/internal/config"
	"github.com/DedovInside/AutoInspect/backend/internal/domain"
	"github.com/DedovInside/AutoInspect/backend/internal/repository"
	"github.com/DedovInside/AutoInspect/backend/internal/repository/postgres"
	"github.com/google/uuid"
)

const (
	maxCarServiceImageSizeBytes = 10 << 20
	imageContentTypeJPEG        = "image/jpeg"
	imageContentTypePNG         = "image/png"
	imageContentTypeWEBP        = "image/webp"
)

type CarServiceProfileService struct {
	profileRepo repository.CarServiceProfileRepository
	imageRepo   repository.CarServiceImageRepository
	fileRepo    repository.FileRepository
	db          *postgres.DB
	s3Cfg       *config.S3Config
}

type UploadCarServiceImageInput struct {
	UserID           uuid.UUID
	File             multipart.File
	OriginalFilename string
	IsPrimary        bool
}

type CarServiceImageWithURL struct {
	Image     *domain.CarServiceImage
	URL       string
	ExpiresAt time.Time
}

func NewCarServiceProfileService(
	db *postgres.DB,
	profileRepo repository.CarServiceProfileRepository,
	imageRepo repository.CarServiceImageRepository,
	fileRepo repository.FileRepository,
	s3Cfg *config.S3Config,
) *CarServiceProfileService {
	return &CarServiceProfileService{
		profileRepo: profileRepo,
		imageRepo:   imageRepo,
		fileRepo:    fileRepo,
		db:          db,
		s3Cfg:       s3Cfg,
	}
}

func (s *CarServiceProfileService) GetMyProfile(
	ctx context.Context,
	userID uuid.UUID,
) (*domain.CarServiceProfile, error) {
	if userID == uuid.Nil {
		return nil, domain.ErrInvalidInput
	}

	return s.profileRepo.GetByUserID(ctx, userID)
}

func (s *CarServiceProfileService) UpdateMyProfile(
	ctx context.Context,
	input *domain.UpdateCarServiceProfileInput,
) (*domain.CarServiceProfile, error) {
	if err := validateUpdateCarServiceProfileInput(input); err != nil {
		return nil, err
	}

	normalized := normalizeUpdateCarServiceProfileInput(input)
	if err := s.profileRepo.Update(ctx, normalized); err != nil {
		return nil, err
	}

	return s.profileRepo.GetByUserID(ctx, normalized.UserID)
}

func (s *CarServiceProfileService) SetMyProfileActive(
	ctx context.Context,
	userID uuid.UUID,
	isActive bool,
) (*domain.CarServiceProfile, error) {
	if userID == uuid.Nil {
		return nil, domain.ErrInvalidInput
	}

	if err := s.profileRepo.SetActive(ctx, userID, isActive); err != nil {
		return nil, err
	}

	return s.profileRepo.GetByUserID(ctx, userID)
}

func (s *CarServiceProfileService) UploadImage(
	ctx context.Context,
	input *UploadCarServiceImageInput,
) (*CarServiceImageWithURL, error) {
	if err := validateUploadCarServiceImageInput(input); err != nil {
		return nil, err
	}

	if s.fileRepo == nil || s.s3Cfg == nil {
		return nil, domain.ErrInternal
	}

	profile, err := s.profileRepo.GetByUserID(ctx, input.UserID)
	if err != nil {
		return nil, err
	}

	contentType, size, err := detectUploadMeta(input.File)
	if err != nil {
		return nil, fmt.Errorf("inspect car service image: %w", err)
	}

	if err := validateCarServiceImageMeta(contentType, size); err != nil {
		return nil, err
	}

	sortOrder, err := s.imageRepo.NextSortOrder(ctx, profile.ID)
	if err != nil {
		return nil, err
	}

	imageID := uuid.New()
	objectKey := carServiceImageObjectKey(profile.ID, imageID, contentType)
	now := time.Now().UTC()
	image := &domain.CarServiceImage{
		ID:               imageID,
		ProfileID:        profile.ID,
		S3Key:            objectKey,
		IsPrimary:        input.IsPrimary,
		SortOrder:        sortOrder,
		OriginalFilename: safeOriginalFilename(input.OriginalFilename),
		ContentType:      contentType,
		SizeBytes:        size,
		CreatedAt:        now,
	}

	if err := s.fileRepo.Upload(ctx, s.s3Cfg.BucketUploads, objectKey, input.File, contentType, size); err != nil {
		return nil, fmt.Errorf("upload car service image: %w", err)
	}

	if err := s.createCarServiceImageRecord(ctx, image); err != nil {
		_ = s.fileRepo.Delete(ctx, s.s3Cfg.BucketUploads, objectKey)
		return nil, err
	}

	return s.carServiceImageWithURL(ctx, image)
}

func (s *CarServiceProfileService) ListImages(
	ctx context.Context,
	userID uuid.UUID,
) ([]*CarServiceImageWithURL, error) {
	if userID == uuid.Nil {
		return nil, domain.ErrInvalidInput
	}

	profile, err := s.profileRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	images, err := s.imageRepo.ListByProfileID(ctx, profile.ID)
	if err != nil {
		return nil, err
	}

	return s.carServiceImagesWithURLs(ctx, images)
}

func (s *CarServiceProfileService) SetPrimaryImage(
	ctx context.Context,
	userID, imageID uuid.UUID,
) (*CarServiceImageWithURL, error) {
	profile, image, err := s.getOwnedCarServiceImage(ctx, userID, imageID)
	if err != nil {
		return nil, err
	}

	if err := s.setPrimaryCarServiceImage(ctx, profile.ID, image.ID); err != nil {
		return nil, err
	}

	image.IsPrimary = true
	return s.carServiceImageWithURL(ctx, image)
}

func (s *CarServiceProfileService) DeleteImage(
	ctx context.Context,
	userID, imageID uuid.UUID,
) error {
	_, image, err := s.getOwnedCarServiceImage(ctx, userID, imageID)
	if err != nil {
		return err
	}

	if err := s.imageRepo.Delete(ctx, image.ID); err != nil {
		return err
	}

	_ = s.fileRepo.Delete(ctx, s.s3Cfg.BucketUploads, image.S3Key)
	return nil
}

func (s *CarServiceProfileService) createCarServiceImageRecord(
	ctx context.Context,
	image *domain.CarServiceImage,
) error {
	if image.IsPrimary {
		return s.createPrimaryCarServiceImageRecord(ctx, image)
	}
	return s.imageRepo.Create(ctx, image)
}

func (s *CarServiceProfileService) createPrimaryCarServiceImageRecord(
	ctx context.Context,
	image *domain.CarServiceImage,
) error {
	if s.db == nil {
		return domain.ErrInternal
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return domain.ErrInternal
	}
	defer func() { _ = tx.Rollback(ctx) }()

	imageRepo := postgres.NewCarServiceImageRepo(tx)
	if err := imageRepo.ClearPrimary(ctx, image.ProfileID); err != nil {
		return err
	}

	if err := imageRepo.Create(ctx, image); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.ErrInternal
	}

	return nil
}

func (s *CarServiceProfileService) setPrimaryCarServiceImage(
	ctx context.Context,
	profileID, imageID uuid.UUID,
) error {
	if s.db == nil {
		return domain.ErrInternal
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return domain.ErrInternal
	}
	defer func() { _ = tx.Rollback(ctx) }()

	imageRepo := postgres.NewCarServiceImageRepo(tx)
	if err := imageRepo.ClearPrimary(ctx, profileID); err != nil {
		return err
	}

	if err := imageRepo.SetPrimary(ctx, imageID); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.ErrInternal
	}

	return nil
}

func (s *CarServiceProfileService) getOwnedCarServiceImage(
	ctx context.Context,
	userID, imageID uuid.UUID,
) (*domain.CarServiceProfile, *domain.CarServiceImage, error) {
	if userID == uuid.Nil || imageID == uuid.Nil {
		return nil, nil, domain.ErrInvalidInput
	}

	profile, err := s.profileRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, nil, err
	}

	image, err := s.imageRepo.GetByID(ctx, imageID)
	if err != nil {
		return nil, nil, err
	}

	if image.ProfileID != profile.ID {
		return nil, nil, domain.ErrForbidden
	}

	return profile, image, nil
}

func (s *CarServiceProfileService) carServiceImagesWithURLs(
	ctx context.Context,
	images []*domain.CarServiceImage,
) ([]*CarServiceImageWithURL, error) {
	out := make([]*CarServiceImageWithURL, 0, len(images))
	for _, image := range images {
		item, err := s.carServiceImageWithURL(ctx, image)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}

	return out, nil
}

func (s *CarServiceProfileService) carServiceImageWithURL(
	ctx context.Context,
	image *domain.CarServiceImage,
) (*CarServiceImageWithURL, error) {
	if image == nil || s.fileRepo == nil || s.s3Cfg == nil {
		return nil, domain.ErrInternal
	}

	expiresAt := time.Now().Add(s.s3Cfg.PresignedURLTTL)

	url, err := s.fileRepo.GetPresignedURL(ctx, s.s3Cfg.BucketUploads, image.S3Key, s.s3Cfg.PresignedURLTTL)
	if err != nil {
		return nil, err
	}

	return &CarServiceImageWithURL{
		Image:     image,
		URL:       url,
		ExpiresAt: expiresAt,
	}, nil
}

func validateUpdateCarServiceProfileInput(input *domain.UpdateCarServiceProfileInput) error {
	if input == nil ||
		input.UserID == uuid.Nil ||
		strings.TrimSpace(input.OrganizationName) == "" ||
		strings.TrimSpace(input.City) == "" ||
		strings.TrimSpace(input.Address) == "" {
		return domain.ErrInvalidInput
	}

	if normalizedOptionalString(input.Phone) == nil &&
		normalizedOptionalString(input.Email) == nil &&
		normalizedOptionalString(input.ContactInfo) == nil &&
		normalizedOptionalString(input.WebsiteURL) == nil {
		return domain.ErrInvalidInput
	}

	return nil
}

func normalizeUpdateCarServiceProfileInput(
	input *domain.UpdateCarServiceProfileInput,
) *domain.UpdateCarServiceProfileInput {
	return &domain.UpdateCarServiceProfileInput{
		UserID:           input.UserID,
		OrganizationName: strings.TrimSpace(input.OrganizationName),
		City:             strings.TrimSpace(input.City),
		Address:          strings.TrimSpace(input.Address),
		Phone:            normalizedOptionalString(input.Phone),
		Email:            normalizedOptionalString(input.Email),
		WebsiteURL:       normalizedOptionalString(input.WebsiteURL),
		ContactInfo:      normalizedOptionalString(input.ContactInfo),
		Description:      normalizedOptionalString(input.Description),
		IsActive:         input.IsActive,
	}
}

func validateUploadCarServiceImageInput(input *UploadCarServiceImageInput) error {
	if input == nil ||
		input.UserID == uuid.Nil ||
		input.File == nil ||
		strings.TrimSpace(input.OriginalFilename) == "" {
		return domain.ErrInvalidInput
	}

	return nil
}

func validateCarServiceImageMeta(contentType string, size int64) error {
	if size <= 0 || size > maxCarServiceImageSizeBytes {
		return domain.ErrInvalidImage
	}

	switch contentType {
	case imageContentTypeJPEG, imageContentTypePNG, imageContentTypeWEBP:
		return nil
	default:
		if strings.HasPrefix(contentType, "image/") {
			return domain.ErrInvalidImage
		}
		return domain.ErrInvalidImage
	}
}

func carServiceImageObjectKey(profileID, imageID uuid.UUID, contentType string) string {
	return path.Join(
		"car-services",
		profileID.String(),
		"images",
		imageID.String()+extensionByContentType(contentType),
	)
}

func safeOriginalFilename(filename string) string {
	name := strings.TrimSpace(filename)
	if name == "" {
		return "image"
	}

	return name
}
