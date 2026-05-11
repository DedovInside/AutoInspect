package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/DedovInside/AutoInspect/backend/internal/domain"
	"github.com/DedovInside/AutoInspect/backend/internal/repository/postgres/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type AuthSessionRepo struct {
	queries *db.Queries
}

func NewAuthSessionRepo(tx DBTX) *AuthSessionRepo {
	return &AuthSessionRepo{
		queries: db.New(tx),
	}
}

func (r *AuthSessionRepo) Create(ctx context.Context, s *domain.AuthSession) error {
	params := db.CreateAuthSessionParams{
		ID:            pgtype.UUID{Bytes: s.ID, Valid: true},
		UserID:        pgtype.UUID{Bytes: s.UserID, Valid: true},
		TokenHash:     s.TokenHash,
		TokenFamilyID: pgtype.UUID{Bytes: s.TokenFamilyID, Valid: true},
		ReplacedByID:  toPgUUIDPtr(s.ReplacedByID),
		UserAgent:     s.UserAgent,
		IpAddress:     s.IPAddress,
		ExpiresAt:     pgtype.Timestamptz{Time: s.ExpiresAt, Valid: true},
		RevokedAt:     toPgTimestamptzPtr(s.RevokedAt),
		RevokedReason: s.RevokedReason,
		CreatedAt:     pgtype.Timestamptz{Time: s.CreatedAt, Valid: true},
		LastUsedAt:    toPgTimestamptzPtr(s.LastUsedAt),
	}
	err := r.queries.CreateAuthSession(ctx, params)
	if err != nil {
		return domain.ErrInternal
	}

	return nil
}

func (r *AuthSessionRepo) GetByTokenHash(ctx context.Context, tokenHash string) (*domain.AuthSession, error) {
	dbSession, err := r.queries.GetAuthSessionByTokenHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, domain.ErrInternal
	}

	return toDomainAuthSession(&dbSession), nil
}

func (r *AuthSessionRepo) Revoke(ctx context.Context,
	id uuid.UUID, revokedReason string, replacedByID *uuid.UUID) error {
	params := db.RevokeAuthSessionParams{
		RevokedReason: &revokedReason,
		ReplacedByID:  toPgUUIDPtr(replacedByID),
		ID:            pgtype.UUID{Bytes: id, Valid: true},
	}
	rowsAffected, err := r.queries.RevokeAuthSession(ctx, params)
	if err != nil {
		return domain.ErrInternal
	}

	if rowsAffected == 0 {
		return domain.ErrNotFound
	}

	return nil
}

func (r *AuthSessionRepo) TouchLastUsed(ctx context.Context, id uuid.UUID, at time.Time) error {
	params := db.TouchLastUsedParams{
		LastUsedAt: pgtype.Timestamptz{Time: at, Valid: true},
		ID:         pgtype.UUID{Bytes: id, Valid: true},
	}

	rowsAffected, err := r.queries.TouchLastUsed(ctx, params)

	if err != nil {
		return domain.ErrInternal
	}

	if rowsAffected == 0 {
		return domain.ErrNotFound
	}

	return nil
}

func (r *AuthSessionRepo) RevokeFamily(ctx context.Context, familyID uuid.UUID, revokedReason string) error {
	params := db.RevokeFamilyParams{
		RevokedReason: &revokedReason,
		TokenFamilyID: pgtype.UUID{Bytes: familyID, Valid: true},
	}

	err := r.queries.RevokeFamily(ctx, params)
	if err != nil {
		return domain.ErrInternal
	}

	return nil
}

func toDomainAuthSession(dbSession *db.AuthSession) *domain.AuthSession {
	return &domain.AuthSession{
		ID:            fromPgUUID(dbSession.ID),
		UserID:        fromPgUUID(dbSession.UserID),
		TokenHash:     dbSession.TokenHash,
		TokenFamilyID: fromPgUUID(dbSession.TokenFamilyID),
		ReplacedByID:  fromPgUUIDPtr(dbSession.ReplacedByID),
		UserAgent:     dbSession.UserAgent,
		IPAddress:     dbSession.IpAddress,
		ExpiresAt:     dbSession.ExpiresAt.Time,
		RevokedAt:     fromPgTimestamptzPtr(dbSession.RevokedAt),
		RevokedReason: dbSession.RevokedReason,
		CreatedAt:     dbSession.CreatedAt.Time,
		UpdatedAt:     fromPgTimestamptzPtr(dbSession.UpdatedAt),
		LastUsedAt:    fromPgTimestamptzPtr(dbSession.LastUsedAt),
	}
}
