package middleware

import (
    "errors"
    "strings"
    "time"

    "github.com/gofiber/fiber/v2"
    "github.com/gofiber/fiber/v2/middleware/limiter"

    "latihan-fiber/helper"
)

func RequireAuth(jwtManager *helper.JWTManager) fiber.Handler {
    return func(c *fiber.Ctx) error {
        token, err := bearerToken(c)
        if err != nil {
            c.Set("WWW-Authenticate", `Bearer realm="api"`)
            return helper.Fail(c, fiber.StatusUnauthorized,
                "header Authorization tidak ada atau salah bentuk")
        }

        authUser, err := jwtManager.Parse(token)
        if err != nil {
            c.Set("WWW-Authenticate", `Bearer realm="api"`)

            if errors.Is(err, helper.ErrExpiredToken) {
                return helper.Fail(c, fiber.StatusUnauthorized, "access token kedaluwarsa")
            }
            return helper.Fail(c, fiber.StatusUnauthorized, "access token tidak valid")
        }

        c.Locals(helper.LocalsAuthUser, authUser)
        return c.Next()
    }
}

func bearerToken(c *fiber.Ctx) (string, error) {
    header := c.Get(fiber.HeaderAuthorization)
    if header == "" {
        return "", errors.New("header kosong")
    }

    parts := strings.SplitN(header, " ", 2)
    if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
        return "", errors.New("format bukan Bearer")
    }

    token := strings.TrimSpace(parts[1])
    if token == "" {
        return "", errors.New("token kosong")
    }

    return token, nil
}

// membatasi percobaan login untuk mencegah serangan Brute Force
func LoginRateLimiter() fiber.Handler {
    return limiter.New(limiter.Config{
        Max:        5,
        Expiration: 1 * time.Minute,
        KeyGenerator: func(c *fiber.Ctx) string {
            return c.IP()
        },
        LimitReached: func(c *fiber.Ctx) error {
            c.Set("Retry-After", "60")
            return helper.Fail(c, fiber.StatusTooManyRequests,
                "terlalu banyak percobaan login, coba lagi dalam satu menit")
        },
    })
}
// menolak request yang role-nya tidak memiliki permission tertentu
func RequirePermission(perms *helper.PermissionSet, permission string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Ambil data user dari JWT yang sudah divalidasi oleh RequireAuth
		user, ok := helper.CurrentUser(c)
		if !ok {
			// klo tanpa identitas, tolak
			return helper.Fail(c, fiber.StatusUnauthorized, "belum terautentikasi")
		}

		// priksa apakah role user memiliki permission yang diminta
		if !perms.Can(user.Role, permission) {
			return helper.Fail(c, fiber.StatusForbidden,
				"role "+user.Role+" tidak memiliki hak "+permission)
		}

		return c.Next()
	}
}