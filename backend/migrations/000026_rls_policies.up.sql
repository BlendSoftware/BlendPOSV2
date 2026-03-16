-- ─────────────────────────────────────────────────────────────────────────────
-- 000026: Row Level Security — políticas de aislamiento por tenant
--
-- Cada política usa current_setting('app.tenant_id', true) que:
--   - Retorna el UUID del tenant actual (seteado por TenantMiddleware en Go)
--   - Retorna NULL (no lanza error) si la variable no está seteada — esto
--     permite que el usuario de superadmin opere sin tenant context
--
-- El segundo argumento 'true' en current_setting es CRÍTICO: sin él,
-- cualquier conexión sin SET LOCAL app.tenant_id lanzaría un error en lugar
-- de retornar NULL, rompiendo seeds, migrations y operaciones de admin.
-- ─────────────────────────────────────────────────────────────────────────────

-- Función helper que retorna el tenant_id activo o NULL
CREATE OR REPLACE FUNCTION current_tenant_id() RETURNS UUID AS $$
    SELECT NULLIF(current_setting('app.tenant_id', true), '')::UUID
$$ LANGUAGE SQL STABLE;

-- ── Políticas de aislamiento ──────────────────────────────────────────────────
-- Patrón: USING (tenant_id = current_tenant_id() OR current_tenant_id() IS NULL)
-- La condición IS NULL permite que el rol de superadmin (sin tenant context) vea todo.

CREATE POLICY tenant_isolation ON usuarios
    USING (tenant_id = current_tenant_id() OR current_tenant_id() IS NULL);

CREATE POLICY tenant_isolation ON productos
    USING (tenant_id = current_tenant_id() OR current_tenant_id() IS NULL);

CREATE POLICY tenant_isolation ON categorias
    USING (tenant_id = current_tenant_id() OR current_tenant_id() IS NULL);

CREATE POLICY tenant_isolation ON proveedores
    USING (tenant_id = current_tenant_id() OR current_tenant_id() IS NULL);

CREATE POLICY tenant_isolation ON ventas
    USING (tenant_id = current_tenant_id() OR current_tenant_id() IS NULL);

CREATE POLICY tenant_isolation ON venta_items
    USING (tenant_id = current_tenant_id() OR current_tenant_id() IS NULL);

CREATE POLICY tenant_isolation ON sesion_cajas
    USING (tenant_id = current_tenant_id() OR current_tenant_id() IS NULL);

CREATE POLICY tenant_isolation ON movimiento_cajas
    USING (tenant_id = current_tenant_id() OR current_tenant_id() IS NULL);

CREATE POLICY tenant_isolation ON comprobantes
    USING (tenant_id = current_tenant_id() OR current_tenant_id() IS NULL);

CREATE POLICY tenant_isolation ON movimientos_stock
    USING (tenant_id = current_tenant_id() OR current_tenant_id() IS NULL);

CREATE POLICY tenant_isolation ON historial_precios
    USING (tenant_id = current_tenant_id() OR current_tenant_id() IS NULL);

CREATE POLICY tenant_isolation ON compras
    USING (tenant_id = current_tenant_id() OR current_tenant_id() IS NULL);

CREATE POLICY tenant_isolation ON compra_pagos
    USING (tenant_id = current_tenant_id() OR current_tenant_id() IS NULL);

CREATE POLICY tenant_isolation ON compra_items
    USING (tenant_id = current_tenant_id() OR current_tenant_id() IS NULL);

CREATE POLICY tenant_isolation ON promociones
    USING (tenant_id = current_tenant_id() OR current_tenant_id() IS NULL);

CREATE POLICY tenant_isolation ON configuracion_fiscal
    USING (tenant_id = current_tenant_id() OR current_tenant_id() IS NULL);

CREATE POLICY tenant_isolation ON audit_log
    USING (tenant_id = current_tenant_id() OR current_tenant_id() IS NULL);
