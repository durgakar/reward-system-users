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

func TestOpenLoyaltyAwardPoints(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/points/add" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Fatalf("missing auth header")
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"transfer": map[string]string{"transferId": "tx-1"},
			"customer": map[string]int{"activePoints": 1500},
		})
	}))
	defer srv.Close()

	p := NewOpenLoyaltyProvider(config.OpenLoyalty{BaseURL: srv.URL, APIKey: "test-key"})
	res, err := p.AwardPoints(context.Background(), plugin.AwardRequest{
		ClientID: "c1", Points: 500, Reason: "bonus", RuleID: "r1",
		ReferenceID: "ref-1", Metadata: map[string]string{"email": "a@example.com"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.NewBalance != 1500 || res.TransactionID != "tx-1" {
		t.Fatalf("unexpected result: %+v", res)
	}
}

func TestOpenLoyaltyRequiresEmail(t *testing.T) {
	p := NewOpenLoyaltyProvider(config.OpenLoyalty{BaseURL: "http://example.com", APIKey: "k"})
	_, err := p.AwardPoints(context.Background(), plugin.AwardRequest{Points: 1})
	if err == nil {
		t.Fatal("expected error without email metadata")
	}
}
