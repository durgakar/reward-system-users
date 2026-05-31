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
// Docs: https://docs.voucherify.io/api-reference/loyalties/adjust-loyalty-card-balance
type VoucherifyProvider struct {
	cfg    config.Voucherify
	client HTTPDoer
}

func NewVoucherifyProvider(cfg config.Voucherify) *VoucherifyProvider {
	return &VoucherifyProvider{
		cfg:    cfg,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (p *VoucherifyProvider) WithClient(c HTTPDoer) *VoucherifyProvider {
	p.client = c
	return p
}

func (p *VoucherifyProvider) Name() string { return "voucherify" }

// TestConnection verifies credentials and optionally loyalty campaign access.
func (p *VoucherifyProvider) TestConnection(ctx context.Context) error {
	if err := p.validateConfig(); err != nil {
		return err
	}
	_, status, err := p.doJSON(ctx, http.MethodGet, "/vouchers?page=1&limit=1", nil)
	if err != nil {
		return err
	}
	if status >= 300 {
		return fmt.Errorf("voucherify auth check status %d", status)
	}
	return nil
}

func (p *VoucherifyProvider) AwardPoints(ctx context.Context, req plugin.AwardRequest) (*plugin.AwardResult, error) {
	if err := p.validateConfig(); err != nil {
		return nil, err
	}
	loyaltyID := p.loyaltyCampaignID(req)
	if loyaltyID == "" {
		return nil, fmt.Errorf("voucherify loyalty_id is required (set voucherify.loyalty_id or campaign_id in config)")
	}
	if err := p.ensureLoyaltyMember(ctx, loyaltyID, req); err != nil {
		return nil, fmt.Errorf("ensure loyalty member: %w", err)
	}

	payload := map[string]any{
		"points": req.Points,
		"reason": req.Reason,
	}
	if req.ReferenceID != "" {
		payload["source_id"] = req.ReferenceID
	}

	path := fmt.Sprintf("/loyalties/%s/members/%s/balance", urlPathEscape(loyaltyID), urlPathEscape(req.ClientID))
	body, status, err := p.doJSON(ctx, http.MethodPost, path, payload)
	if err != nil {
		return nil, err
	}
	if status >= 300 {
		return nil, fmt.Errorf("voucherify award status %d: %s", status, string(body))
	}

	var parsed struct {
		Points  int `json:"points"`
		Balance struct {
			Points int `json:"points"`
		} `json:"balance"`
		LoyaltyCard struct {
			Points int `json:"points"`
		} `json:"loyalty_card"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("voucherify decode: %w", err)
	}
	balance := parsed.Points
	if balance == 0 {
		balance = parsed.Balance.Points
	}
	if balance == 0 {
		balance = parsed.LoyaltyCard.Points
	}

	return &plugin.AwardResult{
		TransactionID: req.ReferenceID,
		NewBalance:    balance,
		Provider:      p.Name(),
	}, nil
}

func (p *VoucherifyProvider) GetBalance(ctx context.Context, clientID string) (int, error) {
	if err := p.validateConfig(); err != nil {
		return 0, err
	}
	loyaltyID := p.cfg.LoyaltyID
	if loyaltyID == "" {
		return 0, fmt.Errorf("voucherify loyalty_id is required for balance lookup")
	}

	path := fmt.Sprintf("/loyalties/%s/members/%s", urlPathEscape(loyaltyID), urlPathEscape(clientID))
	body, status, err := p.doJSON(ctx, http.MethodGet, path, nil)
	if err != nil {
		return 0, err
	}
	if status == http.StatusNotFound {
		return 0, nil
	}
	if status >= 300 {
		return 0, fmt.Errorf("voucherify balance status %d: %s", status, string(body))
	}

	var parsed struct {
		LoyaltyCard struct {
			Points int `json:"points"`
		} `json:"loyalty_card"`
		Points int `json:"points"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return 0, err
	}
	if parsed.LoyaltyCard.Points > 0 {
		return parsed.LoyaltyCard.Points, nil
	}
	return parsed.Points, nil
}

func (p *VoucherifyProvider) ensureLoyaltyMember(ctx context.Context, loyaltyID string, req plugin.AwardRequest) error {
	customer := map[string]any{
		"source_id": req.ClientID,
	}
	if email := req.Metadata["email"]; email != "" {
		customer["email"] = email
	}
	if name := req.Metadata["name"]; name != "" {
		customer["name"] = name
	}

	payload := map[string]any{"customer": customer}
	path := fmt.Sprintf("/loyalties/%s/members", urlPathEscape(loyaltyID))
	body, status, err := p.doJSON(ctx, http.MethodPost, path, payload)
	if err != nil {
		return err
	}
	// Member already exists — Voucherify returns 409 or similar
	if status >= 300 && status != http.StatusConflict {
		return fmt.Errorf("add member status %d: %s", status, string(body))
	}
	return nil
}

func (p *VoucherifyProvider) validateConfig() error {
	if p.cfg.ApplicationID == "" || p.cfg.SecretKey == "" {
		return fmt.Errorf("voucherify application_id and secret_key are required")
	}
	return nil
}

func (p *VoucherifyProvider) loyaltyCampaignID(req plugin.AwardRequest) string {
	if p.cfg.LoyaltyID != "" {
		return p.cfg.LoyaltyID
	}
	if req.CampaignID != "" {
		return req.CampaignID
	}
	return ""
}

func (p *VoucherifyProvider) doJSON(ctx context.Context, method, path string, payload any) ([]byte, int, error) {
	var body io.Reader
	if payload != nil {
		raw, err := json.Marshal(payload)
		if err != nil {
			return nil, 0, err
		}
		body = bytes.NewReader(raw)
	}
	endpoint := strings.TrimRight(p.cfg.BaseURL, "/") + path
	httpReq, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return nil, 0, err
	}
	if payload != nil {
		httpReq.Header.Set("Content-Type", "application/json")
	}
	p.setHeaders(httpReq)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, 0, fmt.Errorf("voucherify request: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	return respBody, resp.StatusCode, nil
}

func (p *VoucherifyProvider) setHeaders(req *http.Request) {
	req.Header.Set("X-App-Id", p.cfg.ApplicationID)
	req.Header.Set("X-App-Token", p.cfg.SecretKey)
}

func urlPathEscape(v string) string {
	return strings.ReplaceAll(v, " ", "%20")
}
