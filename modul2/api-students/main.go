package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/requestid"
)

//daftar mode yg perlu body (POST, PUT, PATCH)
var metodeBerbody = map[string]bool{
	fiber.MethodPost:  true,
	fiber.MethodPut:   true,
	fiber.MethodPatch: true,
}

//middleware utuk menolak req klo content type bukan JSON (415)
func requireJSON(c *fiber.Ctx) error {
	if metodeBerbody[c.Method()] {
		ct := c.Get("Content-Type")
		if !strings.HasPrefix(ct, fiber.MIMEApplicationJSON) {
			return fail(c, fiber.StatusUnsupportedMediaType, "Content-Type harus application/json")
		}
	}
	return c.Next()
}

func main() {
	//inisialisasi app fiber dgn eror handler
	app := fiber.New(fiber.Config{
		AppName: "Tugas API Students",
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			status := fiber.StatusInternalServerError
			pesan := "terjadi kesalahan pada server"
			if e, ok := err.(*fiber.Error); ok {
				status = e.Code
				pesan = e.Message
			}
			return fail(c, status, pesan)
		},
	})

	//middleware
	app.Use(requestid.New())
	app.Use(logger.New(logger.Config{
		Format: "[${time}] ${locals:requestid} ${method} ${path} ${status} ${latency}\n",
	}))
	app.Use(cors.New())

	//rute API
	api := app.Group("/api/v1")

	studentsGroup := api.Group("/students", requireJSON)

	//middleware JSON untuk students
	studentsGroup.Get("/", listStudents)
	studentsGroup.Get("/:id", getStudent)
	studentsGroup.Post("/", createStudent)
	studentsGroup.Put("/:id", replaceStudent)
	studentsGroup.Patch("/:id", patchStudent)
	studentsGroup.Delete("/:id", deleteStudent)

	//middleware catch unknown endpoint (404)
	app.Use(func(c *fiber.Ctx) error {
		return fail(c, fiber.StatusNotFound, "endpoint tidak ditemukan")
	})

	//jalankan server
	fmt.Println("Server berjalan di http://localhost:3000")
	log.Fatal(app.Listen(":3000"))
}
