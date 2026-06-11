DROP INDEX IF EXISTS idx_readings_tenant;

ALTER TABLE readings DROP COLUMN IF EXISTS tenant_id;
