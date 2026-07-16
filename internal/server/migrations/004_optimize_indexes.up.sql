-- Optimize indexes based on application query patterns.
-- Safe to re-run: all statements use IF NOT EXISTS.

-- client: List() -> WHERE is_delete = false ORDER BY created_at ASC
CREATE INDEX IF NOT EXISTS idx_client_active_created_at
    ON client (created_at ASC)
    WHERE is_delete = false;

-- client_cert: List/DeleteAllForClient -> WHERE client_id = ? AND is_delete = false [ORDER BY created_at]
CREATE INDEX IF NOT EXISTS idx_client_cert_client_active_created_at
    ON client_cert (client_id, created_at ASC)
    WHERE is_delete = false;

-- client_proxy: SyncFromApply/resolveProxyId -> WHERE client_id = ? [AND name = ?]
CREATE UNIQUE INDEX IF NOT EXISTS uk_proxy_client_name
    ON client_proxy (client_id, name);

-- access_rule: ListRules/cachedRules -> WHERE is_delete = false ORDER BY status DESC, created_at ASC
CREATE INDEX IF NOT EXISTS idx_access_rule_active_order
    ON access_rule (status DESC, created_at ASC)
    WHERE is_delete = false;

-- access_rule: UpsertRule/DeleteRule targetQuery -> exact match on rule dimensions
CREATE UNIQUE INDEX IF NOT EXISTS uk_access_rule_target
    ON access_rule (ip, country, region, city, category)
    WHERE is_delete = false;

-- access_log: append-heavy table; index for time-range queries and per-client history
CREATE INDEX IF NOT EXISTS idx_access_log_client_created_at
    ON access_log (client_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_access_log_created_at
    ON access_log (created_at DESC);
