package main

import (
	"encoding/json"
	"os"
)

// Config holds all application configuration.
type Config struct {
	Who      []WhoEntry      `json:"who"`
	DDNS     []DDNSEntry     `json:"ddns"`
	Webhooks []WebhookEntry  `json:"webhooks"`
}

// WhoEntry represents a pre-loaded name-to-IP mapping or alias.
type WhoEntry struct {
	IAM   string   `json:"iam"`
	IP    string   `json:"ip,omitempty"`
	Alias []string `json:"alias,omitempty"`
}

// DDNSEntry represents a single DDNS configuration.
type DDNSEntry struct {
	Provider  string `json:"provider"`
	Domain    string `json:"domain"`
	IPVersion string `json:"ip_version"`
	IAM       string `json:"iam"`
	AccessKey string `json:"access_key"`
	SecretKey string `json:"secret_key"`
	ZoneID    string `json:"zone_id"`
	TTL       int    `json:"ttl"`
}

// WebhookEntry represents a webhook notification configuration.
type WebhookEntry struct {
	IAM     string            `json:"iam"`
	URL     string            `json:"url"`
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers"`
}

// LoadConfig reads configuration from a JSON file.
// Returns an empty config (not an error) if file doesn't exist or path is empty.
func LoadConfig(path string) (*Config, error) {
	if path == "" {
		return &Config{}, nil
	}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &Config{}, nil
	}
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// SaveConfig writes configuration back to a JSON file.
func SaveConfig(path string, cfg *Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o600)
}
