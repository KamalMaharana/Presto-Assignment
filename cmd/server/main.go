package main

import (
	"context"
	"log"
	"log/slog"

	"gin-app/internal/config"
	"gin-app/internal/database"
	"gin-app/internal/logger"
	"gin-app/internal/repository"
	"gin-app/internal/router"
	"gin-app/internal/service"
)

func main() {
	cfg := config.Load()
	appLogger := logger.New(cfg.LogLevel)
	slog.SetDefault(appLogger)

	db, err := database.ConnectPostgres(cfg)
	if err != nil {
		appLogger.Error("failed to connect to database", "error", err)
		log.Fatalf("failed to connect to database: %v", err)
	}

	chargerRepository := repository.NewChargerRepository(db)
	touRepository := repository.NewTOURepository(db)
	touBulkJobRepository := repository.NewTOUBulkJobRepository(db)
	chargerService := service.NewChargerService(chargerRepository)
	touService := service.NewTOUService(chargerRepository, touRepository)
	touBulkService := service.NewTOUBulkService(appLogger, touBulkJobRepository, touService)

	workerCtx, cancelWorker := context.WithCancel(context.Background())
	defer cancelWorker()
	go touBulkService.StartWorker(workerCtx, 0)

	engine := router.SetupRouter(chargerService, touService, touBulkService, appLogger)

	appLogger.Info("server starting", "address", cfg.ServerAddress())
	if err := engine.Run(cfg.ServerAddress()); err != nil {
		appLogger.Error("failed to start server", "error", err)
		log.Fatalf("failed to start server: %v", err)
	}
}
