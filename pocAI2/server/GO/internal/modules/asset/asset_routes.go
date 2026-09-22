package asset

import (
	"github.com/gofiber/fiber/v2"

	"golang-dh/internal/modules/asset/handler"
	"golang-dh/internal/modules/asset/service"
	"golang-dh/middleware"
)

// SetupRoutes registers asset management endpoints on the Fiber router with RBAC enforcement.
func SetupRoutes(router fiber.Router, authMiddleware fiber.Handler, svc *service.AssetService) {
	h := handler.NewAssetHandler(svc)

	assets := router.Group("/assets")
	if authMiddleware != nil {
		assets.Use(authMiddleware)
	}

	// Read operations: allowed for operator, admin, superadmin
	assets.Get("", h.HandleGetAssets)
	assets.Get("/options", h.HandleGetAssetOptions)
	assets.Get("/download", h.HandleDownloadAssets)

	// Mutate operations: RBAC restricted to admin and superadmin
	assets.Post("", middleware.RequireRoles("admin", "superadmin"), h.HandleCreateAsset)
	assets.Patch("/:id", middleware.RequireRoles("admin", "superadmin"), h.HandleUpdateAsset)
	assets.Delete("/:id", middleware.RequireRoles("admin", "superadmin"), h.HandleDeleteAsset)
}
