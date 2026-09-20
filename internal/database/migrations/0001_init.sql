CREATE TABLE nodes (
    id         INTEGER PRIMARY KEY,
    slug       TEXT    NOT NULL UNIQUE,
    name       TEXT    NOT NULL,
    location   TEXT    NOT NULL DEFAULT '',
    token_hash TEXT    NOT NULL,
    interval_s INTEGER NOT NULL DEFAULT 900,
    enabled    INTEGER NOT NULL DEFAULT 1,
    created_at INTEGER NOT NULL,
    last_seen  INTEGER
);

CREATE UNIQUE INDEX idx_nodes_token ON nodes(token_hash);

CREATE TABLE readings (
    id          INTEGER PRIMARY KEY,
    node_id     INTEGER NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    received_at INTEGER NOT NULL,
    rh          REAL    NOT NULL,
    temp        REAL,
    vbat        REAL,
    rssi        INTEGER
);

CREATE INDEX idx_readings_node_time ON readings(node_id, received_at DESC);

-- Hourly rollups, filled by the retention job once raw rows age out.
CREATE TABLE readings_hourly (
    node_id  INTEGER NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    hour     INTEGER NOT NULL,
    samples  INTEGER NOT NULL,
    rh_min   REAL, rh_avg REAL, rh_max REAL,
    t_min    REAL, t_avg  REAL, t_max  REAL,
    vbat_min REAL,
    PRIMARY KEY (node_id, hour)
) WITHOUT ROWID;

CREATE TABLE rules (
    id        INTEGER PRIMARY KEY,
    node_id   INTEGER REFERENCES nodes(id) ON DELETE CASCADE, -- NULL applies to every node
    metric    TEXT    NOT NULL,
    op        TEXT    NOT NULL,
    threshold REAL    NOT NULL,
    for_min   INTEGER NOT NULL DEFAULT 0,
    enabled   INTEGER NOT NULL DEFAULT 1
);

CREATE TABLE alerts (
    id          INTEGER PRIMARY KEY,
    rule_id     INTEGER NOT NULL REFERENCES rules(id) ON DELETE CASCADE,
    node_id     INTEGER NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
    fired_at    INTEGER NOT NULL,
    resolved_at INTEGER,
    value       REAL
);

CREATE INDEX idx_alerts_open ON alerts(rule_id, node_id) WHERE resolved_at IS NULL;
