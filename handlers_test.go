package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// waitFor polls cond until it returns true or the timeout expires.
func waitFor(t *testing.T, timeout time.Duration, cond func() bool) bool {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return cond()
}

func doIam(srv *Server, name, ip string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(http.MethodGet, "/iam/"+name+"/"+ip, nil)
	r.SetPathValue("name", name)
	r.SetPathValue("ip", ip)
	w := httptest.NewRecorder()
	srv.iamHandler(w, r)
	return w
}

func TestIamHandlerWritesBackLastUpdate(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	cfg := &Config{Who: []WhoEntry{{IAM: "alice", IP: "203.0.113.50"}}}
	srv := &Server{
		store:      NewStore(),
		configPath: configPath,
		whoNames:   map[string]bool{"alice": true},
		aliases:    map[string][]string{},
		config:     cfg,
	}
	srv.store.Load("alice", "203.0.113.50", 0)

	before := time.Now().Unix()
	doIam(srv, "alice", "198.51.100.7")

	ok := waitFor(t, 2*time.Second, func() bool {
		loaded, err := LoadConfig(configPath)
		if err != nil || len(loaded.Who) == 0 {
			return false
		}
		return loaded.Who[0].IP == "198.51.100.7" && loaded.Who[0].LastUpdate >= before
	})
	if !ok {
		loaded, _ := LoadConfig(configPath)
		t.Errorf("config not written back with ip and last_update, got: %+v", loaded)
	}
}

func TestIamHandlerRewritesTouchFileOnChange(t *testing.T) {
	touchPath := filepath.Join(t.TempDir(), "alice.json")
	srv := &Server{
		store:   NewStore(),
		aliases: map[string][]string{},
		touch:   []TouchEntry{{IAM: "alice", Path: touchPath}},
	}

	doIam(srv, "alice", "198.51.100.7")

	ok := waitFor(t, 2*time.Second, func() bool {
		if _, err := os.Stat(touchPath); err != nil {
			return false
		}
		rec := readTouchRecord(t, touchPath)
		return rec.IAM == "alice" && len(rec.IP) == 1 &&
			rec.IP[0] == "198.51.100.7" && rec.LastUpdate > 0
	})
	if !ok {
		t.Error("touch file not written after IP change")
	}
}

func TestIamHandlerRewritesAliasTouchFileOnMemberChange(t *testing.T) {
	touchPath := filepath.Join(t.TempDir(), "julia.json")
	srv := &Server{
		store:   NewStore(),
		aliases: map[string][]string{"julia": {"juliav4", "juliav6"}},
		touch:   []TouchEntry{{IAM: "julia", Path: touchPath}},
	}
	srv.store.Load("juliav6", "2001:db8::1", 100)

	doIam(srv, "juliav4", "198.51.100.7")

	ok := waitFor(t, 2*time.Second, func() bool {
		if _, err := os.Stat(touchPath); err != nil {
			return false
		}
		rec := readTouchRecord(t, touchPath)
		return rec.IAM == "julia" && len(rec.IP) == 2
	})
	if !ok {
		t.Error("alias touch file not written after member IP change")
	}
}

func TestIamHandlerDoesNotTouchWhenIPUnchanged(t *testing.T) {
	touchPath := filepath.Join(t.TempDir(), "alice.json")
	srv := &Server{
		store:   NewStore(),
		aliases: map[string][]string{},
		touch:   []TouchEntry{{IAM: "alice", Path: touchPath}},
	}
	srv.store.Load("alice", "198.51.100.7", 100)

	doIam(srv, "alice", "198.51.100.7")

	if waitFor(t, 200*time.Millisecond, func() bool {
		_, err := os.Stat(touchPath)
		return err == nil
	}) {
		t.Error("touch file should not be written when IP is unchanged")
	}
}
