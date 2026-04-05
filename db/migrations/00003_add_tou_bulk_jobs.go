package migrations

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(up00003, down00003)
}

func up00003(ctx context.Context, tx *sql.Tx) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS tou_bulk_jobs (
			id VARCHAR(64) PRIMARY KEY,
			status VARCHAR(32) NOT NULL,
			source_filename VARCHAR(255) NOT NULL,
			source_storage_path TEXT NOT NULL,
			idempotency_key VARCHAR(128) NULL,
			submitted_by VARCHAR(128) NULL,
			total_rows INTEGER NOT NULL DEFAULT 0,
			processed_rows INTEGER NOT NULL DEFAULT 0,
			success_rows INTEGER NOT NULL DEFAULT 0,
			failed_rows INTEGER NOT NULL DEFAULT 0,
			error_reason TEXT NULL,
			started_at TIMESTAMPTZ NULL,
			completed_at TIMESTAMPTZ NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			CHECK (status IN ('queued', 'processing', 'completed', 'completed_with_errors', 'failed', 'cancelled'))
		)`,
		`CREATE TABLE IF NOT EXISTS tou_bulk_job_rows (
			id BIGSERIAL PRIMARY KEY,
			job_id VARCHAR(64) NOT NULL REFERENCES tou_bulk_jobs(id) ON DELETE CASCADE,
			row_number INTEGER NOT NULL,
			charger_id VARCHAR(64) NOT NULL DEFAULT '',
			effective_from VARCHAR(32) NOT NULL DEFAULT '',
			effective_to VARCHAR(32) NOT NULL DEFAULT '',
			start_time VARCHAR(16) NOT NULL DEFAULT '',
			end_time VARCHAR(16) NOT NULL DEFAULT '',
			price_per_kwh VARCHAR(32) NOT NULL DEFAULT '',
			status VARCHAR(16) NOT NULL,
			error_code VARCHAR(64) NULL,
			error_message TEXT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			CHECK (status IN ('pending', 'processed', 'failed', 'skipped'))
		)`,
		`CREATE INDEX IF NOT EXISTS idx_tou_bulk_jobs_status_created_at
			ON tou_bulk_jobs (status, created_at)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS uq_tou_bulk_jobs_idempotency_key_not_null
			ON tou_bulk_jobs (idempotency_key)
			WHERE idempotency_key IS NOT NULL`,
		`CREATE UNIQUE INDEX IF NOT EXISTS uq_tou_bulk_job_rows_job_rownum
			ON tou_bulk_job_rows (job_id, row_number)`,
		`CREATE INDEX IF NOT EXISTS idx_tou_bulk_job_rows_job_status
			ON tou_bulk_job_rows (job_id, status)`,
		`CREATE INDEX IF NOT EXISTS idx_tou_bulk_job_rows_charger_id
			ON tou_bulk_job_rows (charger_id)`,
	}

	for _, q := range queries {
		if _, err := tx.ExecContext(ctx, q); err != nil {
			return fmt.Errorf("migration 00003 up failed: %w", err)
		}
	}

	return nil
}

func down00003(ctx context.Context, tx *sql.Tx) error {
	queries := []string{
		`DROP INDEX IF EXISTS idx_tou_bulk_job_rows_charger_id`,
		`DROP INDEX IF EXISTS idx_tou_bulk_job_rows_job_status`,
		`DROP INDEX IF EXISTS uq_tou_bulk_job_rows_job_rownum`,
		`DROP INDEX IF EXISTS uq_tou_bulk_jobs_idempotency_key_not_null`,
		`DROP INDEX IF EXISTS idx_tou_bulk_jobs_status_created_at`,
		`DROP TABLE IF EXISTS tou_bulk_job_rows`,
		`DROP TABLE IF EXISTS tou_bulk_jobs`,
	}

	for _, q := range queries {
		if _, err := tx.ExecContext(ctx, q); err != nil {
			return fmt.Errorf("migration 00003 down failed: %w", err)
		}
	}

	return nil
}
