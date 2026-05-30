package app

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/durgakar/reward-system-users/internal/campaign"
	"github.com/durgakar/reward-system-users/internal/config"
	"github.com/durgakar/reward-system-users/internal/email"
	"github.com/durgakar/reward-system-users/internal/registry"
	"github.com/durgakar/reward-system-users/internal/rewards"
	"github.com/durgakar/reward-system-users/internal/rules"
	"github.com/durgakar/reward-system-users/internal/segment"
	"github.com/durgakar/reward-system-users/internal/shopping"
)

// Application wires plugins from config into a campaign runner.
type Application struct {
	Config *config.Config
	Runner *campaign.Runner
}

func New(configPath string) (*Application, error) {
	cfg, err := config.Load(configPath)
	if err != nil {
		return nil, err
	}

	segDefs, err := rules.LoadSegments(cfg.SegmentsFile)
	if err != nil {
		return nil, err
	}
	ruleDefs, err := rules.LoadRules(cfg.RulesFile)
	if err != nil {
		return nil, err
	}

	reg := registry.New()
	reg.RegisterSegment(segment.NewStaticProvider(segDefs))
	reg.RegisterSource(shopping.NewCSVSource(cfg.ShoppingDataPath))
	reg.RegisterReward(rewards.NewLedgerProvider())
	reg.RegisterReward(rewards.NewOpenLoyaltyProvider(cfg.OpenLoyalty))
	reg.RegisterReward(rewards.NewVoucherifyProvider(cfg.Voucherify))
	reg.RegisterEmail(email.NewStdoutSender())
	reg.RegisterEmail(email.NewSMTPSender(cfg.SMTP))

	source, err := reg.Source(cfg.ShoppingSource)
	if err != nil {
		return nil, err
	}
	emailSender, err := reg.Email(cfg.EmailSender)
	if err != nil {
		return nil, err
	}
	rewardProvider, err := reg.Reward(cfg.RewardProvider)
	if err != nil {
		return nil, err
	}
	segmentProvider, err := reg.Segment("static")
	if err != nil {
		return nil, err
	}

	runner := &campaign.Runner{
		CampaignID:      cfg.CampaignID,
		Source:          source,
		SegmentProvider: segmentProvider,
		RuleEngine:      rules.NewEngine(ruleDefs),
		EmailSender:     emailSender,
		RewardProvider:  rewardProvider,
		Renderer:        email.NewRenderer(cfg.TemplatesDir),
		DryRun:          cfg.DryRun,
		Logger:          slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})),
	}

	return &Application{Config: cfg, Runner: runner}, nil
}

func (a *Application) Describe() string {
	return fmt.Sprintf(
		"campaign=%s source=%s email=%s rewards=%s dry_run=%v",
		a.Config.CampaignID,
		a.Config.ShoppingSource,
		a.Config.EmailSender,
		a.Config.RewardProvider,
		a.Config.DryRun,
	)
}
