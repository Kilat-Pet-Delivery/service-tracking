package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/Kilat-Pet-Delivery/lib-common/auth"
	"github.com/Kilat-Pet-Delivery/lib-common/database"
	"github.com/Kilat-Pet-Delivery/lib-common/health"
	"github.com/Kilat-Pet-Delivery/lib-common/kafka"
	"github.com/Kilat-Pet-Delivery/lib-common/logger"
	"github.com/Kilat-Pet-Delivery/lib-common/middleware"
	"github.com/Kilat-Pet-Delivery/service-tracking/internal/application"
	"github.com/Kilat-Pet-Delivery/service-tracking/internal/config"
	"github.com/Kilat-Pet-Delivery/service-tracking/internal/events"
	"github.com/Kilat-Pet-Delivery/service-tracking/internal/handler"
	"github.com/Kilat-Pet-Delivery/service-tracking/internal/repository"
	"github.com/Kilat-Pet-Delivery/service-tracking/internal/ws"
)

func main() {
	// Load configuration.
	cfg, err := config.Load()
	if err != nil {
		panic("failed to load configuration: " + err.Error())
	}

	// Initialize logger.
	log, err := logger.NewNamed(cfg.AppEnv, "service-tracking")
	if err != nil {
		panic("failed to initialize logger: " + err.Error())
	}
	defer func() { _ = log.Sync() }()

	// Connect to database.
	dbConfig := database.PostgresConfig{
		Host:     cfg.DBConfig.Host,
		Port:     cfg.DBConfig.Port,
		User:     cfg.DBConfig.User,
		Password: cfg.DBConfig.Password,
		DBName:   cfg.DBConfig.DBName,
		SSLMode:  cfg.DBConfig.SSLMode,
	}
	db, err := database.Connect(dbConfig, log)
	if err != nil {
		log.Fatal("failed to connect to database", zap.Error(err))
	}

	// Run database migrations.
	if cfg.AppEnv == "development" {
		if err := db.AutoMigrate(&repository.TripTrackModel{}, &repository.WaypointModel{}, &repository.ChatMessageModel{}, &repository.SharedTripModel{}); err != nil {
			log.Fatal("failed to auto-migrate database", zap.Error(err))
		}
		log.Info("database migration completed (dev auto-migrate)")
	} else {
		dbURL := dbConfig.DatabaseURL()
		if err := database.RunMigrations(dbURL, "migrations", log); err != nil {
			log.Fatal("failed to run migrations", zap.Error(err))
		}
	}

	// Initialize JWT manager.
	accessExpiry, err := time.ParseDuration(cfg.JWTConfig.AccessExpiry)
	if err != nil {
		accessExpiry = 15 * time.Minute
	}
	refreshExpiry, err := time.ParseDuration(cfg.JWTConfig.RefreshExpiry)
	if err != nil {
		refreshExpiry = 7 * 24 * time.Hour
	}
	jwtManager := auth.NewJWTManager(cfg.JWTConfig.Secret, accessExpiry, refreshExpiry)

	// Initialize Kafka producer.
	producer := kafka.NewProducer(cfg.KafkaConfig.Brokers, log)
	defer func() { _ = producer.Close() }()

	// Initialize WebSocket hub.
	wsHub := ws.NewHub(log)
	go wsHub.Run()

	// Initialize repository.
	trackingRepo := repository.NewGORMTripTrackRepository(db, log)

	// Initialize application service.
	trackingService := application.NewTrackingService(trackingRepo, wsHub, producer, log)

	// Initialize Kafka consumers.
	groupPrefix := cfg.KafkaConfig.GroupPrefix
	if groupPrefix == "" {
		groupPrefix = "tracking"
	}

	bookingConsumer := events.NewBookingEventConsumer(
		cfg.KafkaConfig.Brokers,
		groupPrefix+"-booking-consumer",
		trackingService,
		log,
	)
	defer func() { _ = bookingConsumer.Close() }()

	runnerConsumer := events.NewRunnerEventConsumer(
		cfg.KafkaConfig.Brokers,
		groupPrefix+"-runner-consumer",
		trackingService,
		log,
	)
	defer func() { _ = runnerConsumer.Close() }()

	// Start consumers in background goroutines.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := bookingConsumer.Start(ctx); err != nil && ctx.Err() == nil {
			log.Error("booking event consumer error", zap.Error(err))
		}
	}()

	go func() {
		if err := runnerConsumer.Start(ctx); err != nil && ctx.Err() == nil {
			log.Error("runner event consumer error", zap.Error(err))
		}
	}()

	// Initialize Gin router.
	router := gin.New()
	router.Use(
		middleware.RequestIDMiddleware(),
		middleware.LoggerMiddleware(log),
		middleware.RecoveryMiddleware(log),
		middleware.CORSMiddleware(),
		middleware.SecurityHeadersMiddleware(),
	)

	// Register health check routes.
	healthHandler := health.NewHandler(db, "service-tracking")
	healthHandler.RegisterRoutes(router)

	// Initialize chat service and handler.
	chatRepo := repository.NewGormChatRepository(db)
	chatService := application.NewChatService(chatRepo, wsHub, log)
	chatHandler := handler.NewChatHandler(chatService)

	// Initialize share service and handler.
	shareRepo := repository.NewGormSharedTripRepository(db)
	shareService := application.NewShareService(shareRepo, trackingRepo, log)
	shareHandler := handler.NewShareHandler(shareService)

	// Register tracking REST API routes.
	trackingHandler := handler.NewTrackingHandler(trackingService, wsHub, jwtManager, log)
	apiV1 := router.Group("/api/v1")
	trackingHandler.RegisterRoutes(apiV1, jwtManager)
	chatHandler.RegisterRoutes(apiV1, jwtManager)
	shareHandler.RegisterRoutes(apiV1, jwtManager)

	// Register WebSocket route.
	trackingHandler.RegisterWSRoute(router, jwtManager)

	// Start HTTP server.
	srv := &http.Server{
		Addr:         cfg.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Info("starting service-tracking", zap.String("port", cfg.Port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("server error", zap.Error(err))
		}
	}()

	// Graceful shutdown.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("shutting down service-tracking...")

	// Cancel context to stop consumers.
	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("server forced to shutdown", zap.Error(err))
	}

	log.Info("service-tracking stopped")
}
