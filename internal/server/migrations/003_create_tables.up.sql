-- Create application tables.
-- Safe to re-run: uses IF NOT EXISTS.

CREATE TABLE IF NOT EXISTS client (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    name character varying(255) NOT NULL,
    port_start bigint,
    port_end bigint,
    is_delete boolean NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT timezone('utc', now()),
    updated_at timestamptz NOT NULL DEFAULT timezone('utc', now())
);

CREATE TABLE IF NOT EXISTS client_cert (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    client_id uuid NOT NULL,
    cert_path text NOT NULL,
    expired_at timestamptz NOT NULL,
    is_delete boolean NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT timezone('utc', now()),
    updated_at timestamptz NOT NULL DEFAULT timezone('utc', now())
);

CREATE TABLE IF NOT EXISTS client_proxy (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    client_id uuid NOT NULL,
    name character varying(255) NOT NULL,
    type character varying(255) NOT NULL,
    port character varying(255) NOT NULL,
    local_ip character varying(255) NOT NULL,
    local_port character varying(255) NOT NULL,
    status smallint NOT NULL,
    created_at timestamptz NOT NULL DEFAULT timezone('utc', now()),
    updated_at timestamptz NOT NULL DEFAULT timezone('utc', now())
);

CREATE TABLE IF NOT EXISTS access_log (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    client_id uuid NOT NULL,
    proxy_id uuid NOT NULL,
    ip character varying(128) NOT NULL,
    category character varying(128) NOT NULL,
    country character varying(256),
    region character varying(256),
    city character varying(256),
    status smallint NOT NULL,
    created_at timestamptz NOT NULL DEFAULT timezone('utc', now()),
    updated_at timestamptz NOT NULL DEFAULT timezone('utc', now())
);

CREATE TABLE IF NOT EXISTS access_rule (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    ip character varying(128),
    city character varying(256),
    region character varying(256),
    country character varying(256),
    category character varying(128),
    status smallint NOT NULL,
    is_delete boolean NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT timezone('utc', now()),
    updated_at timestamptz NOT NULL DEFAULT timezone('utc', now())
);
