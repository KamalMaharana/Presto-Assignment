package models

import "time"

const (
	TOUBulkJobStatusQueued              = "queued"
	TOUBulkJobStatusProcessing          = "processing"
	TOUBulkJobStatusCompleted           = "completed"
	TOUBulkJobStatusCompletedWithErrors = "completed_with_errors"
	TOUBulkJobStatusFailed              = "failed"
	TOUBulkJobStatusCancelled           = "cancelled"
)

type TOUBulkJob struct {
	ID                string     `gorm:"primaryKey;size:64" json:"id"`
	Status            string     `gorm:"size:32;not null;index:idx_tou_bulk_jobs_status_created_at,priority:1" json:"status"`
	SourceFilename    string     `gorm:"size:255;not null" json:"source_filename"`
	SourceStoragePath string     `gorm:"type:text;not null" json:"source_storage_path"`
	IdempotencyKey    *string    `gorm:"size:128;uniqueIndex:uq_tou_bulk_jobs_idempotency_key_not_null,where:idempotency_key IS NOT NULL" json:"idempotency_key,omitempty"`
	SubmittedBy       string     `gorm:"size:128" json:"submitted_by,omitempty"`
	TotalRows         int        `gorm:"not null;default:0" json:"total_rows"`
	ProcessedRows     int        `gorm:"not null;default:0" json:"processed_rows"`
	SuccessRows       int        `gorm:"not null;default:0" json:"success_rows"`
	FailedRows        int        `gorm:"not null;default:0" json:"failed_rows"`
	ErrorReason       string     `gorm:"type:text" json:"error_reason,omitempty"`
	StartedAt         *time.Time `json:"started_at,omitempty"`
	CompletedAt       *time.Time `json:"completed_at,omitempty"`
	CreatedAt         time.Time  `gorm:"index:idx_tou_bulk_jobs_status_created_at,priority:2" json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}
