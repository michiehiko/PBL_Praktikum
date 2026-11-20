package helper

import (
	"github.com/gofiber/fiber/v2"
	"latihan-fiber/app/model"
)

// Seluruh response dibentuk dari struct di package model, bukan fiber.Map.
// Bentuk JSON-nya sama persis dengan sebelumnya; bedanya, struct punya nama
// sehingga swag dapat mendokumentasikan isinya (lihat D.3 Modul 12).

func Success(c *fiber.Ctx, status int, message string, data any) error {
	return c.Status(status).JSON(model.SuccessResponse{
		Success: true,
		Message: message,
		Data:    data,
	})
}

func SuccessList(c *fiber.Ctx, message string, data any, meta *model.Meta) error {
	return c.Status(fiber.StatusOK).JSON(model.WebResponse{
		Success: true, Message: message, Data: data, Meta: meta,
	})
}

func Created(c *fiber.Ctx, message string, data any, location string) error {
	c.Set("Location", location)
	return c.Status(fiber.StatusCreated).JSON(model.SuccessResponse{
		Success: true,
		Message: message,
		Data:    data,
	})
}

func NoContent(c *fiber.Ctx) error {
	return c.SendStatus(fiber.StatusNoContent)
}

func SuccessCursor(c *fiber.Ctx, message string, data any, meta *model.CursorMeta) error {
	return c.Status(fiber.StatusOK).JSON(model.CursorPage{
		Success: true,
		Message: message,
		Data:    data,
		Meta:    meta,
	})
}
