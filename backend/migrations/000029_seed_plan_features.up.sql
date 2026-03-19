-- ─────────────────────────────────────────────────────────────────────────────
-- Migration 000029: Seed expanded feature flags into plans JSONB
-- Idempotent: uses UPDATE ... WHERE to set features on known plan UUIDs.
-- ─────────────────────────────────────────────────────────────────────────────

-- Kiosco (free): all advanced features disabled
UPDATE plans
SET features = '{"analytics_avanzados": false, "export_csv": false, "multi_terminal": false, "soporte_prioritario": false, "productos_ilimitados": false, "facturacion": false, "analytics": false}'::jsonb
WHERE id = '00000000-0000-0000-0000-000000000001';

-- Starter: mid-tier — facturacion + multi_terminal, no advanced analytics
UPDATE plans
SET features = '{"analytics_avanzados": false, "export_csv": false, "multi_terminal": true, "soporte_prioritario": false, "productos_ilimitados": true, "facturacion": true, "analytics": false}'::jsonb
WHERE id = '00000000-0000-0000-0000-000000000002';

-- Pro (paid): all features enabled
UPDATE plans
SET features = '{"analytics_avanzados": true, "export_csv": true, "multi_terminal": true, "soporte_prioritario": true, "productos_ilimitados": true, "facturacion": true, "analytics": true}'::jsonb
WHERE id = '00000000-0000-0000-0000-000000000003';
