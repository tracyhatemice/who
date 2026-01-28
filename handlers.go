package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/tracyhatemice/who/ddns"
	"github.com/tracyhatemice/who/webhook"
)

// Server holds the application dependencies.
type Server struct {
	store      *Store
	ddns       *ddns.Dispatcher
	webhook    *webhook.Dispatcher
	verbose    bool
	configPath string
	configMu   sync.Mutex // protects config file writes
	whoNames   map[string]bool
	config     *Config
}

// getClientIP extracts the client IP from the request.
// Priority: X-Forwarded-For (last) > X-Real-IP > RemoteAddr
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For (use last IP - closest to server)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			ip := strings.TrimSpace(ips[len(ips)-1])
			if netIP := net.ParseIP(ip); netIP != nil {
				return netIP.String()
			}
		}
	}

	// Check X-Real-IP
	if realIP := r.Header.Get("X-Real-Ip"); realIP != "" {
		if netIP := net.ParseIP(realIP); netIP != nil {
			return netIP.String()
		}
	}

	// Fall back to RemoteAddr
	if ip, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		if netIP := net.ParseIP(ip); netIP != nil {
			return netIP.String()
		}
	}

	return ""
}

func (s *Server) whoamiHandler(w http.ResponseWriter, r *http.Request) {
	if ip := getClientIP(r); ip != "" {
		_, _ = fmt.Fprintln(w, ip)
	}
}

func (s *Server) iamHandler(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")

	// Check for explicit IP in path, validate it
	var ip string
	if ipParam := r.PathValue("ip"); ipParam != "" {
		if netIP := net.ParseIP(ipParam); netIP != nil {
			ip = netIP.String()
		}
	}

	// Fallback to client IP from headers/RemoteAddr
	if ip == "" {
		ip = getClientIP(r)
		if ip == "" {
			http.Error(w, "valid IP required", http.StatusBadRequest)
			return
		}
	}

	// Store the mapping (thread-safe)
	changed := s.store.Set(name, ip)

	// Trigger side effects if IP changed and name is non-empty
	if changed && name != "" {
		// Write back to config file if name is in who config
		if s.whoNames[name] {
			go s.saveWhoIP(name, ip)
		}
		// Trigger DDNS update (non-blocking)
		if s.ddns != nil {
			s.ddns.TriggerUpdate(name, ip)
		}
		// Trigger webhook notification (non-blocking)
		if s.webhook != nil {
			s.webhook.TriggerWebhook(name, ip)
		}
	}

	_, _ = fmt.Fprintln(w, ip)
}

// saveWhoIP updates the IP for a who entry in the config file.
func (s *Server) saveWhoIP(name, ip string) {
	s.configMu.Lock()
	defer s.configMu.Unlock()

	for i, entry := range s.config.Who {
		if entry.IAM == name {
			s.config.Who[i].IP = ip
			break
		}
	}

	if err := SaveConfig(s.configPath, s.config); err != nil {
		log.Printf("WHO: failed to save config for %s: %v", name, err)
	} else {
		log.Printf("WHO: saved %s -> %s to config", name, ip)
	}
}

func (s *Server) whoisHandler(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")

	ip, ok := s.store.Get(name)
	if !ok {
		http.NotFound(w, r)
		return
	}
	_, _ = fmt.Fprintln(w, ip)
}

// responseCapture wraps ResponseWriter to capture the response body.
type responseCapture struct {
	http.ResponseWriter
	body []byte
}

func (rc *responseCapture) Write(b []byte) (int, error) {
	rc.body = append(rc.body, b...)
	return rc.ResponseWriter.Write(b)
}

// withLogging wraps a handler with verbose logging if enabled.
func (s *Server) withLogging(next http.HandlerFunc) http.HandlerFunc {
	if !s.verbose {
		return next
	}

	return func(w http.ResponseWriter, r *http.Request) {
		rc := &responseCapture{ResponseWriter: w}
		next(rc, r)
		clientIP := getClientIP(r)
		responseIP := strings.TrimSpace(string(rc.body))
		log.Printf("HTTP: %s - - [%s] \"%s %s %s\" - - [ClientIP:%s] [Response:%s]",
			r.RemoteAddr,
			time.Now().Format("02/Jan/2006:15:04:05 -0700"),
			r.Method, r.URL.Path, r.Proto,
			clientIP, responseIP)
	}
}
