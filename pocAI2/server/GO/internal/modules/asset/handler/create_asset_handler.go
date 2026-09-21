package handler

import (
	"strings"

	"github.com/gofiber/fiber/v2"

	"golang-dh/internal/modules/asset/dto"
	"golang-dh/pkg/errs"
	"golang-dh/pkg/response"
)

// HandleCreateAsset handles POST /api/assets.
func (h *AssetHandler) HandleCreateAsset(c *fiber.Ctx) error {
	var req dto.CreateAssetRequest
	if err := c.BodyParser(&req); err != nil {
		return errs.BadRequest("Invalid JSON body", err.Error()).SendResponse(c)
	}

	_, company := extractUserIdentity(c)
	if company == "" {
		company = strings.TrimSpace(req.Perusahaan)
	}
	if company == "" {
		return errs.BadRequest("Company context is required to register asset").SendResponse(c)
	}

	newID, err := h.svc.CreateAsset(c.Context(), req, company)
	if err != nil {
		return errs.BadRequest(err.Error()).SendResponse(c)
	}

	return response.SendCreated(c, fiber.Map{
		"message": "Asset created successfully",
		"assetId": newID,
	}, "Asset created successfully")
}
