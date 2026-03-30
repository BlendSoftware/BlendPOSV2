-- Revert tenant_id addition to producto_hijo.
DROP POLICY IF EXISTS producto_hijo_tenant_isolation ON producto_hijo;
ALTER TABLE producto_hijo DISABLE ROW LEVEL SECURITY;
DROP INDEX IF EXISTS idx_producto_hijo_tenant_id;
ALTER TABLE producto_hijo DROP COLUMN IF EXISTS tenant_id;
