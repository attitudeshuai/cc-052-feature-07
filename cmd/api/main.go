package main

import (
	"cc-052/internal/config"
	"cc-052/internal/handler"
	"cc-052/internal/repository"
	"cc-052/internal/router"
	"cc-052/internal/service"
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-redis/redis/v8"
)

func main() {
	cfg := config.Load()

	// Database
	db, err := repository.NewDB(cfg)
	if err != nil {
		log.Fatalf("failed to connect db: %v", err)
	}
	defer db.Close()

	// Run migrations
	if err := repository.RunMigrations(db, "migrations"); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}

	// Run seed data
	if err := repository.RunSeed(db, "seed"); err != nil {
		log.Printf("warning: seed data error: %v", err)
	}

	// Redis
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr(),
		Password: cfg.RedisPassword,
		DB:       0,
	})
	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("failed to connect redis: %v", err)
	}
	defer rdb.Close()

	// Repositories
	farmRepo := repository.NewFarmRepo(db)
	plotRepo := repository.NewPlotRepo(db)
	batchRepo := repository.NewBatchRepo(db)
	activityRepo := repository.NewActivityRepo(db)
	inspectionRepo := repository.NewInspectionRepo(db)
	codeRepo := repository.NewTraceCodeRepo(db)
	inputMaterialRepo := repository.NewInputMaterialRepo(db)
	_ = inputMaterialRepo

	// Services
	farmSvc := service.NewFarmService(farmRepo)
	plotSvc := service.NewPlotService(plotRepo)
	batchSvc := service.NewBatchService(batchRepo, plotRepo, farmRepo)
	activitySvc := service.NewActivityService(activityRepo, batchRepo)
	inspectionSvc := service.NewInspectionService(inspectionRepo, batchRepo)
	traceCodeSvc := service.NewTraceCodeService(codeRepo, batchRepo, inspectionRepo, activityRepo, plotRepo, farmRepo)

	// Handlers
	farmH := handler.NewFarmHandler(farmSvc)
	plotH := handler.NewPlotHandler(plotSvc)
	batchH := handler.NewBatchHandler(batchSvc)
	activityH := handler.NewActivityHandler(activitySvc)
	inspectionH := handler.NewInspectionHandler(inspectionSvc)
	traceCodeH := handler.NewTraceCodeHandler(traceCodeSvc)
	healthH := handler.NewHealthHandler(db, rdb)

	// Router
	r := router.Setup(farmH, plotH, batchH, activityH, inspectionH, traceCodeH, healthH, rdb)

	srv := &http.Server{
		Addr:    ":" + cfg.ServerPort,
		Handler: r,
	}

	// Graceful shutdown
	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit
		log.Println("shutting down server...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Fatalf("server forced to shutdown: %v", err)
		}
	}()

	log.Printf("server starting on :%s", cfg.ServerPort)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}

func init() {
	// Ensure health command works for Docker HEALTHCHECK
	if len(os.Args) > 1 && os.Args[1] == "health" {
		resp, err := http.Get(fmt.Sprintf("http://localhost:%s/healthz", os.Getenv("SERVER_PORT")))
		if err != nil || resp.StatusCode != 200 {
			os.Exit(1)
		}
		os.Exit(0)
	}
}