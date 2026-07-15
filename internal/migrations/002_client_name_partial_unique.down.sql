-- Roll back indexes added in 002_client_name_partial_unique.up.sql

DROP INDEX IF EXISTS uk_client_name_active;
