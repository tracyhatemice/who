package main

import (
	"fmt"
	"log"
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
	if realIP := r.Header.Get("X-Real-Ip"); realIP != "" {
		_, _ = fmt.Fprintln(w, realIP)
	}
}

func (s *Server) iamHandler(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")

	realIP := r.Header.Get("X-Real-Ip")
	if realIP == "" {
		http.Error(w, "X-Real-Ip header required", http.StatusBadRequest)
		return
	}

	// Store the mapping (thread-safe)
	changed := s.store.Set(name, realIP)

	// Trigger DDNS update if IP changed and name is non-empty (non-blocking)
	if changed && name != "" && s.ddns != nil {
		s.ddns.TriggerUpdate(name, realIP)
	}

	_, _ = fmt.Fprintln(w, realIP)
}

func (s *Server) whoisHandler(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")

	realIP, ok := s.store.Get(name)
	if !ok {
		http.NotFound(w, r)
		return
	}
	_, _ = fmt.Fprintln(w, realIP)
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
		clientRealIP := r.Header.Get("X-Real-Ip")
		responseIP := strings.TrimSpace(string(rc.body))
		log.Printf("%s - - [%s] \"%s %s %s\" - - [RequestRealIP:%s] [Response:%s]",
			r.RemoteAddr,
			time.Now().Format("02/Jan/2006:15:04:05 -0700"),
			r.Method, r.URL.Path, r.Proto,
			clientRealIP, responseIP)
	}
}
