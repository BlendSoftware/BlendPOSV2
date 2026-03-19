DROP INDEX IF EXISTS idx_productos_padre_id;
ALTER TABLE productos DROP COLUMN IF EXISTS padre_id;
ALTER TABLE productos DROP COLUMN IF EXISTS variante_nombre;
ALTER TABLE productos DROP COLUMN IF EXISTS variante_atributos;
