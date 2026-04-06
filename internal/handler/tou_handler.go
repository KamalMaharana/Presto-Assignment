package handler

import (
	"net/http"

	"gin-app/internal/dto"
	"gin-app/internal/mapper"
	"gin-app/internal/middleware"
	"gin-app/internal/service"
	"log/slog"

	"github.com/gin-gonic/gin"
)

type TOUHandler struct {
	logger     *slog.Logger
	touService service.TOUService
}

func NewTOUHandler(logger *slog.Logger, touService service.TOUService) *TOUHandler {
	return &TOUHandler{logger: logger, touService: touService}
}

func (h *TOUHandler) UpsertSchedule(c *gin.Context) {
	requestID := c.GetString(middleware.RequestIDKey)
	chargerID := c.Param("charger_id")
	h.logger.Info("received tou schedule upsert request", "request_id", requestID, "charger_id", chargerID)

	var req dto.UpsertTOUScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("invalid tou schedule upsert payload", "request_id", requestID, "charger_id", chargerID, "error", err)
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:     err.Error(),
			RequestID: requestID,
		})
		return
	}

	if err := h.touService.ReplaceSchedule(chargerID, mapper.ToBOUpsertTOUScheduleInput(req)); err != nil {
		h.logger.Error("failed to upsert tou schedule", "request_id", requestID, "charger_id", chargerID, "error", err)
		status := http.StatusBadRequest
		errorResponse := dto.ErrorResponse{
			Error:     err.Error(),
			RequestID: requestID,
		}
		if service.IsNotFoundError(err) {
			status = http.StatusNotFound
		}
		if service.IsOverlappingScheduleError(err) {
			status = http.StatusConflict
			overlapErr := service.AsOverlappingScheduleError(err)
			errorResponse.Details = map[string]interface{}{
				"charger_id": chargerID,
			}
			if overlapErr != nil {
				errorResponse.Details = map[string]interface{}{
					"charger_id": overlapErr.ChargerID,
					"proposed":   overlapErr.Proposed,
					"existing":   overlapErr.Existing,
				}
			}
		}

		c.JSON(status, errorResponse)
		return
	}
	h.logger.Info("upserted tou schedule successfully",
		"request_id", requestID,
		"charger_id", chargerID,
		"effective_from", req.EffectiveFrom,
		"period_count", len(req.Periods),
	)

	c.JSON(http.StatusOK, dto.BaseResponse{
		StatusCode: http.StatusOK,
		RequestID:  requestID,
		Message:    "tou schedule updated",
		Data:       nil,
	})
}

func (h *TOUHandler) GetScheduleByDate(c *gin.Context) {
	requestID := c.GetString(middleware.RequestIDKey)
	chargerID := c.Param("charger_id")
	date := c.Query("date")
	h.logger.Info("received tou schedule fetch request", "request_id", requestID, "charger_id", chargerID, "date", date)
	if date == "" {
		h.logger.Warn("missing date in tou schedule fetch request", "request_id", requestID, "charger_id", chargerID)
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:     "date query parameter is required (YYYY-MM-DD)",
			RequestID: requestID,
		})
		return
	}

	resp, err := h.touService.GetScheduleByDate(chargerID, date)
	if err != nil {
		h.logger.Error("failed to fetch tou schedule", "request_id", requestID, "charger_id", chargerID, "date", date, "error", err)
		status := http.StatusBadRequest
		if service.IsNotFoundError(err) {
			status = http.StatusNotFound
		}

		c.JSON(status, dto.ErrorResponse{
			Error:     err.Error(),
			RequestID: requestID,
		})
		return
	}
	h.logger.Info("fetched tou schedule successfully", "request_id", requestID, "charger_id", chargerID, "date", date, "period_count", len(resp.Periods))

	c.JSON(http.StatusOK, dto.BaseResponse{
		StatusCode: http.StatusOK,
		RequestID:  requestID,
		Message:    "tou schedule fetched successfully",
		Data:       mapper.ToDTOTOUScheduleResponse(*resp),
	})
}

func (h *TOUHandler) GetRateAtTime(c *gin.Context) {
	requestID := c.GetString(middleware.RequestIDKey)
	chargerID := c.Param("charger_id")
	date := c.Query("date")
	atTime := c.Query("time")
	h.logger.Info("received tou rate fetch request", "request_id", requestID, "charger_id", chargerID, "date", date, "time", atTime)
	if date == "" || atTime == "" {
		h.logger.Warn("missing date/time in tou rate fetch request", "request_id", requestID, "charger_id", chargerID, "date", date, "time", atTime)
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:     "date and time query parameters are required",
			RequestID: requestID,
		})
		return
	}

	resp, err := h.touService.GetRateAt(chargerID, date, atTime)
	if err != nil {
		h.logger.Error("failed to fetch tou rate at time", "request_id", requestID, "charger_id", chargerID, "date", date, "time", atTime, "error", err)
		status := http.StatusBadRequest
		if service.IsNotFoundError(err) {
			status = http.StatusNotFound
		}

		c.JSON(status, dto.ErrorResponse{
			Error:     err.Error(),
			RequestID: requestID,
		})
		return
	}
	h.logger.Info("fetched tou rate at time successfully",
		"request_id", requestID,
		"charger_id", chargerID,
		"date", date,
		"time", atTime,
		"default_applied", resp.DefaultApplied,
	)

	c.JSON(http.StatusOK, dto.BaseResponse{
		StatusCode: http.StatusOK,
		RequestID:  requestID,
		Message:    "tou rate at time fetched successfully",
		Data:       mapper.ToDTOTOURateAtTimeResponse(*resp),
	})
}
