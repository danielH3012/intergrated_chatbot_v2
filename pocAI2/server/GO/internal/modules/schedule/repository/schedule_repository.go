package repository

import (
	"context"
	"fmt"
	"strings"

	"golang-dh/internal/modules/schedule/dto"
	"golang-dh/pkg/database"
)

// ScheduleRepository defines data access methods for schedule records.
type ScheduleRepository interface {
	Create(ctx context.Context, category, start, frequency string) (string, error)
	List(ctx context.Context, categoryFilter string) ([]dto.ScheduleItem, error)
	Delete(ctx context.Context, id string) (int64, error)
	EnsureTable(ctx context.Context) error
}

type scheduleRepository struct {
	db database.PostgreSQLServicer
}

// NewScheduleRepository creates a new instance of ScheduleRepository.
func NewScheduleRepository(db database.PostgreSQLServicer) ScheduleRepository {
	return &scheduleRepository{db: db}
}

func (r *scheduleRepository) EnsureTable(ctx context.Context) error {
	const createSQL = `
	CREATE TABLE IF NOT EXISTS public.schedule (
	    id_schedule uuid NOT NULL DEFAULT gen_random_uuid(),
	    category character varying(125),
	    start character varying(125),
	    frequency character varying(125),
	    CONSTRAINT schedule_pkey PRIMARY KEY (id_schedule)
	);`
	_, err := r.db.Exec(ctx, createSQL)
	return err
}

func (r *scheduleRepository) Create(ctx context.Context, category, start, frequency string) (string, error) {
	query := `
		INSERT INTO public.schedule (category, start, frequency)
		VALUES ($1, $2, $3)
		RETURNING id_schedule;
	`
	var newID string
	err := r.db.QueryRow(ctx, query, category, start, frequency).Scan(&newID)
	if err != nil {
		return "", err
	}
	return newID, nil
}

func (r *scheduleRepository) List(ctx context.Context, categoryFilter string) ([]dto.ScheduleItem, error) {
	var query string
	var args []any

	if strings.TrimSpace(categoryFilter) != "" {
		query = `SELECT id_schedule, category, start, frequency FROM public.schedule WHERE LOWER(category) LIKE LOWER($1) ORDER BY start ASC`
		args = append(args, "%"+strings.TrimSpace(categoryFilter)+"%")
	} else {
		query = `SELECT id_schedule, category, start, frequency FROM public.schedule ORDER BY start ASC`
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var schedules []dto.ScheduleItem
	for rows.Next() {
		var s dto.ScheduleItem
		if err := rows.Scan(&s.IDSchedule, &s.Category, &s.Start, &s.Frequency); err != nil {
			return nil, fmt.Errorf("failed to scan schedule row: %w", err)
		}
		schedules = append(schedules, s)
	}
	if schedules == nil {
		schedules = []dto.ScheduleItem{}
	}
	return schedules, nil
}

func (r *scheduleRepository) Delete(ctx context.Context, id string) (int64, error) {
	tag, err := r.db.Exec(ctx, `DELETE FROM public.schedule WHERE id_schedule = $1`, id)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}
