package dto

import (
	"time"

	"github.com/DedovInside/AutoInspect/backend/internal/domain"
	"github.com/google/uuid"
)

type VehicleMakeResponse struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type VehicleModelResponse struct {
	ID        uuid.UUID `json:"id"`
	MakeID    uuid.UUID `json:"make_id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type VehicleGenerationResponse struct {
	ID          uuid.UUID `json:"id"`
	ModelID     uuid.UUID `json:"model_id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	YearFrom    int       `json:"year_from"`
	YearTo      int       `json:"year_to,omitempty"`
	YearOptions []int     `json:"year_options"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type AdminVehicleMakeResponse struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type AdminVehicleModelResponse struct {
	ID        uuid.UUID `json:"id"`
	MakeID    uuid.UUID `json:"make_id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type AdminVehicleGenerationResponse struct {
	ID        uuid.UUID `json:"id"`
	ModelID   uuid.UUID `json:"model_id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	YearFrom  int       `json:"year_from"`
	YearTo    int       `json:"year_to,omitempty"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type VehicleMakeListResponse struct {
	Items []VehicleMakeResponse `json:"items"`
	Meta  ListMeta              `json:"meta"`
}

type VehicleModelListResponse struct {
	Items []VehicleModelResponse `json:"items"`
	Meta  ListMeta               `json:"meta"`
}

type VehicleGenerationListResponse struct {
	Items []VehicleGenerationResponse `json:"items"`
	Meta  ListMeta                    `json:"meta"`
}

type AdminVehicleMakeListResponse struct {
	Items []AdminVehicleMakeResponse `json:"items"`
	Meta  ListMeta                   `json:"meta"`
}

type AdminVehicleModelListResponse struct {
	Items []AdminVehicleModelResponse `json:"items"`
	Meta  ListMeta                    `json:"meta"`
}

type AdminVehicleGenerationListResponse struct {
	Items []AdminVehicleGenerationResponse `json:"items"`
	Meta  ListMeta                         `json:"meta"`
}

type CreateVehicleMakeRequest struct {
	Name string `json:"name" binding:"required,min=1,max=100"`
	Slug string `json:"slug" binding:"omitempty,max=120"`
}

type UpdateVehicleMakeRequest struct {
	Name string `json:"name" binding:"required,min=1,max=100"`
	Slug string `json:"slug" binding:"omitempty,max=120"`
}

type SetVehicleCatalogActiveRequest struct {
	IsActive bool `json:"is_active"`
}

type CreateVehicleModelRequest struct {
	MakeID uuid.UUID `json:"make_id" binding:"required"`
	Name   string    `json:"name" binding:"required,min=1,max=100"`
	Slug   string    `json:"slug" binding:"omitempty,max=120"`
}

type UpdateVehicleModelRequest struct {
	MakeID uuid.UUID `json:"make_id" binding:"required"`
	Name   string    `json:"name" binding:"required,min=1,max=100"`
	Slug   string    `json:"slug" binding:"omitempty,max=120"`
}

type CreateVehicleGenerationRequest struct {
	ModelID  uuid.UUID `json:"model_id" binding:"required"`
	Name     string    `json:"name" binding:"required,min=1,max=100"`
	Slug     string    `json:"slug" binding:"omitempty,max=120"`
	YearFrom int       `json:"year_from" binding:"required,min=1900,max=2100"`
	YearTo   int       `json:"year_to" binding:"omitempty,min=1900,max=2100"`
}

type UpdateVehicleGenerationRequest struct {
	ModelID  uuid.UUID `json:"model_id" binding:"required"`
	Name     string    `json:"name" binding:"required,min=1,max=100"`
	Slug     string    `json:"slug" binding:"omitempty,max=120"`
	YearFrom int       `json:"year_from" binding:"required,min=1900,max=2100"`
	YearTo   int       `json:"year_to" binding:"omitempty,min=1900,max=2100"`
}

func ToVehicleMakeResponse(vehicleMake *domain.VehicleMake) VehicleMakeResponse {
	if vehicleMake == nil {
		return VehicleMakeResponse{}
	}

	return VehicleMakeResponse{
		ID:        vehicleMake.ID,
		Name:      vehicleMake.Name,
		Slug:      vehicleMake.Slug,
		CreatedAt: vehicleMake.CreatedAt,
		UpdatedAt: vehicleMake.UpdatedAt,
	}
}

func ToVehicleMakeResponseList(makes []*domain.VehicleMake) []VehicleMakeResponse {
	out := make([]VehicleMakeResponse, 0, len(makes))
	for _, vehicleMake := range makes {
		if vehicleMake == nil {
			continue
		}
		out = append(out, ToVehicleMakeResponse(vehicleMake))
	}

	return out
}

func ToAdminVehicleMakeResponse(vehicleMake *domain.VehicleMake) AdminVehicleMakeResponse {
	if vehicleMake == nil {
		return AdminVehicleMakeResponse{}
	}

	return AdminVehicleMakeResponse{
		ID:        vehicleMake.ID,
		Name:      vehicleMake.Name,
		Slug:      vehicleMake.Slug,
		IsActive:  vehicleMake.IsActive,
		CreatedAt: vehicleMake.CreatedAt,
		UpdatedAt: vehicleMake.UpdatedAt,
	}
}

func ToAdminVehicleMakeResponseList(makes []*domain.VehicleMake) []AdminVehicleMakeResponse {
	out := make([]AdminVehicleMakeResponse, 0, len(makes))
	for _, vehicleMake := range makes {
		if vehicleMake == nil {
			continue
		}
		out = append(out, ToAdminVehicleMakeResponse(vehicleMake))
	}

	return out
}

func ToVehicleModelResponse(model *domain.VehicleModel) VehicleModelResponse {
	if model == nil {
		return VehicleModelResponse{}
	}

	return VehicleModelResponse{
		ID:        model.ID,
		MakeID:    model.MakeID,
		Name:      model.Name,
		Slug:      model.Slug,
		CreatedAt: model.CreatedAt,
		UpdatedAt: model.UpdatedAt,
	}
}

func ToVehicleModelResponseList(models []*domain.VehicleModel) []VehicleModelResponse {
	out := make([]VehicleModelResponse, 0, len(models))
	for _, model := range models {
		if model == nil {
			continue
		}
		out = append(out, ToVehicleModelResponse(model))
	}

	return out
}

func ToAdminVehicleModelResponse(model *domain.VehicleModel) AdminVehicleModelResponse {
	if model == nil {
		return AdminVehicleModelResponse{}
	}

	return AdminVehicleModelResponse{
		ID:        model.ID,
		MakeID:    model.MakeID,
		Name:      model.Name,
		Slug:      model.Slug,
		IsActive:  model.IsActive,
		CreatedAt: model.CreatedAt,
		UpdatedAt: model.UpdatedAt,
	}
}

func ToAdminVehicleModelResponseList(models []*domain.VehicleModel) []AdminVehicleModelResponse {
	out := make([]AdminVehicleModelResponse, 0, len(models))
	for _, model := range models {
		if model == nil {
			continue
		}
		out = append(out, ToAdminVehicleModelResponse(model))
	}

	return out
}

func ToVehicleGenerationResponse(generation *domain.VehicleGeneration, yearOptions []int) VehicleGenerationResponse {
	if generation == nil {
		return VehicleGenerationResponse{}
	}

	return VehicleGenerationResponse{
		ID:          generation.ID,
		ModelID:     generation.ModelID,
		Name:        generation.Name,
		Slug:        generation.Slug,
		YearFrom:    generation.YearFrom,
		YearTo:      generation.YearTo,
		YearOptions: yearOptions,
		CreatedAt:   generation.CreatedAt,
		UpdatedAt:   generation.UpdatedAt,
	}
}

func ToAdminVehicleGenerationResponse(generation *domain.VehicleGeneration) AdminVehicleGenerationResponse {
	if generation == nil {
		return AdminVehicleGenerationResponse{}
	}

	return AdminVehicleGenerationResponse{
		ID:        generation.ID,
		ModelID:   generation.ModelID,
		Name:      generation.Name,
		Slug:      generation.Slug,
		YearFrom:  generation.YearFrom,
		YearTo:    generation.YearTo,
		IsActive:  generation.IsActive,
		CreatedAt: generation.CreatedAt,
		UpdatedAt: generation.UpdatedAt,
	}
}

func ToAdminVehicleGenerationResponseList(
	generations []*domain.VehicleGeneration,
) []AdminVehicleGenerationResponse {
	out := make([]AdminVehicleGenerationResponse, 0, len(generations))
	for _, generation := range generations {
		if generation == nil {
			continue
		}
		out = append(out, ToAdminVehicleGenerationResponse(generation))
	}

	return out
}

func NewVehicleMakeListResponse(items []VehicleMakeResponse) VehicleMakeListResponse {
	return VehicleMakeListResponse{
		Items: items,
		Meta:  ListMeta{Count: len(items)},
	}
}

func NewAdminVehicleMakeListResponse(items []AdminVehicleMakeResponse) AdminVehicleMakeListResponse {
	return AdminVehicleMakeListResponse{
		Items: items,
		Meta:  ListMeta{Count: len(items)},
	}
}

func NewAdminVehicleModelListResponse(items []AdminVehicleModelResponse) AdminVehicleModelListResponse {
	return AdminVehicleModelListResponse{
		Items: items,
		Meta:  ListMeta{Count: len(items)},
	}
}

func NewAdminVehicleGenerationListResponse(
	items []AdminVehicleGenerationResponse,
) AdminVehicleGenerationListResponse {
	return AdminVehicleGenerationListResponse{
		Items: items,
		Meta:  ListMeta{Count: len(items)},
	}
}

func NewVehicleModelListResponse(items []VehicleModelResponse) VehicleModelListResponse {
	return VehicleModelListResponse{
		Items: items,
		Meta:  ListMeta{Count: len(items)},
	}
}

func NewVehicleGenerationListResponse(items []VehicleGenerationResponse) VehicleGenerationListResponse {
	return VehicleGenerationListResponse{
		Items: items,
		Meta:  ListMeta{Count: len(items)},
	}
}
