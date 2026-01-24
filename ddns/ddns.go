package ddns

import (
	"log"
)

// Provider defines the interface for DNS providers.
type Provider interface {
	Update(domain, ip string, ttl int) error
}

// Entry represents a DDNS configuration matched to a provider.
type Entry struct {
	IAM       string
	Domain    string
	IPVersion string
	TTL       int
	Provider  Provider
}

// Config holds provider-specific configuration.
type Config struct {
	Provider  string
	Domain    string
	IPVersion string
	IAM       string
	AccessKey string
	SecretKey string
	ZoneID    string
	TTL       int
}

// Dispatcher manages DDNS entries and triggers updates.
type Dispatcher struct {
	entries map[string][]*Entry // keyed by IAM name, multiple entries per IAM
}

// NewDispatcher creates a Dispatcher from configuration.
func NewDispatcher(configs []Config) *Dispatcher {
	d := &Dispatcher{entries: make(map[string][]*Entry)}

	for _, cfg := range configs {
		if cfg.IAM == "" {
			log.Printf("DDNS: skipping entry with empty IAM")
			continue
		}

		var provider Provider
		switch cfg.Provider {
		case "route53":
			provider = NewRoute53(cfg.AccessKey, cfg.SecretKey, cfg.ZoneID)
		default:
			log.Printf("DDNS: unknown provider %q for IAM %q, skipping", cfg.Provider, cfg.IAM)
			continue
		}

		ttl := cfg.TTL
		if ttl <= 0 {
			ttl = 300 // default TTL
		}

		entry := &Entry{
			IAM:       cfg.IAM,
			Domain:    cfg.Domain,
			IPVersion: cfg.IPVersion,
			TTL:       ttl,
			Provider:  provider,
		}

		d.entries[cfg.IAM] = append(d.entries[cfg.IAM], entry)
	}
	return d
}

// TriggerUpdate checks if the name has DDNS configs and updates async.
// This is non-blocking - it spawns a goroutine for each update.
func (d *Dispatcher) TriggerUpdate(name, ip string) {
	entries, ok := d.entries[name]
	if !ok {
		return // No DDNS config for this name
	}

	for _, entry := range entries {
		// Async update - don't block the HTTP handler
		go func(e *Entry) {
			log.Printf("DDNS: updating %s -> %s for IAM %s", e.Domain, ip, name)
			if err := e.Provider.Update(e.Domain, ip, e.TTL); err != nil {
				log.Printf("DDNS: failed to update %s: %v", e.Domain, err)
			} else {
				log.Printf("DDNS: successfully updated %s -> %s", e.Domain, ip)
			}
		}(entry)
	}
}
