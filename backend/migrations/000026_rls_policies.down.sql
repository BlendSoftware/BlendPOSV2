-- Rollback 000026: eliminar políticas RLS y función helper

DROP POLICY IF EXISTS tenant_isolation ON audit_log;
DROP POLICY IF EXISTS tenant_isolation ON configuracion_fiscal;
DROP POLICY IF EXISTS tenant_isolation ON promociones;
DROP POLICY IF EXISTS tenant_isolation ON compra_items;
DROP POLICY IF EXISTS tenant_isolation ON compra_pagos;
DROP POLICY IF EXISTS tenant_isolation ON compras;
DROP POLICY IF EXISTS tenant_isolation ON historial_precios;
DROP POLICY IF EXISTS tenant_isolation ON movimientos_stock;
DROP POLICY IF EXISTS tenant_isolation ON comprobantes;
DROP POLICY IF EXISTS tenant_isolation ON movimiento_cajas;
DROP POLICY IF EXISTS tenant_isolation ON sesion_cajas;
DROP POLICY IF EXISTS tenant_isolation ON venta_items;
DROP POLICY IF EXISTS tenant_isolation ON ventas;
DROP POLICY IF EXISTS tenant_isolation ON proveedores;
DROP POLICY IF EXISTS tenant_isolation ON categorias;
DROP POLICY IF EXISTS tenant_isolation ON productos;
DROP POLICY IF EXISTS tenant_isolation ON usuarios;

DROP FUNCTION IF EXISTS current_tenant_id();
