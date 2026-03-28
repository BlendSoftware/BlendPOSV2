-- ─────────────────────────────────────────────────────────────────────────────
-- Migration 000040: Subscription tiers — Básico / Profesional / Enterprise
-- Adds max_sucursales, max_usuarios to plans; reseeds feature flags.
-- ─────────────────────────────────────────────────────────────────────────────

-- ── 1. Add new limit columns ─────────────────────────────────────────────────
ALTER TABLE plans ADD COLUMN IF NOT EXISTS max_sucursales INT NOT NULL DEFAULT 1;
ALTER TABLE plans ADD COLUMN IF NOT EXISTS max_usuarios   INT NOT NULL DEFAULT 1;

-- ── 2. Upsert the three canonical tiers ──────────────────────────────────────
-- Básico (free) — ID 001
UPDATE plans SET
    nombre          = 'Básico',
    max_terminales  = 1,
    max_productos   = 500,
    max_sucursales  = 1,
    max_usuarios    = 1,
    precio_mensual  = 0,
    features        = '{
        "multi_sucursal": false,
        "transferencias": false,
        "stock_sucursal": false,
        "usuarios_extra": false,
        "vencimientos": false,
        "proveedores": false,
        "compras": false,
        "facturacion_afip": false,
        "facturacion_ri": false,
        "reportes_avanzados": false,
        "ai_assistant": false,
        "apariencia": false,
        "clientes_management": false,
        "api_access": false
    }'::jsonb,
    activo = true
WHERE id = '00000000-0000-0000-0000-000000000001';

-- Profesional — ID 002
UPDATE plans SET
    nombre          = 'Profesional',
    max_terminales  = 5,
    max_productos   = 5000,
    max_sucursales  = 3,
    max_usuarios    = 10,
    precio_mensual  = 4999.00,
    features        = '{
        "multi_sucursal": true,
        "transferencias": true,
        "stock_sucursal": true,
        "usuarios_extra": true,
        "vencimientos": true,
        "proveedores": true,
        "compras": true,
        "facturacion_afip": true,
        "facturacion_ri": false,
        "reportes_avanzados": true,
        "ai_assistant": false,
        "apariencia": true,
        "clientes_management": true,
        "api_access": false
    }'::jsonb,
    activo = true
WHERE id = '00000000-0000-0000-0000-000000000002';

-- Enterprise — ID 003
UPDATE plans SET
    nombre          = 'Enterprise',
    max_terminales  = 0,
    max_productos   = 0,
    max_sucursales  = 0,
    max_usuarios    = 0,
    precio_mensual  = 14999.00,
    features        = '{
        "multi_sucursal": true,
        "transferencias": true,
        "stock_sucursal": true,
        "usuarios_extra": true,
        "vencimientos": true,
        "proveedores": true,
        "compras": true,
        "facturacion_afip": true,
        "facturacion_ri": true,
        "reportes_avanzados": true,
        "ai_assistant": true,
        "apariencia": true,
        "clientes_management": true,
        "api_access": true
    }'::jsonb,
    activo = true
WHERE id = '00000000-0000-0000-0000-000000000003';

-- ── 3. Assign all tenants without a plan to Básico ───────────────────────────
UPDATE tenants
SET plan_id = '00000000-0000-0000-0000-000000000001'
WHERE plan_id IS NULL;
