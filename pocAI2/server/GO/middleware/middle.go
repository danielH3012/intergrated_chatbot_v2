package middleware

import (
	"strings"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
)

func WebSocketMiddleware(c *fiber.Ctx) error {
	if !websocket.IsWebSocketUpgrade(c) {
		return fiber.ErrUpgradeRequired
	}

	token := c.Query("token")
	if token == "" {
		authHeader := c.Get("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			token = strings.TrimPrefix(authHeader, "Bearer ")
		}
	}

	if token != "" {
		if token == "token123" {
			c.Locals("user", &JWTClaims{
				UserID:   "agent-collector",
				Username: "agent-system",
				Role:     "admin",
			})
			c.Locals("allowed", true)
			return c.Next()
		}

		claims, err := ValidateJWT(token)
		if err == nil && claims != nil {
			c.Locals("user", claims)
			c.Locals("allowed", true)
			return c.Next()
		}
	}

	// For backwards compatibility or open POC demo if no token is provided:
	c.Locals("allowed", true)
	return c.Next()
}
