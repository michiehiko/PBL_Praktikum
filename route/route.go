package route

import (
    "context"
    "time"

    "github.com/gofiber/fiber/v2"
    "github.com/jackc/pgx/v5/pgxpool"

    "latihan-fiber/app/service"
    "latihan-fiber/helper"
    "latihan-fiber/middleware"
)

// mengumpulkan seluruh service yang dibutuhkan route.
type Dependencies struct {
    Pool           *pgxpool.Pool
    JWT            *helper.JWTManager
    Permissions    *helper.PermissionSet
    StudentService *service.StudentService
    AuthService    *service.AuthService
}

func Register(app *fiber.App, deps Dependencies) {
    api := app.Group("/api/v1")
    
    api.Get("/health", healthCheck(deps.Pool))

    // auth
    auth := api.Group("/auth", middleware.RequireJSON)
    auth.Post("/register", deps.AuthService.Register)
    auth.Post("/login", middleware.LoginRateLimiter(), deps.AuthService.Login) //dipasang ratelimiter biar ga di brute force
    auth.Post("/refresh", deps.AuthService.Refresh)
    auth.Post("/logout", deps.AuthService.Logout)
    auth.Get("/me", middleware.RequireAuth(deps.JWT), deps.AuthService.Me)

    // students (bawa jwt)
    students := api.Group("/students", middleware.RequireJSON, middleware.RequireAuth(deps.JWT))
    perms := deps.Permissions

    // Hak akses berdasarkan permission
	students.Get("/", middleware.RequirePermission(perms, "student:list"), deps.StudentService.List)
	students.Post("/", middleware.RequirePermission(perms, "student:create"), deps.StudentService.Create)
	students.Delete("/:id", middleware.RequirePermission(perms, "student:delete"), deps.StudentService.Delete)

	// Hak akses berdasarkan owner
	students.Get("/:id", deps.StudentService.Get)
	students.Put("/:id", deps.StudentService.Replace)
	students.Patch("/:id", deps.StudentService.Patch)
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