package handler

import (
	"github.com/gofiber/fiber/v2"

	"golang-dh/middleware"
	"golang-dh/pkg/errs"
)

// HandleGetMe handles GET /api/auth/me.
func (h *AuthHandler) HandleGetMe(c *fiber.Ctx) error {
	userVal := c.Locals("user")
	if userVal == nil {
		return errs.Unauthorized("Missing or invalid authorization context").SendResponse(c)
	}

	claims, ok := userVal.(*middleware.JWTClaims)
	if !ok || claims == nil {
		return errs.Unauthorized("Invalid user identity in token").SendResponse(c)
	}

	user, err := h.svc.GetMe(c.Context(), claims.UserID)
	if err != nil {
		return errs.NotFound("User account not found").SendResponse(c)
	}

	return c.JSON(fiber.Map{
		"user": user,
	})
}
