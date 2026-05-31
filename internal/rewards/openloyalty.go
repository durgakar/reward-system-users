package rewards

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/durgakar/reward-system-users/internal/config"
	"github.com/durgakar/reward-system-users/pkg/plugin"
)

type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// OpenLoyaltyProvider integrates with the open-source Open Loyalty platform.
type OpenLoyaltyProvider struct {
	cfg    config.OpenLoyalty
	client HTTPDoer
}

func NewOpenLoyaltyProvider(cfg config.OpenLoyalty) *OpenLoyaltyProvider {
	return &OpenLoyaltyProvider{
		cfg:    cfg,
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

func (p *OpenLoyaltyProvider) WithClient(c HTTPDoer) *OpenLoyaltyProvider {
	p.client = c
	return p
}

func (p *OpenLoyaltyProvider) Name() string { return "open_loyalty" }

func (p *OpenLoyaltyProvider) AwardPoints(ctx context.Context, req plugin.AwardRequest) (*plugin.AwardResult, error) {
	if p.cfg.APIKey == "" {
		return nil, fmt.Errorf("open_loyalty.api_key is required")
	}
	email := req.Metadata["email"]
	if email == "" {
		return nil, fmt.Errorf("open_loyalty requires client email in metadata")
	}
	payload := map[string]any{
		"customer": map[string]string{"email": email},
		"transfer": map[string]any{
			"points":  req.Points,
			"comment": fmt.Sprintf("%s (%s)", req.Reason, req.RuleID),
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	endpoint := strings.TrimRight(p.cfg.BaseURL, "/") + "/points/add"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.cfg.APIKey)
	if p.cfg.StoreCode != "" {
		httpReq.Header.Set("X-Store-Code", p.cfg.StoreCode)
	}
	if req.ReferenceID != "" {
		httpReq.Header.Set("X-Idempotency-Key", req.ReferenceID)
	}

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("open loyalty request: %w", err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("open loyalty status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var parsed struct {
		Transfer struct {
			ID string `json:"transferId"`
		} `json:"transfer"`
		Customer struct {
			Points int `json:"activePoints"`
		} `json:"customer"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, fmt.Errorf("open loyalty decode: %w", err)
	}
	return &plugin.AwardResult{
		TransactionID: firstNonEmpty(parsed.Transfer.ID, req.ReferenceID),
		NewBalance:    parsed.Customer.Points,
		Provider:      p.Name(),
	}, nil
}

func (p *OpenLoyaltyProvider) GetBalance(ctx context.Context, clientID string) (int, error) {
	if p.cfg.APIKey == "" {
		return 0, fmt.Errorf("open_loyalty.api_key is required")
	}
	q := url.Values{}
	q.Set("identifier", clientID)
	endpoint := strings.TrimRight(p.cfg.BaseURL, "/") + "/customer?" + q.Encode()
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return 0, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+p.cfg.APIKey)
	resp, err := p.client.Do(httpReq)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("open loyalty status %d: %s", resp.StatusCode, string(body))
	}
	var parsed struct {
		ActivePoints int `json:"activePoints"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return 0, err
	}
	return parsed.ActivePoints, nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
