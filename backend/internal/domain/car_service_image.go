package domain

import (
	"time"

	"github.com/google/uuid"
)

type CarServiceImage struct {
	ID               uuid.UUID `json:"id" db:"id"`
	ProfileID        uuid.UUID `json:"profile_id" db:"profile_id"`
	S3Key            string    `json:"s3_key" db:"s3_key"`
	IsPrimary        bool      `json:"is_primary" db:"is_primary"`
	SortOrder        int       `json:"sort_order" db:"sort_order"`
	OriginalFilename string    `json:"original_filename" db:"original_filename"`
	ContentType      string    `json:"content_type" db:"content_type"`
	SizeBytes        int64     `json:"size_bytes" db:"size_bytes"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
}

type CreateCarServiceImageInput struct {
	ProfileID        uuid.UUID
	S3Key            string
	IsPrimary        bool
	SortOrder        int
	OriginalFilename string
	ContentType      string
	SizeBytes        int64
}
