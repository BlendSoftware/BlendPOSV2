-- Add CHECK constraint to unidad_medida (column already exists from 000001)
-- Only allow known values: unidad, kg, gramo
ALTER TABLE productos DROP CONSTRAINT IF EXISTS chk_unidad_medida;
ALTER TABLE productos ADD CONSTRAINT chk_unidad_medida
    CHECK (unidad_medida IN ('unidad', 'kg', 'gramo'));

-- Add weight field to venta_items for weight-based sales
ALTER TABLE venta_items ADD COLUMN IF NOT EXISTS peso NUMERIC(10,3);
