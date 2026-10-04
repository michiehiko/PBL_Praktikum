package route

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"

	"uts-pbe/app/service"
	"uts-pbe/helper"
	"uts-pbe/middleware"
)

type Dependencies struct {
	Pool              *pgxpool.Pool
	JWT               *helper.JWTManager
	Permissions       *helper.PermissionSet
	AuthService       *service.AuthService
	StudentService    *service.StudentService
	CourseService     *service.CourseService
	EnrollmentService *service.EnrollmentService
}

func Register(app *fiber.App, deps Dependencies) {
	api := app.Group("/api/v1")

	// Endpoint Publik
	api.Get("/health", healthCheck(deps.Pool))
	auth := api.Group("/auth", middleware.RequireJSON)
	auth.Post("/login", middleware.LoginRateLimiter(), deps.AuthService.Login) // Endpoint 1

	// Endpoint Terautentikasi
	secured := api.Group("/", middleware.RequireAuth(deps.JWT))

	// Auth Routes (Terautentikasi)
	secured.Get("/auth/me", deps.AuthService.Me) // Endpoint 2

	// Students Routes
	students := secured.Group("/students", middleware.RequireJSON)
	perms := deps.Permissions

	// Admin Only
	students.Get("/", middleware.RequirePermission(perms, "student:list"), deps.StudentService.List)           // Endpoint 3
	students.Post("/", middleware.RequirePermission(perms, "student:create"), deps.StudentService.Create)      // Endpoint 4
	students.Put("/:id", middleware.RequirePermission(perms, "student:update"), deps.StudentService.Replace)   // Endpoint 6
	students.Delete("/:id", middleware.RequirePermission(perms, "student:delete"), deps.StudentService.Delete) // Endpoint 7

	// Admin & Pemilik Data
	secured.Get("/students/:id", deps.StudentService.Get) // Endpoint 5 (RequireJSON dicabut agar bisa GET)

	// Courses Routes
	secured.Get("/courses", deps.CourseService.List) // Endpoint 8 (Semua role)

	// Enrollments Routes
	enrollments := secured.Group("/enrollments", middleware.RequireJSON)
	enrollments.Post("/", deps.EnrollmentService.Create) // Endpoint 9 (Mahasiswa)

	// Admin & Pemilik Data
	secured.Delete("/enrollments/:id", deps.EnrollmentService.Delete) // Endpoint 10
}

func healthCheck(pool *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(c.UserContext(), 2*time.Second)
		defer cancel()

		if err := pool.Ping(ctx); err != nil {
			return helper.Fail(c, fiber.StatusServiceUnavailable, "database tidak dapat dihubungi")
		}
		return helper.Success(c, fiber.StatusOK, "server dan database berjalan", nil)
	}
}
