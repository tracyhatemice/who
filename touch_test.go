package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func readTouchRecord(t *testing.T, path string) TouchRecord {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading touch file: %v", err)
	}
	var rec TouchRecord
	if err := json.Unmarshal(raw, &rec); err != nil {
		t.Fatalf("touch file is not valid JSON: %v\n%s", err, raw)
	}
	return rec
}

func TestWriteTouchFilePlainName(t *testing.T) {
	path := filepath.Join(t.TempDir(), "alice.json")
	srv := &Server{store: NewStore(), aliases: map[string][]string{}}
	srv.store.Load("alice", "203.0.113.50", 12345)

	srv.writeTouchFile(TouchEntry{IAM: "alice", Path: path})

	rec := readTouchRecord(t, path)
	want := TouchRecord{IAM: "alice", IP: []string{"203.0.113.50"}, LastUpdate: 12345}
	if !reflect.DeepEqual(rec, want) {
		t.Errorf("record = %+v, want %+v", rec, want)
	}
}

func TestWriteTouchFileUnknownNameWritesEmptyRecord(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ghost.json")
	srv := &Server{store: NewStore(), aliases: map[string][]string{}}

	srv.writeTouchFile(TouchEntry{IAM: "ghost", Path: path})

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("touch file should exist on write even without data: %v", err)
	}
	rec := readTouchRecord(t, path)
	if rec.IAM != "ghost" || len(rec.IP) != 0 || rec.LastUpdate != 0 {
		t.Errorf("record = %+v, want empty record for ghost", rec)
	}
	// ip must serialize as [], not null
	var asMap map[string]any
	if err := json.Unmarshal(raw, &asMap); err != nil {
		t.Fatal(err)
	}
	if asMap["ip"] == nil {
		t.Errorf("ip should be [], got null in:\n%s", raw)
	}
}

func TestWriteTouchFileAliasFlattensMembers(t *testing.T) {
	path := filepath.Join(t.TempDir(), "julia.json")
	srv := &Server{
		store:   NewStore(),
		aliases: map[string][]string{"julia": {"juliav4", "juliav6"}},
	}
	srv.store.Load("juliav4", "111.111.111.111", 100)
	srv.store.Load("juliav6", "2001:db8::1", 200)

	srv.writeTouchFile(TouchEntry{IAM: "julia", Path: path})

	rec := readTouchRecord(t, path)
	want := TouchRecord{
		IAM:        "julia",
		IP:         []string{"111.111.111.111", "2001:db8::1"},
		LastUpdate: 200,
	}
	if !reflect.DeepEqual(rec, want) {
		t.Errorf("record = %+v, want %+v", rec, want)
	}
}

func TestWriteTouchFileAliasSkipsMembersWithoutIP(t *testing.T) {
	path := filepath.Join(t.TempDir(), "julia.json")
	srv := &Server{
		store:   NewStore(),
		aliases: map[string][]string{"julia": {"juliav4", "juliav6"}},
	}
	srv.store.Load("juliav4", "111.111.111.111", 100)

	srv.writeTouchFile(TouchEntry{IAM: "julia", Path: path})

	rec := readTouchRecord(t, path)
	want := TouchRecord{IAM: "julia", IP: []string{"111.111.111.111"}, LastUpdate: 100}
	if !reflect.DeepEqual(rec, want) {
		t.Errorf("record = %+v, want %+v", rec, want)
	}
}

func TestWriteAllTouchFilesSkipsInvalidEntries(t *testing.T) {
	dir := t.TempDir()
	alicePath := filepath.Join(dir, "alice.json")
	srv := &Server{
		store:   NewStore(),
		aliases: map[string][]string{},
		touch: []TouchEntry{
			{IAM: "alice", Path: alicePath},
			{IAM: "", Path: filepath.Join(dir, "noname.json")},
			{IAM: "bob", Path: ""},
		},
	}
	srv.store.Load("alice", "203.0.113.50", 12345)

	srv.writeAllTouchFiles()

	if _, err := os.Stat(alicePath); err != nil {
		t.Errorf("alice touch file should be created on startup: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "noname.json")); err == nil {
		t.Error("entry without iam should be skipped")
	}
}

func TestTouchEntriesForName(t *testing.T) {
	srv := &Server{
		aliases: map[string][]string{"julia": {"juliav4", "juliav6"}},
		touch: []TouchEntry{
			{IAM: "julia", Path: "/tmp/julia.json"},
			{IAM: "juliav4", Path: "/tmp/juliav4.json"},
			{IAM: "other", Path: "/tmp/other.json"},
		},
	}

	got := srv.touchEntriesFor("juliav4")

	var paths []string
	for _, e := range got {
		paths = append(paths, e.Path)
	}
	want := []string{"/tmp/julia.json", "/tmp/juliav4.json"}
	if !reflect.DeepEqual(paths, want) {
		t.Errorf("touchEntriesFor(juliav4) paths = %v, want %v", paths, want)
	}
}
