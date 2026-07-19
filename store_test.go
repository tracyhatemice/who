package main

import (
	"testing"
	"time"
)

func TestSetStampsTimestampOnChange(t *testing.T) {
	s := NewStore()

	before := time.Now().Unix()
	changed := s.Set("alice", "203.0.113.50")
	after := time.Now().Unix()

	if !changed {
		t.Fatal("Set on new name should report changed")
	}
	ip, ts, ok := s.GetRecord("alice")
	if !ok {
		t.Fatal("GetRecord should find alice")
	}
	if ip != "203.0.113.50" {
		t.Errorf("ip = %q, want 203.0.113.50", ip)
	}
	if ts < before || ts > after {
		t.Errorf("timestamp %d not in [%d, %d]", ts, before, after)
	}
}

func TestSetKeepsTimestampWhenIPUnchanged(t *testing.T) {
	s := NewStore()
	s.Load("alice", "203.0.113.50", 12345)

	changed := s.Set("alice", "203.0.113.50")

	if changed {
		t.Error("Set with same IP should not report changed")
	}
	_, ts, _ := s.GetRecord("alice")
	if ts != 12345 {
		t.Errorf("timestamp = %d, want unchanged 12345", ts)
	}
}

func TestSetUpdatesTimestampWhenIPChanges(t *testing.T) {
	s := NewStore()
	s.Load("alice", "203.0.113.50", 12345)

	before := time.Now().Unix()
	changed := s.Set("alice", "198.51.100.7")

	if !changed {
		t.Fatal("Set with new IP should report changed")
	}
	ip, ts, _ := s.GetRecord("alice")
	if ip != "198.51.100.7" {
		t.Errorf("ip = %q, want 198.51.100.7", ip)
	}
	if ts < before {
		t.Errorf("timestamp = %d, want >= %d", ts, before)
	}
}

func TestLoadPreloadsIPAndTimestamp(t *testing.T) {
	s := NewStore()
	s.Load("alice", "203.0.113.50", 12345)

	ip, ok := s.Get("alice")
	if !ok || ip != "203.0.113.50" {
		t.Errorf("Get = %q, %v; want 203.0.113.50, true", ip, ok)
	}
	_, ts, _ := s.GetRecord("alice")
	if ts != 12345 {
		t.Errorf("timestamp = %d, want 12345", ts)
	}
}

func TestGetRecordMissingName(t *testing.T) {
	s := NewStore()
	if _, _, ok := s.GetRecord("nobody"); ok {
		t.Error("GetRecord on missing name should return ok=false")
	}
}
