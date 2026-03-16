# PENDIENTE — Ítems diferidos de Fase 1

> Última actualización: 2026-03-16
> Ítems que requieren dependencias externas, infraestructura adicional o decisiones de negocio antes de implementarse.

---

## F1-4 — AFIP Sidecar multi-tenant (stateless)

**Estado:** Implementado (2026-03-16)

**Cambios aplicados:**
- `afip-sidecar/schemas.py` — `FacturarRequest` acepta `cert_pem`, `key_pem`, `modo` opcionales + validación anti-path-traversal
- `afip-sidecar/afip_client.py` — `AFIPClient` soporta init desde PEM content (temp dir), Redis keys por-CUIT (`afip:wsaa:{cuit}:{mode}:*`), `cleanup()` + `__del__`
- `afip-sidecar/main.py` — modo stateless (cliente efímero por request con `finally: cleanup()`) + fallback legacy al cliente global
- `backend/internal/infra/afip.go` — `AFIPPayload` incluye `CertPEM`, `KeyPEM`, `Modo`
- `backend/internal/worker/facturacion_worker.go` — `buildAFIPPayload` usa `ObtenerConfiguracionCompleta` para leer PEM certs
- `backend/internal/worker/retry_cron.go` — idem en retry

**Pendiente (Fase 2):**
- Cifrado de clave privada en Vault (HashiCorp) o con clave derivada de `tenant_id`
- El retry cron usa contexto background — no accede al tenant correcto en multi-tenant; requiere scope de tenant en Fase 2

---

## F1-5 — Billing / Suscripciones (MercadoPago)

**Estado:** Diferido
**Razón:** Requiere cuenta MercadoPago Business, webhook URL pública y decisión sobre modelo de precios.

**Trabajo requerido:**
- Integrar MercadoPago Subscriptions API (preapproval) o Checkout Pro para pagos únicos.
- Agregar tabla `suscripciones` con `tenant_id`, `plan_id`, `mp_preapproval_id`, `estado`, `next_billing_date`.
- Webhook `POST /v1/webhooks/mercadopago` para procesar eventos de pago (activar/suspender tenant).
- Panel superadmin: mostrar estado de suscripción y fecha próximo cobro.
- Email de aviso 7 días antes del vencimiento.

**Dependencias:**
- Cuenta MercadoPago con acceso a Subscriptions habilitado
- Dominio público para webhook (no funciona en localhost)
- Decisión de precios (actualmente `precio_mensual` en tabla `plans` pero sin cobro real)

**Archivos a crear:**
- `backend/internal/handler/webhooks.go`
- `backend/internal/service/billing_service.go`
- `backend/migrations/000028_suscripciones.up.sql`

---

## F1-9 — Read Replica para Analytics

**Estado:** Implementado — infraestructura lista (2026-03-16)

**Cambios aplicados:**
- `backend/internal/config/config.go` — `DatabaseReadReplicaURL` + `BindEnv`
- `backend/internal/infra/database.go` — `NewDatabaseReadReplica(primaryDB, replicaURL)` con fallback automático
- `backend/internal/handler/venta_reporte.go` — `GET /v1/ventas/reporte` PoC usando `dbRead`, filtro explícito `tenant_id` (no RLS)
- `backend/internal/router/router.go` — `DBRead` en `Deps`, ruta registrada
- `backend/cmd/server/main.go` — `dbRead := infra.NewDatabaseReadReplica(db, cfg.DatabaseReadReplicaURL)`

**Para activar en producción:**
- Configurar `DATABASE_READ_REPLICA_URL` con DSN de la réplica PostgreSQL
- Sin cambio de código requerido; el fallback a primary es automático cuando la variable está vacía

**Cuando ampliar:** Cuando `COUNT(ventas)` > ~500k por tenant empiece a impactar latencia en queries de dashboard; migrar queries de `DashboardPage` y métricas de superadmin a `dbRead`.

---

## F2-x — Fase 2 (planificada)

Items de la hoja de ruta Fase 2 aún no especificados en detalle:

| ID    | Descripción                                      | Dependencia           |
|-------|--------------------------------------------------|-----------------------|
| F2-1  | White-label: logo y colores por tenant           | —                     |
| F2-2  | App móvil React Native (misma API)               | —                     |
| F2-3  | Reportes exportables (PDF/Excel) por tenant      | —                     |
| F2-4  | Integración ARCA (ex-AFIP) nuevas resoluciones   | AFIP updates          |
| F2-5  | Multi-sucursal: tenants con múltiples ubicaciones | F1-4 completado       |
| F2-6  | Marketplace de integraciones (ML, Tienda Nube)   | —                     |
