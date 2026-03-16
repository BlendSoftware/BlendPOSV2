# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**BlendPOS** is an offline-first Point-of-Sale system for Argentine kiosks and drugstores. It integrates with AFIP/ARCA (Argentine tax authority) for electronic invoicing.

## Services Architecture

```
Frontend (React+Vite :5173) → Backend (Go/Gin :8000) → PostgreSQL :5432
                                        ↓
                                  Redis :6379 (cache + job queue)
                                        ↓
                              AFIP Sidecar (Python/FastAPI :8001)
                                        ↓
                                AFIP/ARCA (external SOAP/XML)
```

- **Backend** (`/backend`): Go 1.24, Gin framework, GORM ORM, golang-migrate
- **Frontend** (`/frontend`): React 19, TypeScript, Vite, Mantine UI, Zustand, Dexie.js (IndexedDB)
- **AFIP Sidecar** (`/afip-sidecar`): Python/FastAPI wrapping pyafipws library for WSAA + WSFEV1

## Development Commands

### Full Stack (Docker)
```bash
docker-compose up -d                              # Start all services (dev, with hot reload)
docker-compose -f docker-compose.prod.yml up -d  # Production
```

### Backend (Go)
```bash
cd backend
go run cmd/server/main.go                         # Run server directly
go build -o ./tmp/blendpos ./cmd/server/main.go   # Build
go test ./tests/...                               # Run all tests
go test ./tests/... -run TestName                 # Run single test
```
Backend tests use `testcontainers-go` (requires Docker) — they spin up real PostgreSQL and Redis instances.

### Frontend
```bash
cd frontend
npm install
npm run dev          # Dev server on :5173
npm run build        # tsc -b && vite build
npm run lint         # ESLint
npm run test         # Vitest (run once)
npm run test:watch   # Vitest watch mode
npm run test:coverage
```

### Database Migrations
```bash
# Migrations run automatically on backend startup via entrypoint.sh
# Manual run:
migrate -path backend/migrations -database "$DATABASE_URL" up
migrate -path backend/migrations -database "$DATABASE_URL" down 1
```

## Backend Architecture

### Layer Structure
```
cmd/server/main.go         → DI wiring, graceful shutdown
internal/config/           → Viper-based env config
internal/infra/            → DB, Redis, AFIP client, circuit breaker, mailer
internal/model/            → GORM models
internal/dto/              → Request/response DTOs
internal/repository/       → Data access (GORM, interface + impl)
internal/service/          → Business logic
internal/handler/          → HTTP handlers (Gin)
internal/middleware/        → Auth, CORS, rate limiting, logging
internal/router/router.go  → Route registration
internal/worker/           → Async workers (invoicing, email, PDF)
migrations/                → SQL migration files (golang-migrate)
```

### Key Patterns
- **Dependency injection**: All dependencies created in `main.go`, injected into handlers/services
- **Repository pattern**: GORM repos behind interfaces — no `AutoMigrate`, only explicit SQL migrations
- **ACID transactions**: Sales use `db.Transaction` (never partial state)
- **Circuit breaker**: All AFIP sidecar calls are wrapped with a circuit breaker
- **Worker pool**: Invoicing, PDF generation, and email are async via goroutine pool + Redis queue
- **No AutoMigrate**: Schema changes go in numbered migration files under `migrations/`

### Middleware Stack (order matters)
MaxBodySize → Gzip → RequestID → Logger → Recovery → CORS → SecurityHeaders → Timeout(30s) → ErrorHandler → RateLimiter → JWT → Audit

### Rate Limiting (Redis-backed, per IP)
- Global: 1000 req/min
- Login: 10/min, Refresh: 20/min
- Price lookup: 60/min

## Frontend Architecture

### Key Directories
```
src/pages/PosTerminal.tsx         → Main POS interface
src/pages/admin/                  → Admin dashboard pages
src/store/                        → Zustand stores (auth, cart, caja, POS UI)
src/offline/db.ts                 → Dexie IndexedDB setup
src/offline/catalog.ts            → Product catalog sync
src/offline/sync.ts               → Offline sale queue (deduplication via offline_id)
src/api/client.ts                 → Axios HTTP client
src/services/api/                 → API service wrappers
```

### Offline-First
Sales can be registered without internet connectivity:
1. Saved to IndexedDB via Dexie.js
2. `sync.ts` queues them with `offline_id` for deduplication
3. `POST /v1/ventas/sync-batch` uploads when connectivity is restored

## AFIP Sidecar

Issues CAE (electronic invoice codes) from AFIP. Called by the backend's `facturacion_service.go` via HTTP.

Key endpoints:
- `GET /health` — includes AFIP connectivity check
- `POST /facturar` — issues invoice, returns `cae` + `cae_vencimiento`

The sidecar requires X.509 certificates in `afip-sidecar/certs/` (gitignored). Set `AFIP_HOMOLOGACION=true` for testing against AFIP's homologation environment.

## Environment Setup

Three `.env` files required (see `.env.example` files in each directory):
- `/backend/.env` — DB, Redis, JWT, AFIP sidecar URL, SMTP
- `/frontend/.env` — `VITE_API_BASE`, `VITE_API_URL`, `VITE_PRINTER_BAUD_RATE`
- `/afip-sidecar/.env` — CUIT, cert paths, homologation flag

Default dev credentials: `admin@blendpos.com` / `1234`

## Database

PostgreSQL 15. Migrations are in `backend/migrations/` as numbered SQL files. The `entrypoint.sh` script runs `migrate ... up` on every container start (idempotent).

Redis is used for: rate limiting, session cache, async job queue. In production, `appendfsync=always` is required to prevent losing sales on crash.
