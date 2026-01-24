package main

import "sync"

// Store provides thread-safe name-to-IP storage.
type Store struct {
	mu   sync.RWMutex
	data map[string]string
}

// NewStore creates a new thread-safe store.
func NewStore() *Store {
	return &Store{data: make(map[string]string)}
}

// Set stores a name-IP mapping and returns true if the IP changed.
func (s *Store) Set(name, ip string) (changed bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	old, exists := s.data[name]
	s.data[name] = ip
	return !exists || old != ip
}

// Get retrieves an IP by name. Returns empty string and false if not found.
func (s *Store) Get(name string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ip, ok := s.data[name]
	return ip, ok
}
