// Package router wires HTTP routes and middleware for the Gin engine.
package router

import (
	"log/slog"

	"gin-app/internal/handler"
	"gin-app/internal/middleware"
	"gin-app/internal/service"

	"github.com/gin-gonic/gin"
)

func SetupRouter(
	chargerService service.ChargerService,
	touService service.TOUService,
	touBulkService service.TOUBulkService,
	log *slog.Logger,
) *gin.Engine {
	engine := gin.New()
	engine.Use(
		middleware.RequestID(),
		middleware.RequestLogger(log),
		gin.Recovery(),
	)

	healthHandler := handler.NewHealthHandler()
	chargerHandler := handler.NewChargerHandler(log, chargerService)
	touHandler := handler.NewTOUHandler(log, touService)
	touBulkHandler := handler.NewTOUBulkHandler(log, touBulkService)

	engine.GET("/health", healthHandler.Check)

	v1 := engine.Group("/api/v1")
	{
		v1.POST("/chargers", chargerHandler.Create)
		v1.GET("/chargers", chargerHandler.List)
		v1.GET("/chargers/:charger_id", chargerHandler.Get)

		v1.PUT("/chargers/:charger_id/tou-rates", touHandler.UpsertSchedule)
		v1.GET("/chargers/:charger_id/tou-rates", touHandler.GetScheduleByDate)
		v1.GET("/chargers/:charger_id/tou-rate", touHandler.GetRateAtTime)

		v1.POST("/tou-bulk-jobs", touBulkHandler.CreateJob)
		v1.GET("/tou-bulk-jobs/:job_id", touBulkHandler.GetJob)
		v1.GET("/tou-bulk-jobs/:job_id/rows", touBulkHandler.ListJobRows)
	}

	return engine
}
