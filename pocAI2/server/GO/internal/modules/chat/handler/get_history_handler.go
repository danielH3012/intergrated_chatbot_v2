package handler

import (
	"strings"

	"github.com/gofiber/fiber/v2"

	"golang-dh/internal/modules/chat/service"
	"golang-dh/middleware"
	"golang-dh/pkg/errs"
)

// ChatHandler manages HTTP chat queries.
type ChatHandler struct {
	svc *service.ChatService
}

// NewChatHandler creates a new ChatHandler.
func NewChatHandler(svc *service.ChatService) *ChatHandler {
	return &ChatHandler{svc: svc}
}

// HandleGetMessages handles GET /messages/:id.
func (h *ChatHandler) HandleGetMessages(c *fiber.Ctx) error {
	var targetUsername string

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
		return errs.BadRequest("username is required").SendResponse(c)
	}

	result, err := h.svc.GetUserHistory(c.Context(), targetUsername)
	if err != nil {
		return errs.InternalServerError(err.Error()).SendResponse(c)
	}

	return c.JSON(result)
}
