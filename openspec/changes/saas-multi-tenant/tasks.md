# Tasks: BlendPOS SaaS Multi-Tenant

## Convenciones

- `[BE]` Backend Go | `[FE]` Frontend React | `[PY]` AFIP Sidecar | `[DB]` Migración SQL | `[OPS]` Infraestructura
- Estimaciones en días-desarrollador (1 dev)
- Las tareas dentro de cada fase son paralelas salvo dependencia explícita (→)

---

## Fase 0 — Fundación de datos (sin downtime) · ~2 semanas

### F0-1 [DB] Migración base multi-tenant

**Descripción:** Agregar `tenant_id` a todas las tablas de negocio, crear tabla `tenants`, activar RLS sin políticas.

**Archivo:** `backend/migrations/000025_add_tenant_id.up.sql`

```
Pasos:
1. Crear tabla tenants (id, slug, nombre, plan_id, cuit, activo, created_at)
2. Crear tabla plans (id, nombre, max_terminales, precio_mensual, features JSONB)
3. Agregar tenant_id UUID NULL a: ventas, productos, usuarios, venta_items,
   sesion_cajas, comprobantes, movimiento_cajas, movimientos_stock,
   categorias, proveedores, historial_precios, compras, promociones,
   configuracion_fiscal, audit_log
4. Insertar tenant "legacy" con slug='legacy'
5. UPDATE SET tenant_id = <legacy-uuid> en todas las tablas
6. ALTER COLUMN tenant_id SET NOT NULL en todas las tablas
7. Crear índices compuestos: (tenant_id, created_at), (tenant_id, barcode),
   (tenant_id, offline_id)
8. UNIQUE constraint: ventas(tenant_id, offline_id)
9. ENABLE ROW LEVEL SECURITY en todas las tablas (sin políticas aún)
```

**Criterio de éxito:** Migración ejecuta sin error, datos existentes preservados, RLS activado pero transparente (sin políticas = sin filtrado).

**Dependencias:** Ninguna.

---

### F0-2 [DB] Migración de políticas RLS

**Archivo:** `backend/migrations/000026_rls_policies.up.sql`

**Descripción:** Crear políticas RLS que usan `current_setting('app.tenant_id')`.

```
Para cada tabla de negocio:
  CREATE POLICY tenant_isolation ON <tabla>
    USING (tenant_id = current_setting('app.tenant_id', true)::UUID);
```

> Nota: el segundo argumento `true` en `current_setting` hace que retorne NULL en lugar de error si la variable no está seteada — permite queries de superadmin sin tenant context.

**Criterio de éxito:** Un `SET LOCAL app.tenant_id = 'X'` seguido de `SELECT` retorna solo filas de tenant X.

**Dependencias:** F0-1

---

### F0-3 [BE] Agregar `device_id` al modelo de usuario y JWT

**Archivos:**
- `backend/internal/model/usuario.go` — agregar campo `DeviceID`
- `backend/internal/service/auth_service.go` — agregar claim `did` al JWT
- `backend/internal/middleware/auth.go` — parsear claim `did`

**Migración asociada:** Parte de F0-1 (agregar `device_id` a `usuarios`).

**Criterio de éxito:** JWT emitido contiene claims `tid` (tenant_id) y `did` (device_id). El middleware los parsea y los pone en contexto.

---

### F0-4 [BE] TenantMiddleware + RLS injection

**Archivo:** `backend/internal/middleware/tenant.go` (nuevo)

**Descripción:** Middleware que extrae `tenant_id` del JWT y ejecuta `SET LOCAL app.tenant_id` en la conexión PostgreSQL antes de cada handler.

```
Flujo:
  JWT válido → extraer claims.TenantID
  → context.WithValue(ctx, ctxKeyTenantID, tenantID)
  → db.Exec("SET LOCAL app.tenant_id = ?", tenantID)
  → c.Next()
```

**Criterio de éxito:** Test de integración confirma que un request con JWT de tenant A no puede ver datos de tenant B, incluso con SQL injection en parámetros.

**Dependencias:** F0-3

---

### F0-5 [BE] `scopedDB()` en todos los repositorios

**Descripción:** Agregar helper `scopedDB(ctx)` a todos los repositorios existentes. Asegura que el `tenant_id` se agrega explícitamente a cada query (doble seguridad sobre RLS).

**Archivos afectados:** Todos los archivos en `backend/internal/repository/`

**Criterio de éxito:** Todos los métodos de repositorio pasan el test de tenant isolation (ver F0-4).

**Dependencias:** F0-4

---

### F0-6 [FE] `tenant_id` en Zustand `useAuthStore` + `device_id` persistente

**Archivos:**
- `frontend/src/store/useAuthStore.ts` — agregar `tenantId` al estado
- `frontend/src/offline/sync.ts` — función `getOrCreateDeviceId()` con `localStorage`
- `frontend/src/offline/sync.ts` — actualizar `generateOfflineId()` para incluir `tenantId:deviceId`

**Criterio de éxito:**
- `getOrCreateDeviceId()` retorna el mismo UUID en recargas sucesivas
- `generateOfflineId()` genera IDs del formato `{tid}:{did}:{ts}:{rand}`
- El `offline_id` en ventas locales es único globalmente

**Dependencias:** F0-3 (JWT con claim `tid`)

---

## Fase 1 — Multi-tenant MVP · ~6 semanas

### F1-1 [BE] Sistema de registro de tenants

**Archivos nuevos:**
- `backend/internal/handler/tenants.go`
- `backend/internal/service/tenant_service.go`
- `backend/internal/repository/tenant_repo.go`
- `backend/internal/model/tenant.go`

**Endpoints nuevos:**
```
POST /v1/public/register    → Crear tenant + usuario admin inicial
GET  /v1/admin/tenants      → Listar tenants (superadmin)
GET  /v1/tenant/me          → Info del tenant actual
PUT  /v1/tenant/me          → Actualizar datos del tenant
```

**Criterio de éxito:**
- Un nuevo comercio puede registrarse y recibir JWT válido en < 10s
- El tenant creado tiene plan "Starter" asignado automáticamente
- El usuario admin inicial tiene rol `admin` y `device_id` generado

---

### F1-2 [BE] Plan enforcement middleware

**Archivo:** `backend/internal/middleware/plan.go` (nuevo)

**Descripción:** Middleware que verifica límites del plan antes de operaciones costosas.

```
Ejemplos de limits:
  - max_terminales: bloquear login desde terminal N+1
  - max_productos: bloquear creación de producto N+1 (solo planes básicos)
```

**Criterio de éxito:** Request de tenant en plan "Kiosco" intentando abrir 4ª terminal recibe `403` con mensaje claro.

---

### F1-3 [BE] Configuración fiscal por tenant

**Descripción:** Adaptar `configuracion_fiscal` para que cada tenant cargue su propio CUIT y certificados AFIP.

**Archivos:**
- `backend/internal/handler/configuracion_fiscal.go` — ya existe, adaptar para multi-tenant
- `backend/internal/service/configuracion_fiscal_service.go` — idem
- `backend/internal/model/configuracion_fiscal.go` — asegurar tenant_id

**Criterio de éxito:** Dos tenants con distintos CUIT pueden facturar simultáneamente sin interferencia.

**Dependencias:** F0-1, F0-5

---

### F1-4 [PY] AFIP Sidecar stateless multi-CUIT

**Descripción:** Refactorizar `afip_client.py` para que el CUIT y paths de certificados vengan en cada request (no en variables de entorno globales).

**Archivos:**
- `afip-sidecar/afip_client.py` — recibir CUIT + cert_path por request
- `afip-sidecar/schemas.py` — agregar `cuit_emisor`, `cert_path`, `key_path` al request schema
- `afip-sidecar/main.py` — instanciar `AFIPClient` por request (stateless)

**Criterio de éxito:** POST `/facturar` con CUIT_A y CUIT_B en requests paralelos retorna CAEs distintos correctamente.

**Dependencias:** F1-3

---

### F1-5 [BE] Billing service básico

**Descripción:** Integración con MercadoPago (mercado local) para suscripciones.

**Archivos nuevos:**
- `backend/internal/service/billing_service.go`
- `backend/internal/handler/billing.go`
- `backend/migrations/000027_billing.up.sql`

**Endpoints:**
```
POST /v1/billing/subscribe      → Crear suscripción (redirige a MP Checkout)
POST /v1/billing/webhook        → Webhook de MercadoPago (actualiza plan)
GET  /v1/billing/status         → Estado de suscripción del tenant
```

**Criterio de éxito:** Un webhook de pago exitoso de MercadoPago activa el plan correspondiente en < 5s.

---

### F1-6 [FE] Onboarding flow

**Archivos nuevos:**
- `frontend/src/pages/OnboardingPage.tsx`
- `frontend/src/pages/RegisterPage.tsx`

**Pasos del onboarding:**
1. Registro (email, nombre del negocio, contraseña)
2. Configuración fiscal (CUIT, punto de venta, certificados AFIP)
3. Carga inicial del catálogo (CSV import o manual)
4. Apertura de primera caja

**Criterio de éxito:** Un nuevo usuario pasa de registro a primera venta en < 10 minutos.

**Dependencias:** F1-1

---

### F1-7 [FE] Indicador de sync en tiempo real

**Descripción:** `PosHeader.tsx` muestra estado de sincronización con indicador visual (verde/amarillo/rojo) y contador de ventas pendientes.

**Archivo:** `frontend/src/components/pos/PosHeader.tsx`

**Lógica:**
```
Verde:    todas las ventas sincronizadas, online
Amarillo: ventas pendientes de sync, online (sync en progreso)
Rojo:     offline o sync fallando, N ventas acumuladas
```

**Criterio de éxito:** El estado cambia en < 2s ante cambios de conectividad.

---

### F1-8 [BE] Endpoint sync-batch multi-tenant

**Descripción:** `POST /v1/ventas/sync-batch` ya existe. Adaptarlo para:
1. Inyectar `tenant_id` desde JWT (no del body)
2. Usar `INSERT ON CONFLICT (tenant_id, offline_id) DO NOTHING` para idempotencia
3. Retornar lista de `offline_id` procesados (para que el frontend marque como sincronizados)

**Archivo:** `backend/internal/handler/ventas.go`

**Criterio de éxito:**
- 500 ventas en batch procesan en < 3 segundos
- Reenvío del mismo batch retorna `200 OK` sin duplicados
- El stock se descuenta correctamente (incluso si queda negativo — ver RN-S2)

---

### F1-9 [BE] Analytics con réplica de lectura

**Descripción:** Agregar soporte para read replica en la configuración de base de datos. Los endpoints de analytics/reportes usarán la réplica.

**Archivos:**
- `backend/internal/infra/database.go` — agregar `DBReadReplica *gorm.DB`
- `backend/internal/config/config.go` — agregar `DatabaseReadReplicaURL`
- `backend/internal/handler/` — handlers de reportes usan `deps.DBRead` en lugar de `deps.DB`

**Criterio de éxito:** Query de ventas de los últimos 12 meses para tenant con 500k registros responde en < 2s.

> En Fase 1, si no hay réplica configurada, `DBReadReplica` apunta al mismo DB primario. El código ya está preparado para la separación.

---

### F1-10 [BE+FE] Panel superadmin

**Descripción:** Panel interno (ruta `/superadmin`) para gestión de tenants.

**Endpoints:**
```
GET  /v1/superadmin/tenants          → Listar todos los tenants
PUT  /v1/superadmin/tenants/:id/plan → Cambiar plan
PUT  /v1/superadmin/tenants/:id      → Activar/desactivar
GET  /v1/superadmin/metrics          → Métricas globales
```

**Middleware:** Rol `superadmin` verificado antes de cualquier ruta `/v1/superadmin/*`.

---

## Fase 2 — Hardening y performance · ~4 semanas

### F2-1 [BE] Tests de tenant isolation

**Descripción:** Suite de tests e2e que verifican que ningún tenant puede ver datos de otro.

**Archivo:** `backend/tests/e2e/tenant_isolation_test.go`

```
Tests obligatorios:
  - Tenant A no puede listar ventas de Tenant B
  - sync-batch con JWT de Tenant A no registra ventas en Tenant B
  - Tenant A no puede ver productos de Tenant B
  - SQL injection en offline_id no rompe el aislamiento
```

**Criterio de éxito:** 100% de los tests de isolation pasan.

---

### F2-2 [BE] Rate limiting por tenant (además de por IP)

**Descripción:** Agregar rate limiting por `tenant_id` en Redis para evitar que un tenant abuse del sistema.

**Archivo:** `backend/internal/middleware/rate_limiter.go`

**Límites por tenant:**
```
POST /v1/ventas:           500 req/min por tenant (pico de kiosco)
POST /v1/ventas/sync-batch: 10 req/min por tenant
POST /v1/facturar:          60 req/min por tenant
GET  /v1/productos:         200 req/min por tenant
```

---

### F2-3 [FE] PWA: catálogo offline ilimitado

**Descripción:** Asegurar que el catálogo completo se descarga en IndexedDB al iniciar la sesión, con delta sync cada 5 minutos en background.

**Archivo:** `frontend/src/offline/catalog.ts`

```
Flujo:
  Apertura de sesión → GET /v1/productos?since={catalog_version}
  → Upsert en IndexedDB (solo productos modificados)
  → Actualizar catalog_version en localStorage
  → Cada 5 min: delta sync si hay conectividad
```

**Criterio de éxito:**
- Catálogo de 50.000 productos se descarga en < 30s en primera carga
- Delta sync de 100 productos modificados en < 2s
- Búsqueda en catálogo local por barcode en < 50ms

---

### F2-4 [OPS] Monitoreo básico

**Descripción:** Métricas de negocio expuestas en endpoint `/metrics` (formato Prometheus).

**Métricas clave:**
```
blendpos_ventas_total{tenant_id, estado}
blendpos_sync_queue_depth{tenant_id}
blendpos_afip_errors_total{tenant_id, tipo}
blendpos_active_sessions{tenant_id}
blendpos_catalog_version{tenant_id}
```

---

## Resumen de entregables por fase

| Fase | Duración | Entregables clave |
|------|----------|------------------|
| **F0** | 2 semanas | DB multi-tenant, TenantMiddleware, offline_id mejorado |
| **F1** | 6 semanas | Registro, billing, onboarding, sync adaptado, analytics |
| **F2** | 4 semanas | Tests de isolation, rate limiting por tenant, PWA hardening |

**Total estimado: 12 semanas (3 meses)** para MVP SaaS completo con 1 desarrollador full-stack.

---

## Orden de implementación recomendado

```
F0-1 (DB) → F0-2 (RLS) → F0-3 (JWT) → F0-4 (Middleware)
                                              ↓
                                     F0-5 (scopedDB) → F0-6 (FE offline_id)
                                              ↓
                                     F1-1 (registro) → F1-2 (plan enforcement)
                                              ↓
                      F1-3 (cfg fiscal) → F1-4 (sidecar) → F1-5 (billing)
                                              ↓
                              F1-6 (onboarding) + F1-7 (sync indicator)
                                              ↓
                              F1-8 (sync-batch) + F1-9 (read replica)
                                              ↓
                                     F1-10 (superadmin)
                                              ↓
                              F2-1 (isolation tests) + F2-2 (rate limits)
                                              ↓
                                     F2-3 (PWA) + F2-4 (monitoring)
```
