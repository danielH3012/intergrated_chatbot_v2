package response

import (
	"github.com/gofiber/fiber/v2"
)

// StandardResponse is the canonical API response payload across all modules.
type StandardResponse struct {
	Code    int    `json:"code"`
	Status  string `json:"status"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
	Meta    any    `json:"meta,omitempty"`
	Errors  any    `json:"errors,omitempty"`
}

// PaginationMeta encapsulates list pagination details.
type PaginationMeta struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"pageSize"`
	TotalItems int64 `json:"totalItems"`
	TotalPages int   `json:"totalPages"`
}

// SendResponse outputs a formatted JSON response conforming to the enterprise standard.
func SendResponse(c *fiber.Ctx, statusCode int, data any, message string, meta any) error {
	status := "success"
	if statusCode >= 400 {
		status = "error"
	}
	if message == "" {
		if status == "success" {
			message = "Operation completed successfully"
		} else {
			message = "An error occurred"
		}
	}
	return c.Status(statusCode).JSON(StandardResponse{
		Code:    statusCode,
		Status:  status,
		Message: message,
		Data:    data,
		Meta:    meta,
	})
}

// SendSuccess sends a 200 OK response with data.
func SendSuccess(c *fiber.Ctx, data any, message string) error {
	return SendResponse(c, fiber.StatusOK, data, message, nil)
}

// SendCreated sends a 201 Created response with data.
func SendCreated(c *fiber.Ctx, data any, message string) error {
	return SendResponse(c, fiber.StatusCreated, data, message, nil)
}

// SendPaginated sends a 200 OK response with list items and pagination metadata.
func SendPaginated(c *fiber.Ctx, data any, meta PaginationMeta, message string) error {
	return SendResponse(c, fiber.StatusOK, data, message, meta)
}

// SendNoContent sends a 204 No Content response.
func SendNoContent(c *fiber.Ctx) error {
	return c.SendStatus(fiber.StatusNoContent)
}
