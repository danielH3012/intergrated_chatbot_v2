package auth

import (
	"github.com/gofiber/fiber/v2"

	"golang-dh/internal/modules/auth/handler"
	"golang-dh/internal/modules/auth/service"
)

// SetupRoutes mounts authentication endpoints to the Fiber router.
func SetupRoutes(router fiber.Router, authMiddleware fiber.Handler, svc *service.AuthService) {
	h := handler.NewAuthHandler(svc)

	authGroup := router.Group("/auth")
	authGroup.Post("/register", h.HandleRegister)
	authGroup.Post("/login", h.HandleLogin)
	if authMiddleware != nil {
		authGroup.Get("/me", authMiddleware, h.HandleGetMe)
	} else {
		authGroup.Get("/me", h.HandleGetMe)
	}
}
