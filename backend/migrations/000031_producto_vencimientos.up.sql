-- Lotes de producto con fecha de vencimiento
CREATE TABLE lotes_producto (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id),
  producto_id UUID NOT NULL REFERENCES productos(id),
  codigo_lote VARCHAR(100),
  fecha_vencimiento DATE NOT NULL,
  cantidad INTEGER NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_lotes_tenant ON lotes_producto(tenant_id);
CREATE INDEX idx_lotes_producto ON lotes_producto(tenant_id, producto_id);
CREATE INDEX idx_lotes_vencimiento ON lotes_producto(tenant_id, fecha_vencimiento);

ALTER TABLE lotes_producto ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation_lotes ON lotes_producto USING (tenant_id = current_tenant_id() OR current_tenant_id() IS NULL);

-- Flag on producto for "this product tracks expiry"
ALTER TABLE productos ADD COLUMN IF NOT EXISTS controla_vencimiento BOOLEAN NOT NULL DEFAULT false;
