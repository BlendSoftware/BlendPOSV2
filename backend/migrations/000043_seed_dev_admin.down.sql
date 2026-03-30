-- Rollback: remove the dev admin user and tenant seeded in 000043.
DELETE FROM usuarios WHERE username = 'admin@blendpos.com' AND tenant_id = (SELECT id FROM tenants WHERE slug = 'devshop');
-- Note: we don't delete the tenant/sucursal to avoid cascading issues with other data.
