package webhook

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"time"
)

// Entry represents a webhook configuration.
type Entry struct {
	IAM     string
	URL     string
	Method  string
	Headers map[string]string
}

// Config holds webhook configuration from main config.
type Config struct {
	IAM     string
	URL     string
	Method  string
	Headers map[string]string
}

// Dispatcher manages webhook entries and triggers notifications.
type Dispatcher struct {
	entries map[string][]*Entry // IAM → webhooks
	client  *http.Client
}

// Payload is the webhook notification payload.
type Payload struct {
	IAM       string `json:"iam"`
	IP        string `json:"ip"`
	Timestamp string `json:"timestamp"`
}

// NewDispatcher creates a Dispatcher from configuration.
func NewDispatcher(configs []Config) *Dispatcher {
	d := &Dispatcher{
		entries: make(map[string][]*Entry),
		client:  &http.Client{},
	}

	for _, cfg := range configs {
		if cfg.IAM == "" || cfg.URL == "" {
			log.Printf("WEBHOOK: skipping entry with empty IAM or URL")
			continue
		}

		method := cfg.Method
		if method == "" {
			method = "POST"
		}

		entry := &Entry{
			IAM:     cfg.IAM,
			URL:     cfg.URL,
			Method:  method,
			Headers: cfg.Headers,
		}

		d.entries[cfg.IAM] = append(d.entries[cfg.IAM], entry)
	}

	return d
}

// TriggerWebhook checks if the name has webhook configs and sends notifications.
func (d *Dispatcher) TriggerWebhook(name, ip string) {
	entries, ok := d.entries[name]
	if !ok {
		return
	}

	for _, entry := range entries {
		// Async send
		go d.send(entry, name, ip)
	}
}

// send sends a webhook notification.
func (d *Dispatcher) send(entry *Entry, name, ip string) {
	payload := Payload{
		IAM:       name,
		IP:        ip,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		log.Printf("WEBHOOK: failed to marshal payload for %s: %v", entry.URL, err)
		return
	}

	req, err := http.NewRequest(entry.Method, entry.URL, bytes.NewReader(body))
	if err != nil {
		log.Printf("WEBHOOK: failed to create request for %s: %v", entry.URL, err)
		return
	}

	// Set default content-type
	req.Header.Set("Content-Type", "application/json")

	// Apply custom headers
	for k, v := range entry.Headers {
		req.Header.Set(k, v)
	}

	// Use fixed 5-second timeout
	client := &http.Client{Timeout: 5 * time.Second}

	log.Printf("WEBHOOK: sending %s to %s for IAM %s", entry.Method, entry.URL, name)

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("WEBHOOK: failed to send to %s: %v", entry.URL, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		log.Printf("WEBHOOK: successfully sent to %s (status %d)", entry.URL, resp.StatusCode)
	} else {
		log.Printf("WEBHOOK: received non-2xx response from %s: %d", entry.URL, resp.StatusCode)
	}
}
