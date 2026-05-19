package service

import (
	"context"
	"testing"

	"github.com/DedovInside/AutoInspect/backend/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestCarServiceProfileServiceListSpecializationOptions(t *testing.T) {
	t.Parallel()

	svc := NewCarServiceProfileService(
		nil,
		nil,
		nil,
		&fakeDamageTypeRepo{active: map[string]*domain.DamageType{
			"dent": {Code: "dent", NameRU: "вмятина", IsActive: true},
		}},
		&fakePartCategoryRepo{active: map[string]*domain.PartCategory{
			"*":    {Code: "*", NameRU: "любая деталь", IsActive: true},
			"hood": {Code: "hood", NameRU: "капот", IsActive: true},
		}},
		nil,
		nil,
		nil,
	)

	got, err := svc.ListSpecializationOptions(context.Background())

	require.NoError(t, err)
	require.Len(t, got.DamageTypes, 1)
	require.Len(t, got.PartCategories, 2)
	require.Equal(t, "dent", got.DamageTypes[0].Code)
	require.ElementsMatch(t, []string{"*", "hood"}, partCategoryCodes(got.PartCategories))
}

func TestCarServiceProfileServiceNormalizeAndValidateSpecializationsDeduplicatesAndKeepsAnyPart(t *testing.T) {
	t.Parallel()

	svc := NewCarServiceProfileService(
		nil,
		nil,
		nil,
		&fakeDamageTypeRepo{active: map[string]*domain.DamageType{
			"dent": {Code: "dent", IsActive: true},
		}},
		&fakePartCategoryRepo{active: map[string]*domain.PartCategory{
			"*":    {Code: "*", IsActive: true},
			"hood": {Code: "hood", IsActive: true},
		}},
		nil,
		nil,
		nil,
	)

	got, err := svc.normalizeAndValidateSpecializations(context.Background(), []domain.CarServiceSpecializationInput{
		{DamageTypeCode: " dent ", PartCategoryCode: " * "},
		{DamageTypeCode: "dent", PartCategoryCode: "*"},
		{DamageTypeCode: "dent", PartCategoryCode: "hood"},
	})

	require.NoError(t, err)
	require.Equal(t, []domain.CarServiceSpecializationInput{
		{DamageTypeCode: "dent", PartCategoryCode: "*"},
		{DamageTypeCode: "dent", PartCategoryCode: "hood"},
	}, got)
}

func TestCarServiceProfileServiceNormalizeAndValidateSpecializationsRejectsUnknownCodes(t *testing.T) {
	t.Parallel()

	svc := NewCarServiceProfileService(
		nil,
		nil,
		nil,
		&fakeDamageTypeRepo{active: map[string]*domain.DamageType{"dent": {Code: "dent"}}},
		&fakePartCategoryRepo{active: map[string]*domain.PartCategory{"hood": {Code: "hood"}}},
		nil,
		nil,
		nil,
	)

	_, err := svc.normalizeAndValidateSpecializations(context.Background(), []domain.CarServiceSpecializationInput{
		{DamageTypeCode: "scratch", PartCategoryCode: "hood"},
	})
	require.ErrorIs(t, err, domain.ErrInvalidInput)

	_, err = svc.normalizeAndValidateSpecializations(context.Background(), []domain.CarServiceSpecializationInput{
		{DamageTypeCode: "dent", PartCategoryCode: "*"},
	})
	require.ErrorIs(t, err, domain.ErrInvalidInput)
}

func TestCarServiceProfileServiceUpdateMyProfileNormalizesOptionalFields(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	profile := &domain.CarServiceProfile{
		ID:               uuid.New(),
		UserID:           userID,
		OrganizationName: "Old",
		City:             "Москва",
		Address:          "Old address",
		IsActive:         true,
	}
	repo := &fakeProfileRepo{
		byUserID: map[uuid.UUID]*domain.CarServiceProfile{userID: profile},
		byID:     map[uuid.UUID]*domain.CarServiceProfile{profile.ID: profile},
	}
	svc := NewCarServiceProfileService(nil, repo, nil, nil, nil, nil, nil, nil)

	got, err := svc.UpdateMyProfile(context.Background(), &domain.UpdateCarServiceProfileInput{
		UserID:           userID,
		OrganizationName: " New Service ",
		City:             " Москва ",
		Address:          " Новый адрес ",
		Phone:            ptr(" +79990000000 "),
		Email:            ptr(" "),
		ContactInfo:      ptr(" Telegram: @service "),
		Description:      ptr(" Описание "),
		IsActive:         true,
	})

	require.NoError(t, err)
	require.Equal(t, "New Service", got.OrganizationName)
	require.Equal(t, "Москва", got.City)
	require.Equal(t, "Новый адрес", got.Address)
	require.Equal(t, "+79990000000", *got.Phone)
	require.Nil(t, got.Email)
	require.Equal(t, "Telegram: @service", *got.ContactInfo)
	require.Equal(t, "Описание", *got.Description)
}

type fakeDamageTypeRepo struct {
	active map[string]*domain.DamageType
	all    []*domain.DamageType
}

func (r *fakeDamageTypeRepo) ListActive(context.Context) ([]*domain.DamageType, error) {
	out := make([]*domain.DamageType, 0, len(r.active))
	for _, item := range r.active {
		out = append(out, item)
	}
	return out, nil
}

func (r *fakeDamageTypeRepo) ListAll(context.Context) ([]*domain.DamageType, error) {
	if r.all != nil {
		return r.all, nil
	}
	return r.ListActive(context.Background())
}

func (r *fakeDamageTypeRepo) ExistsActive(_ context.Context, code string) (bool, error) {
	_, ok := r.active[code]
	return ok, nil
}

type fakePartCategoryRepo struct {
	active map[string]*domain.PartCategory
}

func (r *fakePartCategoryRepo) ListActive(context.Context) ([]*domain.PartCategory, error) {
	out := make([]*domain.PartCategory, 0, len(r.active))
	for _, item := range r.active {
		out = append(out, item)
	}
	return out, nil
}

func (r *fakePartCategoryRepo) ExistsActive(_ context.Context, code string) (bool, error) {
	_, ok := r.active[code]
	return ok, nil
}

func partCategoryCodes(items []*domain.PartCategory) []string {
	codes := make([]string, 0, len(items))
	for _, item := range items {
		if item != nil {
			codes = append(codes, item.Code)
		}
	}
	return codes
}

type fakeProfileRepo struct {
	byID     map[uuid.UUID]*domain.CarServiceProfile
	byUserID map[uuid.UUID]*domain.CarServiceProfile
}

func (r *fakeProfileRepo) Create(_ context.Context, profile *domain.CarServiceProfile) error {
	if r.byID == nil {
		r.byID = make(map[uuid.UUID]*domain.CarServiceProfile)
	}
	if r.byUserID == nil {
		r.byUserID = make(map[uuid.UUID]*domain.CarServiceProfile)
	}
	r.byID[profile.ID] = profile
	r.byUserID[profile.UserID] = profile
	return nil
}

func (r *fakeProfileRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.CarServiceProfile, error) {
	if profile, ok := r.byID[id]; ok {
		return profile, nil
	}
	return nil, domain.ErrNotFound
}

func (r *fakeProfileRepo) GetByUserID(_ context.Context, userID uuid.UUID) (*domain.CarServiceProfile, error) {
	if profile, ok := r.byUserID[userID]; ok {
		return profile, nil
	}
	return nil, domain.ErrNotFound
}

func (r *fakeProfileRepo) Update(_ context.Context, input *domain.UpdateCarServiceProfileInput) error {
	profile, err := r.GetByUserID(context.Background(), input.UserID)
	if err != nil {
		return err
	}
	profile.OrganizationName = input.OrganizationName
	profile.City = input.City
	profile.Address = input.Address
	profile.Phone = input.Phone
	profile.Email = input.Email
	profile.WebsiteURL = input.WebsiteURL
	profile.ContactInfo = input.ContactInfo
	profile.Description = input.Description
	profile.IsActive = input.IsActive
	return nil
}

func (r *fakeProfileRepo) SetActive(_ context.Context, userID uuid.UUID, isActive bool) error {
	profile, err := r.GetByUserID(context.Background(), userID)
	if err != nil {
		return err
	}
	profile.IsActive = isActive
	return nil
}
