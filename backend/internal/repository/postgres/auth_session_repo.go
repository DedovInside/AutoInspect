package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/DedovInside/AutoInspect/backend/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type AuthSessionRepo struct {
	db *DB
}

func NewAuthSessionRepo(db *DB) *AuthSessionRepo {
	return &AuthSessionRepo{db: db}
}

const authSessionSelectCols = `
	id, user_id, token_hash, token_family_id, replaced_by_id,
	user_agent, ip_address,
	expires_at, revoked_at, revoked_reason,
	created_at, updated_at, last_used_at`

func (r *AuthSessionRepo) Create(ctx context.Context, s *domain.AuthRefreshSession) error {
	query := `
		INSERT INTO auth_refresh_sessions (
			id, user_id, token_hash, token_family_id, replaced_by_id,
			user_agent, ip_address,
			expires_at, revoked_at, revoked_reason,
			created_at, last_used_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7,
			$8, $9, $10,
			$11, $12
		)`

	_, err := r.db.pool.Exec(ctx, query,
		s.ID, s.UserID, s.TokenHash, s.TokenFamilyID, s.ReplacedByID,
		s.UserAgent, s.IPAddress,
		s.ExpiresAt, s.RevokedAt, s.RevokedReason,
		s.CreatedAt, s.LastUsedAt,
	)
	if err != nil {
		return fmt.Errorf("AuthSessionRepo.Create: %w", err)
	}
	return nil
}

func (r *AuthSessionRepo) GetByTokenHash(ctx context.Context, tokenHash string) (*domain.AuthRefreshSession, error) {
	query := `SELECT ` + authSessionSelectCols + ` FROM auth_refresh_sessions WHERE token_hash = $1`
	row := r.db.pool.QueryRow(ctx, query, tokenHash)
	return scanAuthSession(row)
}

func (r *AuthSessionRepo) Revoke(ctx context.Context, id uuid.UUID, revokedReason string, replacedByID *uuid.UUID) error {
	query := `
		UPDATE auth_refresh_sessions
		SET revoked_at = NOW(), revoked_reason = $1, replaced_by_id = $2
		WHERE id = $3 AND revoked_at IS NULL`

	tag, err := r.db.pool.Exec(ctx, query, revokedReason, replacedByID, id)
	if err != nil {
		return fmt.Errorf("AuthSessionRepo.Revoke: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("AuthSessionRepo.Revoke: %w", domain.ErrNotFound)
	}
	return nil
}

func (r *AuthSessionRepo) TouchLastUsed(ctx context.Context, id uuid.UUID, at time.Time) error {
	query := `UPDATE auth_refresh_sessions SET last_used_at = $1 WHERE id = $2`
	tag, err := r.db.pool.Exec(ctx, query, at, id)
	if err != nil {
		return fmt.Errorf("AuthSessionRepo.TouchLastUsed: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("AuthSessionRepo.TouchLastUsed: %w", domain.ErrNotFound)
	}
	return nil
}

func (r *AuthSessionRepo) RevokeFamily(ctx context.Context, familyID uuid.UUID, revokedReason string) error {
	query := `
		UPDATE auth_refresh_sessions
		SET revoked_at = NOW(), revoked_reason = $1
		WHERE token_family_id = $2 AND revoked_at IS NULL`

	_, err := r.db.pool.Exec(ctx, query, revokedReason, familyID)
	if err != nil {
		return fmt.Errorf("AuthSessionRepo.RevokeFamily: %w", err)
	}
	return nil
}

func scanAuthSession(row pgx.Row) (*domain.AuthRefreshSession, error) {
	s := &domain.AuthRefreshSession{}
	err := row.Scan(
		&s.ID, &s.UserID, &s.TokenHash, &s.TokenFamilyID, &s.ReplacedByID,
		&s.UserAgent, &s.IPAddress,
		&s.ExpiresAt, &s.RevokedAt, &s.RevokedReason,
		&s.CreatedAt, &s.UpdatedAt, &s.LastUsedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("scanAuthSession: %w", err)
	}
	return s, nil
}
