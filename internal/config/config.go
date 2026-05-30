package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config drives runtime plugin selection and paths.
type Config struct {
	CampaignID       string `yaml:"campaign_id"`
	SegmentsFile     string `yaml:"segments_file"`
	RulesFile        string `yaml:"rules_file"`
	TemplatesDir     string `yaml:"templates_dir"`
	ShoppingSource   string `yaml:"shopping_source"`
	ShoppingDataPath string `yaml:"shopping_data_path"`
	EmailSender      string `yaml:"email_sender"`
	RewardProvider   string `yaml:"reward_provider"`
	SMTP             SMTP   `yaml:"smtp"`
	OpenLoyalty      OpenLoyalty `yaml:"open_loyalty"`
	Voucherify       Voucherify  `yaml:"voucherify"`
	DryRun           bool   `yaml:"dry_run"`
}

type SMTP struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	From     string `yaml:"from"`
}

type OpenLoyalty struct {
	BaseURL string `yaml:"base_url"`
	APIKey  string `yaml:"api_key"`
	StoreCode string `yaml:"store_code"`
}

type Voucherify struct {
	ApplicationID string `yaml:"application_id"`
	SecretKey     string `yaml:"secret_key"`
	BaseURL       string `yaml:"base_url"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	cfg.applyDefaults()
	return &cfg, nil
}

func (c *Config) applyDefaults() {
	if c.CampaignID == "" {
		c.CampaignID = "default"
	}
	if c.SegmentsFile == "" {
		c.SegmentsFile = "config/segments.yaml"
	}
	if c.RulesFile == "" {
		c.RulesFile = "config/rules.yaml"
	}
	if c.TemplatesDir == "" {
		c.TemplatesDir = "templates"
	}
	if c.ShoppingSource == "" {
		c.ShoppingSource = "csv"
	}
	if c.ShoppingDataPath == "" {
		c.ShoppingDataPath = "data/sample_clients.csv"
	}
	if c.EmailSender == "" {
		c.EmailSender = "stdout"
	}
	if c.RewardProvider == "" {
		c.RewardProvider = "ledger"
	}
	if c.SMTP.Port == 0 {
		c.SMTP.Port = 587
	}
	if c.OpenLoyalty.BaseURL == "" {
		c.OpenLoyalty.BaseURL = "http://localhost:8181/api"
	}
	if c.Voucherify.BaseURL == "" {
		c.Voucherify.BaseURL = "https://api.voucherify.io/v1"
	}
}
