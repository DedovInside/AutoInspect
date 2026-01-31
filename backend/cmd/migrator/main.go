package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres" // драйвер PostgreSQL
	_ "github.com/golang-migrate/migrate/v4/source/file"       // читает .sql файлы из папки
)

func main() {

	// 1. Чтение конфигурации из переменных окружения

	// DATABASE_URL должен быть в формате:
	// postgres://username:password@host:port/database?sslmode=disable
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL environment variable is not set")
	}

	// Путь к папке с миграциями (можно переопределить через env)
	migrationsPath := os.Getenv("MIGRATIONS_PATH")
	if migrationsPath == "" {
		// Пытаемся найти migrations/ относительно текущей директории или exe
		ex, err := os.Executable()
		if err != nil {
			log.Fatal(err)
		}
		baseDir := filepath.Dir(ex)
		migrationsPath = "file://" + filepath.Join(baseDir, "..", "migrations") // предполагаем, что cmd/migrator/ --> backend/migrations/
	}

	// 2. Инициализация мигратора

	// migrate.New создаёт объект мигратора
	// Параметры:
	// - source: откуда читать миграции (file://, s3://, github://, etc.)
	// - database: куда применять (postgres://, mysql://, sqlite://, etc.)

	m, err := migrate.New(migrationsPath, dbURL)

	if err != nil {
		log.Fatalf("Failed to create migrate instance: %v", err)
	}
	defer m.Close() // Закрываем соединение с базой при завершении

	// 3. Парсим команду из аргументов

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	// 4. Выполняем команду

	switch command {

	// Команда up: применить все миграции
	case "up":
		fmt.Println("Applying migrations...")

		if err := m.Up(); err != nil {
			// migrate.ErrNoChange - это не ошибка, просто нечего мигрировать
			if errors.Is(err, migrate.ErrNoChange) {
				fmt.Println("No new migrations to apply")
				return
			}
			log.Fatalf("Migration failed: %v", err)
		}

		fmt.Println("All migrations applied successfully")

	// Команда down: откатить все миграции
	case "down":
		fmt.Println("Rolling back all migrations...")
		fmt.Print("Are you sure? This will delete all data! (yes/no): ")

		var confirmation string
		fmt.Scanln(&confirmation)

		if confirmation != "yes" {
			fmt.Println("Operation cancelled")
			return
		}

		if err := m.Down(); err != nil {
			if errors.Is(err, migrate.ErrNoChange) {
				fmt.Println("No migrations to roll back")
				return
			}
			log.Fatalf("Rollback failed: %v", err)
		}

		fmt.Println("All migrations rolled back successfully")

	// Команда steps: применить или откатить N миграций
	case "steps":
		if len(os.Args) < 3 {
			log.Fatal("Steps command requires a number argument")
		}

		var n int
		_, err := fmt.Sscanf(os.Args[2], "%d", &n)
		if err != nil {
			log.Fatalf("Invalid number of steps: %v", err)
		}

		if err := m.Steps(n); err != nil {
			if errors.Is(err, migrate.ErrNoChange) {
				fmt.Println("No migrations to apply or rollback")
				return
			}
			log.Fatalf("Steps migration failed: %v", err)
		}

		fmt.Printf("Successfully applied %d steps\n", n)

	// Команда goto: мигрировать к конкретной версии
	case "goto":
		if len(os.Args) < 3 {
			log.Fatal("Goto command requires a version argument")
		}

		var version uint
		_, err := fmt.Sscanf(os.Args[2], "%d", &version)
		if err != nil {
			log.Fatalf("Invalid version number: %v", err)
		}

		if err := m.Migrate(version); err != nil {
			if errors.Is(err, migrate.ErrNoChange) {
				fmt.Println("Already at the specified version")
				return
			}
			log.Fatalf("Goto migration failed: %v", err)
		}

		fmt.Printf("Successfully migrated to version %d\n", version)

	// Команда version: показать текущую версию миграции
	case "version":
		version, dirty, err := m.Version()
		if err != nil {
			if errors.Is(err, migrate.ErrNilVersion) {
				fmt.Println("Database is at initial version (no migrations applied)")
				return
			}
			log.Fatalf("Failed to get current version: %v", err)
		}
		fmt.Printf("Current version: %d, Dirty state: %v\n", version, dirty)

		// dirty - флаг, указывающий, была ли прервана миграция
		// Если dirty == true, нужно вручную исправить состояние базы

		if dirty {
			fmt.Println("WARNING: Database is in dirty state!")
			fmt.Println("Last migration was interrupted. Manual intervention required.")
			fmt.Println("Use 'force <VERSION>' to reset the state.")
		}

	// Команда force: принудительно установить версию миграции
	// (используется для исправления dirty состояния)

	case "force":
		if len(os.Args) < 3 {
			log.Fatal("Force command requires a version argument")
		}

		var version uint
		if _, err := fmt.Sscanf(os.Args[2], "%d", &version); err != nil {
			log.Fatalf("Invalid version number: %v", err)
		}

		fmt.Printf("Forcing version to %d...\n", version)
		fmt.Printf("WARNING: This does NOT run migrations! Continue? (yes/no): ")

		var confirmation string
		fmt.Scanln(&confirmation)

		if confirmation != "yes" {
			fmt.Println("Operation cancelled")
			return
		}

		if err := m.Force(int(version)); err != nil {
			log.Fatalf("Failed to force version: %v", err)
		}

		fmt.Printf("Successfully forced version to %d\n", version)

	// Команда drop: удалить все таблицы из базы данных
	case "drop":
		fmt.Println("Dropping all tables from the database...")
		fmt.Print("Are you sure? This will delete ALL DATA! (yes/no): ")

		var confirmation string
		fmt.Scanln(&confirmation)

		if confirmation != "yes" {
			fmt.Println("Operation cancelled")
			return
		}

		if err := m.Drop(); err != nil {
			log.Fatalf("Failed to drop database: %v", err)
		}

		fmt.Println("All tables dropped successfully")

	// Неизвестная команда: показать справку
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)

	}
}

// Вспомогательная функция: справка
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
						Default: file://migrations
	`)
}
