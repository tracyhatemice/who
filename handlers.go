package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/tracyhatemice/who/ddns"
)

// Server holds the application dependencies.
type Server struct {
	store   *Store
	ddns    *ddns.Dispatcher
	verbose bool
}

func (s *Server) whoamiHandler(w http.ResponseWriter, r *http.Request) {
	if ip := r.Header.Get("X-Real-Ip"); ip != "" && net.ParseIP(ip) != nil {
		_, _ = fmt.Fprintln(w, ip)
	}
}

func (s *Server) iamHandler(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")

	// Check for explicit IP in path, validate it
	var ip string
	if ipParam := r.PathValue("ip"); ipParam != "" && net.ParseIP(ipParam) != nil {
		ip = ipParam
	} else {
		// Fallback to X-Real-Ip header, validate it
		ip = r.Header.Get("X-Real-Ip")
		if ip == "" || net.ParseIP(ip) == nil {
			http.Error(w, "valid X-Real-Ip header required", http.StatusBadRequest)
			return
		}
	}

	// Store the mapping (thread-safe)
	changed := s.store.Set(name, ip)

	// Trigger DDNS update if IP changed and name is non-empty (non-blocking)
	if changed && name != "" && s.ddns != nil {
		s.ddns.TriggerUpdate(name, ip)
	}

	_, _ = fmt.Fprintln(w, ip)
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
		clientIP := r.Header.Get("X-Real-Ip")
		responseIP := strings.TrimSpace(string(rc.body))
		log.Printf("HTTP: %s - - [%s] \"%s %s %s\" - - [RequestIP:%s] [Response:%s]",
			r.RemoteAddr,
			time.Now().Format("02/Jan/2006:15:04:05 -0700"),
			r.Method, r.URL.Path, r.Proto,
			clientIP, responseIP)
	}
}
