SHELL := /bin/bash

.PHONY: setup run dev db-up db-down ensure-env migrate-up migrate-down migrate-status migrate-create

GOOSE_DIR := db/migrations

ensure-env:
	@if [ ! -f .env ]; then cp .env.example .env; fi

setup:
	@$(MAKE) ensure-env
	@go mod tidy
	@go install github.com/air-verse/air@latest
	@echo "Setup completed. Run 'make db-up' and then 'make dev' or 'make run'."

run: ensure-env
	@set -a; source .env; set +a; go run ./cmd/migrate up && go run ./cmd/server

dev: ensure-env
	@set -a; source .env; set +a; \
	go run ./cmd/migrate up; \
	if command -v air >/dev/null 2>&1; then \
		air -c .air.toml; \
	else \
		echo "air not found in PATH. Running via go run..."; \
		go run github.com/air-verse/air@latest -c .air.toml; \
	fi

db-up:
	@docker compose up -d postgres

db-down:
	@if [ -n "$(SERVICE)" ]; then \
		echo "Stopping compose service: $(SERVICE)"; \
		docker compose stop "$(SERVICE)"; \
	elif [ -n "$(CONTAINER)" ]; then \
		echo "Stopping container: $(CONTAINER)"; \
		docker stop "$(CONTAINER)" >/dev/null && docker rm "$(CONTAINER)" >/dev/null; \
	else \
		echo "Stopping all compose services"; \
		docker compose down; \
	fi

migrate-up: ensure-env
	@set -a; source .env; set +a; go run ./cmd/migrate up

migrate-down: ensure-env
	@set -a; source .env; set +a; go run ./cmd/migrate down

migrate-status: ensure-env
	@set -a; source .env; set +a; go run ./cmd/migrate status

migrate-create:
	@if [ -z "$(NAME)" ]; then echo "Usage: make migrate-create NAME=create_table"; exit 1; fi
	@mkdir -p "$(GOOSE_DIR)"
	@go run github.com/pressly/goose/v3/cmd/goose@latest -dir "$(GOOSE_DIR)" create "$(NAME)" go
