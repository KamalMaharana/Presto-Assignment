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

### Why this schema
- Charger-specific rates are stored directly by `charger_id`.
- Daily TOU windows are represented as minute ranges for fast lookups.
- `effective_from`/`effective_to` supports schedule changes over time.
- Normalized model avoids duplicating charger metadata in pricing rows.

## API specification

Base path: `/api/v1`

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

### 1) Create charger
`POST /chargers`

Request:
```json
{
  "id": "charger-001",
  "name": "Depot Charger A",
  "location": "Seattle Depot",
  "timezone": "America/Los_Angeles"
}
```

Response `201`:
```json
{
  "id": "charger-001",
  "name": "Depot Charger A",
  "location": "Seattle Depot",
  "timezone": "America/Los_Angeles",
  "created_at": "2026-03-31T13:00:00Z",
  "updated_at": "2026-03-31T13:00:00Z"
}
```

### 2) Upsert TOU schedule for a charger
`PUT /chargers/:charger_id/tou-rates`

Request:
```json
{
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
}
```

Validation:
- periods must be contiguous, non-overlapping
- first period starts `00:00`
- last period ends `24:00`
- `price_per_kwh > 0`

Response `200`:
```json
{
  "message": "tou schedule updated",
  "request_id": "..."
}
```

### 3) Get full TOU schedule for charger + date
`GET /chargers/:charger_id/tou-rates?date=2026-04-15`

Response `200`:
```json
{
  "charger_id": "charger-001",
  "timezone": "America/Los_Angeles",
  "effective_from": "2026-04-01",
  "effective_to": "2026-12-31",
  "periods": [
    { "start_time": "00:00", "end_time": "06:00", "price_per_kwh": 0.15 },
    { "start_time": "06:00", "end_time": "12:00", "price_per_kwh": 0.20 }
  ]
}
```

### 4) Get TOU rate for charger + date + time
`GET /chargers/:charger_id/tou-rate?date=2026-04-15&time=14:30`

Response `200`:
```json
{
  "charger_id": "charger-001",
  "timezone": "America/Los_Angeles",
  "date": "2026-04-15",
  "time": "14:30",
  "price_per_kwh": 0.30,
  "period_start": "14:00",
  "period_end": "18:00",
  "effective_from": "2026-04-01"
}
```

### Error response shape
```json
{
  "error": "human-readable message",
  "request_id": "..."
}
```

## Optional considerations

### Time zones
- Each charger stores an IANA timezone.
- Date-based schedule selection is resolved in the charger's timezone.
- API responses include the timezone used for interpretation.

### Bulk updates
- Recommended endpoint design: `PUT /chargers/tou-rates/bulk`
- Payload includes an array of `{ charger_id, effective_from, periods }`.
- Run update per charger within a DB transaction chunk for atomicity and retry safety.

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
