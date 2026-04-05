// Package migrations contains Goose database migrations.
package migrations

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(up00002, down00002)
}

func up00002(ctx context.Context, tx *sql.Tx) error {
	query := `
		ALTER TABLE chargers
		ADD COLUMN IF NOT EXISTS default_price_per_kwh NUMERIC(10,4) NOT NULL DEFAULT 0.2000
	`
	if _, err := tx.ExecContext(ctx, query); err != nil {
		return fmt.Errorf("migration 00002 up failed: %w", err)
	}

	return nil
}

func down00002(ctx context.Context, tx *sql.Tx) error {
	query := `
		ALTER TABLE chargers
		DROP COLUMN IF EXISTS default_price_per_kwh
	`
	if _, err := tx.ExecContext(ctx, query); err != nil {
		return fmt.Errorf("migration 00002 down failed: %w", err)
	}

	return nil
}
