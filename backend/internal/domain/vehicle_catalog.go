package domain

import (
	"time"

	"github.com/google/uuid"
)

const (
	VehicleYearMin = 1900
	VehicleYearMax = 2100
)

type VehicleMake struct {
	ID        uuid.UUID `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	Slug      string    `json:"slug" db:"slug"`
	IsActive  bool      `json:"is_active" db:"is_active"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

type VehicleModel struct {
	ID        uuid.UUID `json:"id" db:"id"`
	MakeID    uuid.UUID `json:"make_id" db:"make_id"`
	Name      string    `json:"name" db:"name"`
	Slug      string    `json:"slug" db:"slug"`
	IsActive  bool      `json:"is_active" db:"is_active"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

type VehicleGeneration struct {
	ID        uuid.UUID `json:"id" db:"id"`
	ModelID   uuid.UUID `json:"model_id" db:"model_id"`
	Name      string    `json:"name" db:"name"`
	Slug      string    `json:"slug" db:"slug"`
	YearFrom  int       `json:"year_from" db:"year_from"`
	YearTo    int       `json:"year_to,omitempty" db:"year_to"`
	IsActive  bool      `json:"is_active" db:"is_active"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

type VehicleGenerationDetails struct {
	Make       *VehicleMake
	Model      *VehicleModel
	Generation *VehicleGeneration
}

type CreateVehicleMakeInput struct {
	Name string
	Slug string
}

type UpdateVehicleMakeInput struct {
	ID   uuid.UUID
	Name string
	Slug string
}

type SetVehicleMakeActiveInput struct {
	ID       uuid.UUID
	IsActive bool
}

type CreateVehicleModelInput struct {
	MakeID uuid.UUID
	Name   string
	Slug   string
}

type UpdateVehicleModelInput struct {
	ID     uuid.UUID
	MakeID uuid.UUID
	Name   string
	Slug   string
}

type SetVehicleModelActiveInput struct {
	ID       uuid.UUID
	IsActive bool
}

type CreateVehicleGenerationInput struct {
	ModelID  uuid.UUID
	Name     string
	Slug     string
	YearFrom int
	YearTo   int
}

type UpdateVehicleGenerationInput struct {
	ID       uuid.UUID
	ModelID  uuid.UUID
	Name     string
	Slug     string
	YearFrom int
	YearTo   int
}

type SetVehicleGenerationActiveInput struct {
	ID       uuid.UUID
	IsActive bool
}

func (g *VehicleGeneration) ContainsYear(year int) bool {
	if g == nil {
		return false
	}

	if year < g.YearFrom {
		return false
	}

	if g.YearTo > 0 && year > g.YearTo {
		return false
	}

	return true
}

func (g *VehicleGeneration) YearOptions(currentYear int) []int {
	if g == nil {
		return nil
	}

	yearTo := g.YearTo
	if yearTo == 0 || yearTo > currentYear {
		yearTo = currentYear
	}

	if yearTo < g.YearFrom {
		return nil
	}

	years := make([]int, 0, yearTo-g.YearFrom+1)
	for year := yearTo; year >= g.YearFrom; year-- {
		years = append(years, year)
	}

	return years
}

func (d *VehicleGenerationDetails) ToCarInfo(year int) CarInfo {
	if d == nil || d.Make == nil || d.Model == nil || d.Generation == nil {
		return CarInfo{Year: year}
	}

	return CarInfo{
		Make:       d.Make.Name,
		Model:      d.Model.Name,
		Generation: d.Generation.Name,
		Year:       year,
	}
}
