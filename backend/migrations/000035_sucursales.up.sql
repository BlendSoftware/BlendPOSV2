CREATE TABLE sucursales (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    nombre VARCHAR(200) NOT NULL,
    direccion TEXT,
    telefono VARCHAR(50),
    activa BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_sucursales_tenant ON sucursales(tenant_id);

ALTER TABLE sucursales ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_sucursales ON sucursales
    USING (tenant_id = current_tenant_id() OR current_tenant_id() IS NULL);

-- Add sucursal_id to usuarios (nullable — null means "all branches" for admin/supervisor)
ALTER TABLE usuarios ADD COLUMN IF NOT EXISTS sucursal_id UUID REFERENCES sucursales(id);

-- Add sucursal_id to sesion_cajas
ALTER TABLE sesion_cajas ADD COLUMN IF NOT EXISTS sucursal_id UUID REFERENCES sucursales(id);

-- Add sucursal_id to ventas
ALTER TABLE ventas ADD COLUMN IF NOT EXISTS sucursal_id UUID REFERENCES sucursales(id);
