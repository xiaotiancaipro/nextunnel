-- Roll back rename of client_proxy to proxy.

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.tables
        WHERE table_schema = current_schema()
          AND table_name = 'client_proxy'
    ) AND NOT EXISTS (
        SELECT 1
        FROM information_schema.tables
        WHERE table_schema = current_schema()
          AND table_name = 'proxy'
    ) THEN
        ALTER TABLE client_proxy RENAME TO proxy;
    END IF;
END $$;
