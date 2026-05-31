# reward-system-users

Pluggable Go service for **segment-based email campaigns** and **rule-driven loyalty rewards**. Clients are grouped by shopping habits; YAML rules decide when to award points and which email template to send.

## Architecture

```
ShoppingSource ──► SegmentProvider ──► RuleEngine ──► Actions
     (CSV)            (YAML)           (YAML)      ├─ RewardProvider (ledger / Open Loyalty / Voucherify)
                                                   └─ EmailSender (stdout / SMTP)
```

Every integration point is a Go interface in [`pkg/plugin`](pkg/plugin/plugin.go). Swap implementations via `config/config.yaml` without changing the orchestrator.

| Plugin | Interface | Built-in options |
|--------|-----------|------------------|
| Client data | `ShoppingSource` | `csv` (demo) — add Shopify, PostgreSQL, etc. |
| Segments | `SegmentProvider` | `static` (YAML) — add CRM rules |
| Rules | YAML + `RuleEngine` | Operators: `eq`, `gte`, `lte`, `in`, … |
| Rewards | `RewardProvider` | `ledger`, `open_loyalty`, `voucherify` |
| Email | `EmailSender` | `stdout`, `smtp` |

## Quick start

**Requirements:** Go 1.22+

```bash
# Install Go on macOS (if needed)
brew install go

# Clone to ~/reward-system-users (matches GitHub repo name)
git clone https://github.com/durgakar/reward-system-users.git ~/reward-system-users
cd ~/reward-system-users

# Or use the helper script:
# ./scripts/setup-local.sh

go mod tidy
make dry-run    # preview matches without side effects
make run        # award points + print emails to stdout
make test
make build      # produces bin/reward-system-users
```

**Open in Cursor:** File → Open Folder → `~/reward-system-users`

## Example rule

From [`config/rules.yaml`](config/rules.yaml):

```yaml
- id: high_order_bonus
  name: 500 points for orders $100+
  segment: high_spender          # only clients in this segment
  enabled: true
  condition:
    field: last_order_total_usd
    operator: gte
    value: 100
  actions:
    - type: award_points
      points: 500
    - type: send_email
      template: high_spender_bonus
      subject: "You earned {{.Points}} reward points, {{.Client.FirstName}}!"
```

## Open-source & external reward systems

We recommend a **progressive integration path**:

### 1. Internal ledger (default) — start here

The built-in `ledger` provider keeps points in memory with idempotent awards. Zero dependencies; ideal for local dev and unit tests.

```yaml
reward_provider: ledger
```

### 2. [Open Loyalty](https://github.com/OpenLoyalty/Open-Loyalty) — self-hosted OSS

Full-featured loyalty platform (points, tiers, campaigns). PHP/Symfony stack; run via Docker and point this project at its REST API.

```yaml
reward_provider: open_loyalty
open_loyalty:
  base_url: http://localhost:8181/api
  api_key: YOUR_TOKEN
  store_code: default
```

**Suggested next step:** deploy Open Loyalty locally, create a customer schema matching your `client_id` / email, then switch `reward_provider` from `ledger` to `open_loyalty`.

### 3. [Voucherify](https://www.voucherify.io/) — managed promotions API

Not open source, but excellent for **rule-based promotions** and loyalty campaigns with a REST API and sandbox. Useful when you want production-grade idempotency and analytics before self-hosting.

```yaml
reward_provider: voucherify
voucherify:
  application_id: YOUR_APP_ID
  secret_key: YOUR_SECRET_KEY
```

### 4. Other systems worth evaluating

| System | Type | Notes |
|--------|------|-------|
| [Talon.One](https://www.talon.one/) | Promotion engine | Rule-based cart & loyalty; strong for complex stacking rules |
| [LoyaltyLion](https://loyaltylion.com/) | SaaS | Shopify-focused; use via webhook adapter |
| [Medusa](https://github.com/medusajs/medusa) | OSS commerce | Build a `ShoppingSource` that reads Medusa orders |
| Custom PostgreSQL ledger | Roll your own | Implement `RewardProvider` with your schema |

## Email delivery

**Development:** `email_sender: stdout` prints emails to the terminal.

**Local SMTP testing** with Mailhog:

```bash
make docker-mail
# UI: http://localhost:8025
```

Then set in config:

```yaml
email_sender: smtp
smtp:
  host: localhost
  port: 1025
  from: rewards@example.com
```

**Production:** point SMTP at SendGrid, Amazon SES, Postmark, etc.

## Adding a custom plugin

1. Implement the interface in `pkg/plugin`.
2. Register it in [`internal/app/app.go`](internal/app/app.go).
3. Select it in `config/config.yaml`.

Example skeleton:

```go
type ShopifySource struct{}

func (s *ShopifySource) Name() string { return "shopify" }
func (s *ShopifySource) ListClients(ctx context.Context) ([]plugin.Client, error) { ... }
func (s *ShopifySource) GetProfile(ctx context.Context, id string) (plugin.ClientProfile, error) { ... }
```

## Project layout

```
cmd/reward-system-users/  CLI entrypoint
pkg/plugin/           Public extension interfaces
internal/
  campaign/           Orchestrator
  rules/              YAML rule engine
  segment/            Segment evaluation
  shopping/           Data sources
  email/              Templates + senders
  rewards/            Loyalty backends
config/               Segments, rules, runtime config
templates/            HTML email templates
data/                 Sample CSV clients
```

## License

MIT (add your license file as needed)
