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

	jwtSecret := os.Getenv("JWT_SECRET")
	if len(jwtSecret) < minSecretLength {
		logger.Error("JWT_SECRET tidak diisi atau terlalu pendek", slog.Int("minimal_karakter", minSecretLength))
		os.Exit(1)
	}

	pool, err := database.NewPool(context.Background())
	if err != nil {
		logger.Error("gagal terhubung ke PostgreSQL", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer pool.Close()

	// Inisialisasi MongoDB
	mongoClient, err := database.NewMongoClient(context.Background())
	if err != nil {
		logger.Error("gagal terhubung ke MongoDB", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = mongoClient.Disconnect(ctx)
	}()
	mongoDB := mongoClient.Database(config.GetEnv("MONGO_DATABASE", "praktikum_backend"))

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

	jwtManager := helper.NewJWTManager(jwtSecret, jwtIssuer, time.Duration(accessTTLMin)*time.Minute)

	userRepository := repository.NewUserRepository(pool)
	tokenRepository := repository.NewTokenRepository(pool)
	studentRepository := repository.NewStudentRepository(pool)
	roleRepository := repository.NewRoleRepository(pool)

	rawPermissions, err := roleRepository.LoadPermissions(context.Background())
	if err != nil {
		logger.Error("gagal memuat permission", slog.String("error", err.Error()))
		os.Exit(1)
	}
	permissions := helper.NewPermissionSet(rawPermissions)
	logger.Info("permission dimuat", slog.Any("roles", permissions.KnownRoles()))

	authService := service.NewAuthService(
		userRepository, tokenRepository, jwtManager,
		time.Duration(refreshTTLDays)*24*time.Hour,
	)

	txManager := repository.NewTxManager(pool)
	courseRepository := repository.NewCourseRepository(pool)
	courseService := service.NewCourseService(courseRepository, txManager)
	supervisorRepo := repository.NewSupervisorRepository(pool)
	supervisorService := service.NewSupervisorService(supervisorRepo, txManager)
	studentService := service.NewStudentService(studentRepository, permissions)

	// Inisialisasi Repository MongoDB
	reviewRepository := repository.NewReviewRepository(mongoDB)
	if err := reviewRepository.EnsureIndexes(context.Background()); err != nil {
		logger.Error("gagal membuat index MongoDB", slog.String("error", err.Error()))
		os.Exit(1)
	}

	portfolioRepository := repository.NewPortfolioRepository(mongoDB)
	if err := portfolioRepository.EnsureIndexes(context.Background()); err != nil {
		logger.Error("gagal membuat index Portfolio", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Inisialisasi Service MongoDB
	reviewService := service.NewReviewService(reviewRepository, courseRepository, permissions)
	portfolioService := service.NewPortfolioService(portfolioRepository, studentRepository)

	deps := route.Dependencies{
		Pool:              pool,
		Mongo:             mongoClient,
		JWT:               jwtManager,
		Permissions:       permissions,
		StudentService:    studentService,
		AuthService:       authService,
		CourseService:     courseService,
		SupervisorService: supervisorService,
		ReviewService:     reviewService,
		PortfolioService:  portfolioService,
	}

	app := config.NewApp(logger, deps, allowedOrigins)

	go func() {
		port := "3000"
		if err := app.Listen("127.0.0.1:" + port); err != nil {
			logger.Error("server berhenti", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()
	logger.Info("server berjalan", slog.String("port", "3000"))

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