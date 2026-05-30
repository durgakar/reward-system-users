package rewards

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/durgakar/reward-system-users/internal/config"
	"github.com/durgakar/reward-system-users/pkg/plugin"
)

// VoucherifyProvider integrates with Voucherify's loyalty / promotion API.
// Voucherify is SaaS (not fully OSS) but has a generous dev tier and is widely
// used for rule-based promotions — a good bridge before self-hosting Open Loyalty.
//
// Docs: https://docs.voucherify.io/reference/create-publication
type VoucherifyProvider struct {
	cfg    config.Voucherify
	client *http.Client
}

func NewVoucherifyProvider(cfg config.Voucherify) *VoucherifyProvider {
	return &VoucherifyProvider{
		cfg: cfg,
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

func (p *VoucherifyProvider) Name() string { return "voucherify" }

func (p *VoucherifyProvider) AwardPoints(ctx context.Context, req plugin.AwardRequest) (*plugin.AwardResult, error) {
	if p.cfg.ApplicationID == "" || p.cfg.SecretKey == "" {
		return nil, fmt.Errorf("voucherify application_id and secret_key are required")
	}
	payload := map[string]any{
		"customer": map[string]string{
			"source_id": req.ClientID,
			"email":     req.Metadata["email"],
		},
		"points": req.Points,
		"reason": req.Reason,
	}
	body, _ := json.Marshal(payload)
	url := strings.TrimRight(p.cfg.BaseURL, "/") + "/loyalties/" + urlEscape(req.CampaignID) + "/members/" + urlEscape(req.ClientID) + "/balance"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-App-Id", p.cfg.ApplicationID)
	httpReq.Header.Set("X-App-Token", p.cfg.SecretKey)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("voucherify request: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("voucherify status %d: %s", resp.StatusCode, string(respBody))
	}

	var parsed struct {
		Points int `json:"points"`
	}
	_ = json.Unmarshal(respBody, &parsed)
	return &plugin.AwardResult{
		TransactionID: req.ReferenceID,
		NewBalance:    parsed.Points,
		Provider:      p.Name(),
	}, nil
}

func (p *VoucherifyProvider) GetBalance(ctx context.Context, clientID string) (int, error) {
	if p.cfg.ApplicationID == "" || p.cfg.SecretKey == "" {
		return 0, fmt.Errorf("voucherify credentials required")
	}
	url := strings.TrimRight(p.cfg.BaseURL, "/") + "/customers/" + urlEscape(clientID)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, err
	}
	httpReq.Header.Set("X-App-Id", p.cfg.ApplicationID)
	httpReq.Header.Set("X-App-Token", p.cfg.SecretKey)
	resp, err := p.client.Do(httpReq)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("voucherify status %d: %s", resp.StatusCode, string(body))
	}
	var parsed struct {
		Loyalty struct {
			Points int `json:"points"`
		} `json:"loyalty"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return 0, err
	}
	return parsed.Loyalty.Points, nil
}

func urlEscape(v string) string {
	return strings.ReplaceAll(v, " ", "%20")
}
