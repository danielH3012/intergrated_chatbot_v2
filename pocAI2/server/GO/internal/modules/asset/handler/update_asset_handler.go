package handler

import (
	"strings"

	"github.com/gofiber/fiber/v2"

	"golang-dh/internal/modules/asset/dto"
	"golang-dh/pkg/errs"
	"golang-dh/pkg/response"
)

// HandleUpdateAsset handles PATCH /api/assets/:id.
func (h *AssetHandler) HandleUpdateAsset(c *fiber.Ctx) error {
	assetID := strings.TrimSpace(c.Params("id"))
	if assetID == "" {
		return errs.BadRequest("Asset ID is required in URL parameter").SendResponse(c)
	}

	var req dto.UpdateAssetRequest
	if err := c.BodyParser(&req); err != nil {
		return errs.BadRequest("Invalid JSON body", err.Error()).SendResponse(c)
	}

	role, company := extractUserIdentity(c)
	if err := h.svc.UpdateAsset(c.Context(), assetID, req, role, company); err != nil {
		if strings.Contains(err.Error(), "not found") {
			return errs.NotFound(err.Error()).SendResponse(c)
		}
		if strings.Contains(err.Error(), "forbidden") {
			return errs.Forbidden(err.Error()).SendResponse(c)
		}
		return errs.InternalServerError("Failed to update asset", err.Error()).SendResponse(c)
	}

	return response.SendSuccess(c, fiber.Map{
		"message": "Asset updated successfully",
		"assetId": assetID,
	}, "Asset updated successfully")
}
