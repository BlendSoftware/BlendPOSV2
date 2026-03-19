# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

**Project:** BlendPOS — Offline-first POS for Argentine kiosks. Spec-Driven Development (OpenSpec) workflow.

---

## Stack & Services

```
Frontend: React 19 + TypeScript + Zustand + Dexie.js (IndexedDB) + Mantine UI + react-router-dom v7
Backend:  Go 1.24 + Gin + GORM + PostgreSQL + Redis + Viper config
Sidecar:  Python/FastAPI wrapping pyafipws (AFIP invoicing) — port 8001
PWA:      vite-plugin-pwa (injectManifest strategy, service worker at src/sw.ts)
```

### Key Dependencies
**Frontend:** `@mantine/core`, `zustand`, `dexie`, `axios`, `react-router-dom`, `lucide-react`, `vite-plugin-pwa`
**Backend:** `gorm.io/gorm`, `gin-gonic/gin`, `redis/go-redis`, `golang-jwt/jwt`, `shopspring/decimal`

---

## Build, Test & Lint Commands

### Frontend (`cd frontend`)
```bash
npm run dev              # Vite dev server on :5173 (HMR with polling for WSL/Docker)
npm run build            # tsc -b && vite build → dist/
npm run lint             # ESLint (flat config: TS + React rules)
npm run test             # vitest run (single pass)
npm run test:watch       # vitest watch mode
npm run test:coverage    # vitest with coverage
```

### Backend (`cd backend`)
```bash
go run cmd/server/main.go                      # Start server
air -c .air.toml                               # Hot reload (requires: go install github.com/air-verse/air@latest)
go build -o ./blendpos ./cmd/server/main.go    # Production binary
go test ./...                                  # Unit tests
go test -tags integration ./tests/... -v       # Integration tests (real Postgres 15 + Redis 7 via testcontainers)
go test -tags integration -run TestVentas ./tests/... -v  # Single test
```

### Migrations
```bash
# Auto-runs in Docker via entrypoint
migrate -path backend/migrations -database "$DATABASE_URL" up    # Manual (golang-migrate CLI)
migrate -path backend/migrations -database "$DATABASE_URL" down 1 # Rollback one
```
28 numbered SQL files in `backend/migrations/`. Exit code 1 from migrate = "already up to date" (not an error).

### Full Stack
```bash
docker-compose up -d          # All services (frontend, backend, postgres, redis, afip-sidecar)
docker-compose up -d backend  # Backend only with deps
```

Health check endpoint: `GET /health`

---

## Current Change: SaaS Multi-Tenant

**Status:** in-progress | **Scope:** Architecture + DB + Auth foundation
**Goal:** Evolve from single-tenant to multi-tenant SaaS with offline-first, analytics-first positioning.

### Implementation Phases (from design.md)
1. **Phase 0:** SQL foundation — `tenant_id` on all tables, `tenants` table, RLS ✅
2. **Phase 1:** Tenant auth + JWT extraction (in progress)
3. **Phase 2:** Frontend PWA tenant context + offline sync per tenant
4. **Phase 3:** Analytics (reportes completos desde plan gratuito)
5. **Phase 4+:** Billing + subscription tiers

**Design context:** stored in engram persistent memory (search `sdd/saas-multi-tenant/*`)

---

## Backend Architecture (Go)

### DI & Composition
All deps wired in `cmd/server/main.go` → injected into `router.New(deps)` via `router.Deps` struct. No DI framework — manual constructor injection.

### Middleware Chain (order matters in `router.go`)
1. MaxBodySize(10MB)
2. Gzip compression
3. RequestID — unique per request (audit trail)
4. Logger — zerolog structured logging (uses request ID)
5. Recovery — panic recovery
6. CORS — parsed from `ALLOWED_ORIGINS` env (comma-separated)
7. **Per-route:** tenant extraction (JWT), plan guards, IDOR guard, tenant audit

RequestID MUST come before Logger for audit trail correlation.

### Layer Responsibilities
- **Handlers** (`internal/handler/`) — HTTP semantics, DTO marshaling, auth checks
- **Services** (`internal/service/`) — business logic, transactions, worker dispatch
- **Repositories** (`internal/repository/`) — queries only; mutations through services
- **Middleware** (`internal/middleware/`) — tenant extraction, plan guards, IDOR prevention, audit
- **Workers** (`internal/worker/`) — invoicing, email, PDF (async via goroutine pool + Redis)

### Config (Viper)
All env vars with defaults in `internal/config/config.go`. Notable:
- `DATABASE_READ_REPLICA_URL` — optional read replica for analytics
- `WORKER_POOL_SIZE`, `FACTURACION_WORKERS`, `EMAIL_WORKERS` — worker sizing (0 = fallback to pool)
- `JWT_SECRET` — default is insecure; MUST override in production
- `INTERNAL_API_TOKEN` — shared secret between Go backend and AFIP sidecar

### Patterns
- **Interface + impl** — `repository.ISalesRepo` + `repository.SalesRepo{}`
- **No AutoMigrate** — numbered SQL migration files only
- **Transactions for consistency** — sales never partial state; use `db.Transaction`
- **Decimal arithmetic always** — `shopspring/decimal` for money, never float64
- **Tenant context** — extracted from JWT in middleware, stored in `ctx`, explicit in WHERE + RLS belt-and-suspenders

### Error Handling
```go
// Always: structured log + proper HTTP status + tenant context
logger.Error("operation failed", zap.Error(err), zap.String("tenant_id", tenantID))
c.AbortWithStatusJSON(getHTTPStatus(err), ErrorResponse{Message: err.Error()})
```

---

## Frontend Architecture

### State Management (Zustand)
- **Cart store** (`src/store/`): items, totals, tenant context
- **Caja store** — session state, cash drawer, daily balance
- **Auth store** — JWT, tenant ID, user role, expiry
- **UI store** — modal visible/hidden, current screen
- One store per domain. Prefer multiple small stores over one mega store.

### Offline Pattern
- **Dexie.js** — IndexedDB for catalog, sales draft, sync queue
- **Sync queue** — `src/offline/sync.ts` uses `offline_id` (UUID) for deduplication
- **Delete resilience** — write deletes to sync queue first, then local DB

### Component Structure (Atomic + Container)
```
src/pages/PosTerminal.tsx          ← Page + container logic
src/components/Drawer/
  ├── index.tsx                    ← Container (hooks, state)
  ├── Drawer.container.tsx         ← Pure component (props only)
  └── Drawer.styles.ts             ← Styles export
```

### TypeScript Rules
- **No `any`** — use `unknown`, then narrow with type guards or `as const`
- **Precise types** — domain models with discriminated unions
- **Zustand selectors** — use `useShallow()` to prevent recreation

---

## Testing

### Frontend (vitest + jsdom)
- Environment: jsdom with globals enabled
- Setup: `src/test/setup.ts` (polyfills for Mantine: `matchMedia`, `ResizeObserver`, `fake-indexeddb`)
- Mock API: `vi.mock('src/api/client')`
- **No snapshots** — test behavior not markup

### Backend (testcontainers-go)
- Tag: `//go:build integration`
- Creates real Postgres 15 + Redis 7 containers per test suite
- Helpers in `tests/e2e/e2e_test.go`: `jsonBody()`, `do()` for HTTP requests
- Integration tests take ~30s (container startup)
- Transaction rollback ensures no test pollution

---

## Code Review (GGA)

Code review via Gentleman Guardian Angel. Rules in `AGENTS.md`.
```bash
gga review --pr {PR_NUMBER}   # Review PR
gga review src/                # Review folder
```

---

## Common Gotchas

| Issue | Fix |
|-------|-----|
| Float64 for money | Use `decimal.Decimal` always |
| Forgot `tenant_id` in WHERE | RLS catches it (audit log fires), but explicit filter > implicit |
| Zustand selector recreation | Use `useShallow()` or memoized selector |
| Dexie query without index | Add to schema; unindexed = full-table scan |
| RLS not enabled on table | Feature won't work; `ALTER TABLE ... ENABLE ROW LEVEL SECURITY` |
| Missing `offline_id` | Sync retries duplicate sales; always generate UUID client-side |
| Migration exit code 1 | Means "already up to date", not an error |

---

## Where Things Live

| What | Where |
|------|-------|
| Frontend entry | `src/pages/PosTerminal.tsx` + `src/pages/admin/` |
| Zustand stores | `src/store/{auth,cart,caja,ui}.ts` |
| Offline subsystem | `src/offline/` (db.ts, sync.ts, catalog.ts) |
| Backend handlers | `internal/handler/` |
| Services + repos | `internal/service/`, `internal/repository/` |
| Middleware | `internal/middleware/` (tenant, plan, idor_guard, tenant_audit) |
| Workers | `internal/worker/` (facturacion, retry_cron) |
| Migrations | `backend/migrations/` (28 numbered SQL files) |
| Config | Backend: `internal/config/config.go` + `.env`, Frontend: `.env` + `vite.config.ts` |
| Code review rules | `AGENTS.md` (GGA) |

---

## Principles

1. **Spec-driven** — propose → design → tasks → code; not code-first
2. **Offline-first** — transaction completes locally first, syncs async
3. **Type-safe** — TypeScript strict mode; Go panic only on bugs, never user data
4. **Audit trail** — every tenant operation logged (tenant_id + request_id + user)
5. **Performance** — sub-100ms local txns (IndexedDB); assume 300ms+ remote
6. **Testing** — real DB containers, no mocks on infrastructure; behavior > snapshots
7. **Tenant isolation is SECURITY** — every query needs `tenant_id`, not optional
