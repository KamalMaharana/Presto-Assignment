package main

import (
	"context"
	"fmt"
	"log"
	"os"

	_ "gin-app/db/migrations"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
	"github.com/pressly/goose/v3"
)

const migrationsDir = "db/migrations"

func main() {
	_ = godotenv.Load()

	if len(os.Args) < 2 {
		log.Fatalf("usage: go run ./cmd/migrate <up|down|status|version>")
	}

	command := os.Args[1]
	dbDSN := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		getEnv("DB_USER", "postgres"),
		getEnv("DB_PASS", "postgres"),
		getEnv("DB_HOST", "localhost"),
		getEnv("DB_PORT", "5432"),
		getEnv("DB_NAME", "appdb"),
		getEnv("DB_SSLMODE", "disable"),
	)

	db, err := goose.OpenDBWithDriver("pgx", dbDSN)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := goose.SetDialect("postgres"); err != nil {
		log.Fatalf("failed to set goose dialect: %v", err)
	}

	if err := goose.RunContext(context.Background(), command, db, migrationsDir); err != nil {
		log.Fatalf("goose %s failed: %v", command, err)
	}
}

func getEnv(key string, fallback string) string {
	value, ok := os.LookupEnv(key)
	if !ok || value == "" {
		return fallback
	}

	return value
}
