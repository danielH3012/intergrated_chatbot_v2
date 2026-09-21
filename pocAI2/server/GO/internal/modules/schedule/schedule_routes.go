package schedule

import (
	"github.com/gofiber/fiber/v2"

	"golang-dh/internal/modules/schedule/handler"
	"golang-dh/internal/modules/schedule/service"
)

// SetupRoutes registers schedule endpoints with the Fiber router.
func SetupRoutes(router fiber.Router, svc *service.ScheduleService) {
	h := handler.NewScheduleHandler(svc)

	router.Post("/schedule", h.HandleCreateSchedule)
	router.Get("/schedule", h.HandleGetSchedules)
	router.Delete("/schedule/:id", h.HandleDeleteSchedule)
}
