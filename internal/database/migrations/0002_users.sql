CREATE TABLE users (
    id         INTEGER PRIMARY KEY,
    username   TEXT    NOT NULL UNIQUE COLLATE NOCASE,
    pass_hash  TEXT    NOT NULL,
    role       TEXT    NOT NULL CHECK (role IN ('admin', 'viewer')),
    created_at INTEGER NOT NULL
);

CREATE TABLE sessions (
    id         INTEGER PRIMARY KEY,
    user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT    NOT NULL UNIQUE,
    created_at INTEGER NOT NULL,
    last_used  INTEGER NOT NULL,
    expires_at INTEGER NOT NULL,
    user_agent TEXT    NOT NULL DEFAULT ''
);

CREATE INDEX idx_sessions_user ON sessions(user_id);

CREATE TABLE invites (
    id         INTEGER PRIMARY KEY,
    token_hash TEXT    NOT NULL UNIQUE,
    kind       TEXT    NOT NULL CHECK (kind IN ('join', 'reset')),
    role       TEXT    CHECK (role IN ('admin', 'viewer')),          -- join only
    user_id    INTEGER REFERENCES users(id) ON DELETE CASCADE,       -- reset only
    created_by INTEGER REFERENCES users(id) ON DELETE SET NULL,      -- NULL when issued by userctl
    created_at INTEGER NOT NULL,
    expires_at INTEGER NOT NULL,
    used_at    INTEGER
);

-- One rule per node, metric and direction; the settings screen edits them as a set.
CREATE UNIQUE INDEX idx_rules_node_metric ON rules(node_id, metric, op);

-- Nodes registered before rules existed get the same defaults a new node does.
INSERT INTO rules (node_id, metric, op, threshold, for_min, enabled)
SELECT n.id, d.metric, d.op, d.threshold, d.for_min, d.enabled
FROM nodes n
CROSS JOIN (
    SELECT 'rh' AS metric, 'gt' AS op, 65.0 AS threshold, 60 AS for_min, 1 AS enabled
    UNION ALL SELECT 'rh',      'lt', 30.0,  60, 1
    UNION ALL SELECT 'temp',    'gt', 30.0,  30, 0
    UNION ALL SELECT 'temp',    'lt', 10.0,  30, 0
    UNION ALL SELECT 'vbat',    'lt', 3.4,  180, 1
    UNION ALL SELECT 'offline', 'gt', 0.0,    0, 1
) d
WHERE NOT EXISTS (SELECT 1 FROM rules r WHERE r.node_id = n.id);
