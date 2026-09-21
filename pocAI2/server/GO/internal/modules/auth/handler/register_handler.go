package handler

import (
	"github.com/gofiber/fiber/v2"

	"golang-dh/internal/modules/auth/dto"
	"golang-dh/pkg/errs"
)

// HandleRegister handles POST /api/auth/register.
func (h *AuthHandler) HandleRegister(c *fiber.Ctx) error {
	var req dto.RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return errs.BadRequest("Invalid request payload", err.Error()).SendResponse(c)
	}

	res, err := h.svc.Register(c.Context(), req)
	if err != nil {
		return errs.BadRequest(err.Error()).SendResponse(c)
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"token": res.Token,
		"user":  res.User,
	})
}
