-- client: GetByName/Create -> unique name only among active (non-deleted) clients
-- Safe to re-run: DROP uses IF EXISTS; CREATE uses IF NOT EXISTS.

ALTER TABLE client DROP CONSTRAINT IF EXISTS uni_client_name;
ALTER TABLE client DROP CONSTRAINT IF EXISTS client_name_key;
DROP INDEX IF EXISTS uni_client_name;
DROP INDEX IF EXISTS idx_client_name;

CREATE UNIQUE INDEX IF NOT EXISTS uk_client_name_active
    ON client (name)
    WHERE is_delete = false;
