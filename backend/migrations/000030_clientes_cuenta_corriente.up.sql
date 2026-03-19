-- Clientes (fiado / cuenta corriente)
CREATE TABLE clientes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    nombre VARCHAR(200) NOT NULL,
    telefono VARCHAR(50),
    email VARCHAR(200),
    dni VARCHAR(20),
    limite_credito NUMERIC(12,2) NOT NULL DEFAULT 0,
    saldo_deudor NUMERIC(12,2) NOT NULL DEFAULT 0,
    activo BOOLEAN NOT NULL DEFAULT true,
    notas TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_clientes_tenant ON clientes(tenant_id);
CREATE INDEX idx_clientes_nombre ON clientes(tenant_id, nombre);

-- Movimientos de cuenta corriente (ledger — append-only)
CREATE TABLE movimientos_cuenta (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id),
    cliente_id UUID NOT NULL REFERENCES clientes(id),
    tipo VARCHAR(20) NOT NULL CHECK (tipo IN ('cargo', 'pago', 'ajuste')),
    monto NUMERIC(12,2) NOT NULL,
    saldo_posterior NUMERIC(12,2) NOT NULL,
    referencia_id UUID,
    referencia_tipo VARCHAR(30),
    descripcion TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_mov_cuenta_tenant ON movimientos_cuenta(tenant_id);
CREATE INDEX idx_mov_cuenta_cliente ON movimientos_cuenta(tenant_id, cliente_id);

-- RLS
ALTER TABLE clientes ENABLE ROW LEVEL SECURITY;
ALTER TABLE movimientos_cuenta ENABLE ROW LEVEL SECURITY;
CREATE POLICY tenant_isolation_clientes ON clientes USING (tenant_id = current_setting('app.tenant_id')::uuid);
CREATE POLICY tenant_isolation_mov_cuenta ON movimientos_cuenta USING (tenant_id = current_setting('app.tenant_id')::uuid);

-- Add cliente_id to ventas for linking sales to customers
ALTER TABLE ventas ADD COLUMN cliente_id UUID REFERENCES clientes(id);
