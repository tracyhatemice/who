package main

import (
	"sync"
	"time"
)

// record holds an IP and the unix timestamp of its last change.
type record struct {
	ip         string
	lastUpdate int64
}

// Store provides thread-safe name-to-IP storage.
type Store struct {
	mu   sync.RWMutex
	data map[string]record
}

// NewStore creates a new thread-safe store.
func NewStore() *Store {
	return &Store{data: make(map[string]record)}
}

// Set stores a name-IP mapping and returns true if the IP changed.
// The last-change timestamp is updated only when the IP actually changes.
func (s *Store) Set(name, ip string) (changed bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	old, exists := s.data[name]
	if exists && old.ip == ip {
		return false
	}
	s.data[name] = record{ip: ip, lastUpdate: time.Now().Unix()}
	return true
}

// Load pre-loads a name-IP mapping with a known last-change timestamp,
// without stamping the current time. Used when restoring from config.
func (s *Store) Load(name, ip string, lastUpdate int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[name] = record{ip: ip, lastUpdate: lastUpdate}
}

// Get retrieves an IP by name. Returns empty string and false if not found.
func (s *Store) Get(name string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rec, ok := s.data[name]
	return rec.ip, ok
}

// GetRecord retrieves an IP and its last-change unix timestamp by name.
func (s *Store) GetRecord(name string) (ip string, lastUpdate int64, ok bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rec, ok := s.data[name]
	return rec.ip, rec.lastUpdate, ok
}
