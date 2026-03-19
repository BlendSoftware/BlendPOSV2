-- Per-sucursal stock tracking
CREATE TABLE stock_sucursal (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id),
  producto_id UUID NOT NULL REFERENCES productos(id),
  sucursal_id UUID NOT NULL REFERENCES sucursales(id),
  stock_actual INTEGER NOT NULL DEFAULT 0,
  stock_minimo INTEGER NOT NULL DEFAULT 5,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE(tenant_id, producto_id, sucursal_id)
);
CREATE INDEX idx_stock_sucursal_tenant ON stock_sucursal(tenant_id);
CREATE INDEX idx_stock_sucursal_producto ON stock_sucursal(tenant_id, producto_id);
CREATE INDEX idx_stock_sucursal_sucursal ON stock_sucursal(tenant_id, sucursal_id);
ALTER TABLE stock_sucursal ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation_stock_sucursal ON stock_sucursal USING (tenant_id = current_tenant_id() OR current_tenant_id() IS NULL);

-- Stock transfers between branches
CREATE TABLE transferencias_stock (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id UUID NOT NULL REFERENCES tenants(id),
  sucursal_origen_id UUID NOT NULL REFERENCES sucursales(id),
  sucursal_destino_id UUID NOT NULL REFERENCES sucursales(id),
  estado VARCHAR(20) NOT NULL DEFAULT 'pendiente' CHECK (estado IN ('pendiente', 'completada', 'rechazada', 'cancelada')),
  notas TEXT,
  creado_por UUID NOT NULL REFERENCES usuarios(id),
  completado_por UUID,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  completed_at TIMESTAMPTZ
);
CREATE INDEX idx_transferencias_tenant ON transferencias_stock(tenant_id);
ALTER TABLE transferencias_stock ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation_transferencias ON transferencias_stock USING (tenant_id = current_tenant_id() OR current_tenant_id() IS NULL);

-- Transfer line items
CREATE TABLE transferencia_items (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  transferencia_id UUID NOT NULL REFERENCES transferencias_stock(id) ON DELETE CASCADE,
  producto_id UUID NOT NULL REFERENCES productos(id),
  cantidad INTEGER NOT NULL CHECK (cantidad > 0)
);

-- Add es_deposito flag to sucursales (central warehouse)
ALTER TABLE sucursales ADD COLUMN IF NOT EXISTS es_deposito BOOLEAN NOT NULL DEFAULT false;

-- Add sucursal_id to movimientos_stock for audit trail
ALTER TABLE movimientos_stock ADD COLUMN IF NOT EXISTS sucursal_id UUID REFERENCES sucursales(id);
