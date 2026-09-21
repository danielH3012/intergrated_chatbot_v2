package asset

import (
	"github.com/gofiber/fiber/v2"

	"golang-dh/internal/modules/asset/handler"
	"golang-dh/internal/modules/asset/service"
)

// SetupRoutes registers asset management endpoints on the Fiber router.
func SetupRoutes(router fiber.Router, svc *service.AssetService) {
	h := handler.NewAssetHandler(svc)

	router.Get("/assets", h.HandleGetAssets)
	router.Get("/assets/options", h.HandleGetAssetOptions)
	router.Get("/assets/download", h.HandleDownloadAssets)
	router.Post("/assets", h.HandleCreateAsset)
	router.Patch("/assets/:id", h.HandleUpdateAsset)
	router.Delete("/assets/:id", h.HandleDeleteAsset)
}
