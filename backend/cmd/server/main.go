package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/uaad/backend/internal/config"
	"github.com/uaad/backend/internal/domain"
	"github.com/uaad/backend/internal/handler"
	"github.com/uaad/backend/internal/infra"
	"github.com/uaad/backend/internal/middleware"
	"github.com/uaad/backend/internal/repository"
	"github.com/uaad/backend/internal/service"
	"github.com/uaad/backend/internal/worker"
	"golang.org/x/time/rate"
	gormmysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	_ = godotenv.Load(".env")
	_ = godotenv.Load("../.env")

	cfg := config.Load()
	appCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := gorm.Open(gormmysql.Open(cfg.MySQLDSN()), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("failed to get sql.DB: %v", err)
	}
	cfg.ApplyMySQLPool(sqlDB)

	if err := db.AutoMigrate(
		&domain.User{},
		&domain.Activity{},
		&domain.Enrollment{},
		&domain.Order{},
		&domain.Notification{},
		&domain.UserBehavior{},
		&domain.ActivityScore{},
	); err != nil {
		log.Fatalf("failed to migrate database: %v", err)
	}

	// ── Redis ──────────────────────────────────────────────────────
	rdb := infra.NewRedisClient(cfg)
	stockEngine := service.NewStockEngine(rdb)

	// ── Kafka ─────────────────────────────────────────────────────
	kafkaWriter := infra.NewKafkaWriter(cfg)
	kafkaReader := infra.NewKafkaReader(cfg)

	// ── Dependency Injection ────────────────────────────────────────
	userRepo := repository.NewUserRepository(db)
	authSvc := service.NewAuthService(userRepo, cfg.JWTSecret)
	authHandler := handler.NewAuthHandler(authSvc)

	activityRepo := repository.NewActivityRepository(db)
	enrollmentRepo := repository.NewEnrollmentRepository(db)
	orderRepo := repository.NewOrderRepository(db)
	notifRepo := repository.NewNotificationRepository(db)
	behaviorRepo := repository.NewBehaviorRepository(db)
	recommendRepo := repository.NewRecommendationRepository(db)

	activitySvc := service.NewActivityService(activityRepo, stockEngine)
	notifSvc := service.NewNotificationService(notifRepo)
	enrollmentSvc := service.NewEnrollmentService(db, stockEngine, kafkaWriter, enrollmentRepo, activityRepo, orderRepo)
	orderSvc := service.NewOrderService(orderRepo, activityRepo, stockEngine, notifSvc)
	behaviorSvc := service.NewBehaviorService(behaviorRepo, activityRepo)
	recommendSvc := service.NewRecommendationService(recommendRepo, cfg.Scoring, 5*time.Minute)
	stockReconciler := service.NewStockReconciler(activityRepo, enrollmentRepo, stockEngine, 200)
	activityOfflineJob := service.NewActivityOfflineJob(db)

	activityHandler := handler.NewActivityHandler(activitySvc)
	enrollmentHandler := handler.NewEnrollmentHandler(enrollmentSvc)
	orderHandler := handler.NewOrderHandler(orderSvc)
	notifHandler := handler.NewNotificationHandler(notifSvc)
	behaviorHandler := handler.NewBehaviorHandler(behaviorSvc, cfg)
	recommendHandler := handler.NewRecommendationHandler(recommendSvc)

	// 注册限流：默认约 5 次/分钟、突发 5（防刷）。本地批量 gen_jmeter_data 可在 .env 临时提高，如 REG_RATE_LIMIT_PER_MIN=120
	regPerMin := 5.0
	if s := os.Getenv("REG_RATE_LIMIT_PER_MIN"); s != "" {
		if v, err := strconv.ParseFloat(s, 64); err == nil && v > 0 {
			regPerMin = v
		}
	}
	regBurst := 5
	if s := os.Getenv("REG_RATE_LIMIT_BURST"); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v > 0 {
			regBurst = v
		}
	}
	regLimit := middleware.NewIPRateLimiter(rate.Limit(regPerMin/60.0), regBurst)

	// ── Router ──────────────────────────────────────────────────────
	r := gin.Default()

	r.Use(middleware.RequestID())
	r.Use(middleware.PrometheusMiddleware())
	r.Use(cors.New(middleware.BuildCORSConfig(cfg.AppEnv, cfg.CORSAllowedOrigins)))

	// Prometheus metrics endpoint
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	v1 := r.Group("/api/v1")
	{
		v1.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		auth := v1.Group("/auth")
		{
			auth.POST("/register", middleware.RateLimitMiddleware(regLimit), authHandler.Register)
			auth.POST("/login", authHandler.Login)
		}

		protected := v1.Group("", middleware.JWTAuth(cfg.JWTSecret))
		{
			protected.GET("/auth/profile", authHandler.GetCurrentUser)
		}

		handler.RegisterActivityRoutes(v1, activityHandler, cfg.JWTSecret)
		handler.RegisterEnrollmentRoutes(v1, enrollmentHandler, cfg.JWTSecret)
		handler.RegisterOrderRoutes(v1, orderHandler, cfg.JWTSecret)
		handler.RegisterNotificationRoutes(v1, notifHandler, cfg.JWTSecret)
		handler.RegisterBehaviorRoutes(v1, behaviorHandler, cfg.JWTSecret)
		handler.RegisterRecommendationRoutes(v1, recommendHandler, cfg.JWTSecret)
	}

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// ── Enrollment Worker (Kafka consumer → MySQL) ───────────────────
	enrollWorker := worker.NewEnrollmentWorker(kafkaReader, db, stockEngine, notifSvc, activityRepo)
	go enrollWorker.Run(appCtx)

	// ── Order Expiry Scanner (every 5 minutes) ─────────────────────────
	go func() {
		runPeriodic(appCtx, 5*time.Minute, func(context.Context) {
			closed, err := orderSvc.ScanExpired()
			if err != nil {
				log.Printf("[OrderExpiry] scan error: %v", err)
			} else if closed > 0 {
				log.Printf("[OrderExpiry] closed %d expired orders, stock rolled back", closed)
			}
		})
	}()

	// ── Recommendation Score Recalculation ──────────────────────────────
	go func() {
		interval := time.Duration(cfg.ScoreRecalcIntervalMinutes) * time.Minute
		if interval <= 0 {
			interval = 30 * time.Minute
		}
		runPeriodicWithInitial(appCtx, interval, func(ctx context.Context) {
			if err := recommendSvc.RecalculateAllScores(ctx); err != nil {
				log.Printf("[RecommendScore] periodic recalc error: %v", err)
			}
		})
	}()

	// ── Stock Reconciliation (Redis stock self-heal) ────────────────────
	go func() {
		interval := time.Duration(cfg.StockReconcileMinutes) * time.Minute
		if interval <= 0 {
			interval = 10 * time.Minute
		}
		runPeriodicWithInitial(appCtx, interval, func(ctx context.Context) {
			result, err := stockReconciler.Reconcile(ctx)
			if err != nil {
				log.Printf("[StockReconcile] error: %v", err)
				return
			}
			if result.Repaired > 0 {
				log.Printf("[StockReconcile] repaired=%d checked=%d", result.Repaired, result.Checked)
			}
		})
	}()

	// ── Activity Auto-OFFLINE (SPRINT3 §三 task 8) ──────────────────────
	go func() {
		interval := time.Duration(cfg.ActivityOfflineMinutes) * time.Minute
		if interval <= 0 {
			interval = 15 * time.Minute
		}
		runPeriodicWithInitial(appCtx, interval, func(ctx context.Context) {
			result, err := activityOfflineJob.Run(ctx)
			if err != nil {
				log.Printf("[ActivityOffline] scan error: %v", err)
				return
			}
			if result.OfflineCount > 0 {
				log.Printf("[ActivityOffline] %d activities transitioned to OFFLINE", result.OfflineCount)
			}
		})
	}()

	server := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: r,
	}

	go func() {
		<-appCtx.Done()
		log.Println("shutdown signal received, stopping services...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("http server shutdown error: %v", err)
		}
	}()

	log.Printf("Server starting on :%s", cfg.Port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("failed to run server: %v", err)
	}

	if err := kafkaWriter.Close(); err != nil {
		log.Printf("kafka writer close error: %v", err)
	}
	if err := kafkaReader.Close(); err != nil {
		log.Printf("kafka reader close error: %v", err)
	}
	if err := rdb.Close(); err != nil {
		log.Printf("redis close error: %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		log.Printf("mysql close error: %v", err)
	}
}
