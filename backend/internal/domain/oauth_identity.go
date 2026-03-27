package domain

import (
	"time"

	"github.com/google/uuid"
)

const OAuthProviderYandex = "yandex"

type OAuthIdentity struct {
	ID             uuid.UUID `json:"id" db:"id"`
	UserID         uuid.UUID `json:"user_id" db:"user_id"`
	Provider       string    `json:"provider" db:"provider"`
	ProviderUserID string    `json:"provider_user_id" db:"provider_user_id"`
	Email          *string   `json:"email,omitempty" db:"email"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
}

