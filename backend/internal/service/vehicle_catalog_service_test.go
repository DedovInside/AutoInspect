package service

import (
	"context"
	"testing"
	"time"

	"github.com/DedovInside/AutoInspect/backend/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestVehicleCatalogServiceResolveCarInfo(t *testing.T) {
	generationID := uuid.New()
	repo := &fakeVehicleCatalogRepo{
		details: &domain.VehicleGenerationDetails{
			Make: &domain.VehicleMake{
				ID:   uuid.New(),
				Name: "Toyota",
				Slug: "toyota",
			},
			Model: &domain.VehicleModel{
				ID:   uuid.New(),
				Name: "Camry",
				Slug: "camry",
			},
			Generation: &domain.VehicleGeneration{
				ID:       generationID,
				Name:     "XV70",
				Slug:     "xv70",
				YearFrom: 2017,
				YearTo:   2024,
			},
		},
	}
	service := NewVehicleCatalogService(repo)

	carInfo, err := service.ResolveCarInfo(context.Background(), ResolveVehicleCarInput{
		GenerationID: generationID,
		Year:         2020,
	})

	require.NoError(t, err)
	require.Equal(t, domain.CarInfo{
		Make:       "Toyota",
		Model:      "Camry",
		Generation: "XV70",
		Year:       2020,
	}, carInfo)
	require.Equal(t, generationID, repo.requestedGenerationID)
}

func TestVehicleCatalogServiceResolveCarInfoRejectsYearOutsideGenerationRange(t *testing.T) {
	generationID := uuid.New()
	service := NewVehicleCatalogService(&fakeVehicleCatalogRepo{
		details: &domain.VehicleGenerationDetails{
			Make:  &domain.VehicleMake{Name: "Volkswagen"},
			Model: &domain.VehicleModel{Name: "Polo"},
			Generation: &domain.VehicleGeneration{
				ID:       generationID,
				Name:     "5",
				YearFrom: 2009,
				YearTo:   2015,
			},
		},
	})

	_, err := service.ResolveCarInfo(context.Background(), ResolveVehicleCarInput{
		GenerationID: generationID,
		Year:         2020,
	})

	require.ErrorIs(t, err, domain.ErrInvalidInput)
}

func TestVehicleCatalogServiceRejectsEmptyIDs(t *testing.T) {
	service := NewVehicleCatalogService(&fakeVehicleCatalogRepo{})

	_, err := service.ListModels(context.Background(), uuid.Nil)
	require.ErrorIs(t, err, domain.ErrInvalidInput)

	_, err = service.ListGenerations(context.Background(), uuid.Nil)
	require.ErrorIs(t, err, domain.ErrInvalidInput)

	_, err = service.GetGenerationDetails(context.Background(), uuid.Nil)
	require.ErrorIs(t, err, domain.ErrInvalidInput)

	_, err = service.ResolveCarInfo(context.Background(), ResolveVehicleCarInput{})
	require.ErrorIs(t, err, domain.ErrInvalidInput)
}

func TestVehicleCatalogServiceYearOptionsUsesCurrentYearForOpenEndedGeneration(t *testing.T) {
	service := NewVehicleCatalogService(&fakeVehicleCatalogRepo{})
	service.now = func() time.Time {
		return time.Date(2026, time.July, 10, 0, 0, 0, 0, time.UTC)
	}

	years := service.YearOptions(&domain.VehicleGeneration{
		YearFrom: 2024,
	})

	require.Equal(t, []int{2026, 2025, 2024}, years)
}

func TestVehicleCatalogServiceCreateMakeTrimsAndGeneratesSlug(t *testing.T) {
	repo := &fakeVehicleCatalogRepo{}
	service := NewVehicleCatalogService(repo)
	service.now = func() time.Time {
		return time.Date(2026, time.July, 10, 12, 0, 0, 0, time.UTC)
	}

	got, err := service.CreateMake(context.Background(), &domain.CreateVehicleMakeInput{
		Name: " Volkswagen ",
	})

	require.NoError(t, err)
	require.Equal(t, "Volkswagen", got.Name)
	require.Equal(t, "volkswagen", got.Slug)
	require.True(t, got.IsActive)
	require.Equal(t, got, repo.createdMake)
}

func TestVehicleCatalogServiceCreateGenerationRejectsInvalidYears(t *testing.T) {
	service := NewVehicleCatalogService(&fakeVehicleCatalogRepo{})

	_, err := service.CreateGeneration(context.Background(), &domain.CreateVehicleGenerationInput{
		ModelID:  uuid.New(),
		Name:     "XV70",
		YearFrom: 2024,
		YearTo:   2020,
	})

	require.ErrorIs(t, err, domain.ErrInvalidInput)
}

type fakeVehicleCatalogRepo struct {
	makes                 []*domain.VehicleMake
	models                []*domain.VehicleModel
	generations           []*domain.VehicleGeneration
	details               *domain.VehicleGenerationDetails
	err                   error
	createdMake           *domain.VehicleMake
	createdModel          *domain.VehicleModel
	createdGeneration     *domain.VehicleGeneration
	requestedMakeID       uuid.UUID
	requestedModelID      uuid.UUID
	requestedGenerationID uuid.UUID
}

func (r *fakeVehicleCatalogRepo) ListMakes(context.Context) ([]*domain.VehicleMake, error) {
	return r.makes, r.err
}

func (r *fakeVehicleCatalogRepo) ListMakesForAdmin(context.Context) ([]*domain.VehicleMake, error) {
	return r.makes, r.err
}

func (r *fakeVehicleCatalogRepo) GetMakeByID(context.Context, uuid.UUID) (*domain.VehicleMake, error) {
	if len(r.makes) == 0 {
		return nil, r.err
	}
	return r.makes[0], r.err
}

func (r *fakeVehicleCatalogRepo) CreateMake(_ context.Context, vehicleMake *domain.VehicleMake) error {
	r.createdMake = vehicleMake
	return r.err
}

func (r *fakeVehicleCatalogRepo) UpdateMake(context.Context, domain.UpdateVehicleMakeInput) error {
	return r.err
}

func (r *fakeVehicleCatalogRepo) SetMakeActive(context.Context, domain.SetVehicleMakeActiveInput) error {
	return r.err
}

func (r *fakeVehicleCatalogRepo) ListModelsByMakeID(_ context.Context,
	makeID uuid.UUID) ([]*domain.VehicleModel, error) {
	r.requestedMakeID = makeID
	return r.models, r.err
}

func (r *fakeVehicleCatalogRepo) ListModelsByMakeIDForAdmin(_ context.Context,
	makeID uuid.UUID) ([]*domain.VehicleModel, error) {
	r.requestedMakeID = makeID
	return r.models, r.err
}

func (r *fakeVehicleCatalogRepo) GetModelByID(context.Context, uuid.UUID) (*domain.VehicleModel, error) {
	if len(r.models) == 0 {
		return nil, r.err
	}
	return r.models[0], r.err
}

func (r *fakeVehicleCatalogRepo) CreateModel(_ context.Context, model *domain.VehicleModel) error {
	r.createdModel = model
	return r.err
}

func (r *fakeVehicleCatalogRepo) UpdateModel(context.Context, domain.UpdateVehicleModelInput) error {
	return r.err
}

func (r *fakeVehicleCatalogRepo) SetModelActive(context.Context, domain.SetVehicleModelActiveInput) error {
	return r.err
}

func (r *fakeVehicleCatalogRepo) ListGenerationsByModelID(_ context.Context,
	modelID uuid.UUID) ([]*domain.VehicleGeneration, error) {
	r.requestedModelID = modelID
	return r.generations, r.err
}

func (r *fakeVehicleCatalogRepo) ListGenerationsByModelIDForAdmin(_ context.Context,
	modelID uuid.UUID) ([]*domain.VehicleGeneration, error) {
	r.requestedModelID = modelID
	return r.generations, r.err
}

func (r *fakeVehicleCatalogRepo) GetGenerationByID(context.Context, uuid.UUID) (*domain.VehicleGeneration, error) {
	if len(r.generations) == 0 {
		return nil, r.err
	}
	return r.generations[0], r.err
}

func (r *fakeVehicleCatalogRepo) GetGenerationDetailsByID(_ context.Context,
	generationID uuid.UUID) (*domain.VehicleGenerationDetails, error) {
	r.requestedGenerationID = generationID
	return r.details, r.err
}

func (r *fakeVehicleCatalogRepo) CreateGeneration(_ context.Context,
	generation *domain.VehicleGeneration) error {
	r.createdGeneration = generation
	return r.err
}

func (r *fakeVehicleCatalogRepo) UpdateGeneration(context.Context,
	*domain.UpdateVehicleGenerationInput) error {
	return r.err
}

func (r *fakeVehicleCatalogRepo) SetGenerationActive(context.Context,
	domain.SetVehicleGenerationActiveInput) error {
	return r.err
}
