package config

import (
    "log/slog"

    "github.com/gofiber/fiber/v2"

    "latihan-fiber/helper"
    "latihan-fiber/middleware"
    "latihan-fiber/route"
)

// Menyesuaikan parameter dengan struct Dependencies
func NewApp(logger *slog.Logger, deps route.Dependencies, allowedOrigins string) *fiber.App {
    app := fiber.New(fiber.Config{
        AppName:      "API Students (Clean Arch)",
        ErrorHandler: newErrorHandler(logger),
        
        // body dibatasi biar memori server ga abis
        BodyLimit:    1 * 1024 * 1024, // 1 MB
    })

  //middleware global
    middleware.Register(app, logger, allowedOrigins)
    route.Register(app, deps)

    // 404 untuk route yg gada
    app.Use(func(c *fiber.Ctx) error {
        return helper.Fail(c, fiber.StatusNotFound, "endpoint tidak ditemukan")
    })

    return app
}

func newErrorHandler(logger *slog.Logger) fiber.ErrorHandler {
    return func(c *fiber.Ctx, err error) error {
        status := fiber.StatusInternalServerError
        message := "terjadi error pada server"

        if e, ok := err.(*fiber.Error); ok {
            status = e.Code
            message = e.Message
        }

        logger.Error("unhandled_error",
            slog.String("path", c.Path()),
            slog.Int("status", status),
            slog.String("error", err.Error()),
        )

        return helper.Fail(c, status, message)
    }
}