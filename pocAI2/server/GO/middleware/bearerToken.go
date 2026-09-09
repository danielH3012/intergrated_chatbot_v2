package middleware

import (
	"errors"
	"strings"

	"github.com/gofiber/fiber/v2"
)

func BearerAuth() fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "missing authorization header",
			})
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "invalid authorization format",
			})
		}

		token := parts[1]

		// validasi token (misal cek JWT signature/expiry, atau cek ke DB/cache)
		claims, err := validateToken(token)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "invalid or expired token",
			})
		}

		// simpan info user/claims ke context biar bisa dipakai handler selanjutnya
		c.Locals("user", claims)

		return c.Next()
	}
}

func validateToken(token string) (any, any) {
	if token != "token123" { // contoh validasi sederhana
		return "", errors.New("invalid token")
	}
	return "system", nil // atau return identitas user
}
