DROP POLICY IF EXISTS tenant_isolation_lotes ON lotes_producto;
DROP TABLE IF EXISTS lotes_producto;
ALTER TABLE productos DROP COLUMN IF EXISTS controla_vencimiento;
