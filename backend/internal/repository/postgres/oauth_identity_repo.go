package postgres

import (
	"context"
	"errors"

	"github.com/DedovInside/AutoInspect/backend/internal/domain"
	"github.com/DedovInside/AutoInspect/backend/internal/repository/postgres/db"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

type OAuthIdentityRepo struct {
	queries *db.Queries
}

func NewOAuthIdentityRepo(tx DBTX) *OAuthIdentityRepo {
	return &OAuthIdentityRepo{queries: db.New(tx)}
}

func (r *OAuthIdentityRepo) Create(ctx context.Context, identity *domain.OAuthIdentity) error {
	params := db.CreateOAuthIdentityParams{
		ID:             pgtype.UUID{Bytes: identity.ID, Valid: true},
		UserID:         pgtype.UUID{Bytes: identity.UserID, Valid: true},
		Provider:       identity.Provider,
		ProviderUserID: identity.ProviderUserID,
		Email:          identity.Email,
		CreatedAt:      pgtype.Timestamptz{Time: identity.CreatedAt, Valid: true},
	}
	err := r.queries.CreateOAuthIdentity(ctx, params)

	if err != nil {
		var pgErr *pgconn.PgError

		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.ErrAlreadyExists
		}
		return domain.ErrInternal
	}

	return nil
}

func (r *OAuthIdentityRepo) GetByProviderSubject(ctx context.Context,
	provider, providerUserID string) (*domain.OAuthIdentity, error) {
	params := db.GetOAuthIdentityByProviderSubjectParams{
		Provider:       provider,
		ProviderUserID: providerUserID,
	}

	dbIdentity, err := r.queries.GetOAuthIdentityByProviderSubject(ctx, params)

	if err != nil {

		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, domain.ErrInternal
	}

	return toDomainOAuthIdentity(&dbIdentity), nil
}

func toDomainOAuthIdentity(dbID *db.AuthOauthIdentity) *domain.OAuthIdentity {
	return &domain.OAuthIdentity{
		ID:             fromPgUUID(dbID.ID),
		UserID:         fromPgUUID(dbID.UserID),
		Provider:       dbID.Provider,
		ProviderUserID: dbID.ProviderUserID,
		Email:          dbID.Email,
		CreatedAt:      dbID.CreatedAt.Time,
	}
}
