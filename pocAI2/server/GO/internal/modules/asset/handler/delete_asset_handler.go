package handler

import (
	"strings"

	"github.com/gofiber/fiber/v2"

	"golang-dh/pkg/errs"
	"golang-dh/pkg/response"
)

// HandleDeleteAsset handles DELETE /api/assets/:id.
func (h *AssetHandler) HandleDeleteAsset(c *fiber.Ctx) error {
	assetID := strings.TrimSpace(c.Params("id"))
	if assetID == "" {
		return errs.BadRequest("Asset ID is required in URL parameter").SendResponse(c)
	}

	role, company := extractUserIdentity(c)
	if err := h.svc.DeleteAsset(c.Context(), assetID, role, company); err != nil {
		if strings.Contains(err.Error(), "not found") {
			return errs.NotFound(err.Error()).SendResponse(c)
		}
		if strings.Contains(err.Error(), "forbidden") {
			return errs.Forbidden(err.Error()).SendResponse(c)
		}
		return errs.InternalServerError("Failed to delete asset", err.Error()).SendResponse(c)
	}

	return response.SendSuccess(c, fiber.Map{
		"message": "Asset deleted successfully",
		"assetId": assetID,
	}, "Asset deleted successfully")
}
