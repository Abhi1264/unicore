package handlers

import (
	"errors"

	"github.com/Abhi1264/unicore/api/internal/middleware"
	"github.com/gofiber/fiber/v2"
)

// ErrResponseWritten means the body is already sent; ErrorHandler must no-op.
var ErrResponseWritten = errors.New("response written")

type apiErrorBody struct {
	Error     string `json:"error"`
	Code      string `json:"code,omitempty"`
	RequestID string `json:"request_id,omitempty"`
}

func JSONError(c *fiber.Ctx, status int, code, msg string) error {
	rid, _ := c.Locals(middleware.KeyRequestID).(string)
	if err := c.Status(status).JSON(apiErrorBody{
		Error:     msg,
		Code:      code,
		RequestID: rid,
	}); err != nil {
		return err
	}
	return ErrResponseWritten
}

func JSON(c *fiber.Ctx, status int, data any) error {
	return c.Status(status).JSON(data)
}
