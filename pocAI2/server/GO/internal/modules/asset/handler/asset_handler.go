package handler

import (
	"strings"

	"github.com/gofiber/fiber/v2"

	"golang-dh/internal/modules/asset/service"
	"golang-dh/middleware"
)

// AssetHandler manages HTTP requests for IT assets.
type AssetHandler struct {
	svc *service.AssetService
}

// NewAssetHandler initializes a new AssetHandler.
func NewAssetHandler(svc *service.AssetService) *AssetHandler {
	return &AssetHandler{svc: svc}
}

// extractUserIdentity resolves role and company from RequestIdentity or multi-tenant headers.
func extractUserIdentity(c *fiber.Ctx) (role, company string) {
	if identity := middleware.RequestIdentity(c); identity != nil {
		role = identity.Role
		company = identity.Company
	}
	if company == "" {
		company = strings.TrimSpace(c.Get("X-Company"))
	}
	if company == "" {
		company = strings.TrimSpace(c.Get("X-Company-Name"))
	}
	if company == "" {
		company = strings.TrimSpace(c.Get("X-Tenant-ID"))
	}
	if company == "" && strings.EqualFold(role, "superadmin") {
		company = strings.TrimSpace(c.Query("perusahaan"))
	}
	return role, company
}
