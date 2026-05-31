package config

import (
	"fmt"
	"os"
	"strconv"

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
	RulesSource      string `yaml:"rules_source"`    // file | database
	SegmentsSource   string `yaml:"segments_source"` // file | database
	SMTP             SMTP   `yaml:"smtp"`
	OpenLoyalty      OpenLoyalty `yaml:"open_loyalty"`
	Voucherify       Voucherify  `yaml:"voucherify"`
	Database         Database `yaml:"database"`
	Server           Server   `yaml:"server"`
	Cron             Cron     `yaml:"cron"`
	DryRun           bool     `yaml:"dry_run"`
}

type Database struct {
	Driver string `yaml:"driver"` // postgres | mysql
	URL    string `yaml:"url"`
}

type Server struct {
	Host        string `yaml:"host"`
	Port        int    `yaml:"port"`
	AdminAPIKey string `yaml:"admin_api_key"`
}

type Cron struct {
	Enabled  bool   `yaml:"enabled"`
	Schedule string `yaml:"schedule"`
}

type SMTP struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	From     string `yaml:"from"`
}

type OpenLoyalty struct {
	BaseURL   string `yaml:"base_url"`
	APIKey    string `yaml:"api_key"`
	StoreCode string `yaml:"store_code"`
}

type Voucherify struct {
	ApplicationID string `yaml:"application_id"`
	SecretKey     string `yaml:"secret_key"`
	BaseURL       string `yaml:"base_url"`
	LoyaltyID     string `yaml:"loyalty_id"`
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
	cfg.applyEnv()
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
		c.ShoppingSource = "database"
	}
	if c.ShoppingDataPath == "" {
		c.ShoppingDataPath = "data/sample_clients.csv"
	}
	if c.EmailSender == "" {
		c.EmailSender = "stdout"
	}
	if c.RewardProvider == "" {
		c.RewardProvider = "db_ledger"
	}
	if c.RulesSource == "" {
		c.RulesSource = "database"
	}
	if c.SegmentsSource == "" {
		c.SegmentsSource = "database"
	}
	if c.Database.Driver == "" {
		c.Database.Driver = "postgres"
	}
	if c.Server.Host == "" {
		c.Server.Host = "0.0.0.0"
	}
	if c.Server.Port == 0 {
		c.Server.Port = 8080
	}
	if c.Cron.Schedule == "" {
		c.Cron.Schedule = "0 9 * * *"
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

func (c *Config) applyEnv() {
	if v := os.Getenv("DATABASE_URL"); v != "" {
		c.Database.URL = v
	}
	if v := os.Getenv("DATABASE_DRIVER"); v != "" {
		c.Database.Driver = v
	}
	if v := os.Getenv("PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			c.Server.Port = p
		}
	}
	if v := os.Getenv("ADMIN_API_KEY"); v != "" {
		c.Server.AdminAPIKey = v
	}
	if v := os.Getenv("OPEN_LOYALTY_API_KEY"); v != "" {
		c.OpenLoyalty.APIKey = v
	}
	if v := os.Getenv("VOUCHERIFY_APP_ID"); v != "" {
		c.Voucherify.ApplicationID = v
	}
	if v := os.Getenv("VOUCHERIFY_SECRET_KEY"); v != "" {
		c.Voucherify.SecretKey = v
	}
	if v := os.Getenv("VOUCHERIFY_LOYALTY_ID"); v != "" {
		c.Voucherify.LoyaltyID = v
	}
	if v := os.Getenv("REWARD_PROVIDER"); v != "" {
		c.RewardProvider = v
	}
}

func (c *Config) Addr() string {
	return fmt.Sprintf("%s:%d", c.Server.Host, c.Server.Port)
}

func (c *Config) UsesDatabase() bool {
	return c.RulesSource == "database" || c.SegmentsSource == "database" ||
		c.ShoppingSource == "database" || c.RewardProvider == "db_ledger"
}

func (c *Config) Validate() error {
	if c.UsesDatabase() && c.Database.URL == "" {
		return fmt.Errorf("database.url is required for database-backed configuration")
	}
	if c.ShoppingSource == "csv" && c.ShoppingDataPath == "" {
		return fmt.Errorf("shopping_data_path is required for csv source")
	}
	if c.RulesSource == "file" && c.RulesFile == "" {
		return fmt.Errorf("rules_file is required for file rules source")
	}
	if c.SegmentsSource == "file" && c.SegmentsFile == "" {
		return fmt.Errorf("segments_file is required for file segments source")
	}
	return nil
}

func (c *Config) ValidateProduction(requireAdminKey bool) error {
	if err := c.Validate(); err != nil {
		return err
	}
	if requireAdminKey && c.Server.AdminAPIKey == "" {
		return fmt.Errorf("server.admin_api_key is required in production serve mode")
	}
	return nil
}
