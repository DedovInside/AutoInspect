package domain

import (
	"database/sql/driver"
	"encoding/json"
	"net"
	"time"

	"github.com/google/uuid"
)

// AuditLog представляет запись в журнале аудита
type AuditLog struct {
	ID int64 `json:"id" db:"id"`

	UserID     *uuid.UUID `json:"user_id,omitempty" db:"user_id"`
	Action     string     `json:"action" db:"action"`
	EntityType *string    `json:"entity_type,omitempty" db:"entity_type"`
	EntityID   *uuid.UUID `json:"entity_id,omitempty" db:"entity_id"`

	// Контекст запроса
	IPAddress *net.IP    `json:"ip_address,omitempty" db:"ip_address"`
	UserAgent *string    `json:"user_agent,omitempty" db:"user_agent"`
	RequestID *uuid.UUID `json:"request_id,omitempty" db:"request_id"`

	// Детали
	Details    *AuditDetails `json:"details,omitempty" db:"details"`
	StatusCode *int          `json:"status_code,omitempty" db:"status_code"`

	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// AuditDetails представляет детали события аудита
type AuditDetails map[string]interface{}

// Scan реализует интерфейс sql.Scanner для AuditDetails
func (ad *AuditDetails) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, ad)
}

// Value реализует интерфейс driver.Valuer для AuditDetails
func (ad AuditDetails) Value() (driver.Value, error) {
	return json.Marshal(ad)
}

// AuditLogCreateRequest DTO для создания записи аудита
type AuditLogCreateRequest struct {
	UserID     *uuid.UUID    `json:"user_id,omitempty"`
	Action     string        `json:"action" validate:"required"`
	EntityType *string       `json:"entity_type,omitempty"`
	EntityID   *uuid.UUID    `json:"entity_id,omitempty"`
	IPAddress  *string       `json:"ip_address,omitempty"`
	UserAgent  *string       `json:"user_agent,omitempty"`
	RequestID  *uuid.UUID    `json:"request_id,omitempty"`
	Details    *AuditDetails `json:"details,omitempty"`
	StatusCode *int          `json:"status_code,omitempty"`
}
