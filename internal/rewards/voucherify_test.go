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
	var addMemberCalled bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/loyalties/loyalty-1/members":
			addMemberCalled = true
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]string{"id": "v_123", "code": "LOYAL1"})
		case r.Method == http.MethodPost && r.URL.Path == "/loyalties/loyalty-1/members/c1/balance":
			_ = json.NewEncoder(w).Encode(map[string]any{"points": 900, "loyalty_card": map[string]int{"points": 900}})
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	defer srv.Close()

	p := NewVoucherifyProvider(config.Voucherify{
		BaseURL: srv.URL, ApplicationID: "app-id", SecretKey: "secret", LoyaltyID: "loyalty-1",
	})
	res, err := p.AwardPoints(context.Background(), plugin.AwardRequest{
		ClientID: "c1", Points: 100, ReferenceID: "ref-1", CampaignID: "camp",
		Reason: "bonus", Metadata: map[string]string{"email": "a@example.com"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !addMemberCalled {
		t.Fatal("expected add member call")
	}
	if res.NewBalance != 900 {
		t.Fatalf("got balance %d", res.NewBalance)
	}
}

func TestVoucherifyGetBalance(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"loyalty_card": map[string]int{"points": 42}})
	}))
	defer srv.Close()

	p := NewVoucherifyProvider(config.Voucherify{
		BaseURL: srv.URL, ApplicationID: "app-id", SecretKey: "secret", LoyaltyID: "loyalty-1",
	})
	bal, err := p.GetBalance(context.Background(), "c1")
	if err != nil {
		t.Fatal(err)
	}
	if bal != 42 {
		t.Fatalf("got %d", bal)
	}
}

func TestVoucherifyTestConnection(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-App-Id") != "app-id" {
			t.Fatal("missing app id")
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	defer srv.Close()

	p := NewVoucherifyProvider(config.Voucherify{BaseURL: srv.URL, ApplicationID: "app-id", SecretKey: "secret"})
	if err := p.TestConnection(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestVoucherifyRequiresCredentials(t *testing.T) {
	p := NewVoucherifyProvider(config.Voucherify{})
	_, err := p.AwardPoints(context.Background(), plugin.AwardRequest{Points: 1})
	if err == nil {
		t.Fatal("expected error")
	}
}
