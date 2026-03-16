# Diseño Técnico: BlendPOS SaaS Multi-Tenant

## 1. Arquitectura actual vs. arquitectura objetivo

### Estado actual (single-tenant)

```
┌─────────────────────────────────────────────────────────────┐
│                     SINGLE TENANT                           │
│                                                             │
│   React :5173  ──→  Go/Gin :8000  ──→  PostgreSQL (1 DB)   │
│                           │                                 │
│                      Redis :6379                            │
│                           │                                 │
│                   AFIP Sidecar :8001                        │
└─────────────────────────────────────────────────────────────┘
```

Un único conjunto de tablas, sin aislamiento entre comercios. El `usuario_id` es la única frontera de datos.

### Estado objetivo (multi-tenant, Fase 1)

```
┌─────────────────────────────────────────────────────────────────────┐
│                    MULTI-TENANT (shared database, isolated schemas) │
│                                                                     │
│  Tenant A (kiosco_gomez)          Tenant B (despensa_norte)        │
│  ┌──────────────────────┐         ┌──────────────────────┐         │
│  │ React PWA (device_1) │         │ React PWA (device_1) │         │
│  │ React PWA (device_2) │         └──────────┬───────────┘         │
│  └──────────┬───────────┘                    │                     │
│             │                                │                     │
│             └──────────────┬─────────────────┘                     │
│                            ▼                                        │
│                   Go/Gin :8000                                      │
│              ┌─────────────────────┐                               │
│              │  TenantMiddleware   │ ← extrae tenant_id del JWT    │
│              │  + RLS PostgreSQL   │ ← filtra filas automáticamente│
│              └─────────────────────┘                               │
│                            │                                        │
│                   PostgreSQL (shared DB)                            │
│              ┌─────────────────────────────┐                       │
│              │  schema: public             │                       │
│              │  tenants, plans, billing    │                       │
│              ├─────────────────────────────┤                       │
│              │  tenant_id column en TODAS  │                       │
│              │  las tablas de negocio      │                       │
│              │  + Row Level Security       │                       │
│              └─────────────────────────────┘                       │
│                            │                                        │
│                    Redis (namespaced por tenant)                    │
│                    AFIP Sidecar (stateless, multi-CUIT)            │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 2. Estrategia de aislamiento de tenants: decisión y trade-offs

### Opciones evaluadas

| Estrategia | Aislamiento | Complejidad ops | Costo infra | Escala objetivo |
|-----------|------------|----------------|-------------|----------------|
| **DB separada por tenant** | Máximo | Muy alta (N migraciones) | Alto | >100k tenants |
| **Schema separado por tenant** | Alto | Alta (schema per migration) | Medio | >10k tenants |
| **tenant_id + RLS (elegida)** | Medio-alto | Baja (1 migración) | Bajo | <10k tenants ✓ |

### Decisión: `tenant_id` + PostgreSQL Row Level Security (RLS)

**Justificación:**
- El target para 18 meses es < 10.000 tenants. A este volumen, una DB compartida con RLS es operativamente simple y más que suficiente.
- Una sola pasada de migraciones. Un único backup. Un único pool de conexiones.
- RLS en PostgreSQL garantiza que incluso un bug en el código Go no exponga datos entre tenants — es una segunda línea de defensa a nivel DB.
- El gatillo para pasar a DB separada por tenant es **> 50.000 tenants activos** o cuando un tenant individual supere 1M de registros en `ventas`.

### Implementación de RLS

```sql
-- Migration 000025_add_tenant_id.up.sql

-- 1. Agregar tenant_id a todas las tablas de negocio
ALTER TABLE ventas         ADD COLUMN tenant_id UUID NOT NULL;
ALTER TABLE productos      ADD COLUMN tenant_id UUID NOT NULL;
ALTER TABLE venta_items    ADD COLUMN tenant_id UUID NOT NULL;
ALTER TABLE sesion_cajas   ADD COLUMN tenant_id UUID NOT NULL;
ALTER TABLE comprobantes   ADD COLUMN tenant_id UUID NOT NULL;
ALTER TABLE movimiento_cajas ADD COLUMN tenant_id UUID NOT NULL;
ALTER TABLE movimientos_stock ADD COLUMN tenant_id UUID NOT NULL;
ALTER TABLE usuarios       ADD COLUMN tenant_id UUID NOT NULL;
-- (y el resto de tablas de negocio)

-- 2. Tabla maestra de tenants
CREATE TABLE tenants (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug        VARCHAR(63) UNIQUE NOT NULL,  -- "kiosco-gomez"
    nombre      VARCHAR(255) NOT NULL,
    plan_id     UUID REFERENCES plans(id),
    cuit        VARCHAR(13),
    activo      BOOLEAN DEFAULT true,
    created_at  TIMESTAMPTZ DEFAULT NOW()
);

-- 3. Activar RLS en tablas de negocio
ALTER TABLE ventas ENABLE ROW LEVEL SECURITY;
ALTER TABLE productos ENABLE ROW LEVEL SECURITY;
-- (idem para todas)

-- 4. Política: solo ver filas del propio tenant
-- El tenant_id se inyecta vía SET LOCAL en cada request
CREATE POLICY tenant_isolation ON ventas
    USING (tenant_id = current_setting('app.tenant_id')::UUID);

CREATE POLICY tenant_isolation ON productos
    USING (tenant_id = current_setting('app.tenant_id')::UUID);
-- (idem para todas las tablas)

-- 5. Índices compuestos obligatorios (performance crítica)
CREATE INDEX idx_ventas_tenant_created ON ventas (tenant_id, created_at DESC);
CREATE INDEX idx_productos_tenant_barcode ON productos (tenant_id, barcode);
CREATE INDEX idx_ventas_tenant_offline_id ON ventas (tenant_id, offline_id);
```

---

## 3. Inyección del contexto de tenant en Go

### TenantMiddleware

```go
// internal/middleware/tenant.go

func TenantMiddleware(db *gorm.DB) gin.HandlerFunc {
    return func(c *gin.Context) {
        // El tenant_id viene en el JWT (claim "tid")
        claims, ok := c.MustGet("claims").(*JWTClaims)
        if !ok || claims.TenantID == uuid.Nil {
            c.AbortWithStatusJSON(http.StatusUnauthorized,
                gin.H{"error": "tenant context missing"})
            return
        }

        tenantID := claims.TenantID

        // Inyectar en contexto Go para uso en servicios/repos
        ctx := context.WithValue(c.Request.Context(),
            ctxKeyTenantID, tenantID)
        c.Request = c.Request.WithContext(ctx)

        // Inyectar en sesión PostgreSQL para RLS
        // SET LOCAL aplica solo a la transacción/query actual
        if err := db.WithContext(ctx).Exec(
            "SET LOCAL app.tenant_id = ?", tenantID.String(),
        ).Error; err != nil {
            c.AbortWithStatusJSON(http.StatusInternalServerError,
                gin.H{"error": "tenant context injection failed"})
            return
        }

        c.Next()
    }
}

// Helper para extraer tenant_id en repos y servicios
func TenantIDFromContext(ctx context.Context) (uuid.UUID, error) {
    tid, ok := ctx.Value(ctxKeyTenantID).(uuid.UUID)
    if !ok || tid == uuid.Nil {
        return uuid.Nil, errors.New("tenant_id not in context")
    }
    return tid, nil
}
```

### Repositorio con tenant scope automático

```go
// internal/repository/venta_repo.go

type VentaRepository interface {
    Create(ctx context.Context, v *model.Venta) error
    FindByOfflineID(ctx context.Context, offlineID string) (*model.Venta, error)
    // ...
}

type ventaRepo struct {
    db *gorm.DB
}

// scopedDB extrae tenant_id del contexto y construye la query con scope
// RLS hace el filtrado real en PostgreSQL, pero agregar el WHERE explícito
// evita table scans y hace los query plans predecibles
func (r *ventaRepo) scopedDB(ctx context.Context) (*gorm.DB, error) {
    tid, err := TenantIDFromContext(ctx)
    if err != nil {
        return nil, err
    }
    return r.db.WithContext(ctx).Where("tenant_id = ?", tid), nil
}

func (r *ventaRepo) Create(ctx context.Context, v *model.Venta) error {
    tid, err := TenantIDFromContext(ctx)
    if err != nil {
        return err
    }
    v.TenantID = tid
    return r.db.WithContext(ctx).Create(v).Error
}

func (r *ventaRepo) FindByOfflineID(ctx context.Context, offlineID string) (*model.Venta, error) {
    db, err := r.scopedDB(ctx)
    if err != nil {
        return nil, err
    }
    var v model.Venta
    err = db.Where("offline_id = ?", offlineID).First(&v).Error
    return &v, err
}
```

### JWT con claim de tenant

```go
// internal/service/auth_service.go

type JWTClaims struct {
    jwt.RegisteredClaims
    UserID   uuid.UUID `json:"uid"`
    TenantID uuid.UUID `json:"tid"`   // ← nuevo claim
    Role     string    `json:"role"`
    DeviceID string    `json:"did"`   // ← nuevo: identifica la terminal
}

func (s *authService) GenerateTokenPair(user *model.Usuario) (string, string, error) {
    claims := JWTClaims{
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.cfg.JWTExpiry)),
            IssuedAt:  jwt.NewNumericDate(time.Now()),
        },
        UserID:   user.ID,
        TenantID: user.TenantID,  // ← viene del usuario en DB
        Role:     user.Rol,
        DeviceID: user.DeviceID,  // ← asignado al dispositivo en registro
    }
    // ... firma y retorno
}
```

---

## 4. Offline-first multi-tenant: generación de `offline_id` sin conflictos

### El problema

Con múltiples tenants y múltiples dispositivos por tenant, los IDs locales deben ser globalmente únicos sin coordinación con el servidor.

### Solución: `offline_id` compuesto en el frontend

```typescript
// src/offline/sync.ts

/**
 * Genera un offline_id globalmente único sin llamada al servidor.
 * Formato: {tenant_id}:{device_id}:{timestamp_ms}:{random_hex_4}
 * Ejemplo: "a3f1...uuid:d7b2...uuid:1710000000000:f3a1"
 *
 * Colisión teórica: (1/65536) por ms por dispositivo → prácticamente imposible
 */
export function generateOfflineId(tenantId: string, deviceId: string): string {
  const ts = Date.now().toString();
  const rand = Math.floor(Math.random() * 0xFFFF).toString(16).padStart(4, '0');
  return `${tenantId}:${deviceId}:${ts}:${rand}`;
}

// El device_id se genera una vez y se persiste en localStorage
export function getOrCreateDeviceId(): string {
  const key = 'blendpos_device_id';
  let deviceId = localStorage.getItem(key);
  if (!deviceId) {
    deviceId = crypto.randomUUID();
    localStorage.setItem(key, deviceId);
  }
  return deviceId;
}
```

### Sincronización con el backend

```typescript
// src/offline/sync.ts

interface SyncBatchRequest {
  ventas: OfflineVenta[];
  device_id: string;
  catalog_version: number;  // timestamp de última sync del catálogo
}

export async function syncPendingVentas(): Promise<void> {
  const pending = await db.ventasQueue
    .where('estado').equals('pendiente')
    .sortBy('created_at');  // orden causal por device

  if (pending.length === 0) return;

  // Batches de 500 para no saturar el servidor
  const batches = chunk(pending, 500);

  for (const batch of batches) {
    try {
      const res = await apiClient.post('/v1/ventas/sync-batch', {
        ventas: batch,
        device_id: getOrCreateDeviceId(),
        catalog_version: await getCatalogVersion(),
      } satisfies SyncBatchRequest);

      // Marcar como sincronizadas
      const syncedIds = res.data.synced_ids as string[];
      await db.ventasQueue
        .where('offline_id').anyOf(syncedIds)
        .modify({ estado: 'sincronizada' });

    } catch (err) {
      // Backoff exponencial: 30s → 60s → 120s → 300s
      await scheduleRetry(batch.map(v => v.offline_id), err);
    }
  }
}
```

---

## 5. Redis namespacing por tenant

Todas las keys de Redis se prefijarán con el `tenant_id` para evitar colisiones y permitir invalidación selectiva:

```go
// internal/infra/redis.go

func TenantCacheKey(tenantID uuid.UUID, key string) string {
    return fmt.Sprintf("t:%s:%s", tenantID.String(), key)
}

// Ejemplos de uso:
// Rate limiting:  "t:{tid}:rl:login:{ip}"
// Catálogo:       "t:{tid}:catalog:version"
// Sesión JWT:     "t:{tid}:session:{jti}"
// AFIP token:     "t:{tid}:afip:wsaa_token"  (el CUIT es por tenant)
```

---

## 6. AFIP Sidecar: multi-CUIT

Cada tenant tiene su propio CUIT y certificados AFIP. El sidecar es stateless — recibe el CUIT y la ruta al certificado en cada request.

```python
# afip-sidecar/afip_client.py

class AFIPClient:
    """
    Stateless: no guarda estado de CUIT.
    Los certificados se cargan desde paths configurados por tenant.
    """
    def __init__(self, cuit: str, cert_path: str, key_path: str, homologacion: bool):
        self.cuit = cuit
        self.wsaa = WSAA(cert=cert_path, key=key_path, homologacion=homologacion)
        self.wsfev1 = WSFEv1()
```

```go
// internal/infra/afip_client.go — request al sidecar incluye credenciales del tenant

type AFIPFacturarRequest struct {
    // Credenciales del tenant (vienen de configuracion_fiscal del tenant)
    CUITEmisor   string `json:"cuit_emisor"`
    CertPath     string `json:"cert_path"`    // path en el sidecar
    KeyPath      string `json:"key_path"`
    Homologacion bool   `json:"homologacion"`
    // Datos de la factura
    PuntoDeVenta    int     `json:"punto_de_venta"`
    TipoComprobante int     `json:"tipo_comprobante"`
    ImporteTotal    float64 `json:"importe_total"`
    // ...
}
```

Los certificados de cada tenant se almacenan en `/certs/{tenant_id}/afip.crt` dentro del sidecar (volumen persistente, gitignoreado).

---

## 7. Roadmap de migración y gatillos de escalabilidad

### Fase 0 — Preparación (sin downtime) — 4 semanas

```
Estado actual: single-tenant, sin tenant_id
↓
1. Migración 000025: agregar tenant_id a todas las tablas (nullable)
2. Crear tenant "legacy" para todos los datos existentes
3. Backfill: UPDATE todas las tablas SET tenant_id = 'legacy-uuid'
4. Agregar NOT NULL constraint
5. Activar RLS (sin políticas aún — comportamiento transparente)
6. Deploy Go con TenantMiddleware (hardcodea tenant_id del JWT)
```

### Fase 1 — Multi-tenant MVP — 8 semanas

```
↓
1. Sistema de registro de tenants (signup flow)
2. Gestión de planes y billing (Stripe o MercadoPago)
3. Activar políticas RLS en producción
4. Panel de administración de tenants (superadmin)
5. Onboarding flow: registro → carga catálogo → apertura de caja
6. Frontend: tenant_id en JWT, device_id persistente
```

### Fase 2 — Escalabilidad — gatillos

| Gatillo | Acción |
|---------|--------|
| > 50k tenants activos | Evaluar migración a schema-per-tenant |
| 1 tenant > 1M ventas | DB dedicada para ese tenant (manual, por contrato) |
| AFIP latencia p99 > 5s | Pool de workers AFIP dedicado por tenant |
| Redis > 80% memoria | Redis Cluster o Sentinel |
| CPU Go > 70% sostenido | Horizontal scaling (segundo nodo) — sin cambios de código |
| DB conexiones > 400 | PgBouncer connection pooler |

### Gatillo de sharding

El sharding por tenant_id se activa **solo si**:
1. Un tenant específico tiene > 5M de ventas en la DB, Y
2. Sus queries de analytics degradan p99 > 3s a pesar de índices

La estrategia de sharding sería hash del tenant_id sobre N DBs PostgreSQL, con un router en el middleware de Go. Este escenario es improbable en 24 meses a precio de kiosco/despensa.

---

## 8. Modelo de dominio core

```
┌───────────────────┐         ┌───────────────────┐
│      tenants      │         │      plans        │
│───────────────────│         │───────────────────│
│ id (UUID) PK      │────────▶│ id (UUID) PK      │
│ slug (unique)     │         │ nombre            │
│ nombre            │         │ max_terminales    │
│ plan_id (FK)      │         │ precio_mensual    │
│ cuit              │         │ features (JSONB)  │
│ activo            │         └───────────────────┘
│ created_at        │
└───────────────────┘
         │
         │ (tenant_id FK en todas las tablas de negocio)
         │
         ├──────────────────────────────────────────────┐
         ▼                                              ▼
┌───────────────────┐                       ┌───────────────────┐
│     usuarios      │                       │    productos      │
│───────────────────│                       │───────────────────│
│ id (UUID)         │                       │ id (UUID)         │
│ tenant_id (FK)    │◀──── RLS policy ─────▶│ tenant_id (FK)    │
│ email             │                       │ barcode           │
│ rol               │                       │ nombre            │
│ device_id         │ ← nuevo               │ precio            │
└───────────────────┘                       │ stock             │
                                            └───────────────────┘
         │
         ▼
┌───────────────────┐         ┌───────────────────┐
│      ventas       │         │   sesion_cajas    │
│───────────────────│         │───────────────────│
│ id (UUID)         │         │ id (UUID)         │
│ tenant_id (FK)    │         │ tenant_id (FK)    │
│ offline_id        │ ← UNIQUE│ usuario_id (FK)   │
│ device_id         │ ← nuevo │ monto_apertura    │
│ estado            │         │ estado            │
│ created_at        │         └───────────────────┘
└─────────┬─────────┘
          │
          ▼
┌───────────────────┐         ┌───────────────────┐
│   venta_items     │         │   comprobantes    │
│───────────────────│         │───────────────────│
│ tenant_id (FK)    │         │ tenant_id (FK)    │
│ venta_id (FK)     │         │ venta_id (FK)     │
│ producto_id (FK)  │         │ cae               │
│ precio_unitario   │ ← snap  │ estado            │
│ precio_manual     │ ← nuevo └───────────────────┘
└───────────────────┘
```

### Constraint de unicidad multi-tenant para offline_id

```sql
-- Garantiza idempotencia: mismo offline_id del mismo tenant = misma venta
ALTER TABLE ventas
  ADD CONSTRAINT uq_ventas_tenant_offline
  UNIQUE (tenant_id, offline_id);
```

---

## 9. Stack tecnológico: confirmación y adiciones

| Componente | Tecnología | Versión | Cambios para multi-tenant |
|-----------|-----------|---------|--------------------------|
| Backend | Go + Gin | 1.24 | TenantMiddleware, RLS injection |
| ORM | GORM | 1.25 | `scopedDB()` helper en todos los repos |
| DB | PostgreSQL | 15 | RLS policies, índices compuestos |
| Cache/Queue | Redis | 7 | Namespacing por tenant_id |
| Frontend | React + Vite | 19 | `tenant_id` en Zustand `useAuthStore` |
| Offline | Dexie.js | 4.3 | `device_id` en `generateOfflineId()` |
| AFIP | Python/FastAPI | 0.111 | Multi-CUIT stateless |
| Migrations | golang-migrate | latest | Migración 000025+ |
| Billing | MercadoPago / Stripe | TBD | Nuevo servicio `billing_service.go` |

**No se agrega ningún componente de infraestructura nuevo en Fase 1.** El stack existente soporta la escala objetivo sin modificaciones estructurales.
