package handlers

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DedovInside/AutoInspect/backend/internal/domain"
	"github.com/DedovInside/AutoInspect/backend/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestAuthHandlerUpdateMe(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	userID := uuid.New()
	userRepo := &handlerAuthUserRepo{}
	handler := NewAuthHandler(service.NewAuthService(
		nil,
		userRepo,
		nil,
		nil,
		nil,
		nil,
		nil,
	))

	router := gin.New()
	router.PATCH("/v1/auth/me", withTestUser(userID, domain.RoleUser), handler.UpdateMe)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodPatch,
		"/v1/auth/me",
		bytes.NewReader(mustJSON(t, map[string]any{
			"contact_name":  " Иван ",
			"contact_phone": "+79990000000",
			"contact_email": "ivan@example.com",
		})),
	)
	request.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, userID, userRepo.lastInput.UserID)
	require.Contains(t, recorder.Body.String(), `"contact_name":"Иван"`)
	require.Contains(t, recorder.Body.String(), `"contact_phone":"+79990000000"`)
	require.Contains(t, recorder.Body.String(), `"contact_email":"ivan@example.com"`)
}

type handlerAuthUserRepo struct {
	lastInput domain.UpdateUserContactProfileInput
}

func (r *handlerAuthUserRepo) Create(context.Context, *domain.User) error {
	return nil
}

func (r *handlerAuthUserRepo) GetByID(context.Context, uuid.UUID) (*domain.User, error) {
	return nil, domain.ErrNotFound
}

func (r *handlerAuthUserRepo) GetByEmail(context.Context, string) (*domain.User, error) {
	return nil, domain.ErrNotFound
}

func (r *handlerAuthUserRepo) Update(context.Context, *domain.User) error {
	return nil
}

func (r *handlerAuthUserRepo) UpdateContactProfile(
	_ context.Context,
	input domain.UpdateUserContactProfileInput,
) (*domain.User, error) {
	r.lastInput = input
	return &domain.User{
		ID:           input.UserID,
		Username:     "ivan",
		Email:        "ivan@yandex.ru",
		ContactName:  input.ContactName,
		ContactPhone: input.ContactPhone,
		ContactEmail: input.ContactEmail,
		Role:         domain.RoleUser,
		IsActive:     true,
	}, nil
}

func (r *handlerAuthUserRepo) UpdateRole(context.Context, uuid.UUID, domain.Role) error {
	return nil
}

func (r *handlerAuthUserRepo) Delete(context.Context, uuid.UUID) error {
	return nil
}

func (r *handlerAuthUserRepo) List(context.Context, int, int) ([]*domain.User, error) {
	return nil, nil
}

func (r *handlerAuthUserRepo) UpdateLastLogin(context.Context, uuid.UUID) error {
	return nil
}
