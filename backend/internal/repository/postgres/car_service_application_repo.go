package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/DedovInside/AutoInspect/backend/internal/domain"
	"github.com/DedovInside/AutoInspect/backend/internal/repository/postgres/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

type CarServiceApplicationRepo struct {
	queries *db.Queries
}

func NewCarServiceApplicationRepo(tx DBTX) *CarServiceApplicationRepo {
	return &CarServiceApplicationRepo{queries: db.New(tx)}
}

func (r *CarServiceApplicationRepo) Create(ctx context.Context, application *domain.CarServiceApplication) error {
	params := db.CreateCarServiceApplicationParams{
		ID:               pgtype.UUID{Bytes: application.ID, Valid: true},
		UserID:           pgtype.UUID{Bytes: application.UserID, Valid: true},
		OrganizationName: application.OrganizationName,
		City:             application.City,
		Address:          application.Address,
		Phone:            application.Phone,
		Email:            application.Email,
		ContactInfo:      application.ContactInfo,
		Description:      application.Description,
		Status:           string(application.Status),
		RejectionReason:  application.RejectionReason,
		ReviewedBy:       toPgUUIDPtr(application.ReviewedBy),
		ReviewedAt:       toPgTimestamptzPtr(application.ReviewedAt),
		CreatedProfileID: toPgUUIDPtr(application.CreatedProfileID),
		CreatedAt:        pgtype.Timestamptz{Time: application.CreatedAt, Valid: true},
		UpdatedAt:        pgtype.Timestamptz{Time: application.UpdatedAt, Valid: true},
	}

	if err := r.queries.CreateCarServiceApplication(ctx, params); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.ErrAlreadyExists
		}
		return domain.ErrInternal
	}
	return nil
}

func (r *CarServiceApplicationRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.CarServiceApplication, error) {
	dbApplication, err := r.queries.GetCarServiceApplicationByID(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, domain.ErrInternal
	}
	return toDomainCarServiceApplication(&dbApplication), nil
}

func (r *CarServiceApplicationRepo) GetPendingByUserID(ctx context.Context,
	userID uuid.UUID) (*domain.CarServiceApplication, error) {
	dbApplication, err := r.queries.GetPendingCarServiceApplicationByUserID(ctx, pgtype.UUID{Bytes: userID, Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, domain.ErrInternal
	}

	return toDomainCarServiceApplication(&dbApplication), nil
}

func (r *CarServiceApplicationRepo) ListByUserID(ctx context.Context,
	userID uuid.UUID, limit, offset int) ([]*domain.CarServiceApplication, error) {
	limit32, offset32, err := checkedLimitOffset(limit, offset)
	if err != nil {
		return nil, err
	}

	dbApplications, err := r.queries.ListCarServiceApplicationsByUserID(ctx, db.ListCarServiceApplicationsByUserIDParams{
		UserID: pgtype.UUID{Bytes: userID, Valid: true},
		Limit:  limit32,
		Offset: offset32,
	})
	if err != nil {
		return nil, domain.ErrInternal
	}

	return toDomainCarServiceApplications(dbApplications), nil
}

func (r *CarServiceApplicationRepo) ListForAdmin(ctx context.Context,
	status *domain.CarServiceApplicationStatus, limit, offset int) ([]*domain.CarServiceApplication, error) {
	limit32, offset32, err := checkedLimitOffset(limit, offset)
	if err != nil {
		return nil, err
	}

	var statusPtr *string
	if status != nil {
		s := string(*status)
		statusPtr = &s
	}

	dbApplications, err := r.queries.ListCarServiceApplicationsForAdmin(ctx, db.ListCarServiceApplicationsForAdminParams{
		Limit:  limit32,
		Offset: offset32,
		Status: statusPtr,
	})
	if err != nil {
		return nil, domain.ErrInternal
	}

	return toDomainCarServiceApplications(dbApplications), nil
}

func (r *CarServiceApplicationRepo) Approve(ctx context.Context, input domain.ApproveCarServiceApplicationInput) error {
	now := time.Now().UTC()
	params := db.ApproveCarServiceApplicationParams{
		ID:               pgtype.UUID{Bytes: input.ID, Valid: true},
		ReviewedBy:       pgtype.UUID{Bytes: input.ReviewedBy, Valid: input.ReviewedBy != uuid.Nil},
		ReviewedAt:       pgtype.Timestamptz{Time: now, Valid: true},
		CreatedProfileID: toPgUUIDPtr(input.CreatedProfileID),
		UpdatedAt:        pgtype.Timestamptz{Time: now, Valid: true},
	}

	rowsAffected, err := r.queries.ApproveCarServiceApplication(ctx, params)
	if err != nil {
		return domain.ErrInternal
	}

	if rowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *CarServiceApplicationRepo) Reject(ctx context.Context, input domain.RejectCarServiceApplicationInput) error {
	now := time.Now().UTC()
	params := db.RejectCarServiceApplicationParams{
		ID:              pgtype.UUID{Bytes: input.ID, Valid: true},
		RejectionReason: &input.RejectionReason,
		ReviewedBy:      pgtype.UUID{Bytes: input.ReviewedBy, Valid: input.ReviewedBy != uuid.Nil},
		ReviewedAt:      pgtype.Timestamptz{Time: now, Valid: true},
		UpdatedAt:       pgtype.Timestamptz{Time: now, Valid: true},
	}

	rowsAffected, err := r.queries.RejectCarServiceApplication(ctx, params)
	if err != nil {
		return domain.ErrInternal
	}

	if rowsAffected == 0 {
		return domain.ErrNotFound
	}

	return nil
}

func toDomainCarServiceApplications(dbApplications []db.CarServiceApplication) []*domain.CarServiceApplication {
	applications := make([]*domain.CarServiceApplication, 0, len(dbApplications))
	for i := range dbApplications {
		applications = append(applications, toDomainCarServiceApplication(&dbApplications[i]))
	}

	return applications
}

func toDomainCarServiceApplication(dbApplication *db.CarServiceApplication) *domain.CarServiceApplication {
	return &domain.CarServiceApplication{
		ID:               fromPgUUID(dbApplication.ID),
		UserID:           fromPgUUID(dbApplication.UserID),
		OrganizationName: dbApplication.OrganizationName,
		City:             dbApplication.City,
		Address:          dbApplication.Address,
		Phone:            dbApplication.Phone,
		Email:            dbApplication.Email,
		ContactInfo:      dbApplication.ContactInfo,
		Description:      dbApplication.Description,
		Status:           domain.CarServiceApplicationStatus(dbApplication.Status),
		RejectionReason:  dbApplication.RejectionReason,
		ReviewedBy:       fromPgUUIDPtr(dbApplication.ReviewedBy),
		ReviewedAt:       fromPgTimestamptzPtr(dbApplication.ReviewedAt),
		CreatedProfileID: fromPgUUIDPtr(dbApplication.CreatedProfileID),
		CreatedAt:        dbApplication.CreatedAt.Time,
		UpdatedAt:        dbApplication.UpdatedAt.Time,
	}
}
