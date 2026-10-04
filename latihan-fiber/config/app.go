package config

import (
	"errors"
	"log/slog"

	"github.com/gofiber/fiber/v2"

	"latihan-fiber/app/model"
	"latihan-fiber/helper"
	"latihan-fiber/middleware"
	"latihan-fiber/route"
)

func NewApp(logger *slog.Logger, deps route.Dependencies, allowedOrigins string) *fiber.App {
	app := fiber.New(fiber.Config{
		AppName:      "API Students (Clean Arch)",
		ErrorHandler: newErrorHandler(logger),
		BodyLimit:    1 * 1024 * 1024,
	})

	middleware.Register(app, logger, allowedOrigins)
	route.Register(app, deps)

	app.Use(func(c *fiber.Ctx) error {
        // benerin: Handler return helper.NotFound, bukan helper.Fail
		return helper.NotFound("endpoint tidak ditemukan")
	})

	return app
}

func newErrorHandler(logger *slog.Logger) fiber.ErrorHandler {
	return func(c *fiber.Ctx, err error) error {
		// benerin bug 5: Ambil request ID langsung dari Locals Fiber
		requestID, _ := c.Locals("requestid").(string)
		
		var appErr *helper.AppError

		switch {
		case errors.As(err, &appErr):
			// eror yg direncanakan
		case errors.Is(err, fiber.ErrRequestEntityTooLarge):
			appErr = &helper.AppError{
				Status:  fiber.StatusRequestEntityTooLarge,
				Code:    "PAYLOAD_TOO_LARGE",
				Message: "ukuran body melebihi batas yang diizinkan",
			}
		default:
			var fiberErr *fiber.Error
			if errors.As(err, &fiberErr) {
				appErr = &helper.AppError{
					Status:  fiberErr.Code, 
					Code:    "HTTP_ERROR",
					Message: fiberErr.Message,
				}
			} else {
				appErr = helper.Internal(err)
			}
		}

		// perbaikan bug 3 masih dipake 
		if appErr.Status >= fiber.StatusInternalServerError {
			
			// benerin bug 6: Gunakan Unwrap() karena 'cause' bersifat private
			var errMsg string
			if appErr.Unwrap() != nil {
				errMsg = appErr.Unwrap().Error()
			}
			
			logger.Error("request_failed",
				slog.String("request_id", requestID),
				slog.String("path", c.Path()),
				slog.String("code", appErr.Code),
				slog.Int("status", appErr.Status),
				slog.String("error", errMsg))
		} else {
			logger.Warn("request_rejected",
				slog.String("request_id", requestID),
				slog.String("path", c.Path()),
				slog.String("code", appErr.Code),
				slog.Int("status", appErr.Status))
		}

		return c.Status(appErr.Status).JSON(model.ErrorResponse{
			Success:   false,
			Code:      appErr.Code,
			Message:   appErr.Message,
			Fields:    appErr.Fields,
			RequestID: requestID,
		})
	}
}