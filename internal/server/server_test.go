package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/durgakar/reward-system-users/internal/app"
	"github.com/durgakar/reward-system-users/internal/config"
)

func TestHealthEndpoint(t *testing.T) {
	cfg := &config.Config{
		CampaignID:     "test",
		RulesSource:    "file",
		SegmentsSource: "file",
		ShoppingSource: "csv",
		RulesFile:      "../../config/rules.yaml",
		SegmentsFile:   "../../config/segments.yaml",
		ShoppingDataPath: "../../data/sample_clients.csv",
		RewardProvider: "ledger",
		EmailSender:    "stdout",
		TemplatesDir:   "../../templates",
	}
	application, err := app.NewWithConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}
	srv := New(application, nil)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
}
