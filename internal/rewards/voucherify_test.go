package rewards

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/durgakar/reward-system-users/internal/config"
	"github.com/durgakar/reward-system-users/pkg/plugin"
)

func TestVoucherifyAwardPoints(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST got %s", r.Method)
		}
		if r.Header.Get("X-App-Id") != "app-id" {
			t.Fatalf("missing app id")
		}
		_ = json.NewEncoder(w).Encode(map[string]int{"points": 900})
	}))
	defer srv.Close()

	p := NewVoucherifyProvider(config.Voucherify{
		BaseURL: srv.URL, ApplicationID: "app-id", SecretKey: "secret", LoyaltyID: "loyalty-1",
	})
	res, err := p.AwardPoints(context.Background(), plugin.AwardRequest{
		ClientID: "c1", Points: 100, ReferenceID: "ref-1", CampaignID: "camp",
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.NewBalance != 900 {
		t.Fatalf("got balance %d", res.NewBalance)
	}
}

func TestVoucherifyGetBalance(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"loyalty": map[string]int{"points": 42}})
	}))
	defer srv.Close()

	p := NewVoucherifyProvider(config.Voucherify{
		BaseURL: srv.URL, ApplicationID: "app-id", SecretKey: "secret",
	})
	bal, err := p.GetBalance(context.Background(), "c1")
	if err != nil {
		t.Fatal(err)
	}
	if bal != 42 {
		t.Fatalf("got %d", bal)
	}
}
