package controllers

import (
	"context"
	"strings"
	"time"

	"golang-dh/middleware"
	"golang-dh/models"

	"github.com/gofiber/fiber/v2"
)

type ChatResponse struct {
	ID         string                 `json:"id"`
	ChatID     string                 `json:"chat_id"`
	UserID     string                 `json:"user_id,omitempty"`
	Username   string                 `json:"username,omitempty"`
	Role       string                 `json:"role,omitempty"`
	User       bool                   `json:"user"` // serialised from is_user
	Chat       string                 `json:"chat"`
	Models     string                 `json:"models,omitempty"`
	CreatedAt  time.Time              `json:"created_at"`
	Attachment *models.AttachmentInfo `json:"attachment,omitempty"`
}

func GetMessages(c *fiber.Ctx) error {
	targetUsername := ""
	if userVal := c.Locals("user"); userVal != nil {
		if claims, ok := userVal.(*middleware.JWTClaims); ok && claims != nil {
			targetUsername = strings.TrimSpace(claims.Username)
		}
	}
	if targetUsername == "" {
		targetUsername = strings.TrimSpace(c.Params("id"))
		targetUsername = strings.TrimPrefix(targetUsername, "user_")
	}
	if targetUsername == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "username is required",
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rows, err := DB.Query(ctx,
		`SELECT id, "user", chat, created_at, COALESCE(user_id::text, ''), username, role, models,
		        COALESCE(attachment_name, ''), COALESCE(attachment_url, ''), COALESCE(attachment_type, ''), COALESCE(attachment_size, 0)
		 FROM chat
		 WHERE username ILIKE $1
		 ORDER BY created_at ASC`,
		targetUsername,
	)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	defer rows.Close()

	var result []ChatResponse
	for rows.Next() {
		var (
			id, chatText, createdAtStr, uidStr, uname, role, modelsName string
			attachName, attachURL, attachType                           string
			attachSize                                                  int64
			isUser                                                      bool
		)
		if err := rows.Scan(&id, &isUser, &chatText, &createdAtStr, &uidStr, &uname, &role, &modelsName,
			&attachName, &attachURL, &attachType, &attachSize); err != nil {
			continue
		}

		displayUsername := uname
		if !isUser && role == "system" {
			displayUsername = "QTERA AI"
		} else if displayUsername == "" {
			if isUser {
				displayUsername = "User"
			} else {
				displayUsername = "QTERA AI"
			}
		}

		var parsedTime time.Time
		if t, err := time.Parse("2006-01-02 15:04:05", createdAtStr); err == nil {
			parsedTime = t
		} else if t, err := time.Parse(time.RFC3339, createdAtStr); err == nil {
			parsedTime = t
		} else {
			parsedTime = time.Now()
		}

		var attach *models.AttachmentInfo
		if attachName != "" {
			attach = &models.AttachmentInfo{
				Name: attachName,
				URL:  attachURL,
				Type: attachType,
				Size: attachSize,
			}
		}

		result = append(result, ChatResponse{
			ID:         id,
			ChatID:     targetUsername,
			UserID:     uidStr,
			Username:   displayUsername,
			Role:       role,
			User:       isUser,
			Chat:       chatText,
			Models:     modelsName,
			CreatedAt:  parsedTime,
			Attachment: attach,
		})
	}

	if result == nil {
		result = []ChatResponse{}
	}
	return c.JSON(result)
}
