package handler

import (
	"github.com/gofiber/fiber/v2"

	"golang-dh/internal/modules/company/service"
	"golang-dh/pkg/errs"
)

// CompanyHandler handles incoming HTTP requests for companies.
type CompanyHandler struct {
	svc *service.CompanyService
}

// NewCompanyHandler initializes a new CompanyHandler.
func NewCompanyHandler(svc *service.CompanyService) *CompanyHandler {
	return &CompanyHandler{svc: svc}
}

// HandleGetCompanies handles GET /api/perusahaan.
func (h *CompanyHandler) HandleGetCompanies(c *fiber.Ctx) error {
	ctx := c.Context()
	companies, err := h.svc.GetAllCompanies(ctx)
	if err != nil {
		return errs.InternalServerError("Failed to retrieve companies", err.Error()).SendResponse(c)
	}

	return c.JSON(fiber.Map{
		"companies": companies,
	})
}
