-- Rollback migration 000040: Remove new plan columns and restore old plan names
ALTER TABLE plans DROP COLUMN IF EXISTS max_sucursales;
ALTER TABLE plans DROP COLUMN IF EXISTS max_usuarios;

-- Restore original plan names
UPDATE plans SET nombre = 'Kiosco'   WHERE id = '00000000-0000-0000-0000-000000000001';
UPDATE plans SET nombre = 'Starter'  WHERE id = '00000000-0000-0000-0000-000000000002';
UPDATE plans SET nombre = 'Pro'      WHERE id = '00000000-0000-0000-0000-000000000003';
