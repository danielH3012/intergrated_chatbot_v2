package middleware

import (
	"errors"
	"os"
	"strings"
	"time"

	"golang-dh/models"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

var jwtSecret = []byte(getEnv("JWT_SECRET", "qtera-super-secret-jwt-key-poc-2026"))

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

type JWTClaims struct {
	UserID       string `json:"user_id"`
	Username     string `json:"username"`
	Email        string `json:"email"`
	Role         string `json:"role"`
	Company      string `json:"company,omitempty"`
	IdPerusahaan int    `json:"id_perusahaan,omitempty"`
	jwt.RegisteredClaims
}

// GenerateJWT creates a new JWT token for a user valid for 24 hours
func GenerateJWT(user *models.User) (string, error) {
	claims := JWTClaims{
		UserID:       user.ID, // already a string UUID
		Username:     user.Username,
		Email:        user.Email,
		Role:         user.Role,
		Company:      user.Company,
		IdPerusahaan: user.IdPerusahaan,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "qtera-server",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// ValidateJWT verifies the token string and returns parsed claims
func ValidateJWT(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token claims")
}

// JWTAuth authenticates Bearer tokens for API endpoints with backward-compatible agent token support
func JWTAuth() fiber.Handler {
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

		tokenStr := parts[1]

		// Support fallback for agent telemetry ingestion (token123)
		if tokenStr == "token123" {
			c.Locals("user", &JWTClaims{
				UserID:   "agent-collector",
				Username: "agent-system",
				Role:     "admin",
			})
			return c.Next()
		}

		claims, err := ValidateJWT(tokenStr)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "invalid or expired token",
			})
		}

		c.Locals("user", claims)
		return c.Next()
	}
}

// RequireRoles restricts route access to users with one of the allowed roles
func RequireRoles(allowedRoles ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userVal := c.Locals("user")
		if userVal == nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "unauthorized",
			})
		}

		claims, ok := userVal.(*JWTClaims)
		if !ok || claims == nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "invalid user context",
			})
		}

		for _, role := range allowedRoles {
			if strings.EqualFold(claims.Role, role) {
				return c.Next()
			}
		}

		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "forbidden: insufficient permissions",
			"required_roles": allowedRoles,
			"current_role":  claims.Role,
		})
	}
}
