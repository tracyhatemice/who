package main

import (
	"encoding/json"
	"os"
)

// Config holds all application configuration.
type Config struct {
	DDNS []DDNSEntry `json:"ddns"`
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
