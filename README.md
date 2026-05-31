# reward-system-users

Production-ready Go service for **segment-based email campaigns** and **rule-driven loyalty rewards**.

## Features

- **PostgreSQL / MySQL** — clients, profiles, segments, rules, ledger, campaign history
- **HTTP API** — REST endpoints for CRUD + manual campaign runs
- **Admin UI** — edit segments/rules at `/admin` (no YAML editing)
- **Cron scheduler** — daily campaign runs (`cron.enabled`)
- **Pluggable rewards** — `db_ledger`, in-memory `ledger`, **Open Loyalty**, **Voucherify**
- **Docker + CI** — `docker compose up`, GitHub Actions

## Quick start (Docker — recommended)

```bash
git clone https://github.com/durgakar/reward-system-users.git
cd reward-system-users
docker compose up --build
```

- App: http://localhost:8080
- Admin UI: http://localhost:8080/admin
- Mailhog: http://localhost:8025
- API key (default): `dev-key` — set header `Authorization: Bearer dev-key`

## Local development (Go)

Requires **Go 1.25+**

```bash
brew install go
go mod tidy

# File/CSV mode (no database):
make dry-run

# PostgreSQL mode:
docker compose up -d postgres mailhog
make migrate
make serve
```

### Commands

| Command | Description |
|---------|-------------|
| `reward-system-users serve` | HTTP API + admin UI + cron |
| `reward-system-users migrate` | Apply DB schema + seed data |
| `reward-system-users run` | One-shot campaign (CLI) |

## Architecture

```
PostgreSQL ──► ShoppingSource ──► SegmentProvider ──► RuleEngine ──► Actions
   ▲              (database)        (database)         (database)    ├─ db_ledger / Open Loyalty / Voucherify
   │                                                                    └─ SMTP / stdout
 Admin UI / REST API
```

## API (v1)

All `/api/v1/*` routes require `Authorization: Bearer <admin_api_key>`.

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Liveness |
| GET | `/ready` | DB connectivity |
| GET | `/api/v1/clients` | List clients |
| GET/PUT/DELETE | `/api/v1/segments/{id}` | Manage segments |
| GET/PUT/DELETE | `/api/v1/rules/{id}` | Manage rules |
| POST | `/api/v1/campaigns/run` | Run campaign now |
| GET | `/api/v1/campaigns/runs` | Campaign history |

## Reward integrations

Set in config or environment:

```yaml
reward_provider: db_ledger   # default — PostgreSQL ledger
# reward_provider: open_loyalty
# reward_provider: voucherify
```

Environment variables:

- `OPEN_LOYALTY_API_KEY`
- `VOUCHERIFY_APP_ID`, `VOUCHERIFY_SECRET_KEY`, `VOUCHERIFY_LOYALTY_ID`
- `DATABASE_URL`, `ADMIN_API_KEY`, `REWARD_PROVIDER`

## Production checklist

1. Set strong `server.admin_api_key`
2. Use real SMTP (SendGrid, SES, Postmark)
3. Point `reward_provider` at Open Loyalty or Voucherify
4. Deploy via Docker or Kubernetes using included `Dockerfile`
5. CI runs on every push to `main` (`.github/workflows/ci.yml`)

## Legacy file/CSV mode

For demos without a database, use `config/config.example.yaml`:

```yaml
rules_source: file
segments_source: file
shopping_source: csv
reward_provider: ledger
```

## License

MIT
