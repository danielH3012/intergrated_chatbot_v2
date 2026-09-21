package database

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

// PostgreSQLServicer defines the standardized interface for PostgreSQL database access.
// This interface allows seamless mocking in unit tests.
type PostgreSQLServicer interface {
	GetPool() *pgxpool.Pool
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Begin(ctx context.Context) (pgx.Tx, error)
}

// PostgreSQLService wraps a pgxpool.Pool to implement PostgreSQLServicer.
type PostgreSQLService struct {
	pool *pgxpool.Pool
}

// NewPostgreSQLService creates a new PostgreSQLService from an initialized pool.
func NewPostgreSQLService(pool *pgxpool.Pool) *PostgreSQLService {
	return &PostgreSQLService{pool: pool}
}

func (s *PostgreSQLService) GetPool() *pgxpool.Pool {
	return s.pool
}

func (s *PostgreSQLService) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return s.pool.Query(ctx, sql, args...)
}

func (s *PostgreSQLService) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return s.pool.QueryRow(ctx, sql, args...)
}

func (s *PostgreSQLService) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	return s.pool.Exec(ctx, sql, args...)
}

func (s *PostgreSQLService) Begin(ctx context.Context) (pgx.Tx, error) {
	return s.pool.Begin(ctx)
}

// InitPool initializes a pgxpool connection pool reading credentials from .env.
func InitPool() (*pgxpool.Pool, error) {
	_ = godotenv.Load(".env")

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		host := os.Getenv("DB_HOST")
		if host == "" {
			host = "localhost"
		}
		port := os.Getenv("DB_PORT")
		if port == "" {
			port = "5432"
		}
		user := os.Getenv("DB_USER")
		if user == "" {
			user = "postgres"
		}
		password := os.Getenv("DB_PASSWORD")
		dbname := os.Getenv("DB_NAME")
		if dbname == "" {
			dbname = "poc_ai"
		}
		sslmode := os.Getenv("DB_SSLMODE")
		if sslmode == "" {
			sslmode = "disable"
		}

		if password != "" {
			dsn = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s", user, password, host, port, dbname, sslmode)
		} else {
			dsn = fmt.Sprintf("postgres://%s@%s:%s/%s?sslmode=%s", user, host, port, dbname, sslmode)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w (Target DSN: %s)", err, dsn)
	}

	if err := pool.Ping(ctx); err != nil {
		log.Printf("[PostgreSQL] Warning: ping failed: %v. Target DSN: %s", err, dsn)
	} else {
		log.Printf("[PostgreSQL] Connected successfully to database: %s", dsn)
	}

	return pool, nil
}
