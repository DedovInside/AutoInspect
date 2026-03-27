package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/DedovInside/AutoInspect/backend/internal/domain"
	"github.com/jackc/pgx/v5"
)

type OAuthIdentityRepo struct {
	db *DB
}

func NewOAuthIdentityRepo(db *DB) *OAuthIdentityRepo {
	return &OAuthIdentityRepo{db: db}
}

func (r *OAuthIdentityRepo) Create(ctx context.Context, identity *domain.OAuthIdentity) error {
	query := `
		INSERT INTO auth_oauth_identities (id, user_id, provider, provider_user_id, email, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)`

	_, err := r.db.pool.Exec(ctx, query,
		identity.ID, identity.UserID, identity.Provider, identity.ProviderUserID, identity.Email, identity.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("OAuthIdentityRepo.Create: %w", err)
	}
	return nil
}

func (r *OAuthIdentityRepo) GetByProviderSubject(ctx context.Context, provider, providerUserID string) (*domain.OAuthIdentity, error) {
	query := `
		SELECT id, user_id, provider, provider_user_id, email, created_at
		FROM auth_oauth_identities
		WHERE provider = $1 AND provider_user_id = $2`

	identity := &domain.OAuthIdentity{}
	err := r.db.pool.QueryRow(ctx, query, provider, providerUserID).Scan(
		&identity.ID, &identity.UserID, &identity.Provider, &identity.ProviderUserID, &identity.Email, &identity.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, fmt.Errorf("OAuthIdentityRepo.GetByProviderSubject: %w", err)
	}
	return identity, nil
}
