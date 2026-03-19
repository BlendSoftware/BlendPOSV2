-- Variant attributes for products (size, color, etc.)
-- Example: {"talle": "M", "color": "Azul"}
-- The parent product (es_padre=true) has variante_atributos = '{}'
-- Each child variant has specific attributes filled in.
ALTER TABLE productos ADD COLUMN IF NOT EXISTS variante_atributos JSONB DEFAULT '{}';

-- Variant display name (auto-generated from parent name + attributes)
-- Example: "Remera Básica - M / Azul"
ALTER TABLE productos ADD COLUMN IF NOT EXISTS variante_nombre VARCHAR(200);

-- padre_id links a variant to its parent product
ALTER TABLE productos ADD COLUMN IF NOT EXISTS padre_id UUID REFERENCES productos(id);

-- Index for fast lookup of variants by parent
CREATE INDEX IF NOT EXISTS idx_productos_padre_id ON productos(padre_id) WHERE padre_id IS NOT NULL;
