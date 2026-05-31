package shopping

import (
	"context"

	"github.com/durgakar/reward-system-users/internal/store"
	"github.com/durgakar/reward-system-users/pkg/plugin"
)

// DBSource loads clients and profiles from the SQL store.
type DBSource struct {
	store store.Store
}

func NewDBSource(s store.Store) *DBSource {
	return &DBSource{store: s}
}

func (s *DBSource) Name() string { return "database" }

func (s *DBSource) ListClients(ctx context.Context) ([]plugin.Client, error) {
	return s.store.ListClients(ctx)
}

func (s *DBSource) GetProfile(ctx context.Context, clientID string) (plugin.ClientProfile, error) {
	return s.store.GetProfile(ctx, clientID)
}
