package server

import (
	"net/http"
	"time"

	"github.com/durgakar/reward-system-users/internal/domain"
	"github.com/durgakar/reward-system-users/internal/rules"
	"github.com/durgakar/reward-system-users/internal/segment"
	"github.com/durgakar/reward-system-users/pkg/plugin"
)

type clientDetailResponse struct {
	ID             string    `json:"id"`
	Email          string    `json:"email"`
	FirstName      string    `json:"first_name"`
	LastName       string    `json:"last_name"`
	LifetimeSpend  float64   `json:"lifetime_spend_usd"`
	LastOrderUSD   float64   `json:"last_order_total_usd"`
	LastOrderAt    time.Time `json:"last_order_at"`
	Orders90Days   int       `json:"orders_last_90_days"`
	PreferredCat   string    `json:"preferred_category"`
	DaysSinceOrder int       `json:"days_since_last_order"`
	Segments       []string  `json:"segments"`
	PointsBalance  int       `json:"points_balance"`
	ExpectedRules  []string  `json:"expected_rules"`
}

func (s *Server) handleListClientDetails(w http.ResponseWriter, r *http.Request) {
	if !s.requireStore(w) {
		return
	}
	ctx := r.Context()
	clients, err := s.app.Store.ListClients(ctx)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err)
		return
	}

	segProvider := segment.NewStoreProvider(s.app.Store)
	rulesList, _ := s.app.Store.ListRules(ctx)

	out := make([]clientDetailResponse, 0, len(clients))
	for _, c := range clients {
		profile, err := s.app.Store.GetProfile(ctx, c.ID)
		if err != nil {
			continue
		}
		segments, _ := segProvider.Evaluate(ctx, c, profile)
		if segments == nil {
			segments = []string{}
		}
		balance, _ := s.app.Store.GetBalance(ctx, c.ID)
		expected := expectedRulesForClient(profile, segments, rulesList)
		if expected == nil {
			expected = []string{}
		}

		out = append(out, clientDetailResponse{
			ID: c.ID, Email: c.Email, FirstName: c.FirstName, LastName: c.LastName,
			LifetimeSpend: profile.LifetimeSpendUSD, LastOrderUSD: profile.LastOrderTotalUSD,
			LastOrderAt: profile.LastOrderAt, Orders90Days: profile.OrdersLast90Days,
			PreferredCat: profile.PreferredCategory, DaysSinceOrder: profile.DaysSinceLastOrder,
			Segments: segments, PointsBalance: balance,
			ExpectedRules: expected,
		})
	}
	writeJSON(w, http.StatusOK, out)
}

func expectedRulesForClient(profile plugin.ClientProfile, segments []string, rulesList []domain.RuleDefinition) []string {
	segmentSet := make(map[string]struct{}, len(segments))
	for _, seg := range segments {
		segmentSet[seg] = struct{}{}
	}

	var matched []string
	for _, rule := range rulesList {
		if !rule.Enabled {
			continue
		}
		if rule.Segment != "" {
			if _, ok := segmentSet[rule.Segment]; !ok {
				continue
			}
		}
		ok, err := rules.MatchCondition(rule.Condition, profile)
		if err != nil || !ok {
			continue
		}
		matched = append(matched, rule.Name)
	}
	return matched
}
