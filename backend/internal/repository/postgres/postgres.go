package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DBTX interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type DB struct {
	pool *pgxpool.Pool
}

func New(ctx context.Context, dsn string, maxConns int, connLifetime time.Duration) (*DB, error) {
	config, err := pgxpool.ParseConfig(dsn)

	if err != nil {
		return nil, fmt.Errorf("failed to parse DSN: %w", err)
	}

	config.MaxConns = int32(maxConns)
	config.MinConns = int32(maxConns / 4)
	config.MaxConnLifetime = connLifetime
	config.MaxConnIdleTime = 10 * time.Minute
	config.HealthCheckPeriod = 1 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DB{
		pool: pool,
	}, nil
}

func (db *DB) Close() {
	db.pool.Close()
}

func (db *DB) Begin(ctx context.Context) (pgx.Tx, error) {
	return db.pool.Begin(ctx)
}

func (db *DB) Ping(ctx context.Context) error {
	return db.pool.Ping(ctx)
}

func (db *DB) GetPool() *pgxpool.Pool {
	return db.pool
}

func (db *DB) Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
	return db.pool.Exec(ctx, sql, args...)
}

func (db *DB) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	return db.pool.Query(ctx, sql, args...)
}

func (db *DB) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	return db.pool.QueryRow(ctx, sql, args...)
}
