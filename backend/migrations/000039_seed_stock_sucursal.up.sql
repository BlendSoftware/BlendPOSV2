-- Seed stock_sucursal records for all existing products x active sucursales.
-- This ensures the "Stock por Sucursal" page shows all products and transfers
-- don't fail with "sin stock registrado" for products created before the
-- stock_sucursal table was introduced.
INSERT INTO stock_sucursal (tenant_id, producto_id, sucursal_id, stock_actual, stock_minimo, updated_at)
SELECT p.tenant_id, p.id, s.id, 0, p.stock_minimo, NOW()
FROM productos p
CROSS JOIN sucursales s
WHERE p.activo = true
  AND s.activa = true
  AND p.tenant_id = s.tenant_id
ON CONFLICT (tenant_id, producto_id, sucursal_id) DO NOTHING;
