package handler

import (
	"net/http"
	"strconv"
	"strings"

	"gin-app/internal/dto"
	"gin-app/internal/mapper"
	"gin-app/internal/middleware"
	"gin-app/internal/service"
	"log/slog"

	"github.com/gin-gonic/gin"
)

const idempotencyHeader = "Idempotency-Key"

type TOUBulkHandler struct {
	logger         *slog.Logger
	touBulkService service.TOUBulkService
}

func NewTOUBulkHandler(logger *slog.Logger, touBulkService service.TOUBulkService) *TOUBulkHandler {
	return &TOUBulkHandler{
		logger:         logger,
		touBulkService: touBulkService,
	}
}

func (h *TOUBulkHandler) CreateJob(c *gin.Context) {
	requestID := c.GetString(middleware.RequestIDKey)
	h.logger.Info("received tou bulk job create request", "request_id", requestID)

	fileHeader, err := c.FormFile("file")
	if err != nil {
		h.logger.Warn("missing file in tou bulk job request", "request_id", requestID, "error", err)
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:     "file is required in multipart form field `file`",
			RequestID: requestID,
		})
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		h.logger.Error("failed to open uploaded csv", "request_id", requestID, "error", err)
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:     "failed to read uploaded file",
			RequestID: requestID,
		})
		return
	}
	defer file.Close()

	idempotencyKey := strings.TrimSpace(c.GetHeader(idempotencyHeader))
	submittedBy := strings.TrimSpace(c.GetHeader("X-Submitted-By"))

	job, err := h.touBulkService.CreateJobFromCSV(fileHeader.Filename, file, idempotencyKey, submittedBy, requestID)
	if err != nil {
		h.logger.Error("failed to create tou bulk job", "request_id", requestID, "error", err)
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:     err.Error(),
			RequestID: requestID,
		})
		return
	}
	h.logger.Info("accepted tou bulk job",
		"request_id", requestID,
		"job_id", job.ID,
		"status", job.Status,
		"source_filename", fileHeader.Filename,
	)

	c.JSON(http.StatusAccepted, dto.BaseResponse{
		StatusCode: http.StatusAccepted,
		RequestID:  requestID,
		Message:    "tou bulk job accepted",
		Data: dto.CreateTOUBulkJobResponse{
			JobID:  job.ID,
			Status: job.Status,
		},
	})
}

func (h *TOUBulkHandler) GetJob(c *gin.Context) {
	requestID := c.GetString(middleware.RequestIDKey)
	jobID := c.Param("job_id")
	h.logger.Info("received tou bulk job fetch request", "request_id", requestID, "job_id", jobID)

	job, err := h.touBulkService.GetJob(jobID)
	if err != nil {
		h.logger.Error("failed to fetch tou bulk job", "request_id", requestID, "job_id", jobID, "error", err)
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
	h.logger.Info("fetched tou bulk job", "request_id", requestID, "job_id", jobID, "status", job.Status)

	c.JSON(http.StatusOK, dto.BaseResponse{
		StatusCode: http.StatusOK,
		RequestID:  requestID,
		Message:    "tou bulk job fetched successfully",
		Data:       mapper.ToDTOTOUBulkJobResponse(*job),
	})
}

func (h *TOUBulkHandler) ListJobRows(c *gin.Context) {
	requestID := c.GetString(middleware.RequestIDKey)
	jobID := c.Param("job_id")
	statusFilter := strings.TrimSpace(c.Query("status"))
	h.logger.Info("received tou bulk job rows request", "request_id", requestID, "job_id", jobID, "status_filter", statusFilter)

	limit := 100
	if rawLimit := strings.TrimSpace(c.Query("limit")); rawLimit != "" {
		parsedLimit, err := strconv.Atoi(rawLimit)
		if err != nil {
			h.logger.Warn("invalid limit in tou bulk rows request", "request_id", requestID, "job_id", jobID, "limit", rawLimit)
			c.JSON(http.StatusBadRequest, dto.ErrorResponse{
				Error:     "limit must be a valid integer",
				RequestID: requestID,
			})
			return
		}
		limit = parsedLimit
	}

	offset := 0
	if rawOffset := strings.TrimSpace(c.Query("offset")); rawOffset != "" {
		parsedOffset, err := strconv.Atoi(rawOffset)
		if err != nil {
			h.logger.Warn("invalid offset in tou bulk rows request", "request_id", requestID, "job_id", jobID, "offset", rawOffset)
			c.JSON(http.StatusBadRequest, dto.ErrorResponse{
				Error:     "offset must be a valid integer",
				RequestID: requestID,
			})
			return
		}
		offset = parsedOffset
	}

	rows, err := h.touBulkService.ListRows(jobID, statusFilter, limit, offset)
	if err != nil {
		h.logger.Error("failed to list tou bulk job rows",
			"request_id", requestID,
			"job_id", jobID,
			"status_filter", statusFilter,
			"limit", limit,
			"offset", offset,
			"error", err,
		)
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
	h.logger.Info("listed tou bulk job rows",
		"request_id", requestID,
		"job_id", jobID,
		"status_filter", statusFilter,
		"limit", limit,
		"offset", offset,
		"row_count", len(rows),
	)

	c.JSON(http.StatusOK, dto.BaseResponse{
		StatusCode: http.StatusOK,
		RequestID:  requestID,
		Message:    "tou bulk job rows fetched successfully",
		Data:       mapper.ToDTOTOUBulkJobRowResponses(rows),
	})
}
