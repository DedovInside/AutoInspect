package domain

import (
	"time"

	"github.com/google/uuid"
)

type AuthRefreshSession struct {
	ID            uuid.UUID  `json:"id" db:"id"`
	UserID        uuid.UUID  `json:"user_id" db:"user_id"`
	TokenHash     string     `json:"-" db:"token_hash"`
	TokenFamilyID uuid.UUID  `json:"token_family_id" db:"token_family_id"`
	ReplacedByID  *uuid.UUID `json:"replaced_by_id,omitempty" db:"replaced_by_id"`
	UserAgent     *string    `json:"user_agent,omitempty" db:"user_agent"`
	IPAddress     *string    `json:"ip_address,omitempty" db:"ip_address"`
	ExpiresAt     time.Time  `json:"expires_at" db:"expires_at"`
	RevokedAt     *time.Time `json:"revoked_at,omitempty" db:"revoked_at"`
	RevokedReason *string    `json:"revoked_reason,omitempty" db:"revoked_reason"`
	CreatedAt     time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt     *time.Time `json:"updated_at,omitempty" db:"updated_at"`
	LastUsedAt    *time.Time `json:"last_used_at,omitempty" db:"last_used_at"`
}

