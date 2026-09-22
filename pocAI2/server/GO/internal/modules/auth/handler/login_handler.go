package handler

import (
	"strconv"

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

	// Set identity & multi-tenant headers (Tag Samurai style)
	c.Set("X-User-ID", res.User.ID)
	c.Set("X-User-Role", res.User.Role)
	c.Set("X-User-Name", res.User.Username)
	c.Set("X-Company", res.User.Company)
	if res.User.IdPerusahaan > 0 {
		c.Set("X-Company-ID", strconv.Itoa(res.User.IdPerusahaan))
	}

	token := res.Token
	if token == "" {
		token = "sess_" + res.User.ID
	}

	return c.JSON(fiber.Map{
		"message": "login successful",
		"token":   token,
		"user":    res.User,
	})
}
