package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/ilyakaznacheev/cleanenv"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	defaultSeedPath               = "data/vehicle_catalog.seed.json"
	defaultEnvFile                = ".env"
	backendEnvFileFromRoot        = "backend/.env"
	deploymentsEnvFileFromBackend = "../deployments/.env.prod"
	deploymentsEnvFileFromRoot    = "deployments/.env.prod"
)

type allowedSeederConfigFile int

const (
	allowedDefaultEnv allowedSeederConfigFile = iota
	allowedBackendEnvFromRoot
	allowedDeploymentsEnvFromBackend
	allowedDeploymentsEnvFromRoot
)

type seederEnv struct {
	DatabaseURL     string `env:"DATABASE_URL"`
	DatabaseURLFile string `env:"DATABASE_URL_FILE"`
}

type vehicleCatalogSeed struct {
	Makes []vehicleMakeSeed `json:"makes"`
}

type vehicleMakeSeed struct {
	Name   string             `json:"name"`
	Slug   string             `json:"slug"`
	Models []vehicleModelSeed `json:"models"`
}

type vehicleModelSeed struct {
	Name        string                  `json:"name"`
	Slug        string                  `json:"slug"`
	Generations []vehicleGenerationSeed `json:"generations"`
}

type vehicleGenerationSeed struct {
	Name     string `json:"name"`
	Slug     string `json:"slug"`
	YearFrom int    `json:"year_from"`
	YearTo   *int   `json:"year_to"`
}

func main() {
	if err := run(); err != nil {
		log.Printf("vehicle catalog seeder failed: %v", err)
		os.Exit(1)
	}
}

func run() error {
	seedPath := flag.String("file", defaultSeedPath, "path to vehicle catalog seed JSON")
	flag.Parse()

	if err := loadSeederEnv(); err != nil {
		return err
	}

	dbURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if dbURL == "" {
		return errors.New("DATABASE_URL environment variable is not set")
	}

	seed, err := readSeed(*seedPath)
	if err != nil {
		return err
	}
	if err := validateSeed(seed); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		return fmt.Errorf("connect database: %w", err)
	}
	defer pool.Close()

	if err := seedVehicleCatalog(ctx, pool, seed); err != nil {
		return err
	}

	log.Printf("vehicle catalog seed applied successfully: makes=%d", len(seed.Makes))
	return nil
}

func loadSeederEnv() error {
	configFile, err := resolveSeederConfigFile(os.Getenv("CONFIG_FILE"))
	if err != nil {
		return err
	}

	return loadAllowedSeederEnv(configFile)
}

func loadAllowedSeederEnv(configFile allowedSeederConfigFile) error {
	switch configFile {
	case allowedDefaultEnv:
		return loadSeederEnvFile(defaultEnvFile, statDefaultEnvFile())
	case allowedBackendEnvFromRoot:
		return loadSeederEnvFile(backendEnvFileFromRoot, statBackendEnvFileFromRoot())
	case allowedDeploymentsEnvFromBackend:
		return loadSeederEnvFile(deploymentsEnvFileFromBackend, statDeploymentsEnvFileFromBackend())
	case allowedDeploymentsEnvFromRoot:
		return loadSeederEnvFile(deploymentsEnvFileFromRoot, statDeploymentsEnvFileFromRoot())
	default:
		return errors.New("unsupported seeder config file")
	}
}

func loadSeederEnvFile(envFile string, statErr error) error {
	if statErr == nil {
		var fileCfg seederEnv
		if err := cleanenv.ReadConfig(envFile, &fileCfg); err != nil {
			return fmt.Errorf("read config from %s: %w", envFile, err)
		}

		if os.Getenv("DATABASE_URL") == "" && fileCfg.DatabaseURL != "" {
			_ = os.Setenv("DATABASE_URL", fileCfg.DatabaseURL)
		}

		if err := applyDatabaseURLFile(fileCfg.DatabaseURLFile); err != nil {
			return err
		}
	} else {
		if errors.Is(statErr, os.ErrNotExist) {
			return applyDatabaseURLFile("")
		}
		return fmt.Errorf("check config file %s: %w", envFile, statErr)
	}

	return nil
}

func statDefaultEnvFile() error {
	_, err := os.Stat(defaultEnvFile)
	return err
}

func statBackendEnvFileFromRoot() error {
	_, err := os.Stat(backendEnvFileFromRoot)
	return err
}

func statDeploymentsEnvFileFromBackend() error {
	_, err := os.Stat(deploymentsEnvFileFromBackend)
	return err
}

func statDeploymentsEnvFileFromRoot() error {
	_, err := os.Stat(deploymentsEnvFileFromRoot)
	return err
}

func resolveSeederConfigFile(rawPath string) (allowedSeederConfigFile, error) {
	rawPath = strings.TrimSpace(rawPath)
	if rawPath == "" {
		return allowedDefaultEnv, nil
	}

	workDir, err := os.Getwd()
	if err != nil {
		return 0, fmt.Errorf("resolve working directory: %w", err)
	}

	allowedFiles := map[allowedSeederConfigFile]string{
		allowedDefaultEnv:                defaultEnvFile,
		allowedBackendEnvFromRoot:        backendEnvFileFromRoot,
		allowedDeploymentsEnvFromBackend: deploymentsEnvFileFromBackend,
		allowedDeploymentsEnvFromRoot:    deploymentsEnvFileFromRoot,
	}

	configPath := rawPath
	if !filepath.IsAbs(configPath) {
		configPath = filepath.Join(workDir, configPath)
	}
	configPath, err = filepath.Abs(filepath.Clean(configPath))
	if err != nil {
		return 0, fmt.Errorf("resolve config file path: %w", err)
	}

	for file, allowedPath := range allowedFiles {
		resolvedAllowedPath, err := filepath.Abs(filepath.Join(workDir, allowedPath))
		if err != nil {
			return 0, fmt.Errorf("resolve allowed config file path: %w", err)
		}
		if pathsEqual(configPath, resolvedAllowedPath) {
			return file, nil
		}
	}

	return 0, fmt.Errorf(
		"CONFIG_FILE must point to %s, %s, %s or %s",
		defaultEnvFile,
		backendEnvFileFromRoot,
		deploymentsEnvFileFromBackend,
		deploymentsEnvFileFromRoot,
	)
}

func pathsEqual(left, right string) bool {
	if filepath.Separator == '\\' {
		return strings.EqualFold(left, right)
	}

	return left == right
}

func applyDatabaseURLFile(fileCfgValue string) error {
	if os.Getenv("DATABASE_URL") != "" {
		return nil
	}

	path := strings.TrimSpace(os.Getenv("DATABASE_URL_FILE"))
	if path == "" {
		path = strings.TrimSpace(fileCfgValue)
	}
	if path == "" {
		return nil
	}
	if path != "/run/secrets/database_url" {
		return fmt.Errorf("DATABASE_URL_FILE must point to /run/secrets/database_url")
	}

	data, err := os.ReadFile("/run/secrets/database_url")
	if err != nil {
		return fmt.Errorf("DATABASE_URL_FILE: %w", err)
	}

	value := strings.TrimRight(string(data), "\r\n")
	if strings.TrimSpace(value) == "" {
		return errors.New("DATABASE_URL_FILE points to an empty secret file")
	}

	_ = os.Setenv("DATABASE_URL", value)
	return nil
}

func readSeed(seedPath string) (*vehicleCatalogSeed, error) {
	cleanPath := filepath.Clean(seedPath)
	data, err := os.ReadFile(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("read seed file %s: %w", cleanPath, err)
	}

	var seed vehicleCatalogSeed
	if err := json.Unmarshal(data, &seed); err != nil {
		return nil, fmt.Errorf("parse seed file %s: %w", cleanPath, err)
	}

	return &seed, nil
}

func validateSeed(seed *vehicleCatalogSeed) error {
	if seed == nil || len(seed.Makes) == 0 {
		return errors.New("seed must contain at least one make")
	}

	for _, makeItem := range seed.Makes {
		if err := validateMakeSeed(makeItem); err != nil {
			return err
		}
	}

	return nil
}

func validateMakeSeed(makeItem vehicleMakeSeed) error {
	if strings.TrimSpace(makeItem.Name) == "" || strings.TrimSpace(makeItem.Slug) == "" {
		return errors.New("make name and slug are required")
	}

	for _, modelItem := range makeItem.Models {
		if err := validateModelSeed(makeItem.Slug, modelItem); err != nil {
			return err
		}
	}

	return nil
}

func validateModelSeed(makeSlug string, modelItem vehicleModelSeed) error {
	if strings.TrimSpace(modelItem.Name) == "" || strings.TrimSpace(modelItem.Slug) == "" {
		return fmt.Errorf("model name and slug are required for make %q", makeSlug)
	}

	for _, generationItem := range modelItem.Generations {
		if err := validateGenerationSeed(modelItem.Slug, generationItem); err != nil {
			return err
		}
	}

	return nil
}

func validateGenerationSeed(modelSlug string, generationItem vehicleGenerationSeed) error {
	if strings.TrimSpace(generationItem.Name) == "" || strings.TrimSpace(generationItem.Slug) == "" {
		return fmt.Errorf("generation name and slug are required for model %q", modelSlug)
	}
	if generationItem.YearFrom < 1900 || generationItem.YearFrom > 2100 {
		return fmt.Errorf("invalid year_from for generation %q", generationItem.Slug)
	}
	if generationItem.YearTo != nil && (*generationItem.YearTo < generationItem.YearFrom || *generationItem.YearTo > 2100) {
		return fmt.Errorf("invalid year_to for generation %q", generationItem.Slug)
	}

	return nil
}

func seedVehicleCatalog(ctx context.Context, pool *pgxpool.Pool, seed *vehicleCatalogSeed) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	for _, makeItem := range seed.Makes {
		makeID, err := upsertMake(ctx, tx, makeItem)
		if err != nil {
			return err
		}

		for _, modelItem := range makeItem.Models {
			modelID, err := upsertModel(ctx, tx, makeID, modelItem)
			if err != nil {
				return err
			}

			for _, generationItem := range modelItem.Generations {
				if err := upsertGeneration(ctx, tx, modelID, generationItem); err != nil {
					return err
				}
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

func upsertMake(ctx context.Context, tx pgx.Tx, makeItem vehicleMakeSeed) (uuid.UUID, error) {
	const query = `
INSERT INTO vehicle_makes (name, slug, is_active, updated_at)
VALUES ($1, $2, true, CURRENT_TIMESTAMP)
ON CONFLICT (slug) DO UPDATE
SET name = EXCLUDED.name,
    is_active = true,
    updated_at = CURRENT_TIMESTAMP
RETURNING id;`

	var id uuid.UUID
	if err := tx.QueryRow(ctx, query, strings.TrimSpace(makeItem.Name), strings.TrimSpace(makeItem.Slug)).Scan(&id); err != nil {
		return uuid.Nil, fmt.Errorf("upsert make %q: %w", makeItem.Slug, err)
	}

	return id, nil
}

func upsertModel(ctx context.Context, tx pgx.Tx, makeID uuid.UUID, modelItem vehicleModelSeed) (uuid.UUID, error) {
	const query = `
INSERT INTO vehicle_models (make_id, name, slug, is_active, updated_at)
VALUES ($1, $2, $3, true, CURRENT_TIMESTAMP)
ON CONFLICT (make_id, slug) DO UPDATE
SET name = EXCLUDED.name,
    is_active = true,
    updated_at = CURRENT_TIMESTAMP
RETURNING id;`

	var id uuid.UUID
	if err := tx.QueryRow(ctx, query, makeID, strings.TrimSpace(modelItem.Name), strings.TrimSpace(modelItem.Slug)).Scan(&id); err != nil {
		return uuid.Nil, fmt.Errorf("upsert model %q: %w", modelItem.Slug, err)
	}

	return id, nil
}

func upsertGeneration(ctx context.Context, tx pgx.Tx, modelID uuid.UUID, generationItem vehicleGenerationSeed) error {
	const query = `
INSERT INTO vehicle_generations (model_id, name, slug, year_from, year_to, is_active, updated_at)
VALUES ($1, $2, $3, $4, $5, true, CURRENT_TIMESTAMP)
ON CONFLICT (model_id, slug, year_from) DO UPDATE
SET name = EXCLUDED.name,
    year_to = EXCLUDED.year_to,
    is_active = true,
    updated_at = CURRENT_TIMESTAMP;`

	_, err := tx.Exec(
		ctx,
		query,
		modelID,
		strings.TrimSpace(generationItem.Name),
		strings.TrimSpace(generationItem.Slug),
		generationItem.YearFrom,
		generationItem.YearTo,
	)
	if err != nil {
		return fmt.Errorf("upsert generation %q: %w", generationItem.Slug, err)
	}

	return nil
}
