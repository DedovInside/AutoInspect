package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/DedovInside/AutoInspect/backend/internal/domain"
	"github.com/DedovInside/AutoInspect/backend/internal/repository/postgres/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

type UserRepo struct {
	queries *db.Queries
}

func NewUserRepo(tx DBTX) *UserRepo {
	return &UserRepo{
		queries: db.New(tx),
	}
}

func (r *UserRepo) Create(ctx context.Context, user *domain.User) error {
	params := db.CreateUserParams{
		ID:            pgtype.UUID{Bytes: user.ID, Valid: true},
		Username:      user.Username,
		Email:         user.Email,
		FirstName:     user.FirstName,
		LastName:      user.LastName,
		DisplayName:   user.DisplayName,
		AvatarUrl:     user.AvatarURL,
		ContactName:   user.ContactName,
		ContactPhone:  user.ContactPhone,
		ContactEmail:  user.ContactEmail,
		PasswordHash:  user.PasswordHash,
		Role:          string(user.Role),
		EmailVerified: &user.EmailVerified,
		IsActive:      &user.IsActive,
		CreatedAt:     pgtype.Timestamptz{Time: user.CreatedAt, Valid: true},
		UpdatedAt:     pgtype.Timestamptz{Time: user.UpdatedAt, Valid: true},
	}
	err := r.queries.CreateUser(ctx, params)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.ErrAlreadyExists
		}
		return domain.ErrInternal
	}

	return nil
}

func (r *UserRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	pgID := pgtype.UUID{Bytes: id, Valid: true}
	dbUser, err := r.queries.GetUserByID(ctx, pgID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, domain.ErrInternal
	}

	return toDomainUser(&dbUser), nil
}

func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	dbUser, err := r.queries.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, domain.ErrInternal
	}

	return toDomainUser(&dbUser), nil
}

func (r *UserRepo) Update(ctx context.Context, user *domain.User) error {
	params := db.UpdateUserParams{
		Username:      user.Username,
		Email:         user.Email,
		FirstName:     user.FirstName,
		LastName:      user.LastName,
		DisplayName:   user.DisplayName,
		AvatarUrl:     user.AvatarURL,
		ContactName:   user.ContactName,
		ContactPhone:  user.ContactPhone,
		ContactEmail:  user.ContactEmail,
		PasswordHash:  user.PasswordHash,
		Role:          string(user.Role),
		EmailVerified: &user.EmailVerified,
		IsActive:      &user.IsActive,
		ID:            pgtype.UUID{Bytes: user.ID, Valid: true},
	}
	rowsAffected, err := r.queries.UpdateUser(ctx, params)
	if err != nil {
		return domain.ErrInternal
	}

	if rowsAffected == 0 {
		return domain.ErrNotFound
	}

	return nil
}

func (r *UserRepo) UpdateContactProfile(
	ctx context.Context,
	input domain.UpdateUserContactProfileInput,
) (*domain.User, error) {
	dbUser, err := r.queries.UpdateUserContactProfile(ctx, db.UpdateUserContactProfileParams{
		ID:           pgtype.UUID{Bytes: input.UserID, Valid: true},
		ContactName:  input.ContactName,
		ContactPhone: input.ContactPhone,
		ContactEmail: input.ContactEmail,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, domain.ErrInternal
	}

	return toDomainUser(&dbUser), nil
}

func (r *UserRepo) UpdateRole(ctx context.Context, id uuid.UUID, role domain.Role) error {
	rowsAffected, err := r.queries.UpdateUserRole(ctx, db.UpdateUserRoleParams{
		ID:   pgtype.UUID{Bytes: id, Valid: true},
		Role: string(role),
	})
	if err != nil {
		return domain.ErrInternal
	}

	if rowsAffected == 0 {
		return domain.ErrNotFound
	}

	return nil
}

func (r *UserRepo) Delete(ctx context.Context, id uuid.UUID) error {
	pgID := pgtype.UUID{Bytes: id, Valid: true}
	rowsAffected, err := r.queries.DeleteUser(ctx, pgID)
	if err != nil {
		return domain.ErrInternal
	}

	if rowsAffected == 0 {
		return domain.ErrNotFound
	}

	return nil
}

func (r *UserRepo) UpdateLastLogin(ctx context.Context, id uuid.UUID) error {
	pgID := pgtype.UUID{Bytes: id, Valid: true}

	rowsAffected, err := r.queries.UpdateLastLogin(ctx, pgID)
	if err != nil {
		return domain.ErrInternal
	}

	if rowsAffected == 0 {
		return domain.ErrNotFound
	}

	return nil
}

func (r *UserRepo) List(ctx context.Context, limit, offset int) ([]*domain.User, error) {
	params := db.ListUsersParams{
		// #nosec
		Limit: int32(limit),
		// #nosec
		Offset: int32(offset),
	}
	dbUsers, err := r.queries.ListUsers(ctx, params)
	if err != nil {
		return nil, domain.ErrInternal
	}

	users := make([]*domain.User, len(dbUsers))
	for i := range dbUsers {
		users[i] = toDomainUser(&dbUsers[i])
	}

	return users, nil
}

func toDomainUser(dbUser *db.User) *domain.User {
	emailVerified := false

	if dbUser.EmailVerified != nil {
		emailVerified = *dbUser.EmailVerified
	}

	isActive := false

	if dbUser.IsActive != nil {
		isActive = *dbUser.IsActive
	}

	createdAt := dbUser.CreatedAt.Time
	updatedAt := dbUser.UpdatedAt.Time

	var lastLoginTime *time.Time

	if dbUser.LastLogin.Valid {
		lastLoginTime = &dbUser.LastLogin.Time
	}

	id, _ := uuid.FromBytes(dbUser.ID.Bytes[:])
	return &domain.User{
		ID:            id,
		Username:      dbUser.Username,
		Email:         dbUser.Email,
		FirstName:     dbUser.FirstName,
		LastName:      dbUser.LastName,
		DisplayName:   dbUser.DisplayName,
		AvatarURL:     dbUser.AvatarUrl,
		ContactName:   dbUser.ContactName,
		ContactPhone:  dbUser.ContactPhone,
		ContactEmail:  dbUser.ContactEmail,
		PasswordHash:  dbUser.PasswordHash,
		Role:          domain.Role(dbUser.Role),
		EmailVerified: emailVerified,
		IsActive:      isActive,
		CreatedAt:     createdAt,
		UpdatedAt:     updatedAt,
		LastLogin:     lastLoginTime,
	}
}
