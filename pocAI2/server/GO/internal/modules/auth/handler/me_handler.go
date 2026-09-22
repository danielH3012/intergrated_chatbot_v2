package handler

import (
	"github.com/gofiber/fiber/v2"

	"golang-dh/middleware"
	"golang-dh/pkg/errs"
)

// HandleGetMe handles GET /api/auth/me.
func (h *AuthHandler) HandleGetMe(c *fiber.Ctx) error {
	identity := middleware.RequestIdentity(c)
	if identity == nil {
		return errs.Unauthorized("Missing or invalid authorization context (X-User-ID header required)").SendResponse(c)
	}

	user, err := h.svc.GetMe(c.Context(), identity.UserID)
	if err != nil {
		// If user not in local DB but identity was passed via header (e.g. gateway/integration)
		return c.JSON(fiber.Map{
			"user": fiber.Map{
				"id":       identity.UserID,
				"username": identity.Username,
				"role":     identity.Role,
				"email":    identity.Email,
				"company":  identity.Company,
			},
		})
	}

	return c.JSON(fiber.Map{
		"user": user,
	})
}
