# BlendPOS — Spec-Driven Development Document

---

## 1. Visión General

**BlendPOS** es un sistema de punto de venta (POS) diseñado para kioscos y comercios minoristas de Argentina, con arquitectura **Offline-First** y modelo de negocio **SaaS Multi-Tenant**.

### Propuesta de Valor

Un kiosquero abre su comercio, se registra en 30 segundos, carga sus productos y empieza a facturar con AFIP sin saber nada de tecnología. BlendPOS garantiza:

- **Invulnerabilidad operativa:** Si se corta internet, el POS sigue funcionando. Las ventas se registran en IndexedDB (Dexie.js) y sincronizan automáticamente cuando vuelve la conexión. Autonomía mínima: 48 horas offline.
- **Aislamiento de datos:** Cada comercio tiene sus datos blindados con triple capa de seguridad: PostgreSQL RLS + middleware scoped DB + audit callbacks.
- **Facturación electrónica transparente:** Un sidecar Python habla con AFIP. El comerciante no sabe qué es un certificado, un CAE o un punto de venta — BlendPOS lo resuelve.
- **Escalabilidad por diseño:** Miles de comercios en la misma infraestructura, diferenciados por plan (Kiosco/Pro), con enforcement automático de límites.

### Stack

| Capa | Tecnología | Rol |
|------|-----------|-----|
| Frontend | React 19 + TypeScript + Zustand + Dexie.js + Mantine UI | POS UI, offline-first, PWA |
| Backend | Go 1.24 + Gin + GORM + PostgreSQL 15 + Redis 7 | API, auth, lógica de negocio, workers |
| Sidecar AFIP | Python + FastAPI + pyafipws | Facturación electrónica (stateless) |
| Infra | Docker Compose (dev), Railway/Fly.io (prod target) | Orquestación de servicios |

### Arquitectura Multi-Tenant (implementada)

```
Request → JWT Auth → Tenant Extraction → Scoped DB (WHERE tenant_id=?) → RLS (PostgreSQL policy)
                                              ↓
                                     Audit Callback (post-SELECT verification)
                                              ↓
                                     IDOR Guard (cross-tenant access prevention)
```

Triple capa de aislamiento — si una falla, las otras dos atrapan la fuga.

---

## 2. Estado Actual — Qué Está Hecho

### Backend (Go) — ~90% completo
- Tenant registration, CRUD, plan management, superadmin panel
- JWT auth (access 8h + refresh 24h) con revocación vía Redis
- RLS + scoped DB + audit middleware + IDOR guard
- Plan enforcement (max_productos, max_terminales) con cache Redis
- Ventas: CRUD + batch sync con dedup por offline_id + compensación de stock
- Facturación: worker async + retry cron + circuit breaker al sidecar
- 28 migraciones SQL (schema completo + multi-tenant + billing foundation)
- Billing: modelo Subscription + handler subscribe/webhook (MercadoPago interface)
- Config: Viper con read replica support, SMTP, worker pool sizing
- ~20 handlers de dominio (productos, caja, inventario, compras, proveedores, categorías, etc.)

### Frontend (React) — ~85% completo
- POS Terminal funcional con escaneo, carrito, pagos, modales
- 16+ páginas admin (dashboard, productos, categorías, inventario, facturación, compras, etc.)
- SuperAdmin panel con métricas, gestión de tenants, cambio de plan
- Onboarding multi-step (bienvenida, config fiscal, productos, usuarios)
- Registro de tenant
- 7 Zustand stores (auth, cart, caja, UI, printer, promociones, sale)
- Offline: Dexie.js con sync queue, retry con backoff exponencial, dedup
- API client con refresh proactivo de JWT
- Auth store con tokens en memoria (no localStorage)

### AFIP Sidecar — Funcional
- Endpoint `/invoice` stateless
- WSAA token caching
- Certificados de test incluidos
- Dockerizado

### Infraestructura
- Docker Compose con 5 servicios (postgres, redis, backend, frontend, afip-sidecar)
- Health checks configurados
- Hot reload: Air (backend) + Vite HMR (frontend)

---

## 3. Qué Falta — Tareas para Producción

### Fase 3: Producción y Cobro (semanas estimadas: 4-5)

Todo lo que falta para que un kiosquero pueda registrarse, pagar, y operar en producción real.

---

#### T1: CI/CD Pipeline
**Prioridad:** CRÍTICA — sin esto, todo lo demás es inseguro de deployar.

- [ ] **T1.1** Crear `.github/workflows/ci.yml` — pipeline principal
  - Trigger: push a `main`, PRs a `main`
  - Jobs paralelos: lint-frontend, lint-backend, test-frontend, test-backend
  - Frontend: `npm ci && npm run lint && npm run test`
  - Backend: `go vet ./... && go test ./... && go test -tags integration ./tests/... -v`
  - Testcontainers: needs `services: postgres:15, redis:7` en el workflow
- [ ] **T1.2** Agregar `golangci-lint` al backend
  - Crear `.golangci.yml` con reglas: `errcheck`, `govet`, `staticcheck`, `unused`, `gosimple`
  - Integrar en CI como job separado
- [ ] **T1.3** Crear `.github/workflows/deploy.yml` — deploy automático
  - Trigger: push a `main` (post-CI success)
  - Target: Railway o Fly.io (definir con Juani)
  - Secrets: DATABASE_URL, REDIS_URL, JWT_SECRET, INTERNAL_API_TOKEN, AFIP creds
- [ ] **T1.4** Branch protection en `main`
  - Require CI pass + 1 approval antes de merge
  - No force push

---

#### T2: MercadoPago — Billing Real
**Prioridad:** CRÍTICA — sin cobro no hay negocio.

- [ ] **T2.1** Implementar verificación de firma X-Signature en webhook
  - Validar HMAC-SHA256 del header `X-Signature` contra `MP_WEBHOOK_SECRET`
  - Rechazar requests sin firma válida (seguridad crítica)
- [ ] **T2.2** Flujo completo de suscripción
  - Backend: crear preference de MercadoPago con plan seleccionado
  - Manejar estados: `authorized`, `paused`, `cancelled`, `pending`
  - Actualizar `subscription.status` y `tenant.plan_id` según webhook
- [ ] **T2.3** Frontend: página de billing para el tenant
  - Mostrar plan actual, fecha de renovación, estado
  - Botón "Cambiar Plan" con redirect a checkout MercadoPago
  - Historial de pagos (últimos 12 meses)
- [ ] **T2.4** Lógica de gracia y suspensión
  - Si pago falla: 7 días de gracia, luego degradar a plan gratuito
  - Notificar al tenant por email (3 días antes de expiración, día de corte)
- [ ] **T2.5** Agregar env vars: `MP_ACCESS_TOKEN`, `MP_WEBHOOK_SECRET`, `MP_PUBLIC_KEY`
  - Documentar en config.go con defaults vacíos
  - Validar al startup que no estén vacíos en producción

---

#### T3: PWA Producción
**Prioridad:** ALTA — el kiosquero necesita instalar esto como app.

- [ ] **T3.1** Verificar/completar service worker (`src/sw.ts`)
  - Estrategia: precache shell de la app + runtime cache para API
  - Workbox: `precacheAndRoute` para assets estáticos
  - Network-first para API calls, cache-first para assets
- [ ] **T3.2** Manifest completo (`manifest.json`)
  - name, short_name, icons (192x192, 512x512), theme_color, background_color
  - display: `standalone`, orientation: `portrait`
  - start_url: `/`
- [ ] **T3.3** Prompt de instalación
  - Interceptar `beforeinstallprompt` event
  - Mostrar banner custom de Mantine (no el default del browser)
  - Guardar en localStorage si el usuario lo descartó (no molestar por 7 días)
- [ ] **T3.4** Update flow
  - Detectar nueva versión del SW → mostrar toast "Actualización disponible"
  - `skipWaiting()` + `clients.claim()` al aceptar
- [ ] **T3.5** Testear offline real
  - DevTools → Network → Offline → verificar que POS sigue operando
  - Verificar que sync queue procesa al reconectar

---

#### T4: Email y Notificaciones
**Prioridad:** MEDIA — necesario para billing y onboarding.

- [ ] **T4.1** Templates de email (HTML)
  - Bienvenida post-registro
  - Confirmación de pago
  - Aviso de expiración de plan (3 días antes)
  - Plan degradado por falta de pago
  - Reset de contraseña
- [ ] **T4.2** Servicio de email en el backend
  - Implementar `mailer.Send(to, template, data)` con los templates
  - SMTP config ya existe en Viper — conectar con implementación real
  - Usar worker async para no bloquear requests
- [ ] **T4.3** Endpoint de reset de contraseña
  - `POST /v1/auth/forgot-password` → genera token + envía email
  - `POST /v1/auth/reset-password` → valida token + actualiza password
  - Token expira en 1 hora, single-use

---

#### T5: Analytics y Reportes
**Prioridad:** MEDIA — diferenciador del producto ("analytics desde el plan gratuito").

- [ ] **T5.1** Queries de reportes en read replica
  - Ventas por día/semana/mes (totales, cantidad, ticket promedio)
  - Top 10 productos más vendidos
  - Ventas por medio de pago
  - Horarios pico de venta
  - Usar `DATABASE_READ_REPLICA_URL` para no impactar transaccional
- [ ] **T5.2** Endpoints de reportes
  - `GET /v1/reportes/ventas?desde=&hasta=&agrupacion=dia|semana|mes`
  - `GET /v1/reportes/productos/top?limit=10&desde=&hasta=`
  - `GET /v1/reportes/medios-pago?desde=&hasta=`
  - Todos filtrados por `tenant_id` (obligatorio)
- [ ] **T5.3** Frontend: Dashboard de analytics
  - Gráficos con Recharts o Chart.js (evaluar peso del bundle)
  - Filtros de fecha con DatePicker de Mantine
  - Cards de resumen: ventas hoy, esta semana, este mes
  - Tabla de top productos
- [ ] **T5.4** Export a CSV/PDF
  - Botón "Exportar" en cada reporte
  - CSV: generado en frontend (sin hit al backend)
  - PDF: opcional, fase posterior

---

#### T6: Hardening de Seguridad
**Prioridad:** ALTA — antes de exponer a internet.

- [ ] **T6.1** Rotar JWT_SECRET
  - Generar secret seguro de 256 bits
  - Validar en startup: no permitir default "dev_secret_change_in_production" en APP_ENV=production
  - Fail-fast con log.Fatal si no está configurado
- [ ] **T6.2** Rate limiting por tenant
  - Redis-based: `tenant:{id}:rate` con TTL de 1 minuto
  - Límites: 100 req/min plan gratuito, 500 req/min plan Pro
  - Header `X-RateLimit-Remaining` en responses
- [ ] **T6.3** HTTPS y security headers
  - HSTS, X-Content-Type-Options, X-Frame-Options, CSP
  - Configurar en middleware (no en proxy solo)
- [ ] **T6.4** Auditoría de dependencias
  - `npm audit` en CI (fail on high/critical)
  - `govulncheck ./...` en CI para Go
- [ ] **T6.5** Input validation comprehensiva
  - Verificar que TODOS los DTOs usan `binding:"required"` donde corresponde en Gin
  - Frontend: Zod schemas para forms críticos (registro, config fiscal)

---

#### T7: Device Management
**Prioridad:** BAJA — mejora operativa, no blocker.

- [ ] **T7.1** Endpoint de listado de dispositivos
  - `GET /v1/tenant/devices` — listar devices activos del tenant
  - Basado en `device_id` (did) del JWT
  - Mostrar: último acceso, IP, user-agent
- [ ] **T7.2** Revocar dispositivo
  - `DELETE /v1/tenant/devices/:id` — revocar tokens del device via Redis
  - Invalidar todos los JWT con ese `did`
- [ ] **T7.3** Frontend: sección de dispositivos en admin
  - Lista de terminales activas con estado online/offline
  - Botón "Desconectar" por device

---

#### T8: Onboarding y UX de Primer Uso
**Prioridad:** MEDIA — impacta conversión de registro a uso real.

- [ ] **T8.1** Flujo de primer login con cambio de contraseña obligatorio
  - Frontend: detectar `mustChangePassword` del auth store
  - Modal que bloquea toda interacción hasta cambiar password
  - Redirect a onboarding después del cambio
- [ ] **T8.2** Wizard de carga de productos
  - Importar desde CSV (plantilla descargable)
  - Carga masiva: parsear CSV, validar, preview, confirmar
  - Feedback de errores fila por fila
- [ ] **T8.3** Demo mode
  - Productos de ejemplo pre-cargados para que el kiosquero pruebe
  - Banner "Estás en modo demo" con botón "Empezar en serio"
  - Limpiar data demo al activar

---

#### T9: Plan Feature Flags
**Prioridad:** MEDIA — diferenciar planes más allá de límites numéricos.

- [ ] **T9.1** Definir features por plan
  - Plan Kiosco (gratuito): POS básico, 1 terminal, 100 productos, reportes básicos
  - Plan Pro (pago): Multi-terminal, productos ilimitados, analytics avanzados, export, soporte prioritario
  - Almacenar en `plans.features` (JSONB existente)
- [ ] **T9.2** Middleware de feature flags
  - Leer `plan.features` del tenant (cache Redis, TTL 5min)
  - Si feature no habilitada: 403 con mensaje claro ("Upgrade al plan Pro para usar analytics avanzados")
- [ ] **T9.3** Frontend: feature gating
  - Hook `useFeature(flag: string): boolean`
  - Componentes bloqueados muestran overlay con CTA de upgrade
  - No ocultar features — mostrar que existen pero están locked

---

#### T10: Sucursales (Multi-Branch)
**Prioridad:** MEDIA — necesario cuando un tenant tiene más de un local.

- [ ] **T10.1** Migración: tabla `sucursales` (id, tenant_id, nombre, direccion, telefono, activa, rango_pdv_desde, rango_pdv_hasta)
- [ ] **T10.2** Asignar usuarios a sucursal: campo `sucursal_id` en `usuarios`
- [ ] **T10.3** Asignar sesiones de caja a sucursal: campo `sucursal_id` en `sesion_cajas`
- [ ] **T10.4** Stock por sucursal: campo `sucursal_id` en `productos` o tabla `stock_sucursal` (producto_id, sucursal_id, stock_actual, stock_minimo)
- [ ] **T10.5** Ventas por sucursal: campo `sucursal_id` en `ventas` (derivado del cajero/caja)
- [ ] **T10.6** Backend: CRUD de sucursales, filtros en reportes
- [ ] **T10.7** Frontend: selector de sucursal en admin, filtro en reportes
- [ ] **T10.8** Reportes consolidados vs por sucursal

**Modelo de datos:**
- Catálogo (productos, categorías, proveedores) = compartido por tenant
- Stock = POR SUCURSAL (cada local tiene su inventario)
- Ventas/Caja = POR SUCURSAL (cada punto de venta pertenece a un local)
- Usuarios = asignados a 1 sucursal (o "todas" para admin/supervisor)

---

## 4. Orden de Ejecución Recomendado

```
T1 (CI/CD)  ─────────────────────────────────────────────→  PRIMERO (fundación)
    ↓
T6 (Seguridad) ──────→  T2 (MercadoPago) ──────→  T4 (Email)
    ↓                         ↓
T3 (PWA)                 T9 (Feature Flags)
    ↓                         ↓
T5 (Analytics)           T8 (Onboarding UX)
    ↓
T7 (Devices)  ────────────────────────────────────────────→  ÚLTIMO (nice-to-have)
```

**Crítico path:** T1 → T6 → T2 → T3 → Lanzamiento beta con 5 kioscos.

---

## 5. Riesgos

| Riesgo | Impacto | Mitigación |
|--------|---------|------------|
| MercadoPago webhook sin firma | Cualquiera puede simular pagos | T2.1 es blocker para producción |
| JWT_SECRET default en prod | Tokens forjables | T6.1 fail-fast en startup |
| Sin CI/CD, regressions pasan | Bugs en prod sin detectar | T1 es la primera tarea |
| AFIP sidecar como SPOF | No se factura si cae | Circuit breaker + retry cron ya implementados |
| Offline sync con datos corruptos | Ventas perdidas | Dedup por offline_id + compensación de stock ya implementados |
| Plan gratuito abusado | Costo de infra sin revenue | T9 feature flags + T6.2 rate limiting |
