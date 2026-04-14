package postgres

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/DedovInside/AutoInspect/backend/internal/domain"
	"github.com/google/uuid"
)

type AuditLogRepo struct {
	db *DB
}

func NewAuditLogRepo(db *DB) *AuditLogRepo {
	return &AuditLogRepo{db: db}
}

const auditLogSelectCols = `
	id, user_id, action,
	entity_type, entity_id,
	ip_address, user_agent, request_id,
	details, status_code,
	created_at`

// Create создаёт запись в журнале аудита.
func (r *AuditLogRepo) Create(ctx context.Context, log *domain.AuditLog) error {
	query := `
		INSERT INTO audit_logs (
			user_id, action,
			entity_type, entity_id,
			ip_address, user_agent, request_id,
			details, status_code,
			created_at
		) VALUES (
			$1, $2,
			$3, $4,
			$5, $6, $7,
			$8, $9,
			$10
		)
		RETURNING id`

	log.CreatedAt = time.Now()

	// ip_address хранится как inet в PostgreSQL — передаём строку
	var ipStr *string
	if log.IPAddress != nil {
		s := log.IPAddress.String()
		ipStr = &s
	}

	err := r.db.pool.QueryRow(ctx, query,
		log.UserID, log.Action,
		log.EntityType, log.EntityID,
		ipStr, log.UserAgent, log.RequestID,
		log.Details, log.StatusCode,
		log.CreatedAt,
	).Scan(&log.ID)
	if err != nil {
		return fmt.Errorf("AuditLogRepo.Create: %w", err)
	}
	return nil
}

// GetByUserID возвращает записи аудита по пользователю с пагинацией.
func (r *AuditLogRepo) GetByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.AuditLog, error) {
	query := `
		SELECT ` + auditLogSelectCols + `
		FROM audit_logs
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.db.pool.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("AuditLogRepo.GetByUserID: %w", err)
	}
	defer rows.Close()

	return collectAuditLogs(rows)
}

// GetByAction возвращает записи аудита по типу действия с пагинацией.
func (r *AuditLogRepo) GetByAction(ctx context.Context, action string, limit, offset int) ([]*domain.AuditLog, error) {
	query := `
		SELECT ` + auditLogSelectCols + `
		FROM audit_logs
		WHERE action = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.db.pool.Query(ctx, query, action, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("AuditLogRepo.GetByAction: %w", err)
	}
	defer rows.Close()

	return collectAuditLogs(rows)
}

// List возвращает все записи аудита с пагинацией.
func (r *AuditLogRepo) List(ctx context.Context, limit, offset int) ([]*domain.AuditLog, error) {
	query := `
		SELECT ` + auditLogSelectCols + `
		FROM audit_logs
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`

	rows, err := r.db.pool.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("AuditLogRepo.List: %w", err)
	}
	defer rows.Close()

	return collectAuditLogs(rows)
}

// --- вспомогательные функции сканирования ---

func collectAuditLogs(rows interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
}) ([]*domain.AuditLog, error) {
	var logs []*domain.AuditLog
	for rows.Next() {
		l := &domain.AuditLog{}
		var ipStr *string
		err := rows.Scan(
			&l.ID, &l.UserID, &l.Action,
			&l.EntityType, &l.EntityID,
			&ipStr, &l.UserAgent, &l.RequestID,
			&l.Details, &l.StatusCode,
			&l.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("collectAuditLogs: %w", err)
		}
		if ipStr != nil {
			ip := net.ParseIP(*ipStr)
			l.IPAddress = &ip
		}
		logs = append(logs, l)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("collectAuditLogs rows: %w", err)
	}
	return logs, nil
}
