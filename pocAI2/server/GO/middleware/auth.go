package middleware

import (
	"errors"
	"os"
	"strconv"
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

// Identity is a type alias to JWTClaims for Tag Samurai BETS-V2 header-based identity compatibility.
type Identity = JWTClaims

// RequestIdentity extracts the user identity and multi-tenant context from request headers or context
func RequestIdentity(c *fiber.Ctx) *Identity {
	if val := c.Locals("user"); val != nil {
		if id, ok := val.(*Identity); ok && id != nil {
			return id
		}
	}

	userID := strings.TrimSpace(c.Get("X-User-ID"))
	if userID == "" {
		return nil
	}

	role := strings.TrimSpace(c.Get("X-User-Role"))
	if role == "" {
		role = strings.TrimSpace(c.Get("X-Role"))
	}
	username := strings.TrimSpace(c.Get("X-User-Name"))
	if username == "" {
		username = strings.TrimSpace(c.Get("X-Username"))
	}
	email := strings.TrimSpace(c.Get("X-User-Email"))

	// Multi-tenant company context extraction
	company := strings.TrimSpace(c.Get("X-Company"))
	if company == "" {
		company = strings.TrimSpace(c.Get("X-Company-Name"))
	}
	if company == "" {
		company = strings.TrimSpace(c.Get("X-Tenant-ID"))
	}

	companyIDStr := strings.TrimSpace(c.Get("X-Company-ID"))
	companyID, _ := strconv.Atoi(companyIDStr)

	id := &Identity{
		UserID:       userID,
		Role:         role,
		Username:     username,
		Email:        email,
		Company:      company,
		IdPerusahaan: companyID,
	}
	c.Locals("user", id)
	return id
}

// HeaderAuth authenticates incoming requests using headers (X-User-ID, X-User-Role)
// ala Tag Samurai BETS-V2 / KrakenD Gateway pattern.
func HeaderAuth() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// 1. Try reading identity from headers (Tag Samurai style)
		if id := RequestIdentity(c); id != nil {
			return c.Next()
		}

		// 2. Backward compatibility fallback: Bearer token or agent telemetry (token123)
		authHeader := c.Get("Authorization")
		if authHeader != "" {
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) == 2 && parts[0] == "Bearer" {
				tokenStr := parts[1]
				if tokenStr == "token123" {
					c.Locals("user", &Identity{
						UserID:   "agent-collector",
						Username: "agent-system",
						Role:     "admin",
					})
					return c.Next()
				}
				claims, err := ValidateJWT(tokenStr)
				if err == nil && claims != nil {
					c.Locals("user", claims)
					return c.Next()
				}
			}
		}

		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "missing or invalid X-User-ID header",
		})
	}
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
	return HeaderAuth()
}

// RequireRoles restricts route access to users with one of the allowed roles
func RequireRoles(allowedRoles ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id := RequestIdentity(c)
		if id == nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "unauthorized",
			})
		}

		for _, role := range allowedRoles {
			if strings.EqualFold(id.Role, role) {
				return c.Next()
			}
		}

		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error":          "forbidden: insufficient permissions",
			"required_roles": allowedRoles,
			"current_role":   id.Role,
		})
	}
}
