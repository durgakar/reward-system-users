-- PostgreSQL schema for reward-system-users

CREATE TABLE IF NOT EXISTS clients (
    id          TEXT PRIMARY KEY,
    email       TEXT NOT NULL UNIQUE,
    first_name  TEXT NOT NULL DEFAULT '',
    last_name   TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS client_profiles (
    client_id              TEXT PRIMARY KEY REFERENCES clients(id) ON DELETE CASCADE,
    lifetime_spend_usd     DOUBLE PRECISION NOT NULL DEFAULT 0,
    last_order_total_usd   DOUBLE PRECISION NOT NULL DEFAULT 0,
    last_order_at          TIMESTAMPTZ,
    orders_last_90_days    INT NOT NULL DEFAULT 0,
    average_order_usd      DOUBLE PRECISION NOT NULL DEFAULT 0,
    preferred_category     TEXT NOT NULL DEFAULT '',
    days_since_last_order  INT NOT NULL DEFAULT 0,
    updated_at             TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS segments (
    id          TEXT PRIMARY KEY,
    description TEXT NOT NULL DEFAULT '',
    match_json  JSONB NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS rules (
    id            TEXT PRIMARY KEY,
    name          TEXT NOT NULL,
    description   TEXT NOT NULL DEFAULT '',
    segment       TEXT NOT NULL DEFAULT '',
    condition_json JSONB NOT NULL,
    actions_json  JSONB NOT NULL,
    enabled       BOOLEAN NOT NULL DEFAULT TRUE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS point_balances (
    client_id  TEXT PRIMARY KEY REFERENCES clients(id) ON DELETE CASCADE,
    balance    INT NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS point_transactions (
    id            BIGSERIAL PRIMARY KEY,
    client_id     TEXT NOT NULL REFERENCES clients(id) ON DELETE CASCADE,
    points        INT NOT NULL,
    reason        TEXT NOT NULL DEFAULT '',
    rule_id       TEXT NOT NULL DEFAULT '',
    campaign_id   TEXT NOT NULL DEFAULT '',
    reference_id  TEXT NOT NULL UNIQUE,
    provider      TEXT NOT NULL DEFAULT 'db_ledger',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS campaign_runs (
    id                BIGSERIAL PRIMARY KEY,
    campaign_id       TEXT NOT NULL,
    status            TEXT NOT NULL DEFAULT 'running',
    clients_processed INT NOT NULL DEFAULT 0,
    points_awarded    INT NOT NULL DEFAULT 0,
    emails_sent       INT NOT NULL DEFAULT 0,
    errors_count      INT NOT NULL DEFAULT 0,
    started_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finished_at       TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_point_transactions_client ON point_transactions(client_id);
CREATE INDEX IF NOT EXISTS idx_campaign_runs_started ON campaign_runs(started_at DESC);
