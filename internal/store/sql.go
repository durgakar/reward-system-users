package store

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"os"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/durgakar/reward-system-users/internal/domain"
	"github.com/durgakar/reward-system-users/pkg/plugin"
)

//go:embed seed.sql
var seedSQL embed.FS

type SQLStore struct {
	db     *sql.DB
	driver string
}

type Options struct {
	Driver string // postgres or mysql
	URL    string
}

func Open(opts Options) (*SQLStore, error) {
	driver := opts.Driver
	if driver == "" {
		driver = "postgres"
	}
	dsn := opts.URL
	if dsn == "" {
		dsn = os.Getenv("DATABASE_URL")
	}
	if dsn == "" {
		return nil, fmt.Errorf("database URL is required")
	}

	dbDriver := driver
	if driver == "postgres" {
		dbDriver = "pgx"
	}

	db, err := sql.Open(dbDriver, dsn)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)

	s := &SQLStore{db: db, driver: driver}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := s.Ping(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping db: %w", err)
	}
	return s, nil
}

func (s *SQLStore) Close() error { return s.db.Close() }

func (s *SQLStore) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

func (s *SQLStore) Migrate(ctx context.Context, sql string) error {
	for _, stmt := range splitSQL(sql) {
		if _, err := s.db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("migrate stmt: %w", err)
		}
	}
	return nil
}

func splitSQL(sql string) []string {
	parts := strings.Split(sql, ";")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func (s *SQLStore) ph(q string) string {
	if s.driver != "mysql" {
		return q
	}
	// naive $N -> ? conversion for simple queries
	for i := 20; i >= 1; i-- {
		q = strings.ReplaceAll(q, fmt.Sprintf("$%d", i), "?")
	}
	return q
}

func (s *SQLStore) ListClients(ctx context.Context) ([]plugin.Client, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, email, first_name, last_name FROM clients ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []plugin.Client
	for rows.Next() {
		var c plugin.Client
		if err := rows.Scan(&c.ID, &c.Email, &c.FirstName, &c.LastName); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *SQLStore) GetProfile(ctx context.Context, clientID string) (plugin.ClientProfile, error) {
	q := s.ph(`SELECT client_id, lifetime_spend_usd, last_order_total_usd, last_order_at,
		orders_last_90_days, average_order_usd, preferred_category, days_since_last_order
		FROM client_profiles WHERE client_id = $1`)
	var p plugin.ClientProfile
	var lastOrder sql.NullTime
	err := s.db.QueryRowContext(ctx, q, clientID).Scan(
		&p.ClientID, &p.LifetimeSpendUSD, &p.LastOrderTotalUSD, &lastOrder,
		&p.OrdersLast90Days, &p.AverageOrderUSD, &p.PreferredCategory, &p.DaysSinceLastOrder,
	)
	if err != nil {
		return plugin.ClientProfile{}, err
	}
	if lastOrder.Valid {
		p.LastOrderAt = lastOrder.Time
	}
	return p, nil
}

func (s *SQLStore) UpsertClient(ctx context.Context, c plugin.Client, p plugin.ClientProfile) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, s.ph(`INSERT INTO clients (id, email, first_name, last_name)
		VALUES ($1,$2,$3,$4) ON CONFLICT (id) DO UPDATE SET email=EXCLUDED.email, first_name=EXCLUDED.first_name, last_name=EXCLUDED.last_name`),
		c.ID, c.Email, c.FirstName, c.LastName); err != nil {
		return err
	}

	var lastOrder any
	if p.LastOrderAt.IsZero() {
		lastOrder = nil
	} else {
		lastOrder = p.LastOrderAt
	}

	if s.driver == "mysql" {
		_, err = tx.ExecContext(ctx, `INSERT INTO client_profiles
			(client_id, lifetime_spend_usd, last_order_total_usd, last_order_at, orders_last_90_days,
			 average_order_usd, preferred_category, days_since_last_order, updated_at)
			VALUES (?,?,?,?,?,?,?,?,NOW())
			ON DUPLICATE KEY UPDATE lifetime_spend_usd=VALUES(lifetime_spend_usd),
			last_order_total_usd=VALUES(last_order_total_usd), last_order_at=VALUES(last_order_at),
			orders_last_90_days=VALUES(orders_last_90_days), average_order_usd=VALUES(average_order_usd),
			preferred_category=VALUES(preferred_category), days_since_last_order=VALUES(days_since_last_order),
			updated_at=NOW()`,
			p.ClientID, p.LifetimeSpendUSD, p.LastOrderTotalUSD, lastOrder, p.OrdersLast90Days,
			p.AverageOrderUSD, p.PreferredCategory, p.DaysSinceLastOrder)
	} else {
		_, err = tx.ExecContext(ctx, `INSERT INTO client_profiles
			(client_id, lifetime_spend_usd, last_order_total_usd, last_order_at, orders_last_90_days,
			 average_order_usd, preferred_category, days_since_last_order, updated_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,NOW())
			ON CONFLICT (client_id) DO UPDATE SET
			lifetime_spend_usd=EXCLUDED.lifetime_spend_usd,
			last_order_total_usd=EXCLUDED.last_order_total_usd,
			last_order_at=EXCLUDED.last_order_at,
			orders_last_90_days=EXCLUDED.orders_last_90_days,
			average_order_usd=EXCLUDED.average_order_usd,
			preferred_category=EXCLUDED.preferred_category,
			days_since_last_order=EXCLUDED.days_since_last_order,
			updated_at=NOW()`,
			p.ClientID, p.LifetimeSpendUSD, p.LastOrderTotalUSD, lastOrder, p.OrdersLast90Days,
			p.AverageOrderUSD, p.PreferredCategory, p.DaysSinceLastOrder)
	}
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (s *SQLStore) ListSegments(ctx context.Context) ([]domain.SegmentDefinition, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, description, match_json FROM segments ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.SegmentDefinition
	for rows.Next() {
		var seg domain.SegmentDefinition
		var matchRaw []byte
		if err := rows.Scan(&seg.ID, &seg.Description, &matchRaw); err != nil {
			return nil, err
		}
		seg.Match, err = decodeMatch(matchRaw)
		if err != nil {
			return nil, err
		}
		out = append(out, seg)
	}
	return out, rows.Err()
}

func (s *SQLStore) GetSegment(ctx context.Context, id string) (domain.SegmentDefinition, error) {
	q := s.ph(`SELECT id, description, match_json FROM segments WHERE id = $1`)
	var seg domain.SegmentDefinition
	var matchRaw []byte
	err := s.db.QueryRowContext(ctx, q, id).Scan(&seg.ID, &seg.Description, &matchRaw)
	if err != nil {
		return domain.SegmentDefinition{}, err
	}
	seg.Match, err = decodeMatch(matchRaw)
	return seg, err
}

func (s *SQLStore) UpsertSegment(ctx context.Context, seg domain.SegmentDefinition) error {
	matchRaw, err := encodeMatch(seg.Match)
	if err != nil {
		return err
	}
	if s.driver == "mysql" {
		_, err = s.db.ExecContext(ctx, `INSERT INTO segments (id, description, match_json, updated_at)
			VALUES (?,?,?,NOW()) ON DUPLICATE KEY UPDATE description=VALUES(description),
			match_json=VALUES(match_json), updated_at=NOW()`, seg.ID, seg.Description, matchRaw)
		return err
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO segments (id, description, match_json, updated_at)
		VALUES ($1,$2,$3,NOW()) ON CONFLICT (id) DO UPDATE SET description=EXCLUDED.description,
		match_json=EXCLUDED.match_json, updated_at=NOW()`, seg.ID, seg.Description, matchRaw)
	return err
}

func (s *SQLStore) DeleteSegment(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, s.ph(`DELETE FROM segments WHERE id = $1`), id)
	return err
}

func (s *SQLStore) ListRules(ctx context.Context) ([]domain.RuleDefinition, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, name, description, segment, condition_json, actions_json, enabled FROM rules ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.RuleDefinition
	for rows.Next() {
		var r domain.RuleDefinition
		var condRaw, actRaw []byte
		if err := rows.Scan(&r.ID, &r.Name, &r.Description, &r.Segment, &condRaw, &actRaw, &r.Enabled); err != nil {
			return nil, err
		}
		r.Condition, err = decodeCondition(condRaw)
		if err != nil {
			return nil, err
		}
		r.Actions, err = decodeActions(actRaw)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *SQLStore) GetRule(ctx context.Context, id string) (domain.RuleDefinition, error) {
	q := s.ph(`SELECT id, name, description, segment, condition_json, actions_json, enabled FROM rules WHERE id = $1`)
	var r domain.RuleDefinition
	var condRaw, actRaw []byte
	err := s.db.QueryRowContext(ctx, q, id).Scan(&r.ID, &r.Name, &r.Description, &r.Segment, &condRaw, &actRaw, &r.Enabled)
	if err != nil {
		return domain.RuleDefinition{}, err
	}
	r.Condition, err = decodeCondition(condRaw)
	if err != nil {
		return r, err
	}
	r.Actions, err = decodeActions(actRaw)
	return r, err
}

func (s *SQLStore) UpsertRule(ctx context.Context, rule domain.RuleDefinition) error {
	condRaw, err := encodeCondition(rule.Condition)
	if err != nil {
		return err
	}
	actRaw, err := encodeActions(rule.Actions)
	if err != nil {
		return err
	}
	if s.driver == "mysql" {
		_, err = s.db.ExecContext(ctx, `INSERT INTO rules (id, name, description, segment, condition_json, actions_json, enabled, updated_at)
			VALUES (?,?,?,?,?,?,?,NOW()) ON DUPLICATE KEY UPDATE name=VALUES(name), description=VALUES(description),
			segment=VALUES(segment), condition_json=VALUES(condition_json), actions_json=VALUES(actions_json),
			enabled=VALUES(enabled), updated_at=NOW()`,
			rule.ID, rule.Name, rule.Description, rule.Segment, condRaw, actRaw, rule.Enabled)
		return err
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO rules (id, name, description, segment, condition_json, actions_json, enabled, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,NOW()) ON CONFLICT (id) DO UPDATE SET name=EXCLUDED.name,
		description=EXCLUDED.description, segment=EXCLUDED.segment, condition_json=EXCLUDED.condition_json,
		actions_json=EXCLUDED.actions_json, enabled=EXCLUDED.enabled, updated_at=NOW()`,
		rule.ID, rule.Name, rule.Description, rule.Segment, condRaw, actRaw, rule.Enabled)
	return err
}

func (s *SQLStore) DeleteRule(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, s.ph(`DELETE FROM rules WHERE id = $1`), id)
	return err
}

func (s *SQLStore) AwardPoints(ctx context.Context, req plugin.AwardRequest, provider string) (*plugin.AwardResult, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	var existing int
	err = tx.QueryRowContext(ctx, s.ph(`SELECT 1 FROM point_transactions WHERE reference_id = $1`), req.ReferenceID).Scan(&existing)
	if err == nil {
		bal, _ := s.getBalanceTx(ctx, tx, req.ClientID)
		return &plugin.AwardResult{TransactionID: req.ReferenceID, NewBalance: bal, Provider: provider}, nil
	}
	if err != sql.ErrNoRows {
		return nil, err
	}

	if _, err := tx.ExecContext(ctx, s.ph(`INSERT INTO point_transactions
		(client_id, points, reason, rule_id, campaign_id, reference_id, provider)
		VALUES ($1,$2,$3,$4,$5,$6,$7)`),
		req.ClientID, req.Points, req.Reason, req.RuleID, req.CampaignID, req.ReferenceID, provider); err != nil {
		return nil, err
	}

	if s.driver == "mysql" {
		_, err = tx.ExecContext(ctx, `INSERT INTO point_balances (client_id, balance, updated_at)
			VALUES (?, ?, NOW()) ON DUPLICATE KEY UPDATE balance = balance + VALUES(balance), updated_at = NOW()`,
			req.ClientID, req.Points)
	} else {
		_, err = tx.ExecContext(ctx, `INSERT INTO point_balances (client_id, balance, updated_at)
			VALUES ($1, $2, NOW()) ON CONFLICT (client_id) DO UPDATE SET
			balance = point_balances.balance + EXCLUDED.balance, updated_at = NOW()`,
			req.ClientID, req.Points)
	}
	if err != nil {
		return nil, err
	}

	bal, err := s.getBalanceTx(ctx, tx, req.ClientID)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &plugin.AwardResult{TransactionID: req.ReferenceID, NewBalance: bal, Provider: provider}, nil
}

func (s *SQLStore) getBalanceTx(ctx context.Context, tx *sql.Tx, clientID string) (int, error) {
	var bal int
	err := tx.QueryRowContext(ctx, s.ph(`SELECT balance FROM point_balances WHERE client_id = $1`), clientID).Scan(&bal)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return bal, err
}

func (s *SQLStore) GetBalance(ctx context.Context, clientID string) (int, error) {
	var bal int
	err := s.db.QueryRowContext(ctx, s.ph(`SELECT balance FROM point_balances WHERE client_id = $1`), clientID).Scan(&bal)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return bal, err
}

func (s *SQLStore) StartCampaignRun(ctx context.Context, campaignID string) (int64, error) {
	if s.driver == "mysql" {
		res, err := s.db.ExecContext(ctx, `INSERT INTO campaign_runs (campaign_id, status) VALUES (?, 'running')`, campaignID)
		if err != nil {
			return 0, err
		}
		return res.LastInsertId()
	}
	var id int64
	err := s.db.QueryRowContext(ctx, `INSERT INTO campaign_runs (campaign_id, status) VALUES ($1, 'running') RETURNING id`, campaignID).Scan(&id)
	return id, err
}

func (s *SQLStore) FinishCampaignRun(ctx context.Context, runID int64, status string, summary CampaignSummary) error {
	q := s.ph(`UPDATE campaign_runs SET status=$1, clients_processed=$2, points_awarded=$3,
		emails_sent=$4, errors_count=$5, finished_at=NOW() WHERE id=$6`)
	_, err := s.db.ExecContext(ctx, q, status, summary.ClientsProcessed, summary.PointsAwarded,
		summary.EmailsSent, summary.ErrorsCount, runID)
	return err
}

func (s *SQLStore) ListCampaignRuns(ctx context.Context, limit int) ([]CampaignRun, error) {
	if limit <= 0 {
		limit = 20
	}
	q := s.ph(`SELECT id, campaign_id, status, clients_processed, points_awarded, emails_sent,
		errors_count, started_at, finished_at FROM campaign_runs ORDER BY started_at DESC LIMIT $1`)
	rows, err := s.db.QueryContext(ctx, q, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []CampaignRun
	for rows.Next() {
		var r CampaignRun
		var finished sql.NullTime
		if err := rows.Scan(&r.ID, &r.CampaignID, &r.Status, &r.ClientsProcessed, &r.PointsAwarded,
			&r.EmailsSent, &r.ErrorsCount, &r.StartedAt, &finished); err != nil {
			return nil, err
		}
		if finished.Valid {
			t := finished.Time
			r.FinishedAt = &t
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *SQLStore) SeedIfEmpty(ctx context.Context) error {
	var n int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM clients`).Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	raw, err := seedSQL.ReadFile("seed.sql")
	if err != nil {
		return err
	}
	return s.Migrate(ctx, string(raw))
}
