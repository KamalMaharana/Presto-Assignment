package repository

import (
	"time"

	"gin-app/internal/models"

	"gorm.io/gorm"
)

type TOUBulkJobRepository interface {
	CreateJobWithRows(job *models.TOUBulkJob, rows []models.TOUBulkJobRow) error
	GetJobByID(jobID string) (*models.TOUBulkJob, error)
	GetJobByIdempotencyKey(key string) (*models.TOUBulkJob, error)
	ClaimNextQueuedJob() (*models.TOUBulkJob, error)
	ListJobRows(jobID string, status string, limit int, offset int) ([]models.TOUBulkJobRow, error)
	ListAllJobRows(jobID string) ([]models.TOUBulkJobRow, error)
	UpdateRowsStatus(rowIDs []uint, status string, errorCode string, errorMessage string) error
	UpdateJobStatus(jobID string, status string, errorReason string, completedAt *time.Time) error
	RefreshJobCounters(jobID string) (*models.TOUBulkJob, error)
}

type touBulkJobRepository struct {
	db *gorm.DB
}

func NewTOUBulkJobRepository(db *gorm.DB) TOUBulkJobRepository {
	return &touBulkJobRepository{db: db}
}

func (r *touBulkJobRepository) CreateJobWithRows(job *models.TOUBulkJob, rows []models.TOUBulkJobRow) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(job).Error; err != nil {
			return err
		}
		if len(rows) == 0 {
			return nil
		}
		return tx.Create(&rows).Error
	})
}

func (r *touBulkJobRepository) GetJobByID(jobID string) (*models.TOUBulkJob, error) {
	var job models.TOUBulkJob
	if err := r.db.Where("id = ?", jobID).First(&job).Error; err != nil {
		return nil, err
	}
	return &job, nil
}

func (r *touBulkJobRepository) GetJobByIdempotencyKey(key string) (*models.TOUBulkJob, error) {
	var job models.TOUBulkJob
	if err := r.db.Where("idempotency_key = ?", key).First(&job).Error; err != nil {
		return nil, err
	}
	return &job, nil
}

func (r *touBulkJobRepository) ClaimNextQueuedJob() (*models.TOUBulkJob, error) {
	var claimed models.TOUBulkJob
	query := `
UPDATE tou_bulk_jobs
SET status = ?, started_at = NOW(), updated_at = NOW(), error_reason = NULL
WHERE id = (
	SELECT id
	FROM tou_bulk_jobs
	WHERE status = ?
	ORDER BY created_at ASC
	LIMIT 1
	FOR UPDATE SKIP LOCKED
)
RETURNING *`

	if err := r.db.Raw(query, models.TOUBulkJobStatusProcessing, models.TOUBulkJobStatusQueued).Scan(&claimed).Error; err != nil {
		return nil, err
	}
	if claimed.ID == "" {
		return nil, nil
	}
	return &claimed, nil
}

func (r *touBulkJobRepository) ListJobRows(jobID string, status string, limit int, offset int) ([]models.TOUBulkJobRow, error) {
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}
	if offset < 0 {
		offset = 0
	}

	query := r.db.Where("job_id = ?", jobID)
	if status != "" {
		query = query.Where("status = ?", status)
	}

	var rows []models.TOUBulkJobRow
	if err := query.Order("row_number ASC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *touBulkJobRepository) ListAllJobRows(jobID string) ([]models.TOUBulkJobRow, error) {
	var rows []models.TOUBulkJobRow
	if err := r.db.Where("job_id = ?", jobID).Order("row_number ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *touBulkJobRepository) UpdateRowsStatus(rowIDs []uint, status string, errorCode string, errorMessage string) error {
	if len(rowIDs) == 0 {
		return nil
	}
	return r.db.Model(&models.TOUBulkJobRow{}).
		Where("id IN ?", rowIDs).
		Updates(map[string]interface{}{
			"status":        status,
			"error_code":    errorCode,
			"error_message": errorMessage,
			"updated_at":    time.Now().UTC(),
		}).Error
}

func (r *touBulkJobRepository) UpdateJobStatus(jobID string, status string, errorReason string, completedAt *time.Time) error {
	updates := map[string]interface{}{
		"status":       status,
		"error_reason": errorReason,
		"updated_at":   time.Now().UTC(),
	}
	if completedAt != nil {
		updates["completed_at"] = *completedAt
	}
	return r.db.Model(&models.TOUBulkJob{}).Where("id = ?", jobID).Updates(updates).Error
}

func (r *touBulkJobRepository) RefreshJobCounters(jobID string) (*models.TOUBulkJob, error) {
	type rowAggregate struct {
		Total     int
		Processed int
		Success   int
		Failed    int
	}

	var aggregate rowAggregate
	if err := r.db.Raw(`
SELECT
	COUNT(*) AS total,
	COALESCE(SUM(CASE WHEN status IN ('processed', 'failed', 'skipped') THEN 1 ELSE 0 END), 0) AS processed,
	COALESCE(SUM(CASE WHEN status = 'processed' THEN 1 ELSE 0 END), 0) AS success,
	COALESCE(SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END), 0) AS failed
FROM tou_bulk_job_rows
WHERE job_id = ?`, jobID).Scan(&aggregate).Error; err != nil {
		return nil, err
	}

	if err := r.db.Model(&models.TOUBulkJob{}).
		Where("id = ?", jobID).
		Updates(map[string]interface{}{
			"total_rows":     aggregate.Total,
			"processed_rows": aggregate.Processed,
			"success_rows":   aggregate.Success,
			"failed_rows":    aggregate.Failed,
			"updated_at":     time.Now().UTC(),
		}).Error; err != nil {
		return nil, err
	}

	return r.GetJobByID(jobID)
}
