package mapper

import (
	"time"

	"gin-app/internal/bo"
	"gin-app/internal/dto"
)

func ToDTOTOUBulkJobResponse(job bo.TOUBulkJob) dto.TOUBulkJobResponse {
	response := dto.TOUBulkJobResponse{
		JobID:          job.ID,
		Status:         job.Status,
		SourceFilename: job.SourceFilename,
		TotalRows:      job.TotalRows,
		ProcessedRows:  job.ProcessedRows,
		SuccessRows:    job.SuccessRows,
		FailedRows:     job.FailedRows,
		ErrorReason:    job.ErrorReason,
		CreatedAt:      job.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:      job.UpdatedAt.UTC().Format(time.RFC3339),
	}
	if job.StartedAt != nil {
		response.StartedAt = job.StartedAt.UTC().Format(time.RFC3339)
	}
	if job.CompletedAt != nil {
		response.CompletedAt = job.CompletedAt.UTC().Format(time.RFC3339)
	}
	return response
}

func ToDTOTOUBulkJobRowResponses(rows []bo.TOUBulkJobRow) []dto.TOUBulkJobRowResponse {
	response := make([]dto.TOUBulkJobRowResponse, 0, len(rows))
	for _, row := range rows {
		response = append(response, dto.TOUBulkJobRowResponse{
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
	return response
}
