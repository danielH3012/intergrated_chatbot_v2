package handler

import (
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"

	"golang-dh/internal/modules/asset/dto"
	"golang-dh/pkg/errs"
)

// HandleGetAssets handles GET /api/assets.
func (h *AssetHandler) HandleGetAssets(c *fiber.Ctx) error {
	_, company := extractUserIdentity(c)

	if company == "" {
		return c.JSON(fiber.Map{
			"assets": []dto.AssetItem{},
			"pagination": fiber.Map{
				"page":       1,
				"pageSize":   10,
				"totalItems": 0,
				"totalPages": 1,
			},
		})
	}

	page, err := strconv.Atoi(c.Query("page", "1"))
	if err != nil || page < 1 {
		page = 1
	}

	pageSizeStr := c.Query("pageSize", "10")
	allRecords := pageSizeStr == "all"
	pageSize, err := strconv.Atoi(pageSizeStr)
	if err != nil || pageSize < 1 {
		pageSize = 10
	}

	filter := dto.AssetListFilter{
		Perusahaan: company,
		Search:     strings.TrimSpace(c.Query("search")),
		Category:   strings.TrimSpace(c.Query("category")),
		Brand:      strings.TrimSpace(c.Query("brand")),
		Location:   strings.TrimSpace(c.Query("location")),
		Status:     strings.TrimSpace(c.Query("status")),
		Sort:       strings.TrimSpace(c.Query("sort", "createdAt")),
		Order:      strings.TrimSpace(c.Query("order", "DESC")),
		Page:       page,
		PageSize:   pageSize,
		AllRecords: allRecords,
	}

	result, err := h.svc.ListAssets(c.Context(), filter)
	if err != nil {
		return errs.InternalServerError("Failed to retrieve assets", err.Error()).SendResponse(c)
	}

	return c.JSON(fiber.Map{
		"assets": result.Assets,
		"pagination": fiber.Map{
			"page":       result.Page,
			"pageSize":   result.PageSize,
			"totalItems": result.TotalItems,
			"totalPages": result.TotalPages,
		},
	})
}
