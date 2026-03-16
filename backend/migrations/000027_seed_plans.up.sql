-- ─────────────────────────────────────────────────────────────────────────────
-- Migration 000027: Seed plans + assign legacy tenant to Starter
-- ─────────────────────────────────────────────────────────────────────────────

-- ── 1. Insert canonical plans ──────────────────────────────────────────────────
-- Use fixed UUIDs so they can be referenced in application code and future migrations.
INSERT INTO plans (id, nombre, max_terminales, max_productos, precio_mensual, features, activo)
VALUES
    -- Kiosco: single terminal, capped catalog, no AFIP invoicing
    ('00000000-0000-0000-0000-000000000001',
     'Kiosco', 1, 500, 0,
     '{"facturacion": false, "analytics": false, "multi_terminal": false}',
     true),
    -- Starter: 2 terminals, unlimited catalog, AFIP invoicing included
    ('00000000-0000-0000-0000-000000000002',
     'Starter', 2, 0, 0,
     '{"facturacion": true, "analytics": false, "multi_terminal": true}',
     true),
    -- Pro: up to 10 terminals, analytics dashboard, all features
    ('00000000-0000-0000-0000-000000000003',
     'Pro', 10, 0, 0,
     '{"facturacion": true, "analytics": true, "multi_terminal": true}',
     true)
ON CONFLICT (id) DO NOTHING;

-- ── 2. Assign the legacy tenant to Starter plan ───────────────────────────────
UPDATE tenants
SET plan_id = '00000000-0000-0000-0000-000000000002'
WHERE slug = 'legacy'
  AND plan_id IS NULL;
