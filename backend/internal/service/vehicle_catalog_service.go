package service

import (
	"context"
	"strings"
	"time"

	"github.com/DedovInside/AutoInspect/backend/internal/domain"
	"github.com/DedovInside/AutoInspect/backend/internal/repository"
	"github.com/google/uuid"
)

type VehicleCatalogService struct {
	catalogRepo repository.VehicleCatalogRepository
	now         func() time.Time
}

type ResolveVehicleCarInput struct {
	GenerationID uuid.UUID
	Year         int
}

func NewVehicleCatalogService(catalogRepo repository.VehicleCatalogRepository) *VehicleCatalogService {
	return &VehicleCatalogService{
		catalogRepo: catalogRepo,
		now:         time.Now,
	}
}

func (s *VehicleCatalogService) ListMakes(ctx context.Context) ([]*domain.VehicleMake, error) {
	if s == nil || s.catalogRepo == nil {
		return nil, domain.ErrInternal
	}

	return s.catalogRepo.ListMakes(ctx)
}

func (s *VehicleCatalogService) AdminListMakes(ctx context.Context) ([]*domain.VehicleMake, error) {
	if s == nil || s.catalogRepo == nil {
		return nil, domain.ErrInternal
	}

	return s.catalogRepo.ListMakesForAdmin(ctx)
}

func (s *VehicleCatalogService) CreateMake(ctx context.Context,
	input *domain.CreateVehicleMakeInput) (*domain.VehicleMake, error) {
	name, slugValue, err := normalizeVehicleCatalogNameAndSlug(inputName(input), inputSlug(input))
	if err != nil {
		return nil, err
	}

	now := s.currentTime()
	vehicleMake := &domain.VehicleMake{
		ID:        uuid.New(),
		Name:      name,
		Slug:      slugValue,
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.catalogRepo.CreateMake(ctx, vehicleMake); err != nil {
		return nil, err
	}

	return vehicleMake, nil
}

func (s *VehicleCatalogService) UpdateMake(ctx context.Context,
	input *domain.UpdateVehicleMakeInput) (*domain.VehicleMake, error) {
	if input == nil || input.ID == uuid.Nil {
		return nil, domain.ErrInvalidInput
	}

	name, slugValue, err := normalizeVehicleCatalogNameAndSlug(input.Name, input.Slug)
	if err != nil {
		return nil, err
	}

	update := domain.UpdateVehicleMakeInput{
		ID:   input.ID,
		Name: name,
		Slug: slugValue,
	}
	if err := s.catalogRepo.UpdateMake(ctx, update); err != nil {
		return nil, err
	}

	return s.catalogRepo.GetMakeByID(ctx, input.ID)
}

func (s *VehicleCatalogService) SetMakeActive(ctx context.Context,
	input domain.SetVehicleMakeActiveInput) error {
	if input.ID == uuid.Nil {
		return domain.ErrInvalidInput
	}

	return s.catalogRepo.SetMakeActive(ctx, input)
}

func (s *VehicleCatalogService) ListModels(ctx context.Context, makeID uuid.UUID) ([]*domain.VehicleModel, error) {
	if s == nil || s.catalogRepo == nil {
		return nil, domain.ErrInternal
	}

	if makeID == uuid.Nil {
		return nil, domain.ErrInvalidInput
	}

	return s.catalogRepo.ListModelsByMakeID(ctx, makeID)
}

func (s *VehicleCatalogService) AdminListModels(ctx context.Context,
	makeID uuid.UUID) ([]*domain.VehicleModel, error) {
	if s == nil || s.catalogRepo == nil {
		return nil, domain.ErrInternal
	}

	if makeID == uuid.Nil {
		return nil, domain.ErrInvalidInput
	}

	return s.catalogRepo.ListModelsByMakeIDForAdmin(ctx, makeID)
}

func (s *VehicleCatalogService) CreateModel(ctx context.Context,
	input *domain.CreateVehicleModelInput) (*domain.VehicleModel, error) {
	if input == nil || input.MakeID == uuid.Nil {
		return nil, domain.ErrInvalidInput
	}

	name, slugValue, err := normalizeVehicleCatalogNameAndSlug(input.Name, input.Slug)
	if err != nil {
		return nil, err
	}

	now := s.currentTime()
	model := &domain.VehicleModel{
		ID:        uuid.New(),
		MakeID:    input.MakeID,
		Name:      name,
		Slug:      slugValue,
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.catalogRepo.CreateModel(ctx, model); err != nil {
		return nil, err
	}

	return model, nil
}

func (s *VehicleCatalogService) UpdateModel(ctx context.Context,
	input *domain.UpdateVehicleModelInput) (*domain.VehicleModel, error) {
	if input == nil || input.ID == uuid.Nil || input.MakeID == uuid.Nil {
		return nil, domain.ErrInvalidInput
	}

	name, slugValue, err := normalizeVehicleCatalogNameAndSlug(input.Name, input.Slug)
	if err != nil {
		return nil, err
	}

	update := domain.UpdateVehicleModelInput{
		ID:     input.ID,
		MakeID: input.MakeID,
		Name:   name,
		Slug:   slugValue,
	}
	if err := s.catalogRepo.UpdateModel(ctx, update); err != nil {
		return nil, err
	}

	return s.catalogRepo.GetModelByID(ctx, input.ID)
}

func (s *VehicleCatalogService) SetModelActive(ctx context.Context,
	input domain.SetVehicleModelActiveInput) error {
	if input.ID == uuid.Nil {
		return domain.ErrInvalidInput
	}

	return s.catalogRepo.SetModelActive(ctx, input)
}

func (s *VehicleCatalogService) ListGenerations(ctx context.Context,
	modelID uuid.UUID) ([]*domain.VehicleGeneration, error) {
	if s == nil || s.catalogRepo == nil {
		return nil, domain.ErrInternal
	}

	if modelID == uuid.Nil {
		return nil, domain.ErrInvalidInput
	}

	return s.catalogRepo.ListGenerationsByModelID(ctx, modelID)
}

func (s *VehicleCatalogService) AdminListGenerations(ctx context.Context,
	modelID uuid.UUID) ([]*domain.VehicleGeneration, error) {
	if s == nil || s.catalogRepo == nil {
		return nil, domain.ErrInternal
	}

	if modelID == uuid.Nil {
		return nil, domain.ErrInvalidInput
	}

	return s.catalogRepo.ListGenerationsByModelIDForAdmin(ctx, modelID)
}

func (s *VehicleCatalogService) CreateGeneration(ctx context.Context,
	input *domain.CreateVehicleGenerationInput) (*domain.VehicleGeneration, error) {
	if err := validateCreateVehicleGenerationInput(input); err != nil {
		return nil, err
	}

	name, slugValue, err := normalizeVehicleCatalogNameAndSlug(input.Name, input.Slug)
	if err != nil {
		return nil, err
	}

	now := s.currentTime()
	generation := &domain.VehicleGeneration{
		ID:        uuid.New(),
		ModelID:   input.ModelID,
		Name:      name,
		Slug:      slugValue,
		YearFrom:  input.YearFrom,
		YearTo:    input.YearTo,
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.catalogRepo.CreateGeneration(ctx, generation); err != nil {
		return nil, err
	}

	return generation, nil
}

func (s *VehicleCatalogService) UpdateGeneration(ctx context.Context,
	input *domain.UpdateVehicleGenerationInput) (*domain.VehicleGeneration, error) {
	if err := validateUpdateVehicleGenerationInput(input); err != nil {
		return nil, err
	}

	name, slugValue, err := normalizeVehicleCatalogNameAndSlug(input.Name, input.Slug)
	if err != nil {
		return nil, err
	}

	update := domain.UpdateVehicleGenerationInput{
		ID:       input.ID,
		ModelID:  input.ModelID,
		Name:     name,
		Slug:     slugValue,
		YearFrom: input.YearFrom,
		YearTo:   input.YearTo,
	}
	if err := s.catalogRepo.UpdateGeneration(ctx, &update); err != nil {
		return nil, err
	}

	return s.catalogRepo.GetGenerationByID(ctx, input.ID)
}

func (s *VehicleCatalogService) SetGenerationActive(ctx context.Context,
	input domain.SetVehicleGenerationActiveInput) error {
	if input.ID == uuid.Nil {
		return domain.ErrInvalidInput
	}

	return s.catalogRepo.SetGenerationActive(ctx, input)
}

func (s *VehicleCatalogService) GetGenerationDetails(ctx context.Context,
	generationID uuid.UUID) (*domain.VehicleGenerationDetails, error) {
	if s == nil || s.catalogRepo == nil {
		return nil, domain.ErrInternal
	}

	if generationID == uuid.Nil {
		return nil, domain.ErrInvalidInput
	}

	return s.catalogRepo.GetGenerationDetailsByID(ctx, generationID)
}

func (s *VehicleCatalogService) ResolveCarInfo(ctx context.Context,
	input ResolveVehicleCarInput) (domain.CarInfo, error) {
	if input.GenerationID == uuid.Nil || input.Year <= 0 {
		return domain.CarInfo{}, domain.ErrInvalidInput
	}

	details, err := s.GetGenerationDetails(ctx, input.GenerationID)
	if err != nil {
		return domain.CarInfo{}, err
	}

	if details == nil || details.Generation == nil {
		return domain.CarInfo{}, domain.ErrNotFound
	}

	if !details.Generation.ContainsYear(input.Year) {
		return domain.CarInfo{}, domain.ErrInvalidInput
	}

	return details.ToCarInfo(input.Year), nil
}

func (s *VehicleCatalogService) YearOptions(generation *domain.VehicleGeneration) []int {
	if generation == nil {
		return nil
	}

	currentYear := time.Now().Year()
	if s != nil && s.now != nil {
		currentYear = s.now().Year()
	}

	return generation.YearOptions(currentYear)
}

func (s *VehicleCatalogService) currentTime() time.Time {
	if s != nil && s.now != nil {
		return s.now().UTC()
	}

	return time.Now().UTC()
}

func normalizeVehicleCatalogNameAndSlug(nameValue, slugValue string) (
	normalizedName string,
	normalizedSlug string,
	err error,
) {
	nameValue = strings.TrimSpace(nameValue)
	slugValue = strings.TrimSpace(slugValue)

	if nameValue == "" {
		return "", "", domain.ErrInvalidInput
	}

	if slugValue == "" {
		slugValue = slug(nameValue)
	} else {
		slugValue = slug(slugValue)
	}

	if slugValue == "" || slugValue == defaultSlug {
		return "", "", domain.ErrInvalidInput
	}

	return nameValue, slugValue, nil
}

func validateCreateVehicleGenerationInput(input *domain.CreateVehicleGenerationInput) error {
	if input == nil || input.ModelID == uuid.Nil {
		return domain.ErrInvalidInput
	}

	return validateVehicleGenerationYears(input.YearFrom, input.YearTo)
}

func validateUpdateVehicleGenerationInput(input *domain.UpdateVehicleGenerationInput) error {
	if input == nil || input.ID == uuid.Nil || input.ModelID == uuid.Nil {
		return domain.ErrInvalidInput
	}

	return validateVehicleGenerationYears(input.YearFrom, input.YearTo)
}

func validateVehicleGenerationYears(yearFrom, yearTo int) error {
	if yearFrom < domain.VehicleYearMin || yearFrom > domain.VehicleYearMax {
		return domain.ErrInvalidInput
	}

	if yearTo != 0 {
		if yearTo < domain.VehicleYearMin || yearTo > domain.VehicleYearMax || yearTo < yearFrom {
			return domain.ErrInvalidInput
		}
	}

	return nil
}

func inputName(input *domain.CreateVehicleMakeInput) string {
	if input == nil {
		return ""
	}
	return input.Name
}

func inputSlug(input *domain.CreateVehicleMakeInput) string {
	if input == nil {
		return ""
	}
	return input.Slug
}
