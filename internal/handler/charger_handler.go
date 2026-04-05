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

type ChargerHandler struct {
	logger         *slog.Logger
	chargerService service.ChargerService
}

func NewChargerHandler(logger *slog.Logger, chargerService service.ChargerService) *ChargerHandler {
	return &ChargerHandler{logger: logger, chargerService: chargerService}
}

func (h *ChargerHandler) Create(c *gin.Context) {
	var req dto.CreateChargerRequest
	requestID := c.GetString(middleware.RequestIDKey)
	h.logger.Info("received charger create request", "request_id", requestID)
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("failed to bind charger create request", "request_id", requestID, "error", err)
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:     err.Error(),
			RequestID: requestID,
		})
		return
	}

	charger, err := h.chargerService.Create(mapper.ToBOCreateChargerInput(req))
	if err != nil {
		h.logger.Error("failed to create charger", "request_id", requestID, "error", err)
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:     err.Error(),
			RequestID: requestID,
		})
		return
	}

	response := mapper.ToDTOChargerResponse(*charger)
	h.logger.Info("charger created successfully", "request_id", requestID, "charger", response)
	c.JSON(http.StatusCreated, dto.BaseResponse{
		StatusCode: http.StatusCreated,
		RequestID:  requestID,
		Message:    "charger created successfully",
		Data:       response,
	})
}

func (h *ChargerHandler) List(c *gin.Context) {
	requestID := c.GetString(middleware.RequestIDKey)
	h.logger.Info("received charger list request", "request_id", requestID)
	chargers, err := h.chargerService.List()
	if err != nil {
		h.logger.Error("failed to fetch chargers", "request_id", requestID, "error", err)
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:     "failed to fetch chargers",
			RequestID: requestID,
		})
		return
	}

	responses := make([]dto.ChargerResponse, 0, len(chargers))
	for _, charger := range chargers {
		responses = append(responses, mapper.ToDTOChargerResponse(charger))
	}

	h.logger.Info("chargers fetched successfully", "request_id", requestID, "chargers", responses)
	c.JSON(http.StatusOK, dto.BaseResponse{
		StatusCode: http.StatusOK,
		RequestID:  requestID,
		Message:    "chargers fetched successfully",
		Data:       responses,
	})
}

func (h *ChargerHandler) Get(c *gin.Context) {
	requestID := c.GetString(middleware.RequestIDKey)
	chargerID := c.Param("charger_id")
	h.logger.Info("received charger get request", "request_id", requestID, "charger_id", chargerID)
	charger, err := h.chargerService.GetByID(chargerID)
	if err != nil {
		h.logger.Error("failed to get charger", "request_id", requestID, "charger_id", chargerID, "error", err)
		status := http.StatusBadRequest
		message := err.Error()
		if service.IsNotFoundError(err) {
			status = http.StatusNotFound
			message = "charger not found"
		}

		c.JSON(status, gin.H{
			"error":      message,
			"request_id": requestID,
		})
		return
	}

	response := mapper.ToDTOChargerResponse(*charger)
	h.logger.Info("charger fetched successfully", "request_id", requestID, "charger", response)
	c.JSON(http.StatusOK, dto.BaseResponse{
		StatusCode: http.StatusOK,
		RequestID:  requestID,
		Message:    "charger fetched successfully",
		Data:       response,
	})
}
