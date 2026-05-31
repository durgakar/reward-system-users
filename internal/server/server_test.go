package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/durgakar/reward-system-users/internal/app"
	"github.com/durgakar/reward-system-users/internal/config"
)

func TestHealthEndpoint(t *testing.T) {
	cfg := fileModeConfig()
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

func TestAdminUI(t *testing.T) {
	cfg := fileModeConfig()
	application, err := app.NewWithConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}
	srv := New(application, nil)

	for _, path := range []string{"/admin/", "/admin/style.css", "/admin/app.js"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		srv.Handler().ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s: status %d body=%q", path, rec.Code, rec.Body.String())
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusTemporaryRedirect {
		t.Fatalf("/admin redirect: status %d", rec.Code)
	}
}

func fileModeConfig() *config.Config {
	return &config.Config{
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
}
