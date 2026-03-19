-- Fix: unique index on sesion_cajas must include tenant_id for multi-tenant isolation.
-- Without it, two different tenants can't have punto_de_venta=1 open simultaneously.
DROP INDEX IF EXISTS uq_caja_abierta_por_punto;
CREATE UNIQUE INDEX uq_caja_abierta_por_punto ON sesion_cajas (tenant_id, punto_de_venta) WHERE estado = 'abierta';
