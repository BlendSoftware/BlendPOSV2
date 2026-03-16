-- Rollback: remove legacy tenant plan assignment and seed plans
UPDATE tenants SET plan_id = NULL WHERE slug = 'legacy';
DELETE FROM plans WHERE id IN (
    '00000000-0000-0000-0000-000000000001',
    '00000000-0000-0000-0000-000000000002',
    '00000000-0000-0000-0000-000000000003'
);
