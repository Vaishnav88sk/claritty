ALTER TABLE incidents ADD COLUMN received_at TIMESTAMPTZ;
UPDATE incidents SET received_at = detected_at AT TIME ZONE 'UTC';
ALTER TABLE incidents
    ALTER COLUMN received_at SET DEFAULT CURRENT_TIMESTAMP,
    ALTER COLUMN received_at SET NOT NULL;
CREATE INDEX idx_incidents_cluster_received ON incidents (cluster, received_at DESC);
