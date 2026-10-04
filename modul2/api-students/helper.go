package main

import (
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
)

//fungsi pembuat respons
func ok(c *fiber.Ctx, message string, data any) error {
	return c.Status(fiber.StatusOK).JSON(WebResponse{
		Success: true, Message: message, Data: data,
	})
}

func okList(c *fiber.Ctx, message string, data any, meta *Meta) error {
	return c.Status(fiber.StatusOK).JSON(WebResponse{
		Success: true, Message: message, Data: data, Meta: meta,
	})
}

func created(c *fiber.Ctx, message string, data any, location string) error {
	c.Set("Location", location) 
	return c.Status(fiber.StatusCreated).JSON(WebResponse{
		Success: true, Message: message, Data: data,
	})
}

func noContent(c *fiber.Ctx) error {
	return c.SendStatus(fiber.StatusNoContent) 
}

func fail(c *fiber.Ctx, status int, message string) error {
	return c.Status(status).JSON(WebResponse{Success: false, Message: message})
}

func failValidation(c *fiber.Ctx, errs map[string]string) error {
	return c.Status(fiber.StatusUnprocessableEntity).JSON(WebResponse{
		Success: false, Message: "validasi gagal", Errors: errs,
	})
}

//logika query dan string
var allowedSort = map[string]bool{
	"id": true, "nim": true, "name": true, "": true,
} //whitelist kolom yang boleh urutkan

func parseListQuery(c *fiber.Ctx) ListQuery { //membaca param dari Url
	q := ListQuery{
		Page: c.QueryInt("page", 1),
		Limit: c.QueryInt("limit", 10),
		Search: strings.TrimSpace(c.Query("search", "")),
		Sort: strings.TrimSpace(c.Query("sort", "id")),
		Order: strings.ToLower(strings.TrimSpace(c.Query("order", "asc"))),
	}

	//mencegah input negatif
	if q.Page < 1 {
		q.Page = 1
	}
	//mencegah limit terlalu kecil
	if q.Limit < 1 {
		q.Limit = 10
	}
	//batas atas limit
	if q.Limit > 100 {
		q.Limit = 100
	}
	//validasi kolom sort
	if !allowedSort[q.Sort] {
		q.Sort = "id"
	}
	if q.Order != "desc" {
		q.Order = "asc"
	}

	//membaca query is_active
	if raw := c.Query("is_active", ""); raw != "" {
		if b, err := strconv.ParseBool(raw); err == nil {
			q.IsActive = &b
		}
	}

	return q
}