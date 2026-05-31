package rewards

import (
	"bytes"
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

// VoucherifyProvider integrates with Voucherify's loyalty API.
type VoucherifyProvider struct {
	cfg    config.Voucherify
	client HTTPDoer
}

func NewVoucherifyProvider(cfg config.Voucherify) *VoucherifyProvider {
	return &VoucherifyProvider{
		cfg:    cfg,
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

func (p *VoucherifyProvider) WithClient(c HTTPDoer) *VoucherifyProvider {
	p.client = c
	return p
}

func (p *VoucherifyProvider) Name() string { return "voucherify" }

func (p *VoucherifyProvider) AwardPoints(ctx context.Context, req plugin.AwardRequest) (*plugin.AwardResult, error) {
	if p.cfg.ApplicationID == "" || p.cfg.SecretKey == "" {
		return nil, fmt.Errorf("voucherify application_id and secret_key are required")
	}
	loyaltyID := p.cfg.LoyaltyID
	if loyaltyID == "" {
		loyaltyID = req.CampaignID
	}
	if loyaltyID == "" {
		return nil, fmt.Errorf("voucherify loyalty_id or campaign_id is required")
	}

	payload := map[string]any{"points": req.Points}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	endpoint := fmt.Sprintf("%s/loyalties/%s/members/%s/balance",
		strings.TrimRight(p.cfg.BaseURL, "/"), urlPathEscape(loyaltyID), urlPathEscape(req.ClientID))
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	p.setHeaders(httpReq)
	httpReq.Header.Set("Content-Type", "application/json")
	if req.ReferenceID != "" {
		httpReq.Header.Set("X-Idempotency-Key", req.ReferenceID)
	}

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("voucherify request: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("voucherify status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var parsed struct {
		Points int `json:"points"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, fmt.Errorf("voucherify decode: %w", err)
	}
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
	endpoint := strings.TrimRight(p.cfg.BaseURL, "/") + "/customers/" + urlPathEscape(clientID)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return 0, err
	}
	p.setHeaders(httpReq)
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

func (p *VoucherifyProvider) setHeaders(req *http.Request) {
	req.Header.Set("X-App-Id", p.cfg.ApplicationID)
	req.Header.Set("X-App-Token", p.cfg.SecretKey)
}

func urlPathEscape(v string) string {
	return strings.ReplaceAll(v, " ", "%20")
}
