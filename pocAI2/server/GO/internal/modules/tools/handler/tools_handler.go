package handler

import (
	"github.com/gofiber/fiber/v2"

	"golang-dh/internal/modules/tools/dto"
	"golang-dh/internal/modules/tools/service"
	"golang-dh/pkg/errs"
	"golang-dh/pkg/response"
)

// ToolsHandler handles incoming HTTP requests for tool audit logs.
type ToolsHandler struct {
	svc *service.ToolsService
}

// NewToolsHandler creates a new ToolsHandler instance.
func NewToolsHandler(svc *service.ToolsService) *ToolsHandler {
	return &ToolsHandler{svc: svc}
}

// HandleCreateToolsHistory handles POST /api/Tools.
func (h *ToolsHandler) HandleCreateToolsHistory(c *fiber.Ctx) error {
	var req dto.CreateToolsHistoryRequest
	if err := c.BodyParser(&req); err != nil {
		return errs.BadRequest("Invalid request body", err.Error()).SendResponse(c)
	}

	id, err := h.svc.RecordToolExecution(c.Context(), req)
	if err != nil {
		return errs.BadRequest(err.Error()).SendResponse(c)
	}

	return response.SendCreated(c, fiber.Map{
		"message": "success",
		"id":      id,
	}, "Tool execution recorded successfully")
}
