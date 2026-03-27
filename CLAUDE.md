# CLAUDE.md — BlendPOS Agent Guidance

**Project:** BlendPOS — Offline-first POS for Argentine kiosks. Spec-Driven Development (OpenSpec) workflow.

---

## Stack & Services

```
Frontend: React 19 + TypeScript + Zustand + Dexie.js (IndexedDB) + Mantine UI
Backend:  Go 1.24 + Gin + GORM + PostgreSQL + Redis
Sidecar:  Python/FastAPI wrapping pyafipws (AFIP invoicing)
```

### Key Dependencies
**Frontend:** `@mantine/core`, `zustand`, `dexie`, `axios`, `react-router-dom`, `lucide-react`  
**Backend:** `gorm.io/gorm`, `gin-gonic/gin`, `redis/go-redis`, `golang-jwt/jwt`, `shopspring/decimal`

---

## Current Change: SaaS Multi-Tenant

**Status:** in-progress | **Scope:** Architecture + DB + Auth foundation  
**Goal:** Evolve from single-tenant to multi-tenant SaaS with offline-first, analytics-first positioning.

### What's Being Built
- **Tenant isolation:** PostgreSQL RLS (Row Level Security) + `tenant_id` on all tables
- **Offline resilience:** 48h autonomy without internet; catalog + AFIP invoicing in IndexedDB
- **Velocity:** Transactions sub-100ms (p99) in local IndexedDB

### Implementation Phases (from design.md)
1. **Phase 0:** SQL foundation — add `tenant_id`, create `tenants` table, enable RLS
2. **Phase 1:** Tenant auth + JWT extraction
3. **Phase 2:** Frontend PWA tenant context + offline sync per tenant
4. **Phase 3:** Analytics (reportes completos desde plan gratuito)
5. **Phase 4+:** Billing + subscription tiers

**Active artifacts:** `openspec/changes/saas-multi-tenant/{proposal.md, design.md, tasks.md}`

---

## Frontend Rules

### TypeScript Strictness
- **No `any`** — use `unknown`, then narrow with type guards or `as const`
- **Precise types** — model domain (Sale, CartItem, Tenant) with discriminated unions
- **Zustand stores** — one store per domain, typed selectors, no middleware excess

### Offline Pattern
- **Dexie.js** — IndexedDB for catalog, sales draft, queue
- **Sync queue** — `src/offline/sync.ts` uses `offline_id` (UUID) for deduplication
- **Delete resilience** — write deletes to sync queue first, then local DB

### State Management
- **Cart store** (`src/store/`): items, totals, tenant context
- **Caja store** — session state, cash drawer, daily balance
- **Auth store** — JWT, tenant ID, user role, expiry
- **UI store** — modal visible/hidden, current screen
- **No Redux** — Zustand is the pattern; prefer multiple small stores over one mega store

### Component Structure (Atomic + Container)
```
src/pages/PosTerminal.tsx          ← Page + container logic
src/components/Drawer/
  ├── index.tsx                    ← Container (hooks, state)
  ├── Drawer.container.tsx         ← Pure component (props only)
  └── Drawer.styles.ts             ← Styles export
```

### Testing
- Components: `vitest` + `@testing-library/react` (user-centric)
- Stores: Test selectors + reducers independently
- API: Mock via `vi.mock('src/api/client')`
- **No snapshots** for components; test behavior not markup

---

## Backend Rules (Go)

### Architecture
- **DI only in `main.go`** — all deps wired there, injected into handlers/services
- **Interface + impl pattern** — `repository.ISalesRepo` + `repository.SalesRepo{}`
- **No AutoMigrate** — migrations in numbered SQL files under `migrations/`
- **Transactions for consistency** — sales never partial state; use `db.Transaction`

### Tenant Context
- Extract `tenant_id` from JWT in middleware (coming from frontend)
- Store in context: `ctx = context.WithValue(ctx, tenantCtxKey, tenantID)`
- Use in repos: filter by `tenant_id` (RLS + explicit in queries for safety)

### Error Handling
```go
// ❌ Bad
if err != nil {
    c.JSON(400, "error")
}

// ✅ Good
if err != nil {
    logger.Error("operation failed", zap.Error(err), zap.String("tenant_id", tenantID))
    c.AbortWithStatusJSON(getHTTPStatus(err), ErrorResponse{Message: err.Error()})
}
```

### Patterns
- **Repositories** — query only; mutations go through services
- **Services** — business logic, transactions, worker dispatch
- **Handlers** — HTTP semantics + DTO marshaling + auth checks
- **Middleware** — tenant extraction, rate limit, audit logging
- **Workers** — invoicing, email, PDF (async via goroutine pool + Redis queue)

### Decimal Arithmetic (Always)
```go
// ❌ Never
price := 99.99 // float64

// ✅ Always
price := decimal.NewFromString("99.99")
```

### Testing
- Use `testcontainers-go` — spin real PostgreSQL + Redis
- Test transactions — rollback ensures no side effects
- Mock AFIP sidecar circuit breaker

---

## Code Review Checklist (Pre-Commit)

### Frontend (TypeScript/React)
- [ ] No `any` types — specify or narrow
- [ ] Zustand store changes documented in comment (selector, mutation signature)
- [ ] Offline sync logic: tested deduplication, no duplicate `offline_id`
- [ ] Dexie indexes: added if querying by non-PK field
- [ ] Component tests: behavior not markup; no snapshots

### Backend (Go)
- [ ] Tenant context extracted and threaded through request
- [ ] `tenant_id` in WHERE clauses (explicit + RLS belt-and-suspenders)
- [ ] Decimals for all currency arithmetic
- [ ] No SQL injection — parameterized queries only
- [ ] Error logs: include tenant_id + request ID for audit trail

### Shared
- [ ] ✅ No hardcoded env vars — use viper (Go) / import.meta.env (TypeScript)
- [ ] ✅ Migrations + schema changes: SQL files per Go Backend Specs
- [ ] ✅ API contracts: DTOs defined first, handlers consume them
- [ ] ✅ Tests added/updated; coverage not dropped

---

## Common Gotchas

| Issue | Fix |
|-------|-----|
| **Float64 for money** | Use `decimal.Decimal` always |
| **Forgot `tenant_id` in WHERE** | RLS catches it (audit log fires), but explicit filter > implicit |
| **Zustand selector recreation** | Use `useShallow()` or memoized selector |
| **Dexie query without index** | Add to schema or use a full-table scan (log perf warning) |
| **RLS not enabled on table** | Feature won't work; ALTER TABLE ... ENABLE ROW LEVEL SECURITY |
| **Missing offline_id** | Sync retries duplicate sales; always generate UUID client-side |

---

## Workflow Commands

### Spec-Driven Development
```bash
# Explore feature / clarify requirements
pnpm exec openspec explore upgrade-billing

# Propose change (intent + scope)
pnpm exec openspec propose "add subscription tiers"

# Write specs (requirements + scenarios)
pnpm exec openspec spec saas-multi-tenant

# Design (architecture + decisions)
pnpm exec openspec design saas-multi-tenant

# Break into tasks
pnpm exec openspec tasks saas-multi-tenant

# Implement (apply tasks)
pnpm exec openspec apply saas-multi-tenant --task "Phase 0: SQL Foundation"

# Verify (test + code review)
pnpm exec openspec verify saas-multi-tenant

# Archive (move to done)
pnpm exec openspec archive saas-multi-tenant
```

### Dev Stack
```bash
# Full stack
docker-compose up -d

# Frontend only
cd frontend && npm run dev

# Backend only
cd backend && go run cmd/server/main.go

# Migrations
migrate -path backend/migrations -database "$DATABASE_URL" up

# Backend tests (uses testcontainers)
cd backend && go test ./tests/... -v
```

---

## Where Things Live

| What | Where |
|------|-------|
| Frontend entry | `src/pages/PosTerminal.tsx` + `src/pages/admin/` |
| Zustand stores | `src/store/{auth,cart,caja,ui}.ts` |
| Offline subsystem | `src/offline/` (db.ts, sync.ts, catalog.ts) |
| Backend handlers | `internal/handler/` |
| Services + repos | `internal/service/`, `internal/repository/` |
| Migrations | `backend/migrations/` (numbered SQL files) |
| Spec artifacts | `openspec/changes/saas-multi-tenant/` |
| Config (env) | Backend: `.env` + `internal/config/`, Frontend: `.env` + `vite.config.ts` |

---

## Principles

1. **Spec-driven** — propose → design → tasks → code; not code-first
2. **Offline-first** — transaction completes locally first, syncs async
3. **Type-safe** — TypeScript strict mode; Go panic only on bugs, never user data
4. **Audit trail** — every tenant operation logged (tenant_id + request_id + user)
5. **Performance** — sub-100ms local txns (IndexedDB); assume 300ms+ remote
6. **Testing** — real DB containers, no mocks on infrastructure; behavior > snapshots
