-- ─────────────────────────────────────────────────────────────────────────────
-- 000025: Multi-tenant foundation — tenant_id en todas las tablas de negocio
--
-- ESTRATEGIA (sin downtime):
--   1. Crear tablas plans + tenants
--   2. Insertar tenant "legacy" para datos existentes
--   3. Agregar tenant_id NULL a todas las tablas
--   4. Backfill con el UUID del tenant legacy
--   5. Setear NOT NULL
--   6. Crear índices compuestos y ajustar unique constraints
--   7. Activar RLS (sin políticas — transparente hasta migración 000026)
-- ─────────────────────────────────────────────────────────────────────────────

-- ── 1. Tabla de planes ────────────────────────────────────────────────────────
CREATE TABLE plans (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    nombre           VARCHAR(100) NOT NULL,
    max_terminales   INT NOT NULL DEFAULT 1,
    max_productos    INT NOT NULL DEFAULT 0,   -- 0 = sin límite
    precio_mensual   NUMERIC(10,2) NOT NULL DEFAULT 0,
    features         JSONB NOT NULL DEFAULT '{}',
    activo           BOOLEAN NOT NULL DEFAULT true,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO plans (nombre, max_terminales, precio_mensual, features)
VALUES
    ('Starter',  1, 0,    '{"analytics": true, "export_csv": true, "afip": false}'),
    ('Kiosco',   1, 0,    '{"analytics": true, "export_csv": true, "afip": true}'),
    ('Negocio',  3, 0,    '{"analytics": true, "export_csv": true, "afip": true}'),
    ('Pro',      0, 0,    '{"analytics": true, "export_csv": true, "afip": true}');

-- ── 2. Tabla de tenants ───────────────────────────────────────────────────────
CREATE TABLE tenants (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug       VARCHAR(63) UNIQUE NOT NULL,
    nombre     VARCHAR(255) NOT NULL,
    plan_id    UUID REFERENCES plans(id),
    cuit       VARCHAR(13),
    activo     BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Tenant legacy: contiene todos los datos pre-multitenancy
INSERT INTO tenants (slug, nombre, activo)
VALUES ('legacy', 'Legacy Tenant', true);

-- ── 3. Agregar tenant_id NULL a todas las tablas de negocio ───────────────────
ALTER TABLE usuarios              ADD COLUMN IF NOT EXISTS tenant_id UUID;
ALTER TABLE productos             ADD COLUMN IF NOT EXISTS tenant_id UUID;
ALTER TABLE categorias            ADD COLUMN IF NOT EXISTS tenant_id UUID;
ALTER TABLE proveedores           ADD COLUMN IF NOT EXISTS tenant_id UUID;
ALTER TABLE ventas                ADD COLUMN IF NOT EXISTS tenant_id UUID;
ALTER TABLE venta_items           ADD COLUMN IF NOT EXISTS tenant_id UUID;
ALTER TABLE sesion_cajas          ADD COLUMN IF NOT EXISTS tenant_id UUID;
ALTER TABLE movimiento_cajas      ADD COLUMN IF NOT EXISTS tenant_id UUID;
ALTER TABLE comprobantes          ADD COLUMN IF NOT EXISTS tenant_id UUID;
ALTER TABLE movimientos_stock     ADD COLUMN IF NOT EXISTS tenant_id UUID;
ALTER TABLE historial_precios     ADD COLUMN IF NOT EXISTS tenant_id UUID;
ALTER TABLE compras               ADD COLUMN IF NOT EXISTS tenant_id UUID;
ALTER TABLE compra_pagos          ADD COLUMN IF NOT EXISTS tenant_id UUID;
ALTER TABLE compra_items          ADD COLUMN IF NOT EXISTS tenant_id UUID;
ALTER TABLE promociones           ADD COLUMN IF NOT EXISTS tenant_id UUID;
ALTER TABLE configuracion_fiscal  ADD COLUMN IF NOT EXISTS tenant_id UUID;
ALTER TABLE audit_log             ADD COLUMN IF NOT EXISTS tenant_id UUID;

-- device_id en usuarios: identifica el terminal físico
ALTER TABLE usuarios ADD COLUMN IF NOT EXISTS device_id VARCHAR(36);

-- ── 4. Backfill: asignar tenant legacy a todos los registros existentes ────────
DO $$
DECLARE
    legacy_id UUID;
BEGIN
    SELECT id INTO legacy_id FROM tenants WHERE slug = 'legacy';

    UPDATE usuarios             SET tenant_id = legacy_id WHERE tenant_id IS NULL;
    UPDATE productos            SET tenant_id = legacy_id WHERE tenant_id IS NULL;
    UPDATE categorias           SET tenant_id = legacy_id WHERE tenant_id IS NULL;
    UPDATE proveedores          SET tenant_id = legacy_id WHERE tenant_id IS NULL;
    UPDATE ventas               SET tenant_id = legacy_id WHERE tenant_id IS NULL;
    UPDATE venta_items          SET tenant_id = legacy_id WHERE tenant_id IS NULL;
    UPDATE sesion_cajas         SET tenant_id = legacy_id WHERE tenant_id IS NULL;
    UPDATE movimiento_cajas     SET tenant_id = legacy_id WHERE tenant_id IS NULL;
    UPDATE comprobantes         SET tenant_id = legacy_id WHERE tenant_id IS NULL;
    UPDATE movimientos_stock    SET tenant_id = legacy_id WHERE tenant_id IS NULL;
    UPDATE historial_precios    SET tenant_id = legacy_id WHERE tenant_id IS NULL;
    UPDATE compras              SET tenant_id = legacy_id WHERE tenant_id IS NULL;
    UPDATE compra_pagos         SET tenant_id = legacy_id WHERE tenant_id IS NULL;
    UPDATE compra_items         SET tenant_id = legacy_id WHERE tenant_id IS NULL;
    UPDATE promociones          SET tenant_id = legacy_id WHERE tenant_id IS NULL;
    UPDATE configuracion_fiscal SET tenant_id = legacy_id WHERE tenant_id IS NULL;
    UPDATE audit_log            SET tenant_id = legacy_id WHERE tenant_id IS NULL;
END $$;

-- ── 5. NOT NULL + FK a tenants ────────────────────────────────────────────────
ALTER TABLE usuarios             ALTER COLUMN tenant_id SET NOT NULL;
ALTER TABLE productos            ALTER COLUMN tenant_id SET NOT NULL;
ALTER TABLE categorias           ALTER COLUMN tenant_id SET NOT NULL;
ALTER TABLE proveedores          ALTER COLUMN tenant_id SET NOT NULL;
ALTER TABLE ventas               ALTER COLUMN tenant_id SET NOT NULL;
ALTER TABLE venta_items          ALTER COLUMN tenant_id SET NOT NULL;
ALTER TABLE sesion_cajas         ALTER COLUMN tenant_id SET NOT NULL;
ALTER TABLE movimiento_cajas     ALTER COLUMN tenant_id SET NOT NULL;
ALTER TABLE comprobantes         ALTER COLUMN tenant_id SET NOT NULL;
ALTER TABLE movimientos_stock    ALTER COLUMN tenant_id SET NOT NULL;
ALTER TABLE historial_precios    ALTER COLUMN tenant_id SET NOT NULL;
ALTER TABLE compras              ALTER COLUMN tenant_id SET NOT NULL;
ALTER TABLE compra_pagos         ALTER COLUMN tenant_id SET NOT NULL;
ALTER TABLE compra_items         ALTER COLUMN tenant_id SET NOT NULL;
ALTER TABLE promociones          ALTER COLUMN tenant_id SET NOT NULL;
ALTER TABLE configuracion_fiscal ALTER COLUMN tenant_id SET NOT NULL;
ALTER TABLE audit_log            ALTER COLUMN tenant_id SET NOT NULL;

ALTER TABLE usuarios             ADD CONSTRAINT fk_usuarios_tenant    FOREIGN KEY (tenant_id) REFERENCES tenants(id);
ALTER TABLE productos            ADD CONSTRAINT fk_productos_tenant   FOREIGN KEY (tenant_id) REFERENCES tenants(id);
ALTER TABLE categorias           ADD CONSTRAINT fk_categorias_tenant  FOREIGN KEY (tenant_id) REFERENCES tenants(id);
ALTER TABLE proveedores          ADD CONSTRAINT fk_proveedores_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id);
ALTER TABLE ventas               ADD CONSTRAINT fk_ventas_tenant      FOREIGN KEY (tenant_id) REFERENCES tenants(id);
ALTER TABLE sesion_cajas         ADD CONSTRAINT fk_sesiones_tenant    FOREIGN KEY (tenant_id) REFERENCES tenants(id);
ALTER TABLE comprobantes         ADD CONSTRAINT fk_comprobantes_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id);
ALTER TABLE compras              ADD CONSTRAINT fk_compras_tenant     FOREIGN KEY (tenant_id) REFERENCES tenants(id);
ALTER TABLE promociones          ADD CONSTRAINT fk_promociones_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id);
ALTER TABLE configuracion_fiscal ADD CONSTRAINT fk_cfg_fiscal_tenant  FOREIGN KEY (tenant_id) REFERENCES tenants(id);

-- ── 6. Índices compuestos (performance crítica) ───────────────────────────────
-- Ventas: el índice más consultado en analytics y caja
CREATE INDEX idx_ventas_tenant_created      ON ventas        (tenant_id, created_at DESC);
CREATE INDEX idx_ventas_tenant_sesion       ON ventas        (tenant_id, sesion_caja_id);
CREATE INDEX idx_ventas_tenant_estado       ON ventas        (tenant_id, estado);

-- Productos: lookup por barcode es hot-path de cada venta
CREATE INDEX idx_productos_tenant_barcode   ON productos     (tenant_id, codigo_barras);
CREATE INDEX idx_productos_tenant_activo    ON productos     (tenant_id, activo);
CREATE INDEX idx_productos_tenant_nombre    ON productos     (tenant_id, nombre);

-- Sesiones de caja
CREATE INDEX idx_sesiones_tenant_estado     ON sesion_cajas  (tenant_id, estado);
CREATE INDEX idx_sesiones_tenant_pdv        ON sesion_cajas  (tenant_id, punto_de_venta);

-- Comprobantes
CREATE INDEX idx_comprobantes_tenant_estado ON comprobantes  (tenant_id, estado);
CREATE INDEX idx_comprobantes_tenant_venta  ON comprobantes  (tenant_id, venta_id);

-- Audit log
CREATE INDEX idx_audit_tenant_created       ON audit_log     (tenant_id, created_at DESC);

-- Compras
CREATE INDEX idx_compras_tenant_estado      ON compras       (tenant_id, estado);

-- ── 7. Ajuste de unique constraints rotos en multi-tenant ─────────────────────
-- codigo_barras era globalmente único; ahora debe ser único por tenant
ALTER TABLE productos DROP CONSTRAINT IF EXISTS productos_codigo_barras_key;
CREATE UNIQUE INDEX uq_productos_tenant_barcode ON productos (tenant_id, codigo_barras);

-- categorias.nombre era globalmente único
ALTER TABLE categorias DROP CONSTRAINT IF EXISTS categorias_nombre_key;
CREATE UNIQUE INDEX uq_categorias_tenant_nombre ON categorias (tenant_id, nombre);

-- proveedores.cuit era globalmente único; en multi-tenant el mismo proveedor
-- puede existir en múltiples tenants (cada kiosco carga sus propios proveedores)
ALTER TABLE proveedores DROP CONSTRAINT IF EXISTS proveedores_cuit_key;
CREATE UNIQUE INDEX uq_proveedores_tenant_cuit ON proveedores (tenant_id, cuit);

-- offline_id único por tenant (idempotencia de sync) — ya existía un index simple
DROP INDEX IF EXISTS idx_ventas_offline_id;
CREATE UNIQUE INDEX uq_ventas_tenant_offline_id ON ventas (tenant_id, offline_id)
    WHERE offline_id IS NOT NULL;

-- ── 8. Activar RLS (sin políticas aún — comportamiento transparente) ──────────
ALTER TABLE usuarios             ENABLE ROW LEVEL SECURITY;
ALTER TABLE productos            ENABLE ROW LEVEL SECURITY;
ALTER TABLE categorias           ENABLE ROW LEVEL SECURITY;
ALTER TABLE proveedores          ENABLE ROW LEVEL SECURITY;
ALTER TABLE ventas               ENABLE ROW LEVEL SECURITY;
ALTER TABLE venta_items          ENABLE ROW LEVEL SECURITY;
ALTER TABLE sesion_cajas         ENABLE ROW LEVEL SECURITY;
ALTER TABLE movimiento_cajas     ENABLE ROW LEVEL SECURITY;
ALTER TABLE comprobantes         ENABLE ROW LEVEL SECURITY;
ALTER TABLE movimientos_stock    ENABLE ROW LEVEL SECURITY;
ALTER TABLE historial_precios    ENABLE ROW LEVEL SECURITY;
ALTER TABLE compras              ENABLE ROW LEVEL SECURITY;
ALTER TABLE compra_pagos         ENABLE ROW LEVEL SECURITY;
ALTER TABLE compra_items         ENABLE ROW LEVEL SECURITY;
ALTER TABLE promociones          ENABLE ROW LEVEL SECURITY;
ALTER TABLE configuracion_fiscal ENABLE ROW LEVEL SECURITY;
ALTER TABLE audit_log            ENABLE ROW LEVEL SECURITY;
