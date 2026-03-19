ALTER TABLE ventas DROP COLUMN IF EXISTS sucursal_id;
ALTER TABLE sesion_cajas DROP COLUMN IF EXISTS sucursal_id;
ALTER TABLE usuarios DROP COLUMN IF EXISTS sucursal_id;

DROP POLICY IF EXISTS tenant_isolation_sucursales ON sucursales;
DROP TABLE IF EXISTS sucursales;
