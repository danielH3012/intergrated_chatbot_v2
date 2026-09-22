package middleware_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"golang-dh/middleware"

	"github.com/gofiber/fiber/v2"
)

func TestHeaderAuth_Success(t *testing.T) {
	app := fiber.New()
	app.Get("/protected", middleware.HeaderAuth(), func(c *fiber.Ctx) error {
		identity := middleware.RequestIdentity(c)
		if identity == nil {
			return c.Status(fiber.StatusInternalServerError).SendString("nil identity")
		}
		return c.JSON(fiber.Map{
			"user_id":  identity.UserID,
			"role":     identity.Role,
			"username": identity.Username,
		})
	})

	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("X-User-ID", "123")
	req.Header.Set("X-User-Role", "admin")
	req.Header.Set("X-User-Name", "superadmin")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if string(body) == "" {
		t.Fatal("empty response body")
	}
}

func TestHeaderAuth_MissingHeader(t *testing.T) {
	app := fiber.New()
	app.Get("/protected", middleware.HeaderAuth(), func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})

	req := httptest.NewRequest("GET", "/protected", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

func TestRequireRoles(t *testing.T) {
	app := fiber.New()
	app.Get("/admin-only", middleware.HeaderAuth(), middleware.RequireRoles("admin", "superadmin"), func(c *fiber.Ctx) error {
		return c.SendString("welcome admin")
	})

	// 1. Forbidden for operator
	req1 := httptest.NewRequest("GET", "/admin-only", nil)
	req1.Header.Set("X-User-ID", "456")
	req1.Header.Set("X-User-Role", "operator")
	resp1, err := app.Test(req1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp1.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", resp1.StatusCode)
	}

	// 2. Allowed for admin
	req2 := httptest.NewRequest("GET", "/admin-only", nil)
	req2.Header.Set("X-User-ID", "789")
	req2.Header.Set("X-User-Role", "admin")
	resp2, err := app.Test(req2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp2.StatusCode)
	}
}
