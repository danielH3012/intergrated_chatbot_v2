package handler

import (
	"github.com/gofiber/fiber/v2"

	"golang-dh/internal/modules/auth/dto"
	"golang-dh/pkg/errs"
)

// HandleLogin handles POST /api/auth/login.
func (h *AuthHandler) HandleLogin(c *fiber.Ctx) error {
	var req dto.LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return errs.BadRequest("Invalid request payload", err.Error()).SendResponse(c)
	}

	res, err := h.svc.Login(c.Context(), req)
	if err != nil {
		return errs.Unauthorized(err.Error()).SendResponse(c)
	}

	return c.JSON(fiber.Map{
		"token": res.Token,
		"user":  res.User,
	})
}
