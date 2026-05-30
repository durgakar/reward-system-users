//go:build ignore

// Package main demonstrates registering a custom shopping data source.
// Copy this pattern into your own module and register it in internal/app/app.go.
package main

import (
	"context"

	"github.com/durgakar/reward-system-users/pkg/plugin"
)

// PostgresSource is a stub — implement SQL queries against your orders table.
type PostgresSource struct {
	DSN string
}

func (s *PostgresSource) Name() string { return "postgres" }

func (s *PostgresSource) ListClients(_ context.Context) ([]plugin.Client, error) {
	return nil, nil // SELECT id, email, first_name, last_name FROM customers
}

func (s *PostgresSource) GetProfile(_ context.Context, clientID string) (plugin.ClientProfile, error) {
	_ = clientID
	return plugin.ClientProfile{}, nil // aggregate order metrics for clientID
}
