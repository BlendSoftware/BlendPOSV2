ALTER TABLE movimientos_stock DROP COLUMN IF EXISTS sucursal_id;
ALTER TABLE sucursales DROP COLUMN IF EXISTS es_deposito;
DROP TABLE IF EXISTS transferencia_items;
DROP TABLE IF EXISTS transferencias_stock;
DROP TABLE IF EXISTS stock_sucursal;
