package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"latihan-fiber/app/repository"
	"latihan-fiber/app/service"
	"latihan-fiber/config"
	"latihan-fiber/database"
	"latihan-fiber/helper"
	"latihan-fiber/route"
)

const minSecretLength = 32

func main() {
	_ = godotenv.Load()
	logger := config.NewLogger()

	// memeriksa jwt secret sblm server nyala
	jwtSecret := os.Getenv("JWT_SECRET")
	if len(jwtSecret) < minSecretLength {
		logger.Error("JWT_SECRET tidak diisi atau terlalu pendek", slog.Int("minimal_karakter", minSecretLength))
		os.Exit(1)
	}

	pool, err := database.NewPool(context.Background())
	if err != nil {
		logger.Error("gagal terhubung ke database", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer pool.Close()

	// mengambil var env buat jwt
	jwtIssuer := os.Getenv("JWT_ISSUER")
	if jwtIssuer == "" {
		jwtIssuer = "praktikum-backend"
	}

	accessTTLMin, _ := strconv.Atoi(os.Getenv("JWT_ACCESS_TTL_MINUTES"))
	if accessTTLMin == 0 {
		accessTTLMin = 15
	}

	refreshTTLDays, _ := strconv.Atoi(os.Getenv("JWT_REFRESH_TTL_DAYS"))
	if refreshTTLDays == 0 {
		refreshTTLDays = 7
	}

	allowedOrigins := os.Getenv("ALLOWED_ORIGINS")

	// jwt manager
	jwtManager := helper.NewJWTManager(
		jwtSecret,
		jwtIssuer,
		time.Duration(accessTTLMin)*time.Minute,
	)

	// repository
	userRepository := repository.NewUserRepository(pool)
	tokenRepository := repository.NewTokenRepository(pool)
	studentRepository := repository.NewStudentRepository(pool)
	
	// Inisialisasi role repository
	roleRepository := repository.NewRoleRepository(pool)

	// Tpemetaan role ke permission sekali saat aplikasi menyala
	rawPermissions, err := roleRepository.LoadPermissions(context.Background())
	if err != nil {
		logger.Error("gagal memuat permission", slog.String("error", err.Error()))
		os.Exit(1)
	}
	permissions := helper.NewPermissionSet(rawPermissions)
	logger.Info("permission dimuat", slog.Any("roles", permissions.KnownRoles()))

	// service
	// Catatan: Jika kode Anda error di baris ini karena argumen kurang, berarti Anda sudah mengikuti Langkah 4 Modul 6 sepenuhnya.
	// Jika demikian, ubah menjadi: service.NewAuthService(userRepository, tokenRepository, jwtManager, permissions, time.Duration...)
	authService := service.NewAuthService(
		userRepository, tokenRepository, jwtManager,
		time.Duration(refreshTTLDays)*24*time.Hour,
	)
	
	//Inject permissions ke StudentService
	studentService := service.NewStudentService(studentRepository, permissions)

	// dependency injection
	deps := route.Dependencies{
		Pool:           pool,
		JWT:            jwtManager,
		Permissions:    permissions, // TAMBAHAN: Inject ke route
		StudentService: studentService,
		AuthService:    authService,
	}

	// Aplikasi Fiber
	app := config.NewApp(logger, deps, allowedOrigins)
	//port sesuai env/default
	port := "3000"
	if p := os.Getenv("APP_PORT"); p != "" {
		port = p
	}

	go func() {
		if err := app.Listen(":" + port); err != nil {
			logger.Error("server berhenti", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()
	logger.Info("server berjalan", slog.String("port", port))

	// shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("sinyal berhenti diterima, menutup server")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := app.ShutdownWithContext(ctx); err != nil {
		logger.Error("gagal menutup server dengan benar", slog.String("error", err.Error()))
	}
	logger.Info("server berhenti dengan benar")
}