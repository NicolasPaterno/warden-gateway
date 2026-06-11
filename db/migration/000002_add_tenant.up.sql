ALTER TABLE readings ADD COLUMN IF NOT EXISTS tenant_id TEXT NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_readings_tenant
    ON readings (tenant_id, room, type, time DESC);
