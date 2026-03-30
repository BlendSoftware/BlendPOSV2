-- Add tenant_id to producto_hijo for tenant isolation (cross-tenant data leak fix).
ALTER TABLE producto_hijo ADD COLUMN IF NOT EXISTS tenant_id UUID;

-- Backfill tenant_id from the parent product's tenant_id.
UPDATE producto_hijo ph
SET tenant_id = p.tenant_id
FROM productos p
WHERE ph.producto_padre_id = p.id
  AND ph.tenant_id IS NULL;

-- Now make it NOT NULL after backfill.
ALTER TABLE producto_hijo ALTER COLUMN tenant_id SET NOT NULL;

-- Index for tenant-scoped queries.
CREATE INDEX IF NOT EXISTS idx_producto_hijo_tenant_id ON producto_hijo (tenant_id);

-- Enable RLS for tenant isolation.
ALTER TABLE producto_hijo ENABLE ROW LEVEL SECURITY;

CREATE POLICY producto_hijo_tenant_isolation ON producto_hijo
    USING (tenant_id = current_setting('app.current_tenant', true)::uuid);
