package postgres

import (
	"context"

	"github.com/DedovInside/AutoInspect/backend/internal/domain"
	"github.com/DedovInside/AutoInspect/backend/internal/repository/postgres/db"
)

type CarServiceMatchRepo struct {
	queries *db.Queries
}

func NewCarServiceMatchRepo(tx DBTX) *CarServiceMatchRepo {
	return &CarServiceMatchRepo{queries: db.New(tx)}
}

func (r *CarServiceMatchRepo) FindMatching(ctx context.Context,
	criteria []domain.CarServiceMatchCriterion, limit, offset int) ([]*domain.CarServiceMatch, error) {
	limit32, err := intToInt32Checked(limit)
	if err != nil {
		return nil, domain.ErrInvalidInput
	}

	offset32, err := intToInt32Checked(offset)
	if err != nil {
		return nil, domain.ErrInvalidInput
	}

	damageTypes, partCategories := matchCriteriaArrays(criteria)
	if len(damageTypes) == 0 {
		return []*domain.CarServiceMatch{}, nil
	}

	rows, err := r.queries.ListMatchingCarServices(ctx, db.ListMatchingCarServicesParams{
		DamageTypeCodes:   damageTypes,
		PartCategoryCodes: partCategories,
		Limit:             limit32,
		Offset:            offset32,
	})
	if err != nil {
		return nil, domain.ErrInternal
	}

	return toDomainCarServiceMatches(rows), nil
}

func toDomainCarServiceMatches(rows []db.ListMatchingCarServicesRow) []*domain.CarServiceMatch {
	matches := make([]*domain.CarServiceMatch, 0, len(rows))
	for i := range rows {
		matches = append(matches, toDomainCarServiceMatch(&rows[i]))
	}

	return matches
}

func toDomainCarServiceMatch(row *db.ListMatchingCarServicesRow) *domain.CarServiceMatch {
	match := &domain.CarServiceMatch{
		Profile: &domain.CarServiceProfile{
			ID:               fromPgUUID(row.ID),
			UserID:           fromPgUUID(row.UserID),
			OrganizationName: row.OrganizationName,
			City:             row.City,
			Address:          row.Address,
			Phone:            row.Phone,
			Email:            row.Email,
			WebsiteURL:       row.WebsiteUrl,
			ContactInfo:      row.ContactInfo,
			Description:      row.Description,
			IsActive:         row.IsActive,
			CreatedAt:        row.CreatedAt.Time,
			UpdatedAt:        row.UpdatedAt.Time,
		},
		MatchCount: int(row.MatchCount),
	}

	return match
}

func matchCriteriaArrays(criteria []domain.CarServiceMatchCriterion) (damageTypes, partCategories []string) {
	damageTypes = make([]string, 0, len(criteria))
	partCategories = make([]string, 0, len(criteria))

	for _, item := range criteria {
		damageTypes = append(damageTypes, item.DamageTypeCode)
		partCategories = append(partCategories, item.PartCategoryCode)
	}

	return damageTypes, partCategories
}
