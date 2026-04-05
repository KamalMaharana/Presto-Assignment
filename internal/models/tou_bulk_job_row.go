package models

import "time"

const (
	TOUBulkJobRowStatusPending   = "pending"
	TOUBulkJobRowStatusProcessed = "processed"
	TOUBulkJobRowStatusFailed    = "failed"
	TOUBulkJobRowStatusSkipped   = "skipped"
)

type TOUBulkJobRow struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	JobID         string    `gorm:"size:64;not null;index:idx_tou_bulk_job_rows_job_status,priority:1;index:uq_tou_bulk_job_rows_job_rownum,unique,priority:1" json:"job_id"`
	RowNumber     int       `gorm:"not null;index:uq_tou_bulk_job_rows_job_rownum,unique,priority:2" json:"row_number"`
	ChargerID     string    `gorm:"size:64;not null;default:'';index:idx_tou_bulk_job_rows_charger_id" json:"charger_id"`
	EffectiveFrom string    `gorm:"size:32;not null;default:''" json:"effective_from"`
	EffectiveTo   string    `gorm:"size:32;not null;default:''" json:"effective_to"`
	StartTime     string    `gorm:"size:16;not null;default:''" json:"start_time"`
	EndTime       string    `gorm:"size:16;not null;default:''" json:"end_time"`
	PricePerKwh   string    `gorm:"size:32;not null;default:''" json:"price_per_kwh"`
	Status        string    `gorm:"size:16;not null;index:idx_tou_bulk_job_rows_job_status,priority:2" json:"status"`
	ErrorCode     string    `gorm:"size:64" json:"error_code,omitempty"`
	ErrorMessage  string    `gorm:"type:text" json:"error_message,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}
