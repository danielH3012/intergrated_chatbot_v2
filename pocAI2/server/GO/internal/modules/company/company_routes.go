package company

import (
	"github.com/gofiber/fiber/v2"

	"golang-dh/internal/modules/company/handler"
	"golang-dh/internal/modules/company/service"
)

// SetupRoutes registers routes for the company/perusahaan module.
func SetupRoutes(router fiber.Router, svc *service.CompanyService) {
	h := handler.NewCompanyHandler(svc)
	router.Get("/perusahaan", h.HandleGetCompanies)
}
