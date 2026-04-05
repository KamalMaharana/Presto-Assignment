// Package migrations contains Goose database migrations.
package migrations

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(up00001, down00001)
}

func up00001(ctx context.Context, tx *sql.Tx) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS chargers (
			id VARCHAR(64) PRIMARY KEY,
			name VARCHAR(120) NOT NULL,
			location VARCHAR(255),
			timezone VARCHAR(64) NOT NULL DEFAULT 'UTC',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS tou_rate_periods (
			id BIGSERIAL PRIMARY KEY,
			charger_id VARCHAR(64) NOT NULL REFERENCES chargers(id) ON DELETE CASCADE,
			effective_from DATE NOT NULL,
			effective_to DATE NULL,
			start_minute INTEGER NOT NULL,
			end_minute INTEGER NOT NULL,
			price_per_kwh NUMERIC(10,4) NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			CHECK (start_minute >= 0 AND start_minute <= 1439),
			CHECK (end_minute >= 1 AND end_minute <= 1440),
			CHECK (end_minute > start_minute),
			CHECK (price_per_kwh > 0)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_tou_charger_effective_start
			ON tou_rate_periods (charger_id, effective_from, start_minute)`,
		`CREATE INDEX IF NOT EXISTS idx_tou_charger_id
			ON tou_rate_periods (charger_id)`,
		`CREATE INDEX IF NOT EXISTS idx_tou_effective_to
			ON tou_rate_periods (effective_to)`,
	}

	for _, q := range queries {
		if _, err := tx.ExecContext(ctx, q); err != nil {
			return fmt.Errorf("migration 00001 up failed: %w", err)
		}
	}

	return nil
}

func down00001(ctx context.Context, tx *sql.Tx) error {
	queries := []string{
		`DROP TABLE IF EXISTS tou_rate_periods`,
		`DROP TABLE IF EXISTS chargers`,
	}

	for _, q := range queries {
		if _, err := tx.ExecContext(ctx, q); err != nil {
			return fmt.Errorf("migration 00001 down failed: %w", err)
		}
	}

	return nil
}
