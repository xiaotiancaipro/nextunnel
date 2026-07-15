-- Rename legacy proxy table to client_proxy.
-- Safe to re-run: only renames when proxy exists and client_proxy does not.

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.tables
        WHERE table_schema = current_schema()
          AND table_name = 'proxy'
    ) AND NOT EXISTS (
        SELECT 1
        FROM information_schema.tables
        WHERE table_schema = current_schema()
          AND table_name = 'client_proxy'
    ) THEN
        ALTER TABLE proxy RENAME TO client_proxy;
    END IF;
END $$;
