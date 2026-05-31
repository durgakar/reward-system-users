package segment

import (
	"context"

	"github.com/durgakar/reward-system-users/internal/store"
	"github.com/durgakar/reward-system-users/pkg/plugin"
)

// StoreProvider evaluates segments loaded from the database on each call.
type StoreProvider struct {
	store store.Store
}

func NewStoreProvider(s store.Store) *StoreProvider {
	return &StoreProvider{store: s}
}

func (p *StoreProvider) Name() string { return "database" }

func (p *StoreProvider) Evaluate(ctx context.Context, _ plugin.Client, profile plugin.ClientProfile) ([]string, error) {
	defs, err := p.store.ListSegments(ctx)
	if err != nil {
		return nil, err
	}
	static := NewStaticProvider(defs)
	return static.Evaluate(ctx, plugin.Client{}, profile)
}
