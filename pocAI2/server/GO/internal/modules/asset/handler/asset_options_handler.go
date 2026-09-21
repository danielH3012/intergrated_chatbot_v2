package handler

import (
	"github.com/gofiber/fiber/v2"

	"golang-dh/pkg/errs"
)

// HandleGetAssetOptions handles GET /api/assets/options.
func (h *AssetHandler) HandleGetAssetOptions(c *fiber.Ctx) error {
	_, company := extractUserIdentity(c)
	options, err := h.svc.GetOptions(c.Context(), company)
	if err != nil {
		return errs.InternalServerError("Failed to fetch asset options", err.Error()).SendResponse(c)
	}

	return c.JSON(options)
}
