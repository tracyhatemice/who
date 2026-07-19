package main

import (
	"encoding/json"
	"log"
	"os"
)

// TouchRecord is the JSON document written to a touch file.
// The same flattened shape is used for plain names and aliases:
// ip always holds an array (one element for plain names, one per
// resolved alias member), and last_update is the latest change
// among the included names, as a unix timestamp in seconds.
type TouchRecord struct {
	IAM        string   `json:"iam"`
	IP         []string `json:"ip"`
	LastUpdate int64    `json:"last_update"`
}

// buildTouchRecord assembles the current record for a name,
// resolving alias members from the store.
func (s *Server) buildTouchRecord(name string) TouchRecord {
	rec := TouchRecord{IAM: name, IP: []string{}}

	names := []string{name}
	if members, isAlias := s.aliases[name]; isAlias {
		names = members
	}
	for _, n := range names {
		ip, lastUpdate, ok := s.store.GetRecord(n)
		if !ok || ip == "" {
			continue
		}
		rec.IP = append(rec.IP, ip)
		if lastUpdate > rec.LastUpdate {
			rec.LastUpdate = lastUpdate
		}
	}
	return rec
}

// writeTouchFile writes the current record for a touch entry to its path.
func (s *Server) writeTouchFile(entry TouchEntry) {
	rec := s.buildTouchRecord(entry.IAM)
	data, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		log.Printf("TOUCH: failed to marshal record for %s: %v", entry.IAM, err)
		return
	}

	s.touchMu.Lock()
	defer s.touchMu.Unlock()
	if err := os.WriteFile(entry.Path, append(data, '\n'), 0o644); err != nil {
		log.Printf("TOUCH: failed to write %s for %s: %v", entry.Path, entry.IAM, err)
		return
	}
	log.Printf("TOUCH: wrote %s for %s", entry.Path, entry.IAM)
}

// writeAllTouchFiles writes every configured touch file, skipping
// entries that lack an iam or path. Called on startup.
func (s *Server) writeAllTouchFiles() {
	for _, entry := range s.touch {
		if entry.IAM == "" || entry.Path == "" {
			log.Printf("TOUCH: skipping entry with missing iam or path: %+v", entry)
			continue
		}
		s.writeTouchFile(entry)
	}
}

// touchEntriesFor returns the touch entries affected by a change to name:
// entries for the name itself and for any alias that includes it.
func (s *Server) touchEntriesFor(name string) []TouchEntry {
	var entries []TouchEntry
	for _, entry := range s.touch {
		if entry.IAM == "" || entry.Path == "" {
			continue
		}
		if entry.IAM == name {
			entries = append(entries, entry)
			continue
		}
		for _, member := range s.aliases[entry.IAM] {
			if member == name {
				entries = append(entries, entry)
				break
			}
		}
	}
	return entries
}
