-- ─────────────────────────────────────────────────────────────────────────────
-- Migration 000041: Create default "Casa Central" sucursal for tenants without any.
-- Ensures every tenant has at least one sucursal for the independent-branch model.
-- ─────────────────────────────────────────────────────────────────────────────

INSERT INTO sucursales (id, tenant_id, nombre, activa, created_at, updated_at)
SELECT gen_random_uuid(), t.id, 'Casa Central', true, NOW(), NOW()
FROM tenants t
WHERE NOT EXISTS (
    SELECT 1 FROM sucursales s WHERE s.tenant_id = t.id
);
