-- ─────────────────────────────────────────────────────────────────────────────
-- Rollback 000029: Restore original features from migration 000027
-- ─────────────────────────────────────────────────────────────────────────────

UPDATE plans
SET features = '{"facturacion": false, "analytics": false, "multi_terminal": false}'::jsonb
WHERE id = '00000000-0000-0000-0000-000000000001';

UPDATE plans
SET features = '{"facturacion": true, "analytics": false, "multi_terminal": true}'::jsonb
WHERE id = '00000000-0000-0000-0000-000000000002';

UPDATE plans
SET features = '{"facturacion": true, "analytics": true, "multi_terminal": true}'::jsonb
WHERE id = '00000000-0000-0000-0000-000000000003';
