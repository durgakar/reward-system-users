package store

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/durgakar/reward-system-users/internal/domain"
	"github.com/durgakar/reward-system-users/pkg/plugin"
)

// MemoryStore is an in-process Store for local dev when PostgreSQL is unavailable.
type MemoryStore struct {
	mu            sync.RWMutex
	clients       map[string]plugin.Client
	profiles      map[string]plugin.ClientProfile
	segments      map[string]domain.SegmentDefinition
	rules         map[string]domain.RuleDefinition
	balances      map[string]int
	transactions  map[string]struct{}
	campaignRuns  []CampaignRun
	nextRunID     int64
	seeded        bool
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		clients:      make(map[string]plugin.Client),
		profiles:     make(map[string]plugin.ClientProfile),
		segments:     make(map[string]domain.SegmentDefinition),
		rules:        make(map[string]domain.RuleDefinition),
		balances:     make(map[string]int),
		transactions: make(map[string]struct{}),
	}
}

func (s *MemoryStore) Close() error { return nil }

func (s *MemoryStore) Ping(_ context.Context) error { return nil }

func (s *MemoryStore) Migrate(_ context.Context, _ string) error { return nil }

func (s *MemoryStore) ListClients(_ context.Context) ([]plugin.Client, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]plugin.Client, 0, len(s.clients))
	for _, c := range s.clients {
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (s *MemoryStore) GetProfile(_ context.Context, clientID string) (plugin.ClientProfile, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.profiles[clientID]
	if !ok {
		return plugin.ClientProfile{}, fmt.Errorf("client %q not found", clientID)
	}
	return p, nil
}

func (s *MemoryStore) UpsertClient(_ context.Context, c plugin.Client, p plugin.ClientProfile) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clients[c.ID] = c
	s.profiles[p.ClientID] = p
	return nil
}

func (s *MemoryStore) ListSegments(_ context.Context) ([]domain.SegmentDefinition, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]domain.SegmentDefinition, 0, len(s.segments))
	for _, seg := range s.segments {
		out = append(out, seg)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (s *MemoryStore) GetSegment(_ context.Context, id string) (domain.SegmentDefinition, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	seg, ok := s.segments[id]
	if !ok {
		return domain.SegmentDefinition{}, fmt.Errorf("segment %q not found", id)
	}
	return seg, nil
}

func (s *MemoryStore) UpsertSegment(_ context.Context, seg domain.SegmentDefinition) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.segments[seg.ID] = seg
	return nil
}

func (s *MemoryStore) DeleteSegment(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.segments, id)
	return nil
}

func (s *MemoryStore) ListRules(_ context.Context) ([]domain.RuleDefinition, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]domain.RuleDefinition, 0, len(s.rules))
	for _, r := range s.rules {
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (s *MemoryStore) GetRule(_ context.Context, id string) (domain.RuleDefinition, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, ok := s.rules[id]
	if !ok {
		return domain.RuleDefinition{}, fmt.Errorf("rule %q not found", id)
	}
	return r, nil
}

func (s *MemoryStore) UpsertRule(_ context.Context, rule domain.RuleDefinition) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rules[rule.ID] = rule
	return nil
}

func (s *MemoryStore) DeleteRule(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.rules, id)
	return nil
}

func (s *MemoryStore) AwardPoints(_ context.Context, req plugin.AwardRequest, provider string) (*plugin.AwardResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.transactions[req.ReferenceID]; ok {
		return &plugin.AwardResult{
			TransactionID: req.ReferenceID,
			NewBalance:    s.balances[req.ClientID],
			Provider:      provider,
		}, nil
	}
	s.transactions[req.ReferenceID] = struct{}{}
	s.balances[req.ClientID] += req.Points
	return &plugin.AwardResult{
		TransactionID: req.ReferenceID,
		NewBalance:    s.balances[req.ClientID],
		Provider:      provider,
	}, nil
}

func (s *MemoryStore) GetBalance(_ context.Context, clientID string) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.balances[clientID], nil
}

func (s *MemoryStore) StartCampaignRun(_ context.Context, campaignID string) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextRunID++
	id := s.nextRunID
	s.campaignRuns = append(s.campaignRuns, CampaignRun{
		ID: id, CampaignID: campaignID, Status: "running", StartedAt: time.Now(),
	})
	return id, nil
}

func (s *MemoryStore) FinishCampaignRun(_ context.Context, runID int64, status string, summary CampaignSummary) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.campaignRuns {
		if s.campaignRuns[i].ID == runID {
			now := time.Now()
			s.campaignRuns[i].Status = status
			s.campaignRuns[i].ClientsProcessed = summary.ClientsProcessed
			s.campaignRuns[i].PointsAwarded = summary.PointsAwarded
			s.campaignRuns[i].EmailsSent = summary.EmailsSent
			s.campaignRuns[i].ErrorsCount = summary.ErrorsCount
			s.campaignRuns[i].FinishedAt = &now
			return nil
		}
	}
	return fmt.Errorf("run %d not found", runID)
}

func (s *MemoryStore) ListCampaignRuns(_ context.Context, limit int) ([]CampaignRun, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if limit <= 0 {
		limit = 20
	}
	runs := make([]CampaignRun, len(s.campaignRuns))
	copy(runs, s.campaignRuns)
	sort.Slice(runs, func(i, j int) bool { return runs[i].StartedAt.After(runs[j].StartedAt) })
	if len(runs) > limit {
		runs = runs[:limit]
	}
	return runs, nil
}

func (s *MemoryStore) SeedIfEmpty(ctx context.Context) error {
	s.mu.Lock()
	if s.seeded {
		s.mu.Unlock()
		return nil
	}
	s.seeded = true
	s.mu.Unlock()

	clients := []struct {
		c plugin.Client
		p plugin.ClientProfile
	}{
		{plugin.Client{ID: "c-001", Email: "alice@example.com", FirstName: "Alice", LastName: "Nguyen"},
			plugin.ClientProfile{ClientID: "c-001", LifetimeSpendUSD: 820.5, LastOrderTotalUSD: 145, LastOrderAt: time.Date(2026, 5, 20, 0, 0, 0, 0, time.UTC), OrdersLast90Days: 4, AverageOrderUSD: 95, PreferredCategory: "electronics", DaysSinceLastOrder: 10}},
		{plugin.Client{ID: "c-002", Email: "bob@example.com", FirstName: "Bob", LastName: "Smith"},
			plugin.ClientProfile{ClientID: "c-002", LifetimeSpendUSD: 310, LastOrderTotalUSD: 45, LastOrderAt: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC), OrdersLast90Days: 2, AverageOrderUSD: 55, PreferredCategory: "home", DaysSinceLastOrder: 29}},
		{plugin.Client{ID: "c-003", Email: "cara@example.com", FirstName: "Cara", LastName: "Lee"},
			plugin.ClientProfile{ClientID: "c-003", LifetimeSpendUSD: 1200, LastOrderTotalUSD: 210, LastOrderAt: time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC), OrdersLast90Days: 5, AverageOrderUSD: 120, PreferredCategory: "electronics", DaysSinceLastOrder: 12}},
		{plugin.Client{ID: "c-004", Email: "dan@example.com", FirstName: "Dan", LastName: "Patel"},
			plugin.ClientProfile{ClientID: "c-004", LifetimeSpendUSD: 180, LastOrderTotalUSD: 0, LastOrderAt: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC), OrdersLast90Days: 0, AverageOrderUSD: 60, PreferredCategory: "fashion", DaysSinceLastOrder: 118}},
		{plugin.Client{ID: "c-005", Email: "eva@example.com", FirstName: "Eva", LastName: "Garcia"},
			plugin.ClientProfile{ClientID: "c-005", LifetimeSpendUSD: 540, LastOrderTotalUSD: 88, LastOrderAt: time.Date(2026, 5, 25, 0, 0, 0, 0, time.UTC), OrdersLast90Days: 3, AverageOrderUSD: 72, PreferredCategory: "electronics", DaysSinceLastOrder: 5}},
	}
	for _, row := range clients {
		_ = s.UpsertClient(ctx, row.c, row.p)
	}

	segments := []domain.SegmentDefinition{
		{ID: "high_spender", Description: "Clients with lifetime spend over $500", Match: domain.MatchAll{All: []domain.Condition{{Field: "lifetime_spend_usd", Operator: domain.OpGTE, Value: 500}}}},
		{ID: "frequent_buyer", Description: "At least 3 orders in the last 90 days", Match: domain.MatchAll{All: []domain.Condition{{Field: "orders_last_90_days", Operator: domain.OpGTE, Value: 3}}}},
		{ID: "at_risk", Description: "No purchase in 60+ days but historically active", Match: domain.MatchAll{All: []domain.Condition{{Field: "days_since_last_order", Operator: domain.OpGTE, Value: 60}, {Field: "lifetime_spend_usd", Operator: domain.OpGTE, Value: 200}}}},
		{ID: "electronics_fan", Description: "Prefers electronics category", Match: domain.MatchAll{All: []domain.Condition{{Field: "preferred_category", Operator: domain.OpEQ, Value: "electronics"}}}},
	}
	for _, seg := range segments {
		_ = s.UpsertSegment(ctx, seg)
	}

	rules := []domain.RuleDefinition{
		{ID: "high_order_bonus", Name: "500 points for orders $100+", Description: "Award 500 points when the latest order exceeds $100 USD", Segment: "high_spender", Enabled: true,
			Condition: domain.Condition{Field: "last_order_total_usd", Operator: domain.OpGTE, Value: 100},
			Actions: []domain.Action{{Type: domain.ActionAwardPoints, Points: 500}, {Type: domain.ActionSendEmail, Template: "high_spender_bonus", Subject: "You earned {{.Points}} reward points, {{.Client.FirstName}}!"}}},
		{ID: "frequent_buyer_thanks", Name: "Thank frequent buyers", Description: "Send a thank-you email to frequent buyers (no points)", Segment: "frequent_buyer", Enabled: true,
			Condition: domain.Condition{Field: "orders_last_90_days", Operator: domain.OpGTE, Value: 3},
			Actions: []domain.Action{{Type: domain.ActionSendEmail, Template: "frequent_buyer_thanks", Subject: "Thanks for shopping with us, {{.Client.FirstName}}"}}},
		{ID: "win_back_at_risk", Name: "Win-back bonus for inactive clients", Description: "250 points when at-risk clients return with any order", Segment: "at_risk", Enabled: true,
			Condition: domain.Condition{Field: "last_order_total_usd", Operator: domain.OpGTE, Value: 1},
			Actions: []domain.Action{{Type: domain.ActionAwardPoints, Points: 250}, {Type: domain.ActionSendEmail, Template: "welcome_back", Subject: "Welcome back — {{.Points}} points added to your account"}}},
		{ID: "electronics_milestone", Name: "Electronics category milestone", Description: "100 points when electronics fans spend $75+ on latest order", Segment: "electronics_fan", Enabled: true,
			Condition: domain.Condition{Field: "last_order_total_usd", Operator: domain.OpGTE, Value: 75},
			Actions: []domain.Action{{Type: domain.ActionSendEmail, Template: "category_bonus", Subject: "Electronics bonus — {{.Points}} points for you"}, {Type: domain.ActionAwardPoints, Points: 100}}},
	}
	for _, r := range rules {
		_ = s.UpsertRule(ctx, r)
	}
	return nil
}
