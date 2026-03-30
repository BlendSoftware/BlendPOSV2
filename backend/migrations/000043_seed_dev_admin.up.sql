-- ─────────────────────────────────────────────────────────────────────────────
-- Migration 000043: Seed a development admin user with FULL access.
--
-- Creates (if not exists):
--   1. A "dev" tenant (slug: devshop)
--   2. A "Casa Central" sucursal for that tenant
--   3. An admin user: admin@blendpos.com / admin123  (rol: administrador)
--
-- The user has SucursalID = NULL which means access to ALL branches.
-- The rol "administrador" grants full access to every endpoint.
--
-- FOR DEVELOPMENT ONLY — do NOT run in production.
-- ─────────────────────────────────────────────────────────────────────────────

DO $$
DECLARE
    v_tenant_id   UUID;
    v_sucursal_id UUID;
    v_plan_id     UUID;
BEGIN
    -- Get the Basico/free plan (seeded in migration 000027/000040)
    SELECT id INTO v_plan_id FROM plans WHERE nombre = 'Basico' LIMIT 1;
    IF v_plan_id IS NULL THEN
        SELECT id INTO v_plan_id FROM plans WHERE activo = true ORDER BY precio_mensual ASC LIMIT 1;
    END IF;

    -- 1. Create tenant if it doesn't exist
    SELECT id INTO v_tenant_id FROM tenants WHERE slug = 'devshop';
    IF v_tenant_id IS NULL THEN
        INSERT INTO tenants (slug, nombre, plan_id, tipo_negocio, activo, created_at)
        VALUES ('devshop', 'Dev Shop', v_plan_id, 'kiosco', true, NOW())
        RETURNING id INTO v_tenant_id;
    END IF;

    -- 2. Create default sucursal if tenant has none
    SELECT id INTO v_sucursal_id FROM sucursales WHERE tenant_id = v_tenant_id LIMIT 1;
    IF v_sucursal_id IS NULL THEN
        INSERT INTO sucursales (tenant_id, nombre, activa, created_at, updated_at)
        VALUES (v_tenant_id, 'Casa Central', true, NOW(), NOW())
        RETURNING id INTO v_sucursal_id;
    END IF;

    -- 3. Create admin user if username doesn't exist for this tenant
    IF NOT EXISTS (
        SELECT 1 FROM usuarios WHERE username = 'admin@blendpos.com' AND tenant_id = v_tenant_id
    ) THEN
        INSERT INTO usuarios (
            tenant_id, username, nombre, email, password_hash, rol,
            punto_de_venta, sucursal_id, activo, must_change_password, created_at, updated_at
        ) VALUES (
            v_tenant_id,
            'admin@blendpos.com',
            'Admin BlendPOS',
            'admin@blendpos.com',
            '$2a$12$/kDnWGmd6j5RyjLQcDMaROzflY2KmLkX0OuR75ojMPrmLMcw05WlW',  -- admin123
            'administrador',
            NULL,       -- all registers
            NULL,       -- all branches (consolidated view)
            true,
            false,      -- no forced password change
            NOW(),
            NOW()
        );
    END IF;
END $$;
