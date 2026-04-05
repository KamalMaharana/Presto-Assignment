package dto

type CreateTOUBulkJobResponse struct {
	JobID  string `json:"job_id"`
	Status string `json:"status"`
}

type TOUBulkJobResponse struct {
	JobID          string `json:"job_id"`
	Status         string `json:"status"`
	SourceFilename string `json:"source_filename"`
	TotalRows      int    `json:"total_rows"`
	ProcessedRows  int    `json:"processed_rows"`
	SuccessRows    int    `json:"success_rows"`
	FailedRows     int    `json:"failed_rows"`
	ErrorReason    string `json:"error_reason,omitempty"`
	StartedAt      string `json:"started_at,omitempty"`
	CompletedAt    string `json:"completed_at,omitempty"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
}

type TOUBulkJobRowResponse struct {
	RowNumber     int    `json:"row_number"`
	ChargerID     string `json:"charger_id"`
	EffectiveFrom string `json:"effective_from"`
	EffectiveTo   string `json:"effective_to,omitempty"`
	StartTime     string `json:"start_time"`
	EndTime       string `json:"end_time"`
	PricePerKwh   string `json:"price_per_kwh"`
	Status        string `json:"status"`
	ErrorCode     string `json:"error_code,omitempty"`
	ErrorMessage  string `json:"error_message,omitempty"`
}
