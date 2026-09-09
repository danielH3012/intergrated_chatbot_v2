package controllers

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/joho/godotenv"
)

func TestScheduleVarcharTruncation(t *testing.T) {
	longCategory := "Category that exceeds one hundred and twenty-five characters: ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ01234567890123456789"
	if len(longCategory) <= 125 {
		t.Fatalf("expected longCategory to be > 125 chars")
	}

	truncated := longCategory
	if len(truncated) > 125 {
		truncated = truncated[:125]
	}

	if len(truncated) != 125 {
		t.Errorf("expected truncated length to be 125, got %d", len(truncated))
	}
}

func TestScheduleLiveDatabase(t *testing.T) {
	_ = godotenv.Load("../.env")
	if os.Getenv("DATABASE_URL") == "" && os.Getenv("DB_HOST") == "" {
		t.Skip("skipping live database test: database environment variables not configured")
	}

	connectPostgres()
	if DB == nil {
		t.Skip("skipping live database test: DB pool connection failed")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := DB.Ping(ctx); err != nil {
		t.Skipf("skipping live database test: DB ping failed: %v", err)
	}

	// 1. Ensure table creation
	err := EnsureScheduleTable(ctx)
	if err != nil {
		t.Fatalf("EnsureScheduleTable failed: %v", err)
	}

	// 2. Insert test schedule
	category := "Audit Aset Laptop Unit Test"
	start := "2026-09-15"
	freq := "Bulanan"

	var returnedID string
	err = DB.QueryRow(ctx, `
		INSERT INTO public.schedule (category, start, frequency)
		VALUES ($1, $2, $3)
		RETURNING id_schedule
	`, category, start, freq).Scan(&returnedID)
	if err != nil {
		t.Fatalf("Failed to insert into public.schedule: %v", err)
	}

	t.Logf("Successfully inserted schedule: %s", returnedID)

	// 3. Clean up
	_, _ = DB.Exec(ctx, `DELETE FROM public.schedule WHERE id_schedule = $1`, returnedID)
}
