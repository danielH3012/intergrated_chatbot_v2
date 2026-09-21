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

// extractUserIdentity resolves role and company from token claims or fallback headers.
func extractUserIdentity(c *fiber.Ctx) (role, company string) {
	if userVal := c.Locals("user"); userVal != nil {
		if claims, ok := userVal.(*middleware.JWTClaims); ok && claims != nil {
			role = claims.Role
			company = claims.Company
		}
	}
	if company == "" {
		company = strings.TrimSpace(c.Get("X-Company-Name"))
	}
	if company == "" {
		company = strings.TrimSpace(c.Get("X-Company-Id"))
	}
	if company == "" {
		company = strings.TrimSpace(c.Query("perusahaan"))
	}
	return role, company
}
