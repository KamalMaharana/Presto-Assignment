package bo

import "time"

type TOUBulkJob struct {
	ID             string
	Status         string
	SourceFilename string
	TotalRows      int
	ProcessedRows  int
	SuccessRows    int
	FailedRows     int
	ErrorReason    string
	StartedAt      *time.Time
	CompletedAt    *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type TOUBulkJobRow struct {
	RowNumber     int
	ChargerID     string
	EffectiveFrom string
	EffectiveTo   string
	StartTime     string
	EndTime       string
	PricePerKwh   string
	Status        string
	ErrorCode     string
	ErrorMessage  string
}
