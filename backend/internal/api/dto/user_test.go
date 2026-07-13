package dto

import (
	"testing"

	"github.com/DedovInside/AutoInspect/backend/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestToUserResponseIncludesContactProfile(t *testing.T) {
	t.Parallel()

	user := &domain.User{
		ID:           uuid.New(),
		Username:     "ivan",
		Email:        "ivan@yandex.ru",
		ContactName:  ptr("Иван"),
		ContactPhone: ptr("+79990000000"),
		ContactEmail: ptr("ivan@example.com"),
		Role:         domain.RoleUser,
		IsActive:     true,
	}

	got := ToUserResponse(user)

	require.Equal(t, user.ID, got.ID)
	require.Equal(t, "Иван", *got.ContactName)
	require.Equal(t, "+79990000000", *got.ContactPhone)
	require.Equal(t, "ivan@example.com", *got.ContactEmail)
}

func ptr(value string) *string {
	return &value
}
