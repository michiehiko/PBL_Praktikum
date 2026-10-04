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

	"uts-pbe/app/repository"
	"uts-pbe/app/service"
	"uts-pbe/config"
	"uts-pbe/database"
	"uts-pbe/helper"
	"uts-pbe/route"
)

const minSecretLength = 32

func main() {
	_ = godotenv.Load()
	logger := config.NewLogger()

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

	jwtIssuer := os.Getenv("JWT_ISSUER")
	if jwtIssuer == "" {
		jwtIssuer = "siakad-api"
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

	jwtManager := helper.NewJWTManager(
		jwtSecret,
		jwtIssuer,
		time.Duration(accessTTLMin)*time.Minute,
	)

	// Inisialisasi Repository
	roleRepository := repository.NewRoleRepository(pool)
	userRepository := repository.NewUserRepository(pool)
	tokenRepository := repository.NewTokenRepository(pool)
	studentRepository := repository.NewStudentRepository(pool)
	courseRepository := repository.NewCourseRepository(pool)
	enrollmentRepository := repository.NewEnrollmentRepository(pool)

	rawPermissions, err := roleRepository.LoadPermissions(context.Background())
	if err != nil {
		logger.Error("gagal memuat permission", slog.String("error", err.Error()))
		os.Exit(1)
	}
	permissions := helper.NewPermissionSet(rawPermissions)
	logger.Info("permission dimuat", slog.Any("roles", permissions.KnownRoles()))

	// Inisialisasi Service
	authService := service.NewAuthService(
		userRepository, tokenRepository, studentRepository, jwtManager,
		time.Duration(refreshTTLDays)*24*time.Hour,
	)
	studentService := service.NewStudentService(studentRepository, enrollmentRepository, permissions)
	courseService := service.NewCourseService(courseRepository)
	enrollmentService := service.NewEnrollmentService(enrollmentRepository, studentRepository)

	// Inisialisasi Routing
	deps := route.Dependencies{
		Pool:              pool,
		JWT:               jwtManager,
		Permissions:       permissions,
		AuthService:       authService,
		StudentService:    studentService,
		CourseService:     courseService,
		EnrollmentService: enrollmentService,
	}

	app := config.NewApp(logger, deps, allowedOrigins)

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
