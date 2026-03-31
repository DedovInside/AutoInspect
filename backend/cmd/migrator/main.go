package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/ilyakaznacheev/cleanenv"
)

const confirmationYes = "yes"

func main() {
	if err := run(); err != nil {
		log.Printf("migrator failed: %v", err)
		os.Exit(1)
	}
}

func run() error {
	if err := loadMigratorEnv(); err != nil {
		return err
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return errors.New("DATABASE_URL environment variable is not set")
	}

	migrationsPath, err := resolveMigrationsPath()
	if err != nil {
		return err
	}

	m, err := migrate.New(migrationsPath, dbURL)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}
	defer closeMigrator(m)

	if len(os.Args) < 2 {
		printUsage()
		return errors.New("command is required")
	}

	command := os.Args[1]
	args := os.Args[2:]

	if err := executeCommand(m, command, args); err != nil {
		return err
	}

	return nil
}

type migratorEnv struct {
	DatabaseURL    string `env:"DATABASE_URL"`
	MigrationsPath string `env:"MIGRATIONS_PATH"`
}

func loadMigratorEnv() error {
	envFile := os.Getenv("CONFIG_FILE")
	if envFile == "" {
		envFile = ".env"
	}

	if _, err := os.Stat(envFile); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("check config file %s: %w", envFile, err)
	}

	var fileCfg migratorEnv
	if err := cleanenv.ReadConfig(envFile, &fileCfg); err != nil {
		return fmt.Errorf("read config from %s: %w", envFile, err)
	}

	if os.Getenv("DATABASE_URL") == "" && fileCfg.DatabaseURL != "" {
		_ = os.Setenv("DATABASE_URL", fileCfg.DatabaseURL)
	}
	if os.Getenv("MIGRATIONS_PATH") == "" && fileCfg.MigrationsPath != "" {
		_ = os.Setenv("MIGRATIONS_PATH", fileCfg.MigrationsPath)
	}

	return nil
}

func resolveMigrationsPath() (string, error) {
	migrationsPath := os.Getenv("MIGRATIONS_PATH")
	if migrationsPath != "" {
		return migrationsPath, nil
	}

	ex, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("resolve executable path: %w", err)
	}

	baseDir := filepath.Dir(ex)
	return "file://" + filepath.Join(baseDir, "..", "migrations"), nil
}

func closeMigrator(m *migrate.Migrate) {
	if m == nil {
		return
	}

	sourceErr, dbErr := m.Close()
	if sourceErr != nil {
		log.Printf("warning: failed to close migration source: %v", sourceErr)
	}
	if dbErr != nil {
		log.Printf("warning: failed to close migration database connection: %v", dbErr)
	}
}

func executeCommand(m *migrate.Migrate, command string, args []string) error {
	switch command {
	case "up":
		return runUp(m)
	case "down":
		return runDown(m)
	case "steps":
		return runSteps(m, args)
	case "goto":
		return runGoto(m, args)
	case "version":
		return runVersion(m)
	case "force":
		return runForce(m, args)
	case "drop":
		return runDrop(m)
	default:
		printUsage()
		return fmt.Errorf("unknown command: %s", command)
	}
}

func runUp(m *migrate.Migrate) error {
	fmt.Println("Applying migrations...")

	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			fmt.Println("No new migrations to apply")
			return nil
		}
		return fmt.Errorf("migration failed: %w", err)
	}

	fmt.Println("All migrations applied successfully")
	return nil
}

func runDown(m *migrate.Migrate) error {
	fmt.Println("Rolling back all migrations...")

	confirmed, err := askForConfirmation("Are you sure? This will delete all data! (yes/no): ")
	if err != nil {
		return err
	}
	if !confirmed {
		fmt.Println("Operation cancelled")
		return nil
	}

	if err := m.Down(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			fmt.Println("No migrations to roll back")
			return nil
		}
		return fmt.Errorf("rollback failed: %w", err)
	}

	fmt.Println("All migrations rolled back successfully")
	return nil
}

func runSteps(m *migrate.Migrate, args []string) error {
	n, err := parseIntArg(args, "steps")
	if err != nil {
		return err
	}

	if err := m.Steps(n); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			fmt.Println("No migrations to apply or rollback")
			return nil
		}
		return fmt.Errorf("steps migration failed: %w", err)
	}

	fmt.Printf("Successfully applied %d steps\n", n)
	return nil
}

func runGoto(m *migrate.Migrate, args []string) error {
	version, err := parseUintArg(args, "goto")
	if err != nil {
		return err
	}

	if err := m.Migrate(version); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			fmt.Println("Already at the specified version")
			return nil
		}
		return fmt.Errorf("goto migration failed: %w", err)
	}

	fmt.Printf("Successfully migrated to version %d\n", version)
	return nil
}

func runVersion(m *migrate.Migrate) error {
	version, dirty, err := m.Version()
	if err != nil {
		if errors.Is(err, migrate.ErrNilVersion) {
			fmt.Println("Database is at initial version (no migrations applied)")
			return nil
		}
		return fmt.Errorf("failed to get current version: %w", err)
	}

	fmt.Printf("Current version: %d, Dirty state: %v\n", version, dirty)
	if dirty {
		fmt.Println("WARNING: Database is in dirty state!")
		fmt.Println("Last migration was interrupted. Manual intervention required.")
		fmt.Println("Use 'force <VERSION>' to reset the state.")
	}

	return nil
}

func runForce(m *migrate.Migrate, args []string) error {
	version, err := parseUintArg(args, "force")
	if err != nil {
		return err
	}

	fmt.Printf("Forcing version to %d...\n", version)
	confirmed, err := askForConfirmation("WARNING: This does NOT run migrations! Continue? (yes/no): ")
	if err != nil {
		return err
	}
	if !confirmed {
		fmt.Println("Operation cancelled")
		return nil
	}

	if err := m.Force(int(version)); err != nil {
		return fmt.Errorf("failed to force version: %w", err)
	}

	fmt.Printf("Successfully forced version to %d\n", version)
	return nil
}

func runDrop(m *migrate.Migrate) error {
	fmt.Println("Dropping all tables from the database...")
	confirmed, err := askForConfirmation("Are you sure? This will delete ALL DATA! (yes/no): ")
	if err != nil {
		return err
	}
	if !confirmed {
		fmt.Println("Operation cancelled")
		return nil
	}

	if err := m.Drop(); err != nil {
		return fmt.Errorf("failed to drop database: %w", err)
	}

	fmt.Println("All tables dropped successfully")
	return nil
}

func askForConfirmation(prompt string) (bool, error) {
	fmt.Print(prompt)
	var confirmation string
	if _, err := fmt.Scanln(&confirmation); err != nil {
		return false, fmt.Errorf("read confirmation: %w", err)
	}

	return confirmation == confirmationYes, nil
}

func parseIntArg(args []string, command string) (int, error) {
	if len(args) < 1 {
		return 0, fmt.Errorf("%s command requires a number argument", command)
	}

	var n int
	if _, err := fmt.Sscanf(args[0], "%d", &n); err != nil {
		return 0, fmt.Errorf("invalid %s argument: %w", command, err)
	}

	return n, nil
}

func parseUintArg(args []string, command string) (uint, error) {
	if len(args) < 1 {
		return 0, fmt.Errorf("%s command requires a version argument", command)
	}

	var version uint
	if _, err := fmt.Sscanf(args[0], "%d", &version); err != nil {
		return 0, fmt.Errorf("invalid %s version: %w", command, err)
	}

	return version, nil
}

func printUsage() {
	fmt.Println(`
	AutoInspect Database Migrator
	Usage: migrator <command> [arguments]

	Commands:
	up              Apply all pending migrations
	down            Rollback all migrations (requires confirmation)
	steps <N>       Apply N migrations forward (or -N backward)
	goto <VERSION>  Migrate to specific version
	version         Show current migration version
	force <VERSION> Force set version (use only to fix dirty state)
	drop            Drop all tables (requires confirmation)

	Examples:
	migrator up                 # Apply all migrations
	migrator down               # Rollback everything
	migrator steps 1            # Apply next migration
	migrator steps -1           # Rollback last migration
	migrator goto 3             # Migrate to version 3
	migrator version            # Show current version
	migrator force 2            # Force version to 2 (emergency only!)

	Environment Variables:
	DATABASE_URL       Required. PostgreSQL connection string
						Example: postgres://user:pass@localhost:5432/autoinspect?sslmode=disable

	MIGRATIONS_PATH    Optional. Path to migrations folder
						Default: file://<executable>/../migrations

	CONFIG_FILE        Optional. Path to .env-like config file (default: .env)
						Values from process env have higher priority
	`)
}
