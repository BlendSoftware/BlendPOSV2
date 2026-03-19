-- Revert to single-tenant unique index (NOT RECOMMENDED)
DROP INDEX IF EXISTS uq_caja_abierta_por_punto;
CREATE UNIQUE INDEX uq_caja_abierta_por_punto ON sesion_cajas (punto_de_venta) WHERE estado = 'abierta';
