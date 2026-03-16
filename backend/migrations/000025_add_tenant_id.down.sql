-- Rollback 000025: eliminar tenant_id y tablas multi-tenant

-- Desactivar RLS
ALTER TABLE audit_log            DISABLE ROW LEVEL SECURITY;
ALTER TABLE configuracion_fiscal DISABLE ROW LEVEL SECURITY;
ALTER TABLE promociones          DISABLE ROW LEVEL SECURITY;
ALTER TABLE compra_items         DISABLE ROW LEVEL SECURITY;
ALTER TABLE compra_pagos         DISABLE ROW LEVEL SECURITY;
ALTER TABLE compras              DISABLE ROW LEVEL SECURITY;
ALTER TABLE historial_precios    DISABLE ROW LEVEL SECURITY;
ALTER TABLE movimientos_stock    DISABLE ROW LEVEL SECURITY;
ALTER TABLE comprobantes         DISABLE ROW LEVEL SECURITY;
ALTER TABLE movimiento_cajas     DISABLE ROW LEVEL SECURITY;
ALTER TABLE sesion_cajas         DISABLE ROW LEVEL SECURITY;
ALTER TABLE venta_items          DISABLE ROW LEVEL SECURITY;
ALTER TABLE ventas               DISABLE ROW LEVEL SECURITY;
ALTER TABLE proveedores          DISABLE ROW LEVEL SECURITY;
ALTER TABLE categorias           DISABLE ROW LEVEL SECURITY;
ALTER TABLE productos            DISABLE ROW LEVEL SECURITY;
ALTER TABLE usuarios             DISABLE ROW LEVEL SECURITY;

-- Eliminar índices compuestos y unique constraints multi-tenant
DROP INDEX IF EXISTS uq_ventas_tenant_offline_id;
DROP INDEX IF EXISTS uq_proveedores_tenant_cuit;
DROP INDEX IF EXISTS uq_categorias_tenant_nombre;
DROP INDEX IF EXISTS uq_productos_tenant_barcode;
DROP INDEX IF EXISTS idx_audit_tenant_created;
DROP INDEX IF EXISTS idx_compras_tenant_estado;
DROP INDEX IF EXISTS idx_comprobantes_tenant_venta;
DROP INDEX IF EXISTS idx_comprobantes_tenant_estado;
DROP INDEX IF EXISTS idx_sesiones_tenant_pdv;
DROP INDEX IF EXISTS idx_sesiones_tenant_estado;
DROP INDEX IF EXISTS idx_productos_tenant_nombre;
DROP INDEX IF EXISTS idx_productos_tenant_activo;
DROP INDEX IF EXISTS idx_productos_tenant_barcode;
DROP INDEX IF EXISTS idx_ventas_tenant_estado;
DROP INDEX IF EXISTS idx_ventas_tenant_sesion;
DROP INDEX IF EXISTS idx_ventas_tenant_created;

-- Restaurar unique constraints originales
CREATE UNIQUE INDEX IF NOT EXISTS productos_codigo_barras_key ON productos (codigo_barras);
CREATE UNIQUE INDEX IF NOT EXISTS categorias_nombre_key       ON categorias (nombre);
CREATE UNIQUE INDEX IF NOT EXISTS proveedores_cuit_key        ON proveedores (cuit);
CREATE INDEX IF NOT EXISTS idx_ventas_offline_id              ON ventas (offline_id) WHERE offline_id IS NOT NULL;

-- Eliminar FKs y columna tenant_id
ALTER TABLE configuracion_fiscal DROP CONSTRAINT IF EXISTS fk_cfg_fiscal_tenant;
ALTER TABLE promociones          DROP CONSTRAINT IF EXISTS fk_promociones_tenant;
ALTER TABLE compras              DROP CONSTRAINT IF EXISTS fk_compras_tenant;
ALTER TABLE comprobantes         DROP CONSTRAINT IF EXISTS fk_comprobantes_tenant;
ALTER TABLE sesion_cajas         DROP CONSTRAINT IF EXISTS fk_sesiones_tenant;
ALTER TABLE ventas               DROP CONSTRAINT IF EXISTS fk_ventas_tenant;
ALTER TABLE proveedores          DROP CONSTRAINT IF EXISTS fk_proveedores_tenant;
ALTER TABLE categorias           DROP CONSTRAINT IF EXISTS fk_categorias_tenant;
ALTER TABLE productos            DROP CONSTRAINT IF EXISTS fk_productos_tenant;
ALTER TABLE usuarios             DROP CONSTRAINT IF EXISTS fk_usuarios_tenant;

ALTER TABLE audit_log            DROP COLUMN IF EXISTS tenant_id;
ALTER TABLE configuracion_fiscal DROP COLUMN IF EXISTS tenant_id;
ALTER TABLE promociones          DROP COLUMN IF EXISTS tenant_id;
ALTER TABLE compra_items         DROP COLUMN IF EXISTS tenant_id;
ALTER TABLE compra_pagos         DROP COLUMN IF EXISTS tenant_id;
ALTER TABLE compras              DROP COLUMN IF EXISTS tenant_id;
ALTER TABLE historial_precios    DROP COLUMN IF EXISTS tenant_id;
ALTER TABLE movimientos_stock    DROP COLUMN IF EXISTS tenant_id;
ALTER TABLE comprobantes         DROP COLUMN IF EXISTS tenant_id;
ALTER TABLE movimiento_cajas     DROP COLUMN IF EXISTS tenant_id;
ALTER TABLE sesion_cajas         DROP COLUMN IF EXISTS tenant_id;
ALTER TABLE venta_items          DROP COLUMN IF EXISTS tenant_id;
ALTER TABLE ventas               DROP COLUMN IF EXISTS tenant_id;
ALTER TABLE proveedores          DROP COLUMN IF EXISTS tenant_id;
ALTER TABLE categorias           DROP COLUMN IF EXISTS tenant_id;
ALTER TABLE productos            DROP COLUMN IF EXISTS tenant_id;
ALTER TABLE usuarios             DROP COLUMN IF EXISTS tenant_id;
ALTER TABLE usuarios             DROP COLUMN IF EXISTS device_id;

DROP TABLE IF EXISTS tenants;
DROP TABLE IF EXISTS plans;
