package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfigParsesTouchAndLastUpdate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	content := `{
		"who": [
			{"iam": "alice", "ip": "203.0.113.50", "last_update": 12345},
			{"iam": "bob"}
		],
		"touch": [
			{"iam": "alice", "path": "/tmp/alice.json"}
		]
	}`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.Who[0].LastUpdate != 12345 {
		t.Errorf("Who[0].LastUpdate = %d, want 12345", cfg.Who[0].LastUpdate)
	}
	if len(cfg.Touch) != 1 {
		t.Fatalf("len(Touch) = %d, want 1", len(cfg.Touch))
	}
	if cfg.Touch[0].IAM != "alice" || cfg.Touch[0].Path != "/tmp/alice.json" {
		t.Errorf("Touch[0] = %+v, want iam=alice path=/tmp/alice.json", cfg.Touch[0])
	}
}

func TestSaveConfigRoundTripsLastUpdate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	cfg := &Config{
		Who: []WhoEntry{
			{IAM: "alice", IP: "203.0.113.50", LastUpdate: 12345},
			{IAM: "bob"},
		},
	}
	if err := SaveConfig(path, cfg); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}

	loaded, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if loaded.Who[0].LastUpdate != 12345 {
		t.Errorf("Who[0].LastUpdate = %d, want 12345", loaded.Who[0].LastUpdate)
	}

	// Entries that never updated must not gain a last_update key.
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(raw), "last_update") != 1 {
		t.Errorf("config should contain exactly one last_update key, got:\n%s", raw)
	}
}
