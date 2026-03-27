package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/DedovInside/AutoInspect/backend/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type UserRepo struct {
	db *DB
}

func NewUserRepo(db *DB) *UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) Create(ctx context.Context, user *domain.User) error {
	query := `
		INSERT INTO users (
			id, username, email, password_hash, role,
			email_verified, is_active,
			created_at, updated_at,
			api_calls_count
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7,
			$8, $9,
			$10
		)`

	_, err := r.db.pool.Exec(ctx, query,
		user.ID, user.Username, user.Email, user.PasswordHash, user.Role,
		user.EmailVerified, user.IsActive,
		user.CreatedAt, user.UpdatedAt,
		user.APICallsCount,
	)

	if err != nil {
		return fmt.Errorf("create user Error: %w", err)
	}
	return nil
}

func (r *UserRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	query := `
		SELECT id, username, email, password_hash, role,
		       email_verified, is_active,
		       created_at, updated_at, last_login,
		       api_calls_count, api_quota_reset_at
		FROM users
		WHERE id = $1`

	row := r.db.pool.QueryRow(ctx, query, id)
	return scanUser(row)
}

func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `
		SELECT id, username, email, password_hash, role,
		       email_verified, is_active,
		       created_at, updated_at, last_login,
		       api_calls_count, api_quota_reset_at
		FROM users
		WHERE email = $1`

	row := r.db.pool.QueryRow(ctx, query, email)
	return scanUser(row)
}

func (r *UserRepo) Update(ctx context.Context, user *domain.User) error {
	query := `
		UPDATE users SET
			username      = $1,
			email         = $2,
			password_hash = $3,
			role          = $4,
			email_verified = $5,
			is_active     = $6,
			api_calls_count = $7,
			api_quota_reset_at = $8
		WHERE id = $9`

	tag, err := r.db.pool.Exec(ctx, query,
		user.Username, user.Email, user.PasswordHash, user.Role,
		user.EmailVerified, user.IsActive,
		user.APICallsCount, user.APIQuotaResetAt,
		user.ID,
	)
	if err != nil {
		return fmt.Errorf("UserRepo.Update: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("UserRepo.Update: %w", domain.ErrNotFound)
	}
	return nil
}

func (r *UserRepo) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.db.pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("UserRepo.Delete: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("UserRepo.Delete: %w", domain.ErrNotFound)
	}
	return nil
}

func (r *UserRepo) List(ctx context.Context, limit, offset int) ([]*domain.User, error) {
	query := `
		SELECT id, username, email, password_hash, role,
		       email_verified, is_active,
		       created_at, updated_at, last_login,
		       api_calls_count, api_quota_reset_at
		FROM users
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`

	rows, err := r.db.pool.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("UserRepo.List: %w", err)
	}
	defer rows.Close()

	return collectUsers(rows)
}

func (r *UserRepo) UpdateLastLogin(ctx context.Context, id uuid.UUID) error {
	tag, err := r.db.pool.Exec(ctx,
		`UPDATE users SET last_login = NOW() WHERE id = $1`, id,
	)
	if err != nil {
		return fmt.Errorf("UserRepo.UpdateLastLogin: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("UserRepo.UpdateLastLogin: %w", domain.ErrNotFound)
	}
	return nil
}

func scanUser(row pgx.Row) (*domain.User, error) {
	u := &domain.User{}
	err := row.Scan(
		&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.Role,
		&u.EmailVerified, &u.IsActive,
		&u.CreatedAt, &u.UpdatedAt, &u.LastLogin,
		&u.APICallsCount, &u.APIQuotaResetAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("scanUser: %w", err)
	}
	return u, nil
}

func collectUsers(rows pgx.Rows) ([]*domain.User, error) {
	var users []*domain.User
	for rows.Next() {
		u := &domain.User{}
		err := rows.Scan(
			&u.ID, &u.Username, &u.Email, &u.PasswordHash, &u.Role,
			&u.EmailVerified, &u.IsActive,
			&u.CreatedAt, &u.UpdatedAt, &u.LastLogin,
			&u.APICallsCount, &u.APIQuotaResetAt,
		)
		if err != nil {
			return nil, fmt.Errorf("collectUsers: %w", err)
		}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("collectUsers rows: %w", err)
	}
	return users, nil
}
