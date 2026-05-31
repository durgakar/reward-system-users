package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/durgakar/reward-system-users/internal/domain"
	"github.com/durgakar/reward-system-users/pkg/plugin"
)

// Store persists clients, segments, rules, and ledger state.
type Store interface {
	Close() error
	Ping(ctx context.Context) error
	Migrate(ctx context.Context, sql string) error

	ListClients(ctx context.Context) ([]plugin.Client, error)
	GetProfile(ctx context.Context, clientID string) (plugin.ClientProfile, error)
	UpsertClient(ctx context.Context, c plugin.Client, p plugin.ClientProfile) error

	ListSegments(ctx context.Context) ([]domain.SegmentDefinition, error)
	GetSegment(ctx context.Context, id string) (domain.SegmentDefinition, error)
	UpsertSegment(ctx context.Context, seg domain.SegmentDefinition) error
	DeleteSegment(ctx context.Context, id string) error

	ListRules(ctx context.Context) ([]domain.RuleDefinition, error)
	GetRule(ctx context.Context, id string) (domain.RuleDefinition, error)
	UpsertRule(ctx context.Context, rule domain.RuleDefinition) error
	DeleteRule(ctx context.Context, id string) error

	AwardPoints(ctx context.Context, req plugin.AwardRequest, provider string) (*plugin.AwardResult, error)
	GetBalance(ctx context.Context, clientID string) (int, error)

	StartCampaignRun(ctx context.Context, campaignID string) (int64, error)
	FinishCampaignRun(ctx context.Context, runID int64, status string, summary CampaignSummary) error
	ListCampaignRuns(ctx context.Context, limit int) ([]CampaignRun, error)

	SeedIfEmpty(ctx context.Context) error
}

type CampaignSummary struct {
	ClientsProcessed int
	PointsAwarded    int
	EmailsSent       int
	ErrorsCount      int
}

type CampaignRun struct {
	ID               int64      `json:"id"`
	CampaignID       string     `json:"campaign_id"`
	Status           string     `json:"status"`
	ClientsProcessed int        `json:"clients_processed"`
	PointsAwarded    int        `json:"points_awarded"`
	EmailsSent       int        `json:"emails_sent"`
	ErrorsCount      int        `json:"errors_count"`
	StartedAt        time.Time  `json:"started_at"`
	FinishedAt       *time.Time `json:"finished_at,omitempty"`
}

func encodeMatch(m domain.MatchAll) ([]byte, error) {
	return json.Marshal(m)
}

func decodeMatch(b []byte) (domain.MatchAll, error) {
	var m domain.MatchAll
	if err := json.Unmarshal(b, &m); err != nil {
		return domain.MatchAll{}, fmt.Errorf("decode match: %w", err)
	}
	return m, nil
}

func encodeCondition(c domain.Condition) ([]byte, error) {
	return json.Marshal(c)
}

func decodeCondition(b []byte) (domain.Condition, error) {
	var c domain.Condition
	if err := json.Unmarshal(b, &c); err != nil {
		return domain.Condition{}, fmt.Errorf("decode condition: %w", err)
	}
	return c, nil
}

func encodeActions(a []domain.Action) ([]byte, error) {
	return json.Marshal(a)
}

func decodeActions(b []byte) ([]domain.Action, error) {
	var a []domain.Action
	if err := json.Unmarshal(b, &a); err != nil {
		return nil, fmt.Errorf("decode actions: %w", err)
	}
	return a, nil
}
