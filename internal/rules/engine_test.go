package rules

import (
	"testing"
	"time"

	"github.com/durgakar/reward-system-users/internal/domain"
	"github.com/durgakar/reward-system-users/pkg/plugin"
)

func TestEngineHighSpenderRule(t *testing.T) {
	engine := NewEngine([]domain.RuleDefinition{
		{
			ID:      "high_spender_bonus",
			Name:    "500 points for $100+ order",
			Segment: "high_spender",
			Enabled: true,
			Condition: domain.Condition{
				Field:    "last_order_total_usd",
				Operator: domain.OpGTE,
				Value:    100,
			},
			Actions: []domain.Action{
				{Type: domain.ActionAwardPoints, Points: 500},
				{Type: domain.ActionSendEmail, Template: "high_spender_bonus", Subject: "You earned points!"},
			},
		},
	})

	profile := plugin.ClientProfile{
		ClientID:          "c1",
		LastOrderTotalUSD: 150,
	}
	client := plugin.Client{ID: "c1", Email: "a@example.com"}

	outcomes, err := engine.Evaluate(client, profile, []string{"high_spender"})
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if len(outcomes) != 1 {
		t.Fatalf("expected 1 outcome, got %d", len(outcomes))
	}
	if outcomes[0].AwardPoints != 500 {
		t.Fatalf("expected 500 points, got %d", outcomes[0].AwardPoints)
	}
	if outcomes[0].EmailTemplate != "high_spender_bonus" {
		t.Fatalf("unexpected template %q", outcomes[0].EmailTemplate)
	}
}

func TestEngineSegmentGate(t *testing.T) {
	engine := NewEngine([]domain.RuleDefinition{
		{
			ID:      "gated",
			Name:    "gated rule",
			Segment: "vip",
			Enabled: true,
			Condition: domain.Condition{
				Field:    "last_order_total_usd",
				Operator: domain.OpGTE,
				Value:    1,
			},
			Actions: []domain.Action{{Type: domain.ActionAwardPoints, Points: 10}},
		},
	})

	outcomes, err := engine.Evaluate(
		plugin.Client{ID: "c1"},
		plugin.ClientProfile{LastOrderTotalUSD: 200},
		[]string{"regular"},
	)
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if len(outcomes) != 0 {
		t.Fatalf("expected no outcomes without vip segment")
	}
}

func TestEngineDisabledRuleSkipped(t *testing.T) {
	engine := NewEngine([]domain.RuleDefinition{
		{
			ID:      "off",
			Enabled: false,
			Condition: domain.Condition{
				Field:    "last_order_total_usd",
				Operator: domain.OpGTE,
				Value:    1,
			},
			Actions: []domain.Action{{Type: domain.ActionAwardPoints, Points: 99}},
		},
	})
	outcomes, err := engine.Evaluate(
		plugin.Client{ID: "c1"},
		plugin.ClientProfile{LastOrderTotalUSD: 999},
		nil,
	)
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if len(outcomes) != 0 {
		t.Fatalf("disabled rule should not match")
	}
}

func TestCompareTimeField(t *testing.T) {
	ok, err := compare(
		time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
		domain.OpGTE,
		"2026-01-01",
	)
	if err != nil {
		t.Fatalf("compare: %v", err)
	}
	if !ok {
		t.Fatal("expected time comparison to pass")
	}
}
