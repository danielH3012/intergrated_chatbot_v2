package controllers

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"golang-dh/models"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

const createScheduleTableSQL = `
CREATE TABLE IF NOT EXISTS public.schedule
(
    id_schedule uuid NOT NULL DEFAULT gen_random_uuid(),
    category character varying(125) COLLATE pg_catalog."default",
    start character varying(125) COLLATE pg_catalog."default",
    frequency character varying(125) COLLATE pg_catalog."default",
    CONSTRAINT schedule_pkey PRIMARY KEY (id_schedule)
);
`

// EnsureScheduleTable bootstraps the public.schedule table if it doesn't already exist.
func EnsureScheduleTable(ctx context.Context) error {
	if DB == nil {
		return fmt.Errorf("database connection pool is not initialized")
	}
	_, err := DB.Exec(ctx, createScheduleTableSQL)
	if err != nil {
		log.Printf("[EnsureScheduleTable] Failed to bootstrap public.schedule: %v", err)
		return err
	}
	log.Println("[EnsureScheduleTable] Table public.schedule verified/created successfully.")
	return nil
}

// CreateScheduleRequest represents payload for inserting a new schedule.
type CreateScheduleRequest struct {
	Category  string `json:"category" form:"category"`
	Start     string `json:"start" form:"start"`
	Frequency string `json:"frequency" form:"frequency"`
}

// CreateSchedule handles POST /api/schedule to insert a new schedule record.
func CreateSchedule(c *fiber.Ctx) error {
	var req CreateScheduleRequest
	if err := c.BodyParser(&req); err != nil {
		// Fallback to query or form parameters if JSON parse failed
		req.Category = c.FormValue("category")
		req.Start = c.FormValue("start")
		req.Frequency = c.FormValue("frequency")
	}

	if req.Category == "" {
		req.Category = c.Query("category")
	}
	if req.Start == "" {
		req.Start = c.Query("start")
	}
	if req.Frequency == "" {
		req.Frequency = c.Query("frequency")
	}

	category := strings.TrimSpace(req.Category)
	start := strings.TrimSpace(req.Start)
	frequency := strings.TrimSpace(req.Frequency)

	if category == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"ok":    false,
			"error": "field 'category' is required",
		})
	}

	if start == "" {
		start = time.Now().Format("2006-01-02")
	}
	if frequency == "" {
		frequency = "Once"
	}

	// Truncate safely to 125 chars to honor VARCHAR(125) schema constraint
	if len(category) > 125 {
		category = category[:125]
	}
	if len(start) > 125 {
		start = start[:125]
	}
	if len(frequency) > 125 {
		frequency = frequency[:125]
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Ensure table exists
	_ = EnsureScheduleTable(ctx)

	var created models.Schedule
	var returnedID uuid.UUID

	query := `
		INSERT INTO public.schedule (category, start, frequency)
		VALUES ($1, $2, $3)
		RETURNING id_schedule, category, start, frequency
	`

	err := DB.QueryRow(ctx, query, category, start, frequency).Scan(
		&returnedID,
		&created.Category,
		&created.Start,
		&created.Frequency,
	)
	if err != nil {
		log.Printf("[CreateSchedule] Insert error: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"ok":    false,
			"error": fmt.Sprintf("failed to insert schedule: %v", err),
		})
	}

	created.IDSchedule = returnedID.String()

	log.Printf("[CreateSchedule] Successfully inserted schedule: ID=%s, Category=%s, Start=%s, Frequency=%s",
		created.IDSchedule, created.Category, created.Start, created.Frequency)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"ok":      true,
		"message": "Schedule record created successfully",
		"data":    created,
	})
}

// GetSchedules handles GET /api/schedule to list all schedule records.
func GetSchedules(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_ = EnsureScheduleTable(ctx)

	categoryFilter := strings.TrimSpace(c.Query("category"))
	var query string
	var args []interface{}

	if categoryFilter != "" {
		query = `SELECT id_schedule, category, start, frequency FROM public.schedule WHERE LOWER(category) LIKE LOWER($1) ORDER BY start ASC`
		args = append(args, "%"+categoryFilter+"%")
	} else {
		query = `SELECT id_schedule, category, start, frequency FROM public.schedule ORDER BY start ASC`
	}

	rows, err := DB.Query(ctx, query, args...)
	if err != nil {
		log.Printf("[GetSchedules] Query error: %v", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"ok":    false,
			"error": fmt.Sprintf("failed to query schedules: %v", err),
		})
	}
	defer rows.Close()

	var schedules []models.Schedule
	for rows.Next() {
		var s models.Schedule
		var id uuid.UUID
		if err := rows.Scan(&id, &s.Category, &s.Start, &s.Frequency); err == nil {
			s.IDSchedule = id.String()
			schedules = append(schedules, s)
		}
	}

	if schedules == nil {
		schedules = []models.Schedule{}
	}

	return c.JSON(fiber.Map{
		"ok":        true,
		"schedules": schedules,
		"total":     len(schedules),
	})
}

// DeleteSchedule handles DELETE /api/schedule/:id to delete a schedule record by ID.
func DeleteSchedule(c *fiber.Ctx) error {
	idStr := strings.TrimSpace(c.Params("id"))
	if idStr == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"ok":    false,
			"error": "id parameter is required",
		})
	}

	parsedID, err := uuid.Parse(idStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"ok":    false,
			"error": "invalid UUID format for id_schedule",
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res, err := DB.Exec(ctx, `DELETE FROM public.schedule WHERE id_schedule = $1`, parsedID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"ok":    false,
			"error": fmt.Sprintf("failed to delete schedule: %v", err),
		})
	}

	if res.RowsAffected() == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"ok":    false,
			"error": "schedule not found",
		})
	}

	return c.JSON(fiber.Map{
		"ok":      true,
		"message": "Schedule deleted successfully",
		"id":      idStr,
	})
}
