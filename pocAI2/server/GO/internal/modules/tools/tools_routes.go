package tools

import (
	"github.com/gofiber/fiber/v2"

	"golang-dh/internal/modules/tools/handler"
	"golang-dh/internal/modules/tools/service"
)

// SetupRoutes registers routes for tools execution history.
func SetupRoutes(router fiber.Router, authMiddleware fiber.Handler, svc *service.ToolsService) {
	h := handler.NewToolsHandler(svc)
	if authMiddleware != nil {
		router.Post("/Tools", authMiddleware, h.HandleCreateToolsHistory)
	} else {
		router.Post("/Tools", h.HandleCreateToolsHistory)
	}
}
