package handler

import (
	"strings"

	"github.com/gofiber/fiber/v2"

	"golang-dh/internal/modules/schedule/dto"
	"golang-dh/internal/modules/schedule/service"
	"golang-dh/pkg/errs"
	"golang-dh/pkg/response"
)

// ScheduleHandler routes HTTP requests to the ScheduleService.
type ScheduleHandler struct {
	svc *service.ScheduleService
}

// NewScheduleHandler creates a new ScheduleHandler instance.
func NewScheduleHandler(svc *service.ScheduleService) *ScheduleHandler {
	return &ScheduleHandler{svc: svc}
}

// HandleCreateSchedule handles POST /api/schedule.
func (h *ScheduleHandler) HandleCreateSchedule(c *fiber.Ctx) error {
	var req dto.CreateScheduleRequest
	if err := c.BodyParser(&req); err != nil {
		req.Category = c.FormValue("category")
		req.Start = c.FormValue("start")
		req.Frequency = c.FormValue("frequency")
	}
	if req.Category == "" {
		req.Category = c.Query("category")
	}
	if req.Start == "" {
		req.Start = c.Query("start")
	}
	if req.Frequency == "" {
		req.Frequency = c.Query("frequency")
	}

	id, err := h.svc.CreateSchedule(c.Context(), req)
	if err != nil {
		return errs.BadRequest(err.Error()).SendResponse(c)
	}

	return response.SendCreated(c, fiber.Map{"id_schedule": id}, "Schedule created successfully")
}

// HandleGetSchedules handles GET /api/schedule.
func (h *ScheduleHandler) HandleGetSchedules(c *fiber.Ctx) error {
	category := c.Query("category")
	schedules, err := h.svc.GetSchedules(c.Context(), category)
	if err != nil {
		return errs.InternalServerError("Failed to retrieve schedules", err.Error()).SendResponse(c)
	}

	return c.JSON(fiber.Map{
		"schedules": schedules,
	})
}

// HandleDeleteSchedule handles DELETE /api/schedule/:id.
func (h *ScheduleHandler) HandleDeleteSchedule(c *fiber.Ctx) error {
	id := strings.TrimSpace(c.Params("id"))
	if id == "" {
		return errs.BadRequest("Invalid or missing id parameter").SendResponse(c)
	}

	if err := h.svc.DeleteSchedule(c.Context(), id); err != nil {
		if strings.Contains(err.Error(), "not found") {
			return errs.NotFound(err.Error()).SendResponse(c)
		}
		return errs.InternalServerError("Failed to delete schedule", err.Error()).SendResponse(c)
	}

	return response.SendSuccess(c, fiber.Map{"id": id}, "Schedule deleted successfully")
}
