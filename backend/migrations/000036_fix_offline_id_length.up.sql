-- Fix: offline_id format is {tenantId}:{deviceId}:{timestamp}:{random} (~80 chars), not UUID (36 chars)
ALTER TABLE ventas ALTER COLUMN offline_id TYPE VARCHAR(150);
