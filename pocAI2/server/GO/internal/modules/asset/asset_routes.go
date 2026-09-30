package asset

import (
	"github.com/gofiber/fiber/v2"

	"golang-dh/internal/modules/asset/handler"
	"golang-dh/internal/modules/asset/service"
)

// SetupRoutes registers asset management endpoints on the Fiber router.
func SetupRoutes(router fiber.Router, authMiddleware fiber.Handler, svc *service.AssetService) {
	h := handler.NewAssetHandler(svc)

	assets := router.Group("/assets")
	if authMiddleware != nil {
		assets.Use(authMiddleware)
	}

	// Read operations
	assets.Get("", h.HandleGetAssets)
	assets.Get("/options", h.HandleGetAssetOptions)
	assets.Get("/download", h.HandleDownloadAssets)

	// Mutate operations
	assets.Post("", h.HandleCreateAsset)
	assets.Patch("/:id", h.HandleUpdateAsset)
	assets.Delete("/:id", h.HandleDeleteAsset)
}

