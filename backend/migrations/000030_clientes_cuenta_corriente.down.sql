ALTER TABLE ventas DROP COLUMN IF EXISTS cliente_id;

DROP POLICY IF EXISTS tenant_isolation_mov_cuenta ON movimientos_cuenta;
DROP POLICY IF EXISTS tenant_isolation_clientes ON clientes;

DROP TABLE IF EXISTS movimientos_cuenta;
DROP TABLE IF EXISTS clientes;
