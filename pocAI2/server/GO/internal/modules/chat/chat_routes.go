package chat

import (
	"github.com/gofiber/fiber/v2"

	"golang-dh/internal/modules/chat/handler"
	"golang-dh/internal/modules/chat/service"
)

// SetupRoutes registers chat endpoints.
func SetupRoutes(app *fiber.App, authMiddleware fiber.Handler, svc *service.ChatService) {
	h := handler.NewChatHandler(svc)

	if authMiddleware != nil {
		app.Get("/messages/:id", authMiddleware, h.HandleGetMessages)
	} else {
		app.Get("/messages/:id", h.HandleGetMessages)
	}
}
