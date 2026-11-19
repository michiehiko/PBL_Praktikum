package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"latihan-fiber/app/repository"
	"latihan-fiber/app/service"
	"latihan-fiber/config"
	"latihan-fiber/database"
	_ "latihan-fiber/docs" // hasil swag init; mendaftarkan spesifikasi ke swagger
	"latihan-fiber/helper"
	"latihan-fiber/route"
)

const minSecretLength = 32

// Blok komentar di bawah ini adalah informasi UMUM dokumentasi API. swag
// hanya membacanya dari file yang disebut pada flag -g (bawaannya main.go).
//
// @title                       API KRS Praktikum Backend Lanjut
// @version                     1.0
// @description                 API pendaftaran kelas (KRS), ulasan kelas, dan notifikasi.
// @description                 Seluruh endpoint selain /auth memerlukan access token.
// @BasePath                    /api/v1
// @securityDefinitions.apikey  BearerAuth
// @in                          header
// @name                        Authorization
// @description                 Isi dengan: Bearer <access_token>
func main() {
	_ = godotenv.Load()
	logger := config.NewLogger()

	pool, err := database.NewPool(context.Background())
	if err != nil {
		logger.Error("gagal terhubung ke database", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer pool.Close()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	app := startAPI(logger, pool)
	if app == nil {
		os.Exit(1)
	}

	logger.Info("aplikasi berjalan", slog.String("role", "api"))
	<-ctx.Done()

	if err := app.ShutdownWithTimeout(10 * time.Second); err != nil {
		logger.Error("shutdown HTTP tidak rapi", slog.String("error", err.Error()))
	}
	logger.Info("aplikasi berhenti dengan rapi")
}

func startAPI(logger *slog.Logger, pool *pgxpool.Pool) *fiber.App {
	jwtSecret := os.Getenv("JWT_SECRET")
	if len(jwtSecret) < minSecretLength {
		logger.Error("JWT_SECRET tidak diisi atau terlalu pendek",
			slog.Int("minimal_karakter", minSecretLength))
		return nil
	}

	mongoClient, err := database.NewMongoClient(context.Background())
	if err != nil {
		logger.Error("gagal terhubung ke MongoDB", slog.String("error", err.Error()))
		return nil
	}
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
	outboxRepository := repository.NewOutboxRepository(pool)
	notificationRepository := repository.NewNotificationRepository(pool)
	courseRepository := repository.NewCourseRepository(pool)
	txManager := repository.NewTxManager(pool)
	supervisorRepo := repository.NewSupervisorRepository(pool)

	reviewRepository := repository.NewReviewRepository(mongoDB)
	if err := reviewRepository.EnsureIndexes(context.Background()); err != nil {
		logger.Error("gagal membuat index MongoDB", slog.String("error", err.Error()))
		return nil
	}

	portfolioRepository := repository.NewPortfolioRepository(mongoDB)
	if err := portfolioRepository.EnsureIndexes(context.Background()); err != nil {
		logger.Error("gagal membuat index Portfolio", slog.String("error", err.Error()))
		return nil
	}

	rawPermissions, err := roleRepository.LoadPermissions(context.Background())
	if err != nil {
		logger.Error("gagal memuat permission", slog.String("error", err.Error()))
		return nil
	}
	permissions := helper.NewPermissionSet(rawPermissions)

	authService := service.NewAuthService(
		userRepository, tokenRepository, jwtManager,
		time.Duration(refreshTTLDays)*24*time.Hour,
	)
	courseService := service.NewCourseService(courseRepository, outboxRepository, txManager)
	studentService := service.NewStudentService(studentRepository, permissions)
	supervisorService := service.NewSupervisorService(supervisorRepo, txManager)
	reviewService := service.NewReviewService(reviewRepository, courseRepository, permissions)
	portfolioService := service.NewPortfolioService(portfolioRepository, studentRepository)

	inspector := asynq.NewInspector(config.RedisOpt())
	notificationService := service.NewNotificationService(notificationRepository, outboxRepository, inspector)

	deps := route.Dependencies{
		Pool:                pool,
		Mongo:               mongoClient,
		JWT:                 jwtManager,
		Permissions:         permissions,
		StudentService:      studentService,
		AuthService:         authService,
		CourseService:       courseService,
		SupervisorService:   supervisorService,
		ReviewService:       reviewService,
		PortfolioService:    portfolioService,
		NotificationService: notificationService,
		EnableSwagger:       config.GetEnv("SWAGGER_ENABLED", "false") == "true",
	}

	app := config.NewApp(logger, deps, allowedOrigins)

	app.Hooks().OnShutdown(func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = inspector.Close()
		return mongoClient.Disconnect(ctx)
	})

	port := config.GetEnv("APP_PORT", "3000")
	go func() {
		if err := app.Listen(":" + port); err != nil {
			logger.Error("server berhenti", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()
	logger.Info("HTTP berjalan", slog.String("port", port))
	return app
}
