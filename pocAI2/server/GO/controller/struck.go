package controllers

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

// DB is the central PostgreSQL connection pool.
var DB *pgxpool.Pool

// Connect initializes the PostgreSQL connection pool using server/GO/.env and bootstraps the database schema.
func Connect() {
	_ = godotenv.Load(".env")
	connectPostgres()
}

func getDBDSN() string {
	// 1. DATABASE_URL from .env
	if dsn := os.Getenv("DATABASE_URL"); dsn != "" {
		return dsn
	}

	// 2. Constructed from DB_* in .env
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
		return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s", user, password, host, port, dbname, sslmode)
	}
	return fmt.Sprintf("postgres://%s@%s:%s/%s?sslmode=%s", user, host, port, dbname, sslmode)
}

func connectPostgres() {
	dsn := getDBDSN()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatalf("[PostgreSQL] Failed to create connection pool: %v (Target DSN: %s)", err, dsn)
	}
	if err := pool.Ping(ctx); err != nil {
		log.Printf("[PostgreSQL] Warning: Ping failed: %v. Target DSN: %s", err, dsn)
	} else {
		fmt.Printf("[PostgreSQL] Connected successfully to database: %s\n", dsn)
	}
	DB = pool
}
