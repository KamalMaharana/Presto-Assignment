// Package database contains the Postgres database connection logic.
package database

import (
	"fmt"
	"time"

	"gin-app/internal/config"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func ConnectPostgres(cfg config.Config) (*gorm.DB, error) {
	var db *gorm.DB
	var err error

	for attempt := 1; attempt <= cfg.PostgresMaxRetries(); attempt++ {
		db, err = gorm.Open(postgres.Open(cfg.PostgresDSN()), &gorm.Config{})
		if err == nil {
			return db, nil
		}

		time.Sleep(time.Duration(cfg.PostgresRetryDelaySeconds()) * time.Second)
	}

	return nil, fmt.Errorf("postgres connection failed after retries: %w", err)
}
