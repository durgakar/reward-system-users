package rewards

import (
	"context"
	"fmt"
	"sync"

	"github.com/durgakar/reward-system-users/pkg/plugin"
)

// LedgerProvider is a lightweight in-process points ledger.
// Good for prototyping before wiring Open Loyalty or Voucherify.
type LedgerProvider struct {
	mu       sync.RWMutex
	balances map[string]int
	seen     map[string]struct{}
}

func NewLedgerProvider() *LedgerProvider {
	return &LedgerProvider{
		balances: make(map[string]int),
		seen:     make(map[string]struct{}),
	}
}

func (p *LedgerProvider) Name() string { return "ledger" }

func (p *LedgerProvider) AwardPoints(_ context.Context, req plugin.AwardRequest) (*plugin.AwardResult, error) {
	if req.ReferenceID == "" {
		return nil, fmt.Errorf("reference id required for idempotency")
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, ok := p.seen[req.ReferenceID]; ok {
		return &plugin.AwardResult{
			TransactionID: req.ReferenceID,
			NewBalance:    p.balances[req.ClientID],
			Provider:      p.Name(),
		}, nil
	}
	p.seen[req.ReferenceID] = struct{}{}
	p.balances[req.ClientID] += req.Points
	return &plugin.AwardResult{
		TransactionID: req.ReferenceID,
		NewBalance:    p.balances[req.ClientID],
		Provider:      p.Name(),
	}, nil
}

func (p *LedgerProvider) GetBalance(_ context.Context, clientID string) (int, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.balances[clientID], nil
}
