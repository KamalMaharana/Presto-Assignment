package service

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"gin-app/internal/bo"
	"gin-app/internal/models"
	"gin-app/internal/repository"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	defaultBulkStorageDir = "tmp/tou-bulk-jobs"
	defaultBulkPollPeriod = 2 * time.Second
)

var expectedBulkCSVHeader = []string{
	"charger_id",
	"effective_from",
	"effective_to",
	"start_time",
	"end_time",
	"price_per_kwh",
}

type TOUBulkService interface {
	CreateJobFromCSV(filename string, reader io.Reader, idempotencyKey string, submittedBy string, requestID string) (*bo.TOUBulkJob, error)
	GetJob(jobID string) (*bo.TOUBulkJob, error)
	ListRows(jobID string, status string, limit int, offset int) ([]bo.TOUBulkJobRow, error)
	StartWorker(ctx context.Context, pollInterval time.Duration)
}

type touBulkService struct {
	logger     *slog.Logger
	repo       repository.TOUBulkJobRepository
	touService TOUService
	storageDir string
	jobTrace   sync.Map
}

func NewTOUBulkService(
	logger *slog.Logger,
	repo repository.TOUBulkJobRepository,
	touService TOUService,
) TOUBulkService {
	if logger == nil {
		logger = slog.Default()
	}
	return &touBulkService{
		logger:     logger,
		repo:       repo,
		touService: touService,
		storageDir: defaultBulkStorageDir,
	}
}

func (s *touBulkService) CreateJobFromCSV(filename string, reader io.Reader, idempotencyKey string, submittedBy string, requestID string) (*bo.TOUBulkJob, error) {
	requestID = traceRequestID(requestID)
	idempotencyKey = strings.TrimSpace(idempotencyKey)
	if idempotencyKey != "" {
		existing, err := s.repo.GetJobByIdempotencyKey(idempotencyKey)
		if err == nil {
			mapped := toBOBulkJob(existing)
			s.jobTrace.Store(mapped.ID, requestID)
			s.logger.Info("reused existing tou bulk job due to idempotency key",
				"request_id", requestID,
				"job_id", mapped.ID,
				"idempotency_key", idempotencyKey,
			)
			return &mapped, nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	}

	filename = strings.TrimSpace(filename)
	if filename == "" {
		filename = "tou-rates.csv"
	}

	jobID := uuid.NewString()
	s.jobTrace.Store(jobID, requestID)
	if err := os.MkdirAll(s.storageDir, 0o755); err != nil {
		return nil, err
	}

	storageName := fmt.Sprintf("%s-%s", jobID, sanitizeFilename(filename))
	storagePath := filepath.Join(s.storageDir, storageName)
	if err := writeReaderToFile(storagePath, reader); err != nil {
		return nil, err
	}

	rows, err := parseBulkCSVRows(jobID, storagePath)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, errors.New("csv does not contain any data rows")
	}

	var keyPtr *string
	if idempotencyKey != "" {
		keyPtr = &idempotencyKey
	}
	initialFailed := 0
	for _, row := range rows {
		if row.Status == models.TOUBulkJobRowStatusFailed {
			initialFailed++
		}
	}

	job := &models.TOUBulkJob{
		ID:                jobID,
		Status:            models.TOUBulkJobStatusQueued,
		SourceFilename:    filename,
		SourceStoragePath: storagePath,
		IdempotencyKey:    keyPtr,
		SubmittedBy:       strings.TrimSpace(submittedBy),
		TotalRows:         len(rows),
		ProcessedRows:     initialFailed,
		SuccessRows:       0,
		FailedRows:        initialFailed,
	}

	if err := s.repo.CreateJobWithRows(job, rows); err != nil {
		return nil, err
	}

	updated, err := s.repo.RefreshJobCounters(jobID)
	if err != nil {
		return nil, err
	}
	mapped := toBOBulkJob(updated)
	s.logger.Info("created tou bulk job from csv",
		"request_id", requestID,
		"job_id", mapped.ID,
		"total_rows", mapped.TotalRows,
		"source_filename", mapped.SourceFilename,
	)
	return &mapped, nil
}

func (s *touBulkService) GetJob(jobID string) (*bo.TOUBulkJob, error) {
	job, err := s.repo.GetJobByID(strings.TrimSpace(jobID))
	if err != nil {
		return nil, err
	}
	mapped := toBOBulkJob(job)
	return &mapped, nil
}

func (s *touBulkService) ListRows(jobID string, status string, limit int, offset int) ([]bo.TOUBulkJobRow, error) {
	jobID = strings.TrimSpace(jobID)
	if _, err := s.repo.GetJobByID(jobID); err != nil {
		return nil, err
	}

	rows, err := s.repo.ListJobRows(jobID, strings.TrimSpace(status), limit, offset)
	if err != nil {
		return nil, err
	}
	result := make([]bo.TOUBulkJobRow, 0, len(rows))
	for _, row := range rows {
		result = append(result, bo.TOUBulkJobRow{
			RowNumber:     row.RowNumber,
			ChargerID:     row.ChargerID,
			EffectiveFrom: row.EffectiveFrom,
			EffectiveTo:   row.EffectiveTo,
			StartTime:     row.StartTime,
			EndTime:       row.EndTime,
			PricePerKwh:   row.PricePerKwh,
			Status:        row.Status,
			ErrorCode:     row.ErrorCode,
			ErrorMessage:  row.ErrorMessage,
		})
	}
	return result, nil
}

func (s *touBulkService) StartWorker(ctx context.Context, pollInterval time.Duration) {
	if pollInterval <= 0 {
		pollInterval = defaultBulkPollPeriod
	}
	s.logger.Info("tou bulk worker started", "request_id", "system", "poll_interval", pollInterval.String())

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		processedJob, err := s.processNextJob()
		if err != nil {
			s.logger.Error("bulk job processing failed", "request_id", "system", "error", err)
		}
		if processedJob {
			continue
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(pollInterval):
		}
	}
}

func (s *touBulkService) processNextJob() (bool, error) {
	job, err := s.repo.ClaimNextQueuedJob()
	if err != nil {
		return false, err
	}
	if job == nil {
		return false, nil
	}

	requestID := s.requestIDForJob(job.ID)
	s.logger.Info("processing tou bulk job", "request_id", requestID, "job_id", job.ID)

	rows, err := s.repo.ListAllJobRows(job.ID)
	if err != nil {
		return true, s.failJob(job.ID, fmt.Sprintf("failed to list rows: %v", err), requestID)
	}

	groups := make(map[string][]models.TOUBulkJobRow)
	for _, row := range rows {
		if row.Status != models.TOUBulkJobRowStatusPending {
			continue
		}
		key := groupKey(row.ChargerID, row.EffectiveFrom, row.EffectiveTo)
		groups[key] = append(groups[key], row)
	}

	for _, groupedRows := range groups {
		if len(groupedRows) == 0 {
			continue
		}

		rowIDs := make([]uint, 0, len(groupedRows))
		for _, row := range groupedRows {
			rowIDs = append(rowIDs, row.ID)
		}

		chargerID := strings.TrimSpace(groupedRows[0].ChargerID)
		effectiveFrom := strings.TrimSpace(groupedRows[0].EffectiveFrom)
		effectiveTo := strings.TrimSpace(groupedRows[0].EffectiveTo)

		if chargerID == "" || effectiveFrom == "" {
			if updateErr := s.repo.UpdateRowsStatus(rowIDs, models.TOUBulkJobRowStatusFailed, "validation_error", "charger_id and effective_from are required"); updateErr != nil {
				return true, s.failJob(job.ID, fmt.Sprintf("failed to update invalid rows: %v", updateErr), requestID)
			}
			continue
		}

		periods := make([]bo.TOUPeriod, 0, len(groupedRows))
		parseFailed := false
		parseErrMessage := ""
		for _, row := range groupedRows {
			price, parseErr := strconv.ParseFloat(strings.TrimSpace(row.PricePerKwh), 64)
			if parseErr != nil {
				parseFailed = true
				parseErrMessage = fmt.Sprintf("invalid price_per_kwh at row %d", row.RowNumber)
				break
			}
			periods = append(periods, bo.TOUPeriod{
				StartTime:   strings.TrimSpace(row.StartTime),
				EndTime:     strings.TrimSpace(row.EndTime),
				PricePerKwh: price,
			})
		}

		if parseFailed {
			if updateErr := s.repo.UpdateRowsStatus(rowIDs, models.TOUBulkJobRowStatusFailed, "validation_error", parseErrMessage); updateErr != nil {
				return true, s.failJob(job.ID, fmt.Sprintf("failed to update parse failures: %v", updateErr), requestID)
			}
			continue
		}

		err = s.touService.ReplaceSchedule(chargerID, bo.UpsertTOUScheduleInput{
			EffectiveFrom: effectiveFrom,
			EffectiveTo:   effectiveTo,
			Periods:       periods,
		})
		if err != nil {
			if updateErr := s.repo.UpdateRowsStatus(rowIDs, models.TOUBulkJobRowStatusFailed, "processing_error", err.Error()); updateErr != nil {
				return true, s.failJob(job.ID, fmt.Sprintf("failed to update failed rows: %v", updateErr), requestID)
			}
			s.logger.Warn("failed to apply grouped rows for tou bulk job",
				"request_id", requestID,
				"job_id", job.ID,
				"charger_id", chargerID,
				"effective_from", effectiveFrom,
				"effective_to", effectiveTo,
				"error", err,
			)
			continue
		}

		if updateErr := s.repo.UpdateRowsStatus(rowIDs, models.TOUBulkJobRowStatusProcessed, "", ""); updateErr != nil {
			return true, s.failJob(job.ID, fmt.Sprintf("failed to mark rows processed: %v", updateErr), requestID)
		}
	}

	latest, err := s.repo.RefreshJobCounters(job.ID)
	if err != nil {
		return true, s.failJob(job.ID, fmt.Sprintf("failed to refresh counters: %v", err), requestID)
	}

	completedAt := time.Now().UTC()
	finalStatus := models.TOUBulkJobStatusCompleted
	finalErrorReason := ""
	if latest.FailedRows > 0 {
		finalStatus = models.TOUBulkJobStatusCompletedWithErrors
		finalErrorReason = fmt.Sprintf("%d rows failed during processing", latest.FailedRows)
	}
	if err := s.repo.UpdateJobStatus(job.ID, finalStatus, finalErrorReason, &completedAt); err != nil {
		return true, err
	}

	s.logger.Info("completed tou bulk job",
		"request_id", requestID,
		"job_id", job.ID,
		"status", finalStatus,
		"failed_rows", latest.FailedRows,
	)
	s.jobTrace.Delete(job.ID)
	return true, nil
}

func (s *touBulkService) failJob(jobID string, reason string, requestID string) error {
	completedAt := time.Now().UTC()
	s.logger.Error("marking tou bulk job as failed",
		"request_id", requestID,
		"job_id", jobID,
		"reason", reason,
	)
	s.jobTrace.Delete(jobID)
	return s.repo.UpdateJobStatus(jobID, models.TOUBulkJobStatusFailed, reason, &completedAt)
}

func (s *touBulkService) requestIDForJob(jobID string) string {
	if value, ok := s.jobTrace.Load(jobID); ok {
		if id, castOK := value.(string); castOK {
			return traceRequestID(id)
		}
	}
	return "n/a"
}

func traceRequestID(requestID string) string {
	requestID = strings.TrimSpace(requestID)
	if requestID == "" {
		return "n/a"
	}
	return requestID
}

func parseBulkCSVRows(jobID string, path string) ([]models.TOUBulkJobRow, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	header, err := reader.Read()
	if err != nil {
		if errors.Is(err, io.EOF) {
			return nil, errors.New("csv is empty")
		}
		return nil, err
	}

	if err := validateBulkCSVHeader(header); err != nil {
		return nil, err
	}

	rows := make([]models.TOUBulkJobRow, 0)
	rowNumber := 1
	for {
		record, readErr := reader.Read()
		if errors.Is(readErr, io.EOF) {
			break
		}
		rowNumber++
		if readErr != nil {
			rows = append(rows, models.TOUBulkJobRow{
				JobID:        jobID,
				RowNumber:    rowNumber,
				Status:       models.TOUBulkJobRowStatusFailed,
				ErrorCode:    "parse_error",
				ErrorMessage: readErr.Error(),
			})
			continue
		}

		row := models.TOUBulkJobRow{
			JobID:     jobID,
			RowNumber: rowNumber,
			Status:    models.TOUBulkJobRowStatusPending,
		}

		if len(record) != len(expectedBulkCSVHeader) {
			row.Status = models.TOUBulkJobRowStatusFailed
			row.ErrorCode = "validation_error"
			row.ErrorMessage = "row must contain 6 columns"
			rows = append(rows, row)
			continue
		}

		row.ChargerID = strings.TrimSpace(record[0])
		row.EffectiveFrom = strings.TrimSpace(record[1])
		row.EffectiveTo = strings.TrimSpace(record[2])
		row.StartTime = strings.TrimSpace(record[3])
		row.EndTime = strings.TrimSpace(record[4])
		row.PricePerKwh = strings.TrimSpace(record[5])

		if row.ChargerID == "" || row.EffectiveFrom == "" || row.StartTime == "" || row.EndTime == "" || row.PricePerKwh == "" {
			row.Status = models.TOUBulkJobRowStatusFailed
			row.ErrorCode = "validation_error"
			row.ErrorMessage = "charger_id, effective_from, start_time, end_time and price_per_kwh are required"
		}
		rows = append(rows, row)
	}

	return rows, nil
}

func validateBulkCSVHeader(header []string) error {
	if len(header) != len(expectedBulkCSVHeader) {
		return errors.New("invalid csv header")
	}
	for i := range expectedBulkCSVHeader {
		if strings.ToLower(strings.TrimSpace(header[i])) != expectedBulkCSVHeader[i] {
			return errors.New("invalid csv header")
		}
	}
	return nil
}

func writeReaderToFile(path string, reader io.Reader) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, reader)
	return err
}

func sanitizeFilename(name string) string {
	var builder strings.Builder
	for _, char := range name {
		if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') || char == '.' || char == '-' || char == '_' {
			builder.WriteRune(char)
			continue
		}
		builder.WriteByte('_')
	}

	result := strings.TrimSpace(builder.String())
	if result == "" {
		return "tou-rates.csv"
	}
	return result
}

func groupKey(chargerID string, effectiveFrom string, effectiveTo string) string {
	return strings.TrimSpace(chargerID) + "|" + strings.TrimSpace(effectiveFrom) + "|" + strings.TrimSpace(effectiveTo)
}

func toBOBulkJob(job *models.TOUBulkJob) bo.TOUBulkJob {
	return bo.TOUBulkJob{
		ID:             job.ID,
		Status:         job.Status,
		SourceFilename: job.SourceFilename,
		TotalRows:      job.TotalRows,
		ProcessedRows:  job.ProcessedRows,
		SuccessRows:    job.SuccessRows,
		FailedRows:     job.FailedRows,
		ErrorReason:    job.ErrorReason,
		StartedAt:      job.StartedAt,
		CompletedAt:    job.CompletedAt,
		CreatedAt:      job.CreatedAt,
		UpdatedAt:      job.UpdatedAt,
	}
}
