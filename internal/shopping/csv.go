package shopping

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/durgakar/reward-system-users/pkg/plugin"
)

// CSVSource loads demo client + shopping data from a CSV file.
// Expected columns:
// client_id,email,first_name,last_name,lifetime_spend_usd,last_order_total_usd,
// last_order_at,orders_last_90_days,average_order_usd,preferred_category,days_since_last_order
type CSVSource struct {
	path string
}

func NewCSVSource(path string) *CSVSource {
	return &CSVSource{path: path}
}

func (s *CSVSource) Name() string { return "csv" }

func (s *CSVSource) ListClients(_ context.Context) ([]plugin.Client, error) {
	rows, err := s.readAll()
	if err != nil {
		return nil, err
	}
	clients := make([]plugin.Client, 0, len(rows))
	for _, row := range rows {
		clients = append(clients, plugin.Client{
			ID:        row.ClientID,
			Email:     row.Email,
			FirstName: row.FirstName,
			LastName:  row.LastName,
		})
	}
	return clients, nil
}

func (s *CSVSource) GetProfile(_ context.Context, clientID string) (plugin.ClientProfile, error) {
	rows, err := s.readAll()
	if err != nil {
		return plugin.ClientProfile{}, err
	}
	for _, row := range rows {
		if row.ClientID == clientID {
			return row.profile(), nil
		}
	}
	return plugin.ClientProfile{}, fmt.Errorf("client %q not found", clientID)
}

type csvRow struct {
	ClientID           string
	Email              string
	FirstName          string
	LastName           string
	LifetimeSpendUSD   float64
	LastOrderTotalUSD  float64
	LastOrderAt        time.Time
	OrdersLast90Days   int
	AverageOrderUSD    float64
	PreferredCategory  string
	DaysSinceLastOrder int
}

func (r csvRow) profile() plugin.ClientProfile {
	return plugin.ClientProfile{
		ClientID:           r.ClientID,
		LifetimeSpendUSD:   r.LifetimeSpendUSD,
		LastOrderTotalUSD:  r.LastOrderTotalUSD,
		LastOrderAt:        r.LastOrderAt,
		OrdersLast90Days:   r.OrdersLast90Days,
		AverageOrderUSD:    r.AverageOrderUSD,
		PreferredCategory:  r.PreferredCategory,
		DaysSinceLastOrder: r.DaysSinceLastOrder,
	}
}

func (s *CSVSource) readAll() ([]csvRow, error) {
	f, err := os.Open(s.path)
	if err != nil {
		return nil, fmt.Errorf("open csv: %w", err)
	}
	defer f.Close()

	reader := csv.NewReader(f)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("read csv: %w", err)
	}
	if len(records) < 2 {
		return nil, fmt.Errorf("csv must include header and at least one row")
	}

	header := indexHeader(records[0])
	required := []string{
		"client_id", "email", "first_name", "last_name",
		"lifetime_spend_usd", "last_order_total_usd", "last_order_at",
		"orders_last_90_days", "average_order_usd", "preferred_category", "days_since_last_order",
	}
	for _, col := range required {
		if _, ok := header[col]; !ok {
			return nil, fmt.Errorf("csv missing column %q", col)
		}
	}

	rows := make([]csvRow, 0, len(records)-1)
	for _, rec := range records[1:] {
		row, err := parseCSVRow(rec, header)
		if err != nil {
			return nil, err
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func indexHeader(cols []string) map[string]int {
	m := make(map[string]int, len(cols))
	for i, c := range cols {
		m[strings.TrimSpace(strings.ToLower(c))] = i
	}
	return m
}

func parseCSVRow(rec []string, header map[string]int) (csvRow, error) {
	get := func(name string) string {
		return strings.TrimSpace(rec[header[name]])
	}
	lifetime, err := strconv.ParseFloat(get("lifetime_spend_usd"), 64)
	if err != nil {
		return csvRow{}, fmt.Errorf("lifetime_spend_usd: %w", err)
	}
	lastOrder, err := strconv.ParseFloat(get("last_order_total_usd"), 64)
	if err != nil {
		return csvRow{}, fmt.Errorf("last_order_total_usd: %w", err)
	}
	lastAt, err := time.Parse("2006-01-02", get("last_order_at"))
	if err != nil {
		return csvRow{}, fmt.Errorf("last_order_at: %w", err)
	}
	orders90, err := strconv.Atoi(get("orders_last_90_days"))
	if err != nil {
		return csvRow{}, fmt.Errorf("orders_last_90_days: %w", err)
	}
	avg, err := strconv.ParseFloat(get("average_order_usd"), 64)
	if err != nil {
		return csvRow{}, fmt.Errorf("average_order_usd: %w", err)
	}
	daysSince, err := strconv.Atoi(get("days_since_last_order"))
	if err != nil {
		return csvRow{}, fmt.Errorf("days_since_last_order: %w", err)
	}

	return csvRow{
		ClientID:           get("client_id"),
		Email:              get("email"),
		FirstName:          get("first_name"),
		LastName:           get("last_name"),
		LifetimeSpendUSD:   lifetime,
		LastOrderTotalUSD:  lastOrder,
		LastOrderAt:        lastAt,
		OrdersLast90Days:   orders90,
		AverageOrderUSD:    avg,
		PreferredCategory:  get("preferred_category"),
		DaysSinceLastOrder: daysSince,
	}, nil
}
