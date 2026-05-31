// Package plugin defines the extension points for reward-system-users.
// Implement these interfaces to add custom segments, data sources, email
// senders, or reward backends without changing the campaign orchestrator.
package plugin

import (
	"context"
	"time"
)

// Client is the minimal identity used across the pipeline.
type Client struct {
	ID        string   `json:"id"`
	Email     string   `json:"email"`
	FirstName string   `json:"first_name"`
	LastName  string   `json:"last_name"`
	Segment   []string `json:"segment,omitempty"` // optional pre-assigned tags from upstream CRM
}

// ClientProfile holds shopping behaviour metrics for rule evaluation.
type ClientProfile struct {
	ClientID           string
	LifetimeSpendUSD   float64
	LastOrderTotalUSD  float64
	LastOrderAt        time.Time
	OrdersLast90Days   int
	AverageOrderUSD    float64
	PreferredCategory  string
	DaysSinceLastOrder int
	Custom             map[string]any // extension field for bespoke integrations
}

// EmailMessage is passed to EmailSender implementations.
type EmailMessage struct {
	To          string
	Subject     string
	HTMLBody    string
	TextBody    string
	TemplateID  string
	Metadata    map[string]string
}

// AwardRequest is passed to RewardProvider implementations.
type AwardRequest struct {
	ClientID    string
	Points      int
	Reason      string
	RuleID      string
	CampaignID  string
	ReferenceID string // idempotency key
	Metadata    map[string]string
}

// AwardResult is returned after points are granted.
type AwardResult struct {
	TransactionID string
	NewBalance    int
	Provider      string
}

// RuleOutcome describes what a matched rule wants to happen.
type RuleOutcome struct {
	RuleID      string
	RuleName    string
	AwardPoints int
	EmailTemplate string
	EmailSubject  string
	Metadata    map[string]string
}

// SegmentProvider assigns segment IDs to a client from profile data.
type SegmentProvider interface {
	Name() string
	Evaluate(ctx context.Context, client Client, profile ClientProfile) ([]string, error)
}

// ShoppingSource loads client profiles and lists clients for campaigns.
type ShoppingSource interface {
	Name() string
	GetProfile(ctx context.Context, clientID string) (ClientProfile, error)
	ListClients(ctx context.Context) ([]Client, error)
}

// EmailSender delivers campaign emails.
type EmailSender interface {
	Name() string
	Send(ctx context.Context, msg EmailMessage) error
}

// RewardProvider grants and queries loyalty points.
type RewardProvider interface {
	Name() string
	AwardPoints(ctx context.Context, req AwardRequest) (*AwardResult, error)
	GetBalance(ctx context.Context, clientID string) (int, error)
}
