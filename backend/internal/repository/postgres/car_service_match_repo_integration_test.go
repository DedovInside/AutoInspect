//go:build integration

package postgres

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/DedovInside/AutoInspect/backend/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestCarServiceMatchRepoFindMatchingSupportsAnyPartCategory(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	dsn := postgresContainerDSN(t, ctx)

	db, err := New(ctx, dsn, 2, time.Minute)
	require.NoError(t, err)
	t.Cleanup(db.Close)

	prepareCarServiceMatchSchema(t, ctx, db)
	t.Cleanup(func() {
		_, _ = db.Exec(ctx, `DROP TABLE IF EXISTS car_service_specializations`)
		_, _ = db.Exec(ctx, `DROP TABLE IF EXISTS car_service_profiles`)
	})

	anyProfileID := uuid.New()
	exactProfileID := uuid.New()
	otherProfileID := uuid.New()
	insertMatchProfile(t, ctx, db, anyProfileID, "Any Body Repair")
	insertMatchProfile(t, ctx, db, exactProfileID, "Hood Repair")
	insertMatchProfile(t, ctx, db, otherProfileID, "Glass Repair")

	insertSpecialization(t, ctx, db, anyProfileID, "dent", "*")
	insertSpecialization(t, ctx, db, exactProfileID, "dent", "hood")
	insertSpecialization(t, ctx, db, otherProfileID, "scratch", "windshield")

	repo := NewCarServiceMatchRepo(db)
	matches, err := repo.FindMatching(ctx, []domain.CarServiceMatchCriterion{
		{DamageTypeCode: "dent", PartCategoryCode: "hood"},
	}, 10, 0)

	require.NoError(t, err)
	require.Len(t, matches, 2)
	require.ElementsMatch(t, []uuid.UUID{anyProfileID, exactProfileID}, []uuid.UUID{
		matches[0].Profile.ID,
		matches[1].Profile.ID,
	})
	require.Equal(t, 1, matches[0].MatchCount)
	require.Equal(t, 1, matches[1].MatchCount)
}

func postgresContainerDSN(t *testing.T, ctx context.Context) string {
	t.Helper()

	container, err := tcpostgres.Run(
		ctx,
		"postgres:16-alpine",
		tcpostgres.WithDatabase("autoinspect_test"),
		tcpostgres.WithUsername("postgres"),
		tcpostgres.WithPassword("postgres"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		if isTestcontainersProviderUnavailable(err) {
			t.Skipf("testcontainers Docker provider is unavailable: %v", err)
		}
		require.NoError(t, err)
	}

	t.Cleanup(func() {
		require.NoError(t, container.Terminate(context.Background()))
	})

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	return dsn
}

func isTestcontainersProviderUnavailable(err error) bool {
	message := err.Error()
	return strings.Contains(message, "failed to create Docker provider") ||
		strings.Contains(message, "Cannot connect to the Docker daemon") ||
		strings.Contains(message, "rootless Docker is not supported on Windows")
}

func prepareCarServiceMatchSchema(t *testing.T, ctx context.Context, db *DB) {
	t.Helper()

	_, err := db.Exec(ctx, `DROP TABLE IF EXISTS car_service_specializations`)
	require.NoError(t, err)
	_, err = db.Exec(ctx, `DROP TABLE IF EXISTS car_service_profiles`)
	require.NoError(t, err)

	_, err = db.Exec(ctx, `
		CREATE TABLE car_service_profiles (
			id UUID PRIMARY KEY,
			user_id UUID NOT NULL,
			organization_name TEXT NOT NULL,
			city TEXT NOT NULL,
			address TEXT NOT NULL,
			phone TEXT,
			email TEXT,
			website_url TEXT,
			contact_info TEXT,
			description TEXT,
			is_active BOOLEAN NOT NULL DEFAULT TRUE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)
	`)
	require.NoError(t, err)

	_, err = db.Exec(ctx, `
		CREATE TABLE car_service_specializations (
			id UUID PRIMARY KEY,
			profile_id UUID NOT NULL REFERENCES car_service_profiles(id) ON DELETE CASCADE,
			damage_type_code TEXT NOT NULL,
			part_category_code TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)
	`)
	require.NoError(t, err)
}

func insertMatchProfile(t *testing.T, ctx context.Context, db *DB, id uuid.UUID, name string) {
	t.Helper()

	_, err := db.Exec(ctx, `
		INSERT INTO car_service_profiles (
			id, user_id, organization_name, city, address, is_active, created_at, updated_at
		)
		VALUES ($1, $2, $3, 'Москва', 'Тестовая улица', TRUE, now(), now())
	`, id, uuid.New(), name)
	require.NoError(t, err)
}

func insertSpecialization(
	t *testing.T,
	ctx context.Context,
	db *DB,
	profileID uuid.UUID,
	damageTypeCode string,
	partCategoryCode string,
) {
	t.Helper()

	_, err := db.Exec(ctx, `
		INSERT INTO car_service_specializations (
			id, profile_id, damage_type_code, part_category_code, created_at
		)
		VALUES ($1, $2, $3, $4, now())
	`, uuid.New(), profileID, damageTypeCode, partCategoryCode)
	require.NoError(t, err)
}
