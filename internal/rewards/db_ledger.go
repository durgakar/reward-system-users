package rewards

import (
	"context"

	"github.com/durgakar/reward-system-users/internal/store"
	"github.com/durgakar/reward-system-users/pkg/plugin"
)

// DBLedgerProvider persists points in PostgreSQL/MySQL via the store.
type DBLedgerProvider struct {
	store store.Store
}

func NewDBLedgerProvider(s store.Store) *DBLedgerProvider {
	return &DBLedgerProvider{store: s}
}

func (p *DBLedgerProvider) Name() string { return "db_ledger" }

func (p *DBLedgerProvider) AwardPoints(ctx context.Context, req plugin.AwardRequest) (*plugin.AwardResult, error) {
	if req.ReferenceID == "" {
		req.ReferenceID = req.CampaignID + ":" + req.ClientID + ":" + req.RuleID
	}
	return p.store.AwardPoints(ctx, req, p.Name())
}

func (p *DBLedgerProvider) GetBalance(ctx context.Context, clientID string) (int, error) {
	return p.store.GetBalance(ctx, clientID)
}
