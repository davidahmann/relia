package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	ListenAddr string           `yaml:"listen_addr"`
	DB         DBConfig         `yaml:"db"`
	PolicyPath string           `yaml:"policy_path"`
	SigningKey SigningKeyConfig `yaml:"signing_key"`
	Slack      SlackConfig      `yaml:"slack"`
	AWS        AWSConfig        `yaml:"aws"`
}

type DBConfig struct {
	Driver string `yaml:"driver"`
	DSN    string `yaml:"dsn"`
}

type SigningKeyConfig struct {
	KeyID           string `yaml:"key_id"`
	PrivateKeyPath  string `yaml:"private_key_path"`
	PublicKeyPath   string `yaml:"public_key_path"`
	RotateOnStartup bool   `yaml:"rotate_on_startup"`
}

type SlackConfig struct {
	Enabled         bool   `yaml:"enabled"`
	BotToken        string `yaml:"bot_token"`
	SigningSecret   string `yaml:"signing_secret"`
	ApprovalChannel string `yaml:"approval_channel"`
}

type AWSConfig struct {
	STSRegionDefault string `yaml:"sts_region_default"`
}

func Load(path string) (Config, error) {
	// #nosec G304 -- path is operator-provided config path.
	raw, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}

	expanded := os.ExpandEnv(string(raw))
	expanded = strings.ReplaceAll(expanded, "\r\n", "\n")

	var cfg Config
	if err := yaml.Unmarshal([]byte(expanded), &cfg); err != nil {
		return Config{}, err
	}
	return cfg, cfg.Validate()
}

func (c Config) Validate() error {
	if c.ListenAddr == "" {
		return fmt.Errorf("listen_addr is required")
	}
	if c.PolicyPath == "" {
		return fmt.Errorf("policy_path is required")
	}

	if c.Slack.Enabled && c.Slack.SigningSecret == "" {
		return fmt.Errorf("slack.signing_secret is required when slack.enabled=true")
	}

	if c.DB.Driver != "" && c.DB.DSN == "" {
		return fmt.Errorf("db.dsn is required when db.driver is set")
	}

	return nil
}
