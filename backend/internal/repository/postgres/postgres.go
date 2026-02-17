package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type DB struct {
	pool *pgxpool.Pool
}

// Создаём новое подключение к базе данных с пулом соединений

func New(ctx context.Context, dsn string, maxConns int, connLifetime time.Duration) (*DB, error) {
	config, err := pgxpool.ParseConfig(dsn)

	if err != nil {
		return nil, fmt.Errorf("failed to parse DSN: %w", err)
	}

	// Настройка пула соединений
	config.MaxConns = int32(maxConns)
	config.MinConns = int32(maxConns / 4)
	config.MaxConnLifetime = connLifetime
	config.MaxConnIdleTime = 10 * time.Minute
	config.HealthCheckPeriod = 1 * time.Minute

	// Создание пула с настройками
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close() // Закрываем пул перед возвратом ошибки
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DB{
		pool: pool,
	}, nil
}

// CLose закрывает пул соединений
func (db *DB) Close() {
	db.pool.Close()
}

// Ping проверяет соединение с базой данных
func (db *DB) Ping(ctx context.Context) error {
	return db.pool.Ping(ctx)
}

// GetPool возвращает пул соединений для выполнения запросов
func (db *DB) GetPool() *pgxpool.Pool {
	return db.pool
}
