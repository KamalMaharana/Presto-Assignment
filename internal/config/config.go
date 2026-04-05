// Package config provides application configuration loading and access helpers.
package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	AppPort   string
	LogLevel  string
	DBHost    string
	DBPort    string
	DBUser    string
	DBPass    string
	DBName    string
	DBSSLMode string
}

func Load() Config {
	_ = godotenv.Load()

	return Config{
		AppPort:   getEnv("APP_PORT", "8080"),
		LogLevel:  getEnv("LOG_LEVEL", "info"),
		DBHost:    getEnv("DB_HOST", "localhost"),
		DBPort:    getEnv("DB_PORT", "5432"),
		DBUser:    getEnv("DB_USER", "postgres"),
		DBPass:    getEnv("DB_PASS", "postgres"),
		DBName:    getEnv("DB_NAME", "appdb"),
		DBSSLMode: getEnv("DB_SSLMODE", "disable"),
	}
}

func (c Config) ServerAddress() string {
	return fmt.Sprintf(":%s", c.AppPort)
}

func (c Config) PostgresDSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s TimeZone=UTC",
		c.DBHost,
		c.DBPort,
		c.DBUser,
		c.DBPass,
		c.DBName,
		c.DBSSLMode,
	)
}

func (c Config) PostgresMaxRetries() int {
	value := getEnv("DB_CONNECT_RETRIES", "10")
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 1 {
		return 10
	}

	return parsed
}

func (c Config) PostgresRetryDelaySeconds() int {
	value := getEnv("DB_CONNECT_RETRY_DELAY_SECONDS", "2")
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 1 {
		return 2
	}

	return parsed
}

func getEnv(key string, fallback string) string {
	value, exists := os.LookupEnv(key)
	if !exists || value == "" {
		return fallback
	}

	return value
}
