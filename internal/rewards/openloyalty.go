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

// OpenLoyaltyProvider integrates with the open-source Open Loyalty platform.
// Docs: https://github.com/OpenLoyalty/Open-Loyalty
//
// Self-host Open Loyalty, then set open_loyalty.base_url and api_key in config.
type OpenLoyaltyProvider struct {
	cfg    config.OpenLoyalty
	client *http.Client
}

func NewOpenLoyaltyProvider(cfg config.OpenLoyalty) *OpenLoyaltyProvider {
	return &OpenLoyaltyProvider{
		cfg: cfg,
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

func (p *OpenLoyaltyProvider) Name() string { return "open_loyalty" }

func (p *OpenLoyaltyProvider) AwardPoints(ctx context.Context, req plugin.AwardRequest) (*plugin.AwardResult, error) {
	if p.cfg.APIKey == "" {
		return nil, fmt.Errorf("open_loyalty.api_key is required")
	}
	payload := map[string]any{
		"customer": map[string]string{"email": req.Metadata["email"]},
		"transfer": map[string]any{
			"points": req.Points,
			"comment": fmt.Sprintf("%s (%s)", req.Reason, req.RuleID),
		},
	}
	body, _ := json.Marshal(payload)
	url := strings.TrimRight(p.cfg.BaseURL, "/") + "/points/add"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.cfg.APIKey)
	if p.cfg.StoreCode != "" {
		httpReq.Header.Set("X-Store-Code", p.cfg.StoreCode)
	}

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("open loyalty request: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("open loyalty status %d: %s", resp.StatusCode, string(respBody))
	}

	var parsed struct {
		Transfer struct {
			ID string `json:"transferId"`
		} `json:"transfer"`
		Customer struct {
			Points int `json:"activePoints"`
		} `json:"customer"`
	}
	_ = json.Unmarshal(respBody, &parsed)
	return &plugin.AwardResult{
		TransactionID: firstNonEmpty(parsed.Transfer.ID, req.ReferenceID),
		NewBalance:    parsed.Customer.Points,
		Provider:      p.Name(),
	}, nil
}

func (p *OpenLoyaltyProvider) GetBalance(ctx context.Context, clientID string) (int, error) {
	_ = ctx
	_ = clientID
	return 0, fmt.Errorf("open_loyalty GetBalance: implement via GET /customer?email= lookup")
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
