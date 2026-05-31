-- MySQL schema for reward-system-users

CREATE TABLE IF NOT EXISTS clients (
    id          VARCHAR(64) PRIMARY KEY,
    email       VARCHAR(255) NOT NULL UNIQUE,
    first_name  VARCHAR(128) NOT NULL DEFAULT '',
    last_name   VARCHAR(128) NOT NULL DEFAULT '',
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS client_profiles (
    client_id              VARCHAR(64) PRIMARY KEY,
    lifetime_spend_usd     DOUBLE NOT NULL DEFAULT 0,
    last_order_total_usd   DOUBLE NOT NULL DEFAULT 0,
    last_order_at          TIMESTAMP NULL,
    orders_last_90_days    INT NOT NULL DEFAULT 0,
    average_order_usd      DOUBLE NOT NULL DEFAULT 0,
    preferred_category     VARCHAR(64) NOT NULL DEFAULT '',
    days_since_last_order  INT NOT NULL DEFAULT 0,
    updated_at             TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (client_id) REFERENCES clients(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS segments (
    id          VARCHAR(64) PRIMARY KEY,
    description TEXT NOT NULL,
    match_json  JSON NOT NULL,
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS rules (
    id             VARCHAR(64) PRIMARY KEY,
    name           VARCHAR(255) NOT NULL,
    description    TEXT NOT NULL,
    segment        VARCHAR(64) NOT NULL DEFAULT '',
    condition_json JSON NOT NULL,
    actions_json   JSON NOT NULL,
    enabled        BOOLEAN NOT NULL DEFAULT TRUE,
    created_at     TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at     TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS point_balances (
    client_id  VARCHAR(64) PRIMARY KEY,
    balance    INT NOT NULL DEFAULT 0,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (client_id) REFERENCES clients(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS point_transactions (
    id            BIGINT AUTO_INCREMENT PRIMARY KEY,
    client_id     VARCHAR(64) NOT NULL,
    points        INT NOT NULL,
    reason        TEXT NOT NULL,
    rule_id       VARCHAR(64) NOT NULL DEFAULT '',
    campaign_id   VARCHAR(64) NOT NULL DEFAULT '',
    reference_id  VARCHAR(255) NOT NULL UNIQUE,
    provider      VARCHAR(64) NOT NULL DEFAULT 'db_ledger',
    created_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (client_id) REFERENCES clients(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS campaign_runs (
    id                BIGINT AUTO_INCREMENT PRIMARY KEY,
    campaign_id       VARCHAR(64) NOT NULL,
    status            VARCHAR(32) NOT NULL DEFAULT 'running',
    clients_processed INT NOT NULL DEFAULT 0,
    points_awarded    INT NOT NULL DEFAULT 0,
    emails_sent       INT NOT NULL DEFAULT 0,
    errors_count      INT NOT NULL DEFAULT 0,
    started_at        TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    finished_at       TIMESTAMP NULL
);

CREATE INDEX idx_point_transactions_client ON point_transactions(client_id);
CREATE INDEX idx_campaign_runs_started ON campaign_runs(started_at);
