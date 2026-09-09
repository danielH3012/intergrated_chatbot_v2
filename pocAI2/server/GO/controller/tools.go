package controllers

import (
	"context"
	"strings"
	"time"

	"golang-dh/middleware"
	"golang-dh/models"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// CreateToolsHistory records a tool execution into public.tools_history.
func CreateToolsHistory(c *fiber.Ctx) error {
	var body struct {
		Perusahaan string `json:"perusahaan"`
		Tools      string `json:"tools"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	body.Perusahaan = strings.TrimSpace(body.Perusahaan)
	body.Tools = strings.TrimSpace(body.Tools)

	if body.Tools == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "tools parameter is required"})
	}

	userVal := c.Locals("user")
	var userID, username, role string
	if claims, ok := userVal.(*middleware.JWTClaims); ok && claims != nil {
		userID = claims.UserID
		username = claims.Username
		role = claims.Role
		if body.Perusahaan == "" {
			body.Perusahaan = claims.Company
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	entry := models.ToolsHistoryPayload{
		ID:                uuid.New().String(),
		Perusahaan:        body.Perusahaan,
		Tools:             body.Tools,
		CreatedByUserID:   userID,
		CreatedByUsername: username,
		CreatedByRole:     role,
		CreatedAt:         time.Now(),
	}

	_, err := DB.Exec(ctx,
		`INSERT INTO tools_history (id, perusahaan, tools, created_by_user_id, created_by_username, created_by_role, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		entry.ID, entry.Perusahaan, entry.Tools, entry.CreatedByUserID, entry.CreatedByUsername, entry.CreatedByRole, entry.CreatedAt,
	)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "failed to record tool history: " + err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(entry)
}
