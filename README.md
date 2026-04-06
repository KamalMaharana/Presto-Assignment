# EV Charger TOU Pricing Service

Gin + GORM service for storing and retrieving charger-specific Time-of-Use (TOU) rates in $/kWh.

Schema is managed with Goose SQL migrations (`db/migrations`).

## Architecture

```text
.
├── cmd/server/main.go
├── internal/
│   ├── config/        # env loading and config
│   ├── database/      # postgres connection
│   ├── handler/       # HTTP handlers
│   ├── logger/        # structured JSON logger
│   ├── middleware/    # request ID + request logging
│   ├── models/        # GORM models
│   ├── repository/    # DB access layer
│   ├── router/        # route wiring
│   └── service/       # business logic/validation
├── docker-compose.yml # PostgreSQL
├── Makefile
└── .air.toml
```

## Database schema design

### `chargers`
- `id` (PK, string)
- `name` (required)
- `location`
- `timezone` (IANA timezone, example: `America/Los_Angeles`)
- `default_price_per_kwh` (NUMERIC(10,4), required)
- `created_at`, `updated_at`

### `tou_rate_periods`
- `id` (PK)
- `charger_id` (FK-like reference to `chargers.id`, indexed)
- `effective_from` (DATE, required)
- `effective_to` (DATE, nullable)
- `start_minute` (0-1439)
- `end_minute` (1-1440, exclusive)
- `price_per_kwh` (NUMERIC(10,4))
- `created_at`, `updated_at`

### `tou_bulk_jobs`
- `id` (PK, UUID string)
- `status` (`queued`, `processing`, `completed`, `completed_with_errors`, `failed`)
- `source_filename`, `source_storage_path`
- `idempotency_key` (nullable, unique when present)
- `submitted_by`
- `total_rows`, `processed_rows`, `success_rows`, `failed_rows`
- `error_reason`, `started_at`, `completed_at`
- `created_at`, `updated_at`

### `tou_bulk_job_rows`
- `id` (PK)
- `job_id` (indexed)
- `row_number` (unique per job)
- `charger_id`, `effective_from`, `effective_to`, `start_time`, `end_time`, `price_per_kwh`
- `status` (`pending`, `processed`, `failed`)
- `error_code`, `error_message`
- `created_at`, `updated_at`

### Why this schema
- Charger-specific rates are stored directly by `charger_id`.
- Daily TOU windows are represented as minute ranges for fast lookups.
- `effective_from`/`effective_to` supports schedule changes over time.
- Normalized model avoids duplicating charger metadata in pricing rows.
- Bulk ingestion uses async jobs + row-level statuses for retry-safe processing and auditability.

## API specification

Base path: `/api/v1`

### Response shape

All successful API responses are wrapped:

```json
{
  "status_code": 200,
  "request_id": "....",
  "message": "human readable message",
  "data": {}
}
```

Error responses:

```json
{
  "error": "human-readable message",
  "request_id": "...."
}
```

### Endpoints

| Method | Path | Description |
| --- | --- | --- |
| GET | `/health` | Health check |
| POST | `/api/v1/chargers` | Create charger |
| GET | `/api/v1/chargers` | List chargers |
| GET | `/api/v1/chargers/:charger_id` | Get charger by ID |
| PUT | `/api/v1/chargers/:charger_id/tou-rates` | Upsert TOU schedule |
| GET | `/api/v1/chargers/:charger_id/tou-rates?date=YYYY-MM-DD` | Get schedule for a date |
| GET | `/api/v1/chargers/:charger_id/tou-rate?date=YYYY-MM-DD&time=HH:MM` | Get rate at a specific date/time |
| POST | `/api/v1/tou-bulk-jobs` | Create async TOU bulk CSV job |
| GET | `/api/v1/tou-bulk-jobs/:job_id` | Get bulk job summary/status |
| GET | `/api/v1/tou-bulk-jobs/:job_id/rows` | List per-row processing results |

## cURL requests (all endpoints)

Set a base URL first:

```bash
BASE_URL="http://localhost:8080"
```

### Health

```bash
curl --request GET "$BASE_URL/health"
```

### Chargers

Create charger:

```bash
curl --request POST "$BASE_URL/api/v1/chargers" \
  --header "Content-Type: application/json" \
  --data '{
    "id": "charger-001",
    "name": "Depot Charger A",
    "location": "Seattle Depot",
    "timezone": "America/Los_Angeles",
    "default_price_per_kwh": 0.18
  }'
```

List chargers:

```bash
curl --request GET "$BASE_URL/api/v1/chargers"
```

Get charger by ID:

```bash
curl --request GET "$BASE_URL/api/v1/chargers/charger-001"
```

### TOU rates

Upsert TOU schedule:

```bash
curl --request PUT "$BASE_URL/api/v1/chargers/charger-001/tou-rates" \
  --header "Content-Type: application/json" \
  --data '{
    "effective_from": "2026-04-01",
    "effective_to": "2026-12-31",
    "periods": [
      { "start_time": "00:00", "end_time": "06:00", "price_per_kwh": 0.15 },
      { "start_time": "06:00", "end_time": "12:00", "price_per_kwh": 0.20 },
      { "start_time": "12:00", "end_time": "14:00", "price_per_kwh": 0.25 },
      { "start_time": "14:00", "end_time": "18:00", "price_per_kwh": 0.30 },
      { "start_time": "18:00", "end_time": "20:00", "price_per_kwh": 0.25 },
      { "start_time": "20:00", "end_time": "22:00", "price_per_kwh": 0.20 },
      { "start_time": "22:00", "end_time": "24:00", "price_per_kwh": 0.15 }
    ]
  }'
```

Get TOU schedule by date:

```bash
curl --request GET "$BASE_URL/api/v1/chargers/charger-001/tou-rates?date=2026-04-15"
```

Get TOU rate by date and time:

```bash
curl --request GET "$BASE_URL/api/v1/chargers/charger-001/tou-rate?date=2026-04-15&time=14:30"
```

### TOU bulk jobs

Create sample CSV:

```bash
cat > tou-bulk.csv <<'EOF'
charger_id,effective_from,effective_to,start_time,end_time,price_per_kwh
charger-001,2026-05-01,2026-12-31,00:00,06:00,0.15
charger-001,2026-05-01,2026-12-31,06:00,12:00,0.20
charger-001,2026-05-01,2026-12-31,12:00,24:00,0.25
charger-002,2026-05-01,,00:00,24:00,0.18
EOF
```

Create async bulk job:

```bash
curl --request POST "$BASE_URL/api/v1/tou-bulk-jobs" \
  --header "Idempotency-Key: job-2026-05-01" \
  --header "X-Submitted-By: ops@acme.com" \
  --form "file=@tou-bulk.csv"
```

Get job status:

```bash
curl --request GET "$BASE_URL/api/v1/tou-bulk-jobs/<job_id>"
```

List all rows for a job:

```bash
curl --request GET "$BASE_URL/api/v1/tou-bulk-jobs/<job_id>/rows"
```

List only failed rows with pagination:

```bash
curl --request GET "$BASE_URL/api/v1/tou-bulk-jobs/<job_id>/rows?status=failed&limit=50&offset=0"
```

### Bulk CSV format

CSV header must match exactly (case-insensitive, same column order):

```text
charger_id,effective_from,effective_to,start_time,end_time,price_per_kwh
```

Rules:
- Required fields per row: `charger_id`, `effective_from`, `start_time`, `end_time`, `price_per_kwh`
- `effective_to` is optional (blank means open-ended)
- `effective_from`/`effective_to` format: `YYYY-MM-DD`
- `start_time`/`end_time` format: `HH:MM` (`24:00` is allowed for `end_time`)
- Rows are grouped by `charger_id + effective_from + effective_to`; each group is applied as one schedule update

### Bulk job statuses

Job status values:
- `queued`
- `processing`
- `completed`
- `completed_with_errors`
- `failed`

Row status values:
- `pending`
- `processed`
- `failed`

### Notes

- The bulk worker starts with the API process and polls for queued jobs continuously.
- `Idempotency-Key` is optional but recommended for safe retries of job creation.
- For `GET /tou-rate`, if no TOU period applies, the API returns the charger's `default_price_per_kwh` with `default_applied: true`.
- Date parsing and TOU applicability are evaluated using each charger's configured IANA timezone.

## Configuration

The app reads configuration from environment variables (or `.env`):

| Variable | Default | Description |
| --- | --- | --- |
| `APP_PORT` | `8080` | HTTP server port |
| `LOG_LEVEL` | `info` | Application log level |
| `DB_HOST` | `localhost` | PostgreSQL host |
| `DB_PORT` | `5432` | PostgreSQL port |
| `DB_USER` | `postgres` | PostgreSQL user |
| `DB_PASS` | `postgres` | PostgreSQL password |
| `DB_NAME` | `appdb` | PostgreSQL database name |
| `DB_SSLMODE` | `disable` | PostgreSQL SSL mode |
| `DB_CONNECT_RETRIES` | `10` | DB connection retry attempts on startup |
| `DB_CONNECT_RETRY_DELAY_SECONDS` | `2` | Delay between DB connection retries |

## Run locally

Prerequisites: Go, Docker, Docker Compose.

```bash
make setup
make db-up
make migrate-up
make dev
```

If `air` is not on your PATH, `make dev` automatically falls back to `go run github.com/air-verse/air@latest`.

## Migration commands

```bash
make migrate-up
make migrate-down
make migrate-status
make migrate-create NAME=add_new_table
```

## Run with Docker

Build and run API + PostgreSQL:

```bash
docker compose up --build -d
```

Check logs:

```bash
docker compose logs -f app
```

Stop:

```bash
docker compose down
```
