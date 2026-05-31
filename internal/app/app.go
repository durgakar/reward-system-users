package app

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"github.com/durgakar/reward-system-users/internal/campaign"
	"github.com/durgakar/reward-system-users/internal/config"
	"github.com/durgakar/reward-system-users/internal/domain"
	"github.com/durgakar/reward-system-users/internal/email"
	"github.com/durgakar/reward-system-users/internal/registry"
	"github.com/durgakar/reward-system-users/internal/rewards"
	"github.com/durgakar/reward-system-users/internal/rules"
	"github.com/durgakar/reward-system-users/internal/segment"
	"github.com/durgakar/reward-system-users/internal/shopping"
	"github.com/durgakar/reward-system-users/internal/store"
	"github.com/durgakar/reward-system-users/pkg/plugin"
)

// Application wires plugins from config into a campaign runner.
type Application struct {
	Config *config.Config
	Store  store.Store
	Runner *campaign.Runner
	reg    *registry.Registry
	mu     sync.RWMutex
}

func New(configPath string) (*Application, error) {
	cfg, err := config.Load(configPath)
	if err != nil {
		return nil, err
	}
	return NewWithConfig(cfg)
}

func NewWithConfig(cfg *config.Config) (*Application, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	app := &Application{Config: cfg, reg: registry.New()}
	if err := app.initStore(); err != nil {
		return nil, err
	}
	if err := app.initRegistry(); err != nil {
		return nil, err
	}
	if err := app.RebuildRunner(); err != nil {
		return nil, err
	}
	return app, nil
}

func (a *Application) initStore() error {
	if !a.Config.UsesDatabase() {
		return nil
	}
	if a.Config.Database.URL == "" {
		return fmt.Errorf("database.url is required when using database-backed sources")
	}
	s, err := store.Open(store.Options{Driver: a.Config.Database.Driver, URL: a.Config.Database.URL})
	if err != nil {
		return err
	}
	a.Store = s
	return nil
}

func (a *Application) initRegistry() error {
	a.reg.RegisterEmail(email.NewStdoutSender())
	a.reg.RegisterEmail(email.NewSMTPSender(a.Config.SMTP))
	a.reg.RegisterReward(rewards.NewLedgerProvider())
	a.reg.RegisterReward(rewards.NewOpenLoyaltyProvider(a.Config.OpenLoyalty))
	a.reg.RegisterReward(rewards.NewVoucherifyProvider(a.Config.Voucherify))

	if a.Store != nil {
		a.reg.RegisterReward(rewards.NewDBLedgerProvider(a.Store))
		a.reg.RegisterSource(shopping.NewDBSource(a.Store))
		a.reg.RegisterSegment(segment.NewStoreProvider(a.Store))
	}
	a.reg.RegisterSource(shopping.NewCSVSource(a.Config.ShoppingDataPath))
	return nil
}

func (a *Application) loadSegments() ([]plugin.SegmentProvider, error) {
	if a.Config.SegmentsSource == "database" && a.Store != nil {
		return []plugin.SegmentProvider{segment.NewStoreProvider(a.Store)}, nil
	}
	segDefs, err := rules.LoadSegments(a.Config.SegmentsFile)
	if err != nil {
		return nil, err
	}
	return []plugin.SegmentProvider{segment.NewStaticProvider(segDefs)}, nil
}

func (a *Application) loadRules(ctx context.Context) ([]domain.RuleDefinition, error) {
	if a.Config.RulesSource == "database" && a.Store != nil {
		return a.Store.ListRules(ctx)
	}
	return rules.LoadRules(a.Config.RulesFile)
}

func (a *Application) RebuildRunner() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.rebuildRunnerLocked()
}

func (a *Application) rebuildRunnerLocked() error {
	ctx := context.Background()
	segProviders, err := a.loadSegments()
	if err != nil {
		return err
	}
	ruleDefs, err := a.loadRules(ctx)
	if err != nil {
		return err
	}

	sourceName := a.Config.ShoppingSource
	source, err := a.reg.Source(sourceName)
	if err != nil {
		return err
	}
	emailSender, err := a.reg.Email(a.Config.EmailSender)
	if err != nil {
		return err
	}
	rewardProvider, err := a.reg.Reward(a.Config.RewardProvider)
	if err != nil {
		return err
	}

	a.Runner = &campaign.Runner{
		CampaignID:      a.Config.CampaignID,
		Source:          source,
		SegmentProvider: segProviders[0],
		RuleEngine:      rules.NewEngine(ruleDefs),
		EmailSender:     emailSender,
		RewardProvider:  rewardProvider,
		Renderer:        email.NewRenderer(a.Config.TemplatesDir),
		DryRun:          a.Config.DryRun,
		Store:           a.Store,
		Logger:          slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})),
	}
	return nil
}

func (a *Application) RunCampaign(ctx context.Context) ([]campaign.Result, error) {
	a.mu.Lock()
	if err := a.rebuildRunnerLocked(); err != nil {
		a.mu.Unlock()
		return nil, err
	}
	runner := a.Runner
	a.mu.Unlock()
	return runner.Run(ctx)
}

func (a *Application) Describe() string {
	return fmt.Sprintf(
		"campaign=%s source=%s email=%s rewards=%s rules=%s dry_run=%v db=%v",
		a.Config.CampaignID,
		a.Config.ShoppingSource,
		a.Config.EmailSender,
		a.Config.RewardProvider,
		a.Config.RulesSource,
		a.Config.DryRun,
		a.Store != nil,
	)
}

func (a *Application) Migrate(migrationsDir string) error {
	if a.Store == nil {
		return fmt.Errorf("database not configured")
	}
	ctx := context.Background()
	name := "001_init.sql"
	if a.Config.Database.Driver == "mysql" {
		name = "001_init.mysql.sql"
	}
	raw, err := os.ReadFile(filepath.Join(migrationsDir, name))
	if err != nil {
		return err
	}
	if err := a.Store.Migrate(ctx, string(raw)); err != nil {
		return err
	}
	return a.Store.SeedIfEmpty(ctx)
}

func (a *Application) Close() error {
	if a.Store != nil {
		return a.Store.Close()
	}
	return nil
}
