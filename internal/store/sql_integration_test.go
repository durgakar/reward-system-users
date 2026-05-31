//go:build integration

package store

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/durgakar/reward-system-users/internal/domain"
	"github.com/durgakar/reward-system-users/pkg/plugin"
)

func TestSQLStoreIntegration(t *testing.T) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL not set")
	}

	ctx := context.Background()
	s, err := Open(Options{Driver: "postgres", URL: url})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	migration, err := os.ReadFile(filepath.Join("..", "..", "migrations", "001_init.sql"))
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Migrate(ctx, string(migration)); err != nil {
		t.Fatal(err)
	}

	seg := domain.SegmentDefinition{
		ID: "test_seg", Description: "test",
		Match: domain.MatchAll{All: []domain.Condition{{Field: "lifetime_spend_usd", Operator: domain.OpGTE, Value: 1}}},
	}
	if err := s.UpsertSegment(ctx, seg); err != nil {
		t.Fatal(err)
	}

	rule := domain.RuleDefinition{
		ID: "test_rule", Name: "Test", Enabled: true,
		Condition: domain.Condition{Field: "lifetime_spend_usd", Operator: domain.OpGTE, Value: 1},
		Actions:   []domain.Action{{Type: domain.ActionAwardPoints, Points: 10}},
	}
	if err := s.UpsertRule(ctx, rule); err != nil {
		t.Fatal(err)
	}

	client := plugin.Client{ID: "it-1", Email: "it@example.com", FirstName: "It"}
	profile := plugin.ClientProfile{
		ClientID: "it-1", LifetimeSpendUSD: 100, LastOrderAt: time.Now(),
	}
	if err := s.UpsertClient(ctx, client, profile); err != nil {
		t.Fatal(err)
	}

	award, err := s.AwardPoints(ctx, plugin.AwardRequest{
		ClientID: "it-1", Points: 25, ReferenceID: "it-ref-1", RuleID: "test_rule",
	}, "db_ledger")
	if err != nil {
		t.Fatal(err)
	}
	if award.NewBalance != 25 {
		t.Fatalf("balance %d", award.NewBalance)
	}

	// idempotent
	award2, err := s.AwardPoints(ctx, plugin.AwardRequest{
		ClientID: "it-1", Points: 25, ReferenceID: "it-ref-1", RuleID: "test_rule",
	}, "db_ledger")
	if err != nil {
		t.Fatal(err)
	}
	if award2.NewBalance != 25 {
		t.Fatalf("expected idempotent balance 25 got %d", award2.NewBalance)
	}
}
