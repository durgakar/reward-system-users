package campaign

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/durgakar/reward-system-users/internal/email"
	"github.com/durgakar/reward-system-users/internal/rules"
	"github.com/durgakar/reward-system-users/pkg/plugin"
)

// Result summarizes one client's campaign processing.
type Result struct {
	ClientID     string
	Segments     []string
	RuleMatches  int
	PointsAwarded int
	EmailsSent   int
	Errors       []string
}

// Runner orchestrates segment evaluation, rules, rewards, and email delivery.
type Runner struct {
	CampaignID      string
	Source          plugin.ShoppingSource
	SegmentProvider plugin.SegmentProvider
	RuleEngine      *rules.Engine
	EmailSender     plugin.EmailSender
	RewardProvider  plugin.RewardProvider
	Renderer        *email.Renderer
	DryRun          bool
	Logger          *slog.Logger
}

func (r *Runner) Run(ctx context.Context) ([]Result, error) {
	clients, err := r.Source.ListClients(ctx)
	if err != nil {
		return nil, fmt.Errorf("list clients: %w", err)
	}

	results := make([]Result, 0, len(clients))
	for _, client := range clients {
		res := r.processClient(ctx, client)
		results = append(results, res)
	}
	return results, nil
}

func (r *Runner) processClient(ctx context.Context, client plugin.Client) Result {
	res := Result{ClientID: client.ID}

	profile, err := r.Source.GetProfile(ctx, client.ID)
	if err != nil {
		res.Errors = append(res.Errors, err.Error())
		return res
	}

	segments, err := r.SegmentProvider.Evaluate(ctx, client, profile)
	if err != nil {
		res.Errors = append(res.Errors, err.Error())
		return res
	}
	res.Segments = segments

	outcomes, err := r.RuleEngine.Evaluate(client, profile, segments)
	if err != nil {
		res.Errors = append(res.Errors, err.Error())
		return res
	}
	res.RuleMatches = len(outcomes)

	for _, outcome := range outcomes {
		templateData := email.TemplateData{
			Client: email.ClientView{
				ID:        client.ID,
				Email:     client.Email,
				FirstName: client.FirstName,
				LastName:  client.LastName,
			},
			Points:   outcome.AwardPoints,
			RuleName: outcome.RuleName,
			Campaign: r.CampaignID,
			Profile: email.ProfileView{
				LifetimeSpendUSD:  profile.LifetimeSpendUSD,
				LastOrderTotalUSD: profile.LastOrderTotalUSD,
				PreferredCategory: profile.PreferredCategory,
			},
		}

		if outcome.AwardPoints > 0 {
			if r.DryRun {
				r.logInfo("dry-run award points", "client", client.ID, "points", outcome.AwardPoints, "rule", outcome.RuleID)
			} else {
				ref := fmt.Sprintf("%s:%s:%s", r.CampaignID, client.ID, outcome.RuleID)
				award, err := r.RewardProvider.AwardPoints(ctx, plugin.AwardRequest{
					ClientID:    client.ID,
					Points:      outcome.AwardPoints,
					Reason:      outcome.RuleName,
					RuleID:      outcome.RuleID,
					CampaignID:  r.CampaignID,
					ReferenceID: ref,
					Metadata: map[string]string{
						"email": client.Email,
					},
				})
				if err != nil {
					res.Errors = append(res.Errors, fmt.Sprintf("award points: %v", err))
					continue
				}
				res.PointsAwarded += outcome.AwardPoints
				templateData.Points = award.NewBalance
			}
		}

		if outcome.EmailTemplate != "" {
			subject := outcome.EmailSubject
			if subject == "" {
				subject = "An update on your rewards"
			}
			renderedSubject, err := email.RenderSubject(subject, templateData)
			if err != nil {
				res.Errors = append(res.Errors, fmt.Sprintf("render subject: %v", err))
				continue
			}
			htmlBody, textBody, err := r.Renderer.Render(outcome.EmailTemplate, templateData)
			if err != nil {
				res.Errors = append(res.Errors, fmt.Sprintf("render template: %v", err))
				continue
			}
			if r.DryRun {
				r.logInfo("dry-run email", "client", client.Email, "template", outcome.EmailTemplate, "subject", renderedSubject)
				res.EmailsSent++
				continue
			}
			if err := r.EmailSender.Send(ctx, plugin.EmailMessage{
				To:         client.Email,
				Subject:    renderedSubject,
				HTMLBody:   htmlBody,
				TextBody:   textBody,
				TemplateID: outcome.EmailTemplate,
				Metadata: map[string]string{
					"rule_id": outcome.RuleID,
				},
			}); err != nil {
				res.Errors = append(res.Errors, fmt.Sprintf("send email: %v", err))
				continue
			}
			res.EmailsSent++
		}
	}
	return res
}

func (r *Runner) logInfo(msg string, args ...any) {
	if r.Logger != nil {
		r.Logger.Info(msg, args...)
	}
}
