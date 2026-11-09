CREATE TABLE IF NOT EXISTS clusters (
    name         TEXT PRIMARY KEY,
    last_seen    TIMESTAMP NOT NULL,
    health_score FLOAT,
    ready_nodes  INT,
    total_nodes  INT,
    running_pods INT,
    pending_pods INT,
    failed_pods  INT,
    crashloop    INT,
    namespaces   TEXT
);

CREATE TABLE IF NOT EXISTS incidents (
    id                   TEXT PRIMARY KEY,
    cluster              TEXT NOT NULL REFERENCES clusters(name),
    namespace            TEXT,
    severity             TEXT,
    status               TEXT NOT NULL DEFAULT 'INVESTIGATING',
    title                TEXT,
    category             TEXT,
    root_cause           TEXT,
    contributing_factors JSONB,
    affected_services    JSONB,
    affected_namespaces  JSONB,
    remediation_plan     JSONB,
    llm_model            TEXT,
    confidence           INT,
    has_issue            BOOLEAN,
    detected_at          TIMESTAMP NOT NULL,
    resolved_at          TIMESTAMP,
    scan_duration_secs   FLOAT
);

CREATE TABLE IF NOT EXISTS remediation_log (
    id           SERIAL PRIMARY KEY,
    incident_id  TEXT REFERENCES incidents(id),
    cluster      TEXT,
    step_number  INT,
    description  TEXT,
    command      TEXT,
    status       TEXT,
    executed_at  TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_incidents_cluster ON incidents(cluster);
CREATE INDEX IF NOT EXISTS idx_incidents_detected_at ON incidents(detected_at DESC);
CREATE INDEX IF NOT EXISTS idx_incidents_severity ON incidents(severity);
